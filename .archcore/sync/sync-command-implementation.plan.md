---
title: "Implementation Plan: archcore sync Command"
status: accepted
tags:
  - "sync"
---

## Overview

Implement the `archcore sync` command for one-way push of local `.archcore/` documents to a cloud or on-prem server, with incremental diff via SHA-256 manifest.

Related: [ADR: One-Way Push Sync Strategy](./one-way-push-sync-strategy.adr.md) | [Idea: Sync Design](./sync-design.idea.md)

## Settings Validation (Pre-sync Checks)

Before any sync operation, the command validates in order:

1. **`.archcore/` directory exists** — otherwise advise `archcore init`
2. **Settings load and validate** — via `config.Load()`
3. **Sync mode is not `none`** — graceful error: `"sync is disabled — run 'archcore config set sync cloud' or 'archcore init' to configure"`
4. **`project_id` is non-nil** — both cloud and on-prem require it for the API endpoint
5. **Auth token present** — resolve from `ARCHCORE_TOKEN` env var

Extracted into a testable `checkSyncPreconditions(baseDir string, tokenLookup func(string) string)` function that returns a `syncPreconditions` struct or error. The `tokenLookup` parameter avoids direct `os.Getenv` dependency for clean testing.

```go
type syncPreconditions struct {
    Settings  *config.Settings
    ProjectID int
    Token     string
    ServerURL string
}
```

## New Package: `internal/sync/`

Pure domain package — no cobra, no display, no API dependencies.

### manifest.go — State Tracking

- `Manifest` struct with `Version int` and `Files map[string]*FileEntry`
- `FileEntry` with `Hash string` (SHA-256 hex) and `SyncedAt time.Time`
- `LoadManifest(baseDir)` — returns empty manifest if file missing, error if corrupted JSON
- `SaveManifest(baseDir, m)` — atomic write (temp file + rename)
- Manifest file: `.archcore/.sync-state.json` (gitignored)

### hash.go — File Walking and Hashing

- `HashFile(path) (string, error)` — SHA-256 hex digest via streaming `io.Copy`
- `FileState` struct: `RelPath`, `AbsPath`, `Hash`
- `ScanFiles(baseDir) ([]FileState, error)` — walks `vision/`, `knowledge/`, `experience/`, skips subdirs, tolerates missing category dirs

### diff.go — Change Detection

- `DiffAction` type: `created`, `modified`, `deleted`, `unchanged`
- `DiffEntry` struct: `RelPath`, `Action`, `Hash`
- `Diff(current []FileState, manifest *Manifest) []DiffEntry` — compares on-disk state vs manifest
- `HasChanges(entries) bool` — true if any non-unchanged entry
- `FilterByAction(entries, action) []DiffEntry` — filter helper

### payload.go — API Request Construction

- `SyncFilePayload`: `Path`, `Hash`, `Content`, `Action` (JSON fields)
- `SyncPayload`: `Files []SyncFilePayload`
- `BuildPayload(baseDir, entries)` — reads file content for created/modified, skips unchanged, empty content for deleted

## API Extension: `internal/api/client.go`

Additive changes, backward-compatible with existing `NewClient` callers:

- Add `Token string` field to `Client` struct
- `NewAuthenticatedClient(serverURL, token string) *Client` — 30s timeout (sync may be slow)
- `applyAuth(req)` — sets `Authorization: Bearer <token>` if token non-empty; no-op for unauthenticated clients
- `post(ctx, path, body, dest)` helper — JSON marshal, POST, decode response
- `SyncProject(ctx, projectID int, payload *SyncPayload) (*SyncResponse, error)` — `POST /api/v1/projects/{id}/sync`
- `SyncResponse` struct: `Status`, `Synced int`, `Message`

Modify existing `get` helper to call `applyAuth`.

## Command: `cmd/sync.go`

### Flags

| Flag        | Type | Default | Description                          |
| ----------- | ---- | ------- | ------------------------------------ |
| `--dry-run` | bool | false   | Show diff without sending to server  |
| `--force`   | bool | false   | Re-sync all files (ignore manifest)  |
| `--ci`      | bool | false   | Skip interactive confirmation prompt |

### Flow

1. Validate preconditions (settings, sync mode, project_id, token)
2. Load manifest + scan files
3. Calculate diff (or mark all as modified if `--force`)
4. Print diff summary (created/modified/deleted counts with file paths)
5. Exit early if `--dry-run`
6. Interactive confirmation via `huh.NewConfirm` (skip if `--ci`)
7. Build payload, send via `api.SyncProject`
8. Update manifest on success (only for confirmed files)
9. Print success summary

