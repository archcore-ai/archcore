---
title: "Sync Design: One-Way Push to Cloud/On-Prem Server with GraphRAG Indexing"
status: accepted
tags:
  - "sync"
---

## Idea

Push `.archcore/` documents one way, from local to a cloud or on-prem server, and let the server
index them for GraphRAG-based search across one project and across many.

`.archcore/` is the source of truth at the project level, so sync is a push. The server is a
read-only consumer: it indexes, builds the graph, and exposes cross-project search through MCP.
Nothing is pulled back and nothing is merged, because the local files are authoritative.

```
┌─────────────────┐         push          ┌──────────────────────┐
│  .archcore/     │ ──────────────────────▶│  Sync Server         │
│  (source of     │   incremental diff     │  (cloud / on-prem)   │
│   truth)        │                        │                      │
│  any dirs/      │                        │  ┌────────────────┐  │
│  any nesting    │                        │  │ Document Index  │  │
│                 │                        │  │ GraphRAG        │  │
│                 │         MCP search     │  │ Cross-project   │  │
│  Claude Code ◀──│──────────────────────  │  │ Search          │  │
│  (MCP client)   │   enhanced retrieval   │  └────────────────┘  │
└─────────────────┘                        └──────────────────────┘
```

## Status

The core sync path is implemented and tested; the related implementation plan tracks what shipped
and where it differs from the original design. The `sync` command is currently gated: it is hidden
and returns "sync is not available yet — this feature is coming soon".

## Value

- One authoritative copy removes merge conflicts, divergence states, and any question about which
  version wins.
- The server adds retrieval that a local scan cannot provide: a document graph and search across
  every synced project.
- An incremental diff keeps the transfer proportional to the change set rather than to the corpus size.
- Authoring stays offline-capable and git-native; sync is an enhancement, not a dependency.

## Possible Implementation

### Incremental sync through a manifest

File: `.archcore/.sync-state.json`, gitignored. It stores `version`, currently `1`, and a flat
`files` map of relative path to SHA-256 hex digest. It carries no per-file timestamp and no server
metadata. `@internal/sync/manifest.go` defines the structure and its validation.

Validation: at most 10,000 files; each hash a valid SHA-256 digest of 64 hex characters; each path
relative with no `..` segment and no absolute form; no `null` values.

Diff algorithm:

1. Walk `.archcore/` recursively and hash every `.md` file that follows the `slug.type.md` convention.
2. Compare against the manifest and produce four sets: created, modified, deleted, unchanged.
3. Push only the delta: created, modified, and deleted.
4. Update the manifest after a successful response, and only for the files the server confirmed.

The manifest is written atomically through a temporary file and a rename, so a crash mid-write
cannot corrupt it.

### API protocol

The wire contract is normative for both sides, so it is shown in full.

```
POST /api/v1/sync
Authorization: Bearer <token>
Content-Type: application/json

{
  "project_id": 42,
  "project_name": "my-project",
  "created": [
    {
      "path": "use-postgres.adr.md",
      "sha256": "a1b2c3...",
      "frontmatter": { "title": "Use PostgreSQL", "status": "accepted" },
      "content": "full markdown body..."
    }
  ],
  "modified": [ { "path": "mvp-launch.plan.md", "sha256": "d4e5f6...", "frontmatter": {...}, "content": "..." } ],
  "deleted": [ "old-workflow.task-type.md" ]
}
```

Differences from the first sketch of this design:

- The endpoint is `POST /api/v1/sync`; the project id travels in the body, not in the URL.
- `project_id` is optional. WHEN it is omitted and `project_name` is present, the server auto-creates the project.
- Frontmatter `title` and `status` are parsed from the YAML block and sent as structured data.
- `status` is validated against `draft`, `accepted`, and `rejected`.

Response:

```json
{
  "project_id": 42,
  "accepted": [ { "path": "use-postgres.adr.md", "action": "created" } ],
  "deleted": ["old-workflow.task-type.md"],
  "errors": [ { "path": "bad.md", "message": "invalid frontmatter" } ]
}
```

