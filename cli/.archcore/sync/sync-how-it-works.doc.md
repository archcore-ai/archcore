---
title: "How Sync Works in Archcore"
status: accepted
tags:
  - "sync"
---

## Overview

`archcore sync` pushes local `.archcore/` documents to a cloud or on-prem server for indexing, GraphRAG-based search, and cross-project knowledge retrieval.

Core principle: `.archcore/` is the source of truth. Sync pushes in one direction, from local to server. The server consumes documents and never writes back to the local directory. The related ADR records that decision.

## Architecture

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

## Sync modes

The `sync` field in `.archcore/settings.json` sets the mode. The mode decides which fields are valid and where sync pushes to.

| Mode      | `project_id` | `archcore_url` | Server URL                |
| --------- | ------------ | -------------- | ------------------------- |
| `none`    | forbidden    | forbidden      | not applicable — sync off |
| `cloud`   | optional     | forbidden      | `https://app.archcore.ai` |
| `on-prem` | optional     | required       | custom `archcore_url`     |

- `none` — sync is disabled. `archcore sync` prints a message and exits.
- `cloud` — pushes to the Archcore cloud at `https://app.archcore.ai`.
- `on-prem` — pushes to a self-hosted server at the URL in `archcore_url`.

WHILE `project_id` is unset, the server auto-creates a project on the first sync, using the directory name as `project_name`, and the CLI persists the returned ID to `settings.json`. During auto-creation the CLI also sends `repo_url` when the directory is a git repository with an `origin` remote, detected via `git remote get-url origin`.

## CLI usage

```bash
archcore sync              # interactive — shows the diff, asks for confirmation
archcore sync --dry-run    # preview what would be synced, no push
archcore sync --force      # full re-sync, ignores the manifest (all files count as modified)
archcore sync --ci         # non-interactive mode for CI/CD pipelines
```

### Flags

| Flag        | Effect                                                       |
| ----------- | ------------------------------------------------------------ |
| `--dry-run` | Show the diff summary without sending anything to the server |
| `--force`   | Ignore the manifest and re-sync every file as changed        |
| `--ci`      | Skip the interactive confirmation prompt                     |

## What one sync run does

This section describes current CLI behavior, not steps for the reader.

### 1. Validate preconditions

The command checks that `.archcore/` exists, that `settings.json` loads and passes validation, that the sync mode is not `none`, and that an auth token is available through the `ARCHCORE_TOKEN` environment variable.

### 2. Load the manifest

`.archcore/.sync-state.json` records what the last successful sync sent. It maps a relative file path to the SHA-256 hash the file had at that moment.

IF the file is absent, which is the case on a first sync, THEN the CLI creates an empty manifest in memory.

### 3. Scan the files

The CLI walks `.archcore/` recursively and computes a SHA-256 hash for every `.md` file that follows the `slug.type.md` convention. Directories nest to any depth. The walk skips hidden directories, which are `.`-prefixed, and the meta files `settings.json` and `.sync-state.json`.

### 4. Calculate the diff

The CLI compares the scan against the manifest and produces four sets.

| Action        | Meaning                                       |
| ------------- | --------------------------------------------- |
| created       | The file exists on disk but not in the manifest |
| modified      | The file exists in both and the hashes differ  |
| deleted       | The file is in the manifest but not on disk    |
| unchanged     | The file exists in both and the hashes match   |

WITH `--force`, every existing file is marked `modified` regardless of its hash. Deletions are still detected normally.

### 5. Display the diff summary

The CLI prints counts and file paths grouped by action. IF nothing changed, THEN it prints "up to date" and exits.

### 6. Exit on a dry run

IF `--dry-run` is set, THEN the run stops here and sends nothing to the server.

### 7. Ask for confirmation

A `huh.NewConfirm` prompt asks the user to confirm. The prompt is skipped WHILE `--ci` is set.

### 8. Build the payload and send it

For each created or modified file, the CLI reads the full content from disk, parses the YAML frontmatter for `title` and `status`, extracts `doc_type` from the filename (`adr` from `use-postgres.adr.md`), derives `category` from the document type (`adr` maps to `knowledge`), and validates the status against `draft`, `accepted`, and `rejected`.