Graceful exits (print styled message, return nil) for user-facing issues. True errors (I/O, JSON) return errors.

### Registration

Add `newSyncCmd()` to `root.AddCommand(...)` in `cmd/root.go`.

## Unit Tests

### internal/sync/manifest_test.go

- **Table-driven `TestLoadManifest`**: no file (returns empty), valid JSON (2 files), corrupted JSON (error), empty object (empty files map)
- **`TestSaveManifest_Roundtrip`**: save → load → verify hash and timestamp preserved
- **`TestSaveManifest_Atomic`**: verify temp file cleaned up, final file exists

### internal/sync/hash_test.go

- **`TestHashFile`**: known content → 64-char hex output
- **`TestHashFile_NotFound`**: nonexistent file → error
- **`TestHashFile_Deterministic`**: same file hashed twice → same result
- **Table-driven `TestScanFiles`**: empty dirs (0 files), files across categories (2 found), skips subdirs (nested dir ignored), missing category dir (not an error)

### internal/sync/diff_test.go

- **Table-driven `TestDiff`**: first sync (all created), no changes (all unchanged), one modified, one deleted, mixed (created + modified + deleted + unchanged), empty both sides
- **Table-driven `TestHasChanges`**: empty, all unchanged, has created, has deleted, has modified, mixed

### internal/sync/payload_test.go

- **`TestBuildPayload`**: created file has content, deleted file has empty content, unchanged files excluded
- **`TestBuildPayload_MissingFile`**: created entry pointing to nonexistent file → error

### internal/api/client_test.go (extend existing)

- **Table-driven `TestSyncProject`**: success (verify method, path, auth, content-type), server error (500), unauthorized (401), bad response JSON
- **`TestNewAuthenticatedClient`**: verify BaseURL and Token set correctly
- **`TestApplyAuth_NoToken`**: existing unauthenticated client sends no auth header

### cmd/sync_test.go

- **Table-driven `TestCheckSyncPreconditions`**: no .archcore dir, sync none, project_id nil, missing token, valid cloud, valid on-prem
- **`TestCheckSyncPreconditions_CloudServerURL`**: cloud mode resolves to `config.CloudServerURL`
- **`TestCheckSyncPreconditions_OnPremServerURL`**: on-prem mode resolves to configured URL
- **`TestRunSync_DryRun_DoesNotUpdateManifest`**: dry-run leaves manifest unchanged
- **`TestRunSync_Force_ResyncsUnchangedFiles`**: force includes unchanged files as modified
- **`TestRunSync_NoChanges_ShortCircuit`**: up-to-date manifest → no API call
- **`TestRunSync_EndToEnd`**: full flow with httptest server, CI flag, verify manifest updated after sync

## New Files

| File                             | Purpose                           |
| -------------------------------- | --------------------------------- |
| `internal/sync/manifest.go`      | Manifest struct, persistence      |
| `internal/sync/hash.go`          | SHA-256 hashing, file scanning    |
| `internal/sync/diff.go`          | Diff calculation                  |
| `internal/sync/payload.go`       | Sync request payload construction |
| `internal/sync/manifest_test.go` | Manifest tests                    |
| `internal/sync/hash_test.go`     | Hashing/scanning tests            |
| `internal/sync/diff_test.go`     | Diff tests                        |
| `internal/sync/payload_test.go`  | Payload tests                     |
| `cmd/sync.go`                    | Cobra command + preconditions     |
| `cmd/sync_test.go`               | Command tests                     |

## Modified Files

| File                          | Change                                                                   |
| ----------------------------- | ------------------------------------------------------------------------ |
| `internal/api/client.go`      | Token field, applyAuth, post helper, NewAuthenticatedClient, SyncProject |
| `internal/api/client_test.go` | Auth and sync endpoint tests                                             |
| `cmd/root.go`                 | Register `newSyncCmd()`                                                  |

## Implementation Order

1. `internal/sync/` — all 4 source files + tests. Run `go test ./internal/sync/`
2. `internal/api/client.go` — extensions + tests. Run `go test ./internal/api/`
3. `cmd/sync.go` + `cmd/sync_test.go` — command + preconditions. Run `go test ./cmd/`
4. `cmd/root.go` — register command
5. Manual verify: `go build -o archcore . && ./archcore sync --dry-run`

## Key Design Decisions

- **`import archsync`** — avoids conflict with stdlib `sync` package
- **`func(string) string` for token lookup** — clean test injection without `t.Setenv`
- **Atomic manifest write** — crash-safe via temp + rename
- **Graceful exits for user errors** — return nil after styled output (matches doctor/init pattern)
- **Raw UTF-8 content (not base64)** — documents are Markdown text, simpler and debuggable