Status codes: `200` for a full success; `201` when the project was auto-created, with `project_id`
returned and saved to the local `settings.json`; `207` for a partial success where some files were
accepted and some returned errors.

### CLI commands

```bash
archcore sync                    # interactive: diff summary, then confirmation
archcore sync --ci               # non-interactive, for CI
archcore sync --dry-run          # show what would be synced, push nothing
archcore sync --force            # full re-sync, ignore the manifest
```

Run flow:

1. Validate the preconditions: `.archcore/` exists, settings are valid, the sync mode is not `none`.
2. Load the manifest and scan the files with SHA-256 hashing.
3. Calculate the diff, or mark everything modified under `--force`.
4. Print the diff summary with counts and paths.
5. Exit early under `--dry-run`.
6. Confirm through `huh.NewConfirm`, skipped under `--ci`.
7. Build the payload with file contents and parsed frontmatter.
8. Send `POST /api/v1/sync`.
9. Handle the response: update the manifest for confirmed files and report errors.
10. WHEN the server auto-created the project (`201`), persist `project_id` to the settings.

### Authentication

The token travels through the `ARCHCORE_TOKEN` environment variable:

```bash
export ARCHCORE_TOKEN=arc_xxxxx
ARCHCORE_TOKEN=${{ secrets.ARCHCORE_TOKEN }} archcore sync --ci   # CI/CD
```

`Client` in `@internal/api/client.go` sets the `Authorization: Bearer <token>` header from its
`Token` field, and only while the token is non-empty.

### Settings integration

| Sync mode  | `project_id` | `archcore_url` | Server URL                |
| ---------- | ------------ | -------------- | ------------------------- |
| `none`     | forbidden    | forbidden      | not applicable — sync off |
| `cloud`    | optional     | forbidden      | `https://app.archcore.ai` |
| `on-prem`  | optional     | required       | value of `archcore_url`   |

WHILE `project_id` is unset, the server auto-creates a project from `project_name`, derived from
the directory name, and the CLI persists the returned id.

### Package structure

`internal/sync` is imported as `archsync`, which avoids the collision with the standard library `sync`.

- `@internal/sync/manifest.go` — `Manifest` struct, load and save, validation
- `@internal/sync/hash.go` — SHA-256 hashing, file scanning
- `@internal/sync/diff.go` — change detection: created, modified, deleted, unchanged
- `@internal/sync/payload.go` — payload construction, frontmatter parsing
- `@internal/api/client.go` — sync client constructor, `applyAuth`, `Sync()`
- `@cmd/sync.go` — cobra command, preconditions, flow orchestration

### Server to local: read-only enrichment through MCP

The server never writes into `.archcore/`. It enriches search instead:

- The local MCP server (`archcore mcp`) searches local `.archcore/` files only.
- The server MCP layer searches indexed documents across every synced project through GraphRAG.
- Claude Code can use both at once: local for this project, remote for cross-project knowledge.

## Risks and Constraints

- Path traversal: `validateRelPath()` rejects `..` segments and absolute paths in the payload.
- Manifest integrity: validation rejects malformed hashes, `null` values, and excessive file counts.
- Auth header: applied only while the token is non-empty.
- Response size: at most 10 MB of body, and 512 bytes of error context.
- Read-only globals are never pushed as local documents; `ScanFiles` skips the reserved `global/`
  tree and every declared global source.

## Not implemented

| Feature                   | Status   | Notes                                             |
| ------------------------- | -------- | ------------------------------------------------- |
| `archcore login`/`logout` | planned  | The token travels through `ARCHCORE_TOKEN` today  |
| Keychain token storage    | planned  | For local development convenience                 |
| Sync triggers (hooks)     | planned  | A `sync_trigger` setting and a git post-commit hook |
| Chunked sync              | deferred | For a very large repository, above 10K files      |
| Cross-project MCP search  | planned  | A remote MCP endpoint over the server-side index  |
