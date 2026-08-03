---
title: "Implementation Plan: archcore sync Command"
status: accepted
tags:
  - "sync"
---

## Goal

Implement `archcore sync`: a one-way push of local `.archcore/` documents to a cloud or on-prem
server, with an incremental diff driven by a SHA-256 manifest.

## Status

Executed. The command, the `internal/sync` package, and the API extension all shipped. Several
details differ from the plan below; the "Outcome" section lists the verified differences, and the
sync engine contract plus the sync guide describe the behavior that ships today.

The command itself is currently gated: it is hidden and returns "sync is not available yet — this
feature is coming soon".

## Tasks

### 1. Pre-sync precondition checks (`cmd/sync.go`)

- [x] Check that `.archcore/` exists; otherwise advise `archcore init`.
- [x] Load and validate settings through `config.Load()`.
- [x] Reject the sync mode `none` with a graceful error: `sync is disabled — run 'archcore config set sync cloud' or 'archcore init' to configure`.
- [x] Resolve the project: cloud and on-prem both need a project on the server side.
- [x] Extract the checks into a testable `checkSyncPreconditions(baseDir)` that returns a `syncPreconditions` struct or an error.

### 2. New package `internal/sync/`

A pure domain package: no cobra, no display, no API dependencies.

- [x] `manifest.go` — the `Manifest` struct with a version and a file map; `LoadManifest(baseDir)` returning an empty manifest when the file is missing and an error on corrupted JSON; `SaveManifest(baseDir, m)` writing atomically through a temporary file and a rename. The manifest file is `.archcore/.sync-state.json`, gitignored.
- [x] `hash.go` — `HashFile(path)` producing a SHA-256 hex digest through a streaming `io.Copy`; a `FileState` struct with `RelPath`, `AbsPath`, and `Hash`; `ScanFiles(baseDir)` walking the document tree.
- [x] `diff.go` — the `DiffAction` type (`created`, `modified`, `deleted`, `unchanged`); a `DiffEntry` struct; `Diff(current, manifest)`; `HasChanges(entries)`; `FilterByAction(entries, action)`.
- [x] `payload.go` — `SyncFilePayload` and `SyncPayload`; `BuildPayload(baseDir, entries)` reading content for created and modified entries, skipping unchanged ones, and sending empty content for deletions.

### 3. API extension (`internal/api/client.go`)

- [x] Add a `Token` field to `Client`.
- [x] Add `applyAuth(req)`, which sets `Authorization: Bearer <token>` while the token is non-empty and does nothing otherwise.
- [x] Add a constructor for the sync client with a longer timeout than the default.
- [x] Add the sync call and a `SyncResponse` struct.
- [x] Route the existing `get` helper through `applyAuth`.

### 4. Command (`cmd/sync.go`)

- [x] Flags: `--dry-run` (show the diff without sending), `--force` (re-sync every file, ignoring the manifest), `--ci` (skip the confirmation prompt).
- [x] Flow: validate preconditions; load the manifest and scan the files; calculate the diff, or mark everything modified under `--force`; print the diff summary; exit early under `--dry-run`; confirm through `huh.NewConfirm` unless `--ci`; build and send the payload; update the manifest after a successful response; print the summary.
- [x] Use graceful exits — a styled message and a `nil` return — for user-facing issues, and real errors for I/O and JSON failures.
- [x] Register `newSyncCmd()` in `@cmd/root.go`.

### 5. Tests

- [x] `internal/sync/manifest_test.go` — table-driven `TestLoadManifest` (missing file, valid JSON, corrupted JSON, empty object); `TestSaveManifest_Roundtrip`; atomic-write coverage.
- [x] `internal/sync/hash_test.go` — `TestHashFile`, `TestHashFile_NotFound`, `TestHashFile_Deterministic`, and a table-driven `TestScanFiles`.
- [x] `internal/sync/diff_test.go` — table-driven `TestDiff` (first sync, no changes, modified, deleted, mixed, empty) and `TestHasChanges`.
- [x] `internal/sync/payload_test.go` — content present for created entries, empty for deletions, unchanged entries excluded, missing file returns an error.
- [x] `internal/api/client_test.go` — sync call coverage: success with method, path, auth, and content type; server error; unauthorized; malformed response JSON.
- [x] `cmd/sync_test.go` — table-driven `TestCheckSyncPreconditions`; cloud and on-prem server URL resolution; dry-run leaves the manifest unchanged; `--force` re-sends unchanged files; an up-to-date manifest short-circuits without an API call; an end-to-end run against an `httptest` server.

## Outcome

Verified differences between this plan and the shipped code:

- Endpoint: the plan specified `POST /api/v1/projects/{id}/sync`. The code calls `POST /api/v1/sync` through `Client.Sync` in `@internal/api/client.go`.
- Client constructor: `NewAuthenticatedClient` was replaced by `NewSyncClient` with a 120s timeout during the July 2026 audit.
- Preconditions: `syncPreconditions` in `@cmd/sync.go` carries `Settings`, `ProjectID`, `ServerURL`, and `BaseDir`. It has no `Token` field, and no `tokenLookup` injection parameter shipped. `project_id` may be nil, which triggers server-side auto-creation.
- File scan: `ScanFiles` walks `.archcore/` recursively instead of the three fixed category directories, and it skips hidden directories, meta files, the reserved `global/` tree, and declared global sources.
- Manifest: the shipped file maps a relative path to a hash string. The planned per-entry `SyncedAt` timestamp did not ship.

## Acceptance Criteria

- `archcore sync --dry-run` prints the diff and sends nothing. ✅
- The manifest updates only after a confirmed server response, and only for accepted paths. ✅
- `--force` re-sends unchanged files while deletions are still detected normally. ✅
- `go test ./internal/sync/ ./internal/api/ ./cmd/` is green. ✅

## Dependencies

- The decision this plan implements: the related ADR on one-way push sync.
- The contract the implementation is checked against: the related sync engine specification.
- The feature gate that currently hides the command: the related ADR on temporarily disabling sync.

## Key design decisions

- Import `internal/sync` as `archsync`, which avoids the collision with the standard library `sync`.
- Write the manifest atomically through a temporary file and a rename, so a crash mid-write cannot corrupt it.
- Use graceful exits for user errors: styled output and a `nil` return, matching the `doctor` and `init` pattern.
- Send raw UTF-8 content rather than base64: documents are Markdown, so the payload stays debuggable.
