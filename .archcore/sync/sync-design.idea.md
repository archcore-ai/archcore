---
title: "Sync Design: One-Way Push to Cloud/On-Prem Server with GraphRAG Indexing"
status: accepted
tags:
  - "sync"
---

## Context

`.archcore/` is the source of truth at the project level. When a cloud or on-prem sync server is configured, we need a mechanism to push documents to it for indexing, GraphRAG-based search, and cross-project knowledge retrieval.

The sync server indexes documents and builds a graph for enhanced MCP search — both within a single project and across multiple projects.

## Core Model: One-Way Push (Local → Server)

Since `.archcore/` is the **source of truth**, sync is fundamentally a **push** operation. The server is a **read-only consumer** that indexes, builds GraphRAG, and exposes cross-project search via MCP.

```
┌─────────────────┐         push          ┌──────────────────────┐
│  .archcore/     │ ──────────────────────▶│  Sync Server         │
│  (source of     │   incremental diff     │  (cloud / on-prem)   │
│   truth)        │                        │                      │
│                 │                        │  ┌────────────────┐  │
│  vision/        │                        │  │ Document Index  │  │
│  knowledge/     │                        │  │ GraphRAG        │  │
│  experience/    │                        │  │ Cross-project   │  │
│                 │         MCP search     │  │ Search          │  │
│  Claude Code ◀──│──────────────────────  │  └────────────────┘  │
│  (MCP client)   │   enhanced retrieval   │                      │
└─────────────────┘                        └──────────────────────┘
```

No pull/merge needed — local files are authoritative.

## Implementation Status: DONE

Core sync functionality is fully implemented and tested. See [Implementation Plan](./sync-command-implementation.plan.md) for details.

## Incremental Sync via Manifest

Track state with a local manifest:

**File:** `.archcore/.sync-state.json` (gitignored)

```json
{
  "version": 1,
  "files": {
    "knowledge/use-postgres.adr.md": "a1b2c3d4e5f6...",
    "vision/mvp-launch.plan.md": "d4e5f6a1b2c3..."
  }
}
```

Manifest stores only `version` (currently 1) and a flat `files` map of relative paths → SHA-256 hex digests. No per-file timestamps or server metadata — keeping it minimal.

**Validation:** max 10,000 files, valid SHA-256 hashes (64 hex chars), valid category paths, no null values.

**Diff algorithm:**

1. Walk `vision/`, `knowledge/`, `experience/` — SHA-256 hash each `.md` file (flat scan, no subdirs)
2. Compare against manifest → produce 4 sets: **created**, **modified**, **deleted**, **unchanged**
3. Push only the delta to the server (created + modified + deleted)
4. Update manifest on success — only for files confirmed by server response

**Atomic writes:** manifest saved via temp file + rename to prevent corruption on crash.

## API Protocol

```
POST /api/v1/sync
Authorization: Bearer <token>
Content-Type: application/json

{
  "project_id": 42,
  "project_name": "my-project",
  "created": [
    {
      "path": "knowledge/use-postgres.adr.md",
      "sha256": "a1b2c3...",
      "frontmatter": { "title": "Use PostgreSQL", "status": "accepted" },
      "content": "full markdown body..."
    }
  ],
  "modified": [
    {
      "path": "vision/mvp-launch.plan.md",
      "sha256": "d4e5f6...",
      "frontmatter": { "title": "MVP Launch Plan", "status": "draft" },
      "content": "full markdown body..."
    }
  ],
  "deleted": [
    "experience/old-workflow.task-type.md"
  ]
}
```

Key differences from initial design:
- Endpoint is `POST /api/v1/sync` (project ID in body, not URL)
- `project_id` is optional — if omitted with `project_name`, server auto-creates the project
- Frontmatter (title + status) parsed from YAML and sent as structured data
- Status field validated against allowed values (`draft`, `accepted`, `rejected`)

**Response:**

```json
{
  "project_id": 42,
  "accepted": [
    { "path": "knowledge/use-postgres.adr.md", "action": "created" }
  ],
  "deleted": ["experience/old-workflow.task-type.md"],
  "errors": [
    { "path": "vision/bad.md", "message": "invalid frontmatter" }
  ]
}
```

