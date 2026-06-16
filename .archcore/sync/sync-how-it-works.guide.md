---
title: "How Sync Works in Archcore"
status: accepted
tags:
  - "sync"
---

---
title: How Sync Works in Archcore
status: accepted
---

## Overview

`archcore sync` pushes local `.archcore/` documents to a cloud or on-prem server for indexing, GraphRAG-based search, and cross-project knowledge retrieval.

**Core principle:** `.archcore/` is the **source of truth**. Sync is a one-way push (local → server). The server is a read-only consumer — it never writes back to the local directory.

Related: [ADR: One-Way Push Sync Strategy](./one-way-push-sync-strategy.adr.md)

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

## Sync Modes

Sync mode is configured in `.archcore/settings.json` via the `sync` field. It drives what fields are valid and where sync pushes to.

| Mode      | `project_id` | `archcore_url` | Server URL                |
| --------- | ------------ | -------------- | ------------------------- |
| `none`    | forbidden    | forbidden      | N/A — sync disabled       |
| `cloud`   | optional     | forbidden      | `https://app.archcore.ai` |
| `on-prem` | optional     | required       | custom `archcore_url`     |

- **`none`** — sync is fully disabled. Running `archcore sync` prints a message and exits.
- **`cloud`** — pushes to the Archcore cloud at `https://app.archcore.ai`.
- **`on-prem`** — pushes to a self-hosted server at the URL specified in `archcore_url`.

If `project_id` is not set, the server auto-creates a project on first sync using the directory name as `project_name`, and the CLI persists the returned ID to `settings.json`. During auto-creation, the CLI also sends `repo_url` if the directory is a git repository with an `origin` remote (auto-detected via `git remote get-url origin`).

## CLI Usage

```bash
archcore sync              # interactive — shows diff, asks for confirmation
archcore sync --dry-run    # preview what would be synced, no actual push
archcore sync --force      # full re-sync, ignores manifest (treats all files as modified)
archcore sync --ci         # non-interactive mode for CI/CD pipelines
```

### Flags

| Flag        | Effect                                                      |
| ----------- | ----------------------------------------------------------- |
| `--dry-run` | Show diff summary without sending anything to the server    |
| `--force`   | Ignore manifest — re-sync all files as if they were changed |
| `--ci`      | Skip the interactive confirmation prompt                    |

## Sync Pipeline (Step by Step)

When you run `archcore sync`, the following happens:

### 1. Validate Preconditions

The command checks:
- `.archcore/` directory exists
- `settings.json` loads and passes validation
- Sync mode is not `none`
- Auth token is available (via `ARCHCORE_TOKEN` env var)

### 2. Load Manifest

The manifest file `.archcore/.sync-state.json` tracks what was last successfully synced. It maps relative file paths to their SHA-256 hashes at the time of last sync.

If the file doesn't exist (first sync), an empty manifest is created in memory.

### 3. Scan Files

Recursively walks the `.archcore/` directory and computes a SHA-256 hash for each `.md` file matching the `slug.type.md` naming convention. Directories can be nested to any depth. Hidden directories (`.`-prefixed) and meta files (`settings.json`, `.sync-state.json`) are skipped.

### 4. Calculate Diff

Compares the current file scan against the manifest to produce four sets:

| Action        | Meaning                                           |
| ------------- | ------------------------------------------------- |
| **created**   | File exists on disk but not in manifest           |
| **modified**  | File exists in both but SHA-256 hashes differ     |
| **deleted**   | File in manifest but no longer exists on disk     |
| **unchanged** | File exists in both and hashes match              |

With `--force`, all existing files are marked as `modified` regardless of hash comparison (deletions are still detected normally).

### 5. Display Diff Summary

Prints counts and file paths grouped by action. If there are no changes, prints "up to date" and exits.

### 6. Dry-Run Exit

If `--dry-run` is set, the pipeline stops here. Nothing is sent to the server.

### 7. Interactive Confirmation

A `huh.NewConfirm` prompt asks the user to confirm. Skipped when `--ci` is set.

### 8. Build Payload and Send