It then constructs the JSON payload:

```json
{
  "project_id": 42,
  "created": [{ "path": "...", "sha256": "...", "doc_type": "adr", "category": "knowledge", "frontmatter": {...}, "content": "..." }],
  "modified": [{ "path": "...", "sha256": "...", "doc_type": "adr", "category": "knowledge", "frontmatter": {...}, "content": "..." }],
  "deleted": ["path/to/removed.md"]
}
```

WHILE `project_id` is unset, the payload carries `project_name`, and `repo_url` when one was detected:

```json
{
  "project_name": "my-project",
  "repo_url": "https://github.com/org/my-project.git",
  "created": [...]
}
```

The CLI sends the payload with `POST /api/v1/sync`.

### 9. Handle the response

- `200` — every file synced.
- `201` — the project was auto-created; the response carries `project_id`, which the CLI saves to `settings.json`.
- `207` — partial success; some files were accepted and some returned errors.

The CLI reports per-file errors to the user.

### 10. Update the manifest

Only after a successful response, the CLI stores the hash of each created or modified file, removes each deleted file from the manifest, and leaves unchanged entries alone. It saves the manifest atomically: write to a temporary file, then rename.

## Manifest format

File: `.archcore/.sync-state.json`, gitignored.

```json
{
  "version": 1,
  "files": {
    "use-postgres.adr.md": "a1b2c3d4e5f6...",
    "auth/login-flow.guide.md": "d4e5f6a1b2c3..."
  }
}
```

- `version` — always `1`, reserved for future schema evolution.
- `files` — a flat map of relative path to SHA-256 hex digest, 64 lowercase characters.

### Validation rules

- At most 10,000 files.
- Each hash is a valid SHA-256 digest: 64 lowercase hex characters.
- Each path is relative, with no `..` segment and no absolute form.
- No `null` values.

## Authentication

```bash
# Set the token through an environment variable
export ARCHCORE_TOKEN=arc_xxxxx

# CI/CD usage
ARCHCORE_TOKEN=${{ secrets.ARCHCORE_TOKEN }} archcore sync --ci
```

The CLI passes the token in the `Authorization: Bearer <token>` header. `api.NewAuthenticatedClient()` uses a 30-second timeout, longer than the 10-second default, to carry large payloads.

## Package structure

```
internal/sync/          — imported as "archsync" (avoids the stdlib sync conflict)
├── manifest.go         — Manifest struct, load and save, validation
├── hash.go             — SHA-256 hashing, file scanning
├── diff.go             — change detection (created/modified/deleted/unchanged)
├── payload.go          — payload construction, frontmatter parsing
└── *_test.go           — table-driven tests per module

internal/git/git.go     — DetectRepoURL() for origin remote detection
internal/api/client.go  — Sync() method, authenticated client, response handling
cmd/sync.go             — cobra command, preconditions, pipeline orchestration
```

## Security

- Path traversal prevention: `validateRelPath()` rejects `..` segments and absolute paths.
- Manifest validation: rejects malformed hashes, `null` values, and file counts above 10,000.
- Response size limit: 10 MB of response body.
- Error body limit: 512 bytes from an error response.
- Atomic writes: the manifest is saved through a temporary file and a rename, which prevents a corrupted file.

## Error recovery

| Scenario                        | What happens                                                     |
| ------------------------------- | ---------------------------------------------------------------- |
| Sync fails mid-request          | The manifest stays unchanged; the next sync retries the same changes |
| `.sync-state.json` is corrupted | Load fails — delete the file and run `archcore sync --force`     |
| `.sync-state.json` is deleted   | Treated as a first sync; every file counts as created            |
| Server returns 207 (partial)    | The manifest is updated for accepted files only; errors reported |
| Network timeout                 | An error is returned; the manifest is unchanged and a retry is safe |

## MCP integration

Two MCP search layers coexist:

- Local MCP (`archcore mcp`) searches `.archcore/` files in the current project only.
- Server MCP searches indexed documents across every synced project through GraphRAG.

Claude Code can use both at once: the local server for current-project context, the remote server for cross-project retrieval.