**Status codes:**
- `200` — all files synced successfully
- `201` — project auto-created, `project_id` returned and saved to local `settings.json`
- `207` — partial success (some files accepted, some errored)

## CLI Commands

```bash
# Push local changes to server
archcore sync                    # interactive, shows diff summary, confirms
archcore sync --ci               # non-interactive, CI mode
archcore sync --dry-run          # show what would be synced, don't push
archcore sync --force            # full re-sync, ignore manifest
```

### Sync Flow

1. Validate preconditions (`.archcore/` exists, settings valid, sync not `none`, token present)
2. Load manifest + scan files (SHA-256 hashing)
3. Calculate diff (or mark all as modified if `--force`)
4. Print diff summary (created/modified/deleted counts with paths)
5. Exit early if `--dry-run`
6. Interactive confirmation via `huh.NewConfirm` (skip if `--ci`)
7. Build payload with file contents and parsed frontmatter
8. Send via `POST /api/v1/sync`
9. Handle response: update manifest for confirmed files, report errors
10. If project auto-created (201), persist `project_id` to settings

## Authentication

```bash
# Token via env var
export ARCHCORE_TOKEN=arc_xxxxx

# CI/CD — secret in pipeline
ARCHCORE_TOKEN=${{ secrets.ARCHCORE_TOKEN }} archcore sync --ci
```

Token is passed via `ARCHCORE_TOKEN` env var. The `api.NewAuthenticatedClient()` sets `Authorization: Bearer <token>` header on all requests.

**Not yet implemented:** `archcore login`/`logout` commands, keychain storage, browser OAuth flow.

## Settings Integration

Sync mode in `settings.json` drives validation:

| Sync mode  | `project_id` | `archcore_url` | Server URL                    |
| ---------- | ------------ | -------------- | ----------------------------- |
| `none`     | forbidden    | forbidden      | N/A — sync disabled           |
| `cloud`    | optional*    | forbidden      | `https://app.archcore.ai`     |
| `on-prem`  | required     | required       | value of `archcore_url`       |

*If `project_id` is omitted in `cloud` mode, the server auto-creates a project using `project_name` (derived from directory name) and persists the returned ID.

## Package Structure

```
internal/sync/          — import as "archsync" (avoids stdlib sync conflict)
├── manifest.go         — Manifest struct, load/save, validation
├── hash.go             — SHA-256 hashing, file scanning
├── diff.go             — Change detection (created/modified/deleted/unchanged)
├── payload.go          — Sync request payload, frontmatter parsing
└── *_test.go           — Comprehensive table-driven tests

internal/api/client.go  — NewAuthenticatedClient, applyAuth, Sync() method
cmd/sync.go             — Cobra command, preconditions, flow orchestration
```

## Security

- **Path traversal prevention:** `validateRelPath()` rejects `..` segments and absolute paths in payload
- **Manifest validation:** rejects malformed hashes, null values, excessive file counts
- **Auth header:** only applied when token is non-empty
- **Response size limits:** max 10 MB response body, 512 bytes for error context

## Server → Local: Read-Only Enrichment via MCP

The server doesn't push back to `.archcore/`. Instead, it **enhances MCP search**:

- **Local MCP tool** (`archcore mcp`) → searches local `.archcore/` files only (current behavior)
- **Server MCP tool** → searches indexed documents across **all projects** via GraphRAG
- Claude Code can use both simultaneously — local for this project, server for cross-project knowledge

## Not Yet Implemented

| Feature                   | Status  | Notes                                            |
| ------------------------- | ------- | ------------------------------------------------ |
| `archcore login`/`logout` | planned | Token via env var works today                    |
| Keychain token storage    | planned | For local dev UX                                 |
| Sync triggers (hooks)     | planned | `sync_trigger` setting, git post-commit hook     |
| Chunked sync              | deferred| For very large repos (>10k files), pagination    |
| Cross-project MCP search  | planned | Remote MCP endpoint for server-indexed documents |