For each created/modified file:
- Reads full content from disk
- Parses YAML frontmatter to extract `title` and `status`
- Extracts `doc_type` from the filename (e.g., `adr` from `use-postgres.adr.md`)
- Derives `category` from the document type (e.g., `adr` → `knowledge`)
- Validates status against allowed values (`draft`, `accepted`, `rejected`)

Constructs a JSON payload:

```json
{
  "project_id": 42,
  "created": [{ "path": "...", "sha256": "...", "doc_type": "adr", "category": "knowledge", "frontmatter": {...}, "content": "..." }],
  "modified": [{ "path": "...", "sha256": "...", "doc_type": "adr", "category": "knowledge", "frontmatter": {...}, "content": "..." }],
  "deleted": ["path/to/removed.md"]
}
```

When auto-creating a project (`project_id` not set), the payload includes `project_name` and optionally `repo_url`:

```json
{
  "project_name": "my-project",
  "repo_url": "https://github.com/org/my-project.git",
  "created": [...]
}
```

Sends via `POST /api/v1/sync`.

### 9. Handle Response

The server responds with:
- **200** — all files synced successfully
- **201** — project auto-created; `project_id` returned and saved to `settings.json`
- **207** — partial success; some files accepted, some had errors

Per-file errors are reported to the user.

### 10. Update Manifest

Only after a successful response, the manifest is updated:
- Created/modified files → hash stored in manifest
- Deleted files → removed from manifest
- Unchanged files → untouched

The manifest is saved atomically (write to temp file, then rename).

## Manifest Format

File: `.archcore/.sync-state.json` (gitignored)

```json
{
  "version": 1,
  "files": {
    "use-postgres.adr.md": "a1b2c3d4e5f6...",
    "auth/login-flow.guide.md": "d4e5f6a1b2c3..."
  }
}
```

- `version` — always `1` (for future schema evolution)
- `files` — flat map of relative path → SHA-256 hex digest (64 chars, lowercase)

### Validation Rules

- Max 10,000 files
- Hashes must be valid SHA-256 (64 lowercase hex characters)
- Paths must be relative with no `..` segments or absolute paths
- No `null` values

## Authentication

```bash
# Set token via environment variable
export ARCHCORE_TOKEN=arc_xxxxx

# CI/CD usage
ARCHCORE_TOKEN=${{ secrets.ARCHCORE_TOKEN }} archcore sync --ci
```

The token is passed as `Authorization: Bearer <token>` header on the API request. The `api.NewAuthenticatedClient()` uses a 30-second timeout (longer than default 10s to handle large payloads).

## Package Structure

```
internal/sync/          — imported as "archsync" (avoids stdlib sync conflict)
├── manifest.go         — Manifest struct, load/save, validation
├── hash.go             — SHA-256 hashing, file scanning
├── diff.go             — Change detection (created/modified/deleted/unchanged)
├── payload.go          — Sync payload construction, frontmatter parsing
└── *_test.go           — Table-driven tests for each module

internal/git/git.go     — DetectRepoURL() helper for origin remote detection
internal/api/client.go  — Sync() method, authenticated client, response handling
cmd/sync.go             — Cobra command, preconditions, pipeline orchestration
```

## Security

- **Path traversal prevention:** `validateRelPath()` rejects `..` segments and absolute paths
- **Manifest validation:** rejects malformed hashes, null values, file counts over 10,000
- **Response size limit:** max 10 MB response body
- **Error body limit:** max 512 bytes from error responses
- **Atomic writes:** manifest saved via temp file + rename to prevent corruption

## Error Recovery

| Scenario                        | What happens                                                    |
| ------------------------------- | --------------------------------------------------------------- |
| Sync fails mid-request          | Manifest is NOT updated — next sync retries the same changes    |
| `.sync-state.json` is corrupted | Error on load — delete the file and run `archcore sync --force` |
| `.sync-state.json` is deleted   | Treated as first sync — all files marked as created             |
| Server returns 207 (partial)    | Manifest updated only for accepted files; errors reported       |
| Network timeout                 | Error returned — manifest unchanged, safe to retry              |

## MCP Integration

Two MCP search layers coexist:

- **Local MCP** (`archcore mcp`) — searches `.archcore/` files in the current project only
- **Server MCP** — searches indexed documents across all synced projects via GraphRAG

Claude Code can use both simultaneously: local for current project context, server for cross-project knowledge retrieval.