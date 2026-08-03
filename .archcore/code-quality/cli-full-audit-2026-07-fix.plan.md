---
title: "Fix Plan: Full CLI Audit Findings (July 2026)"
status: accepted
tags:
  - "code-quality"
  - "code-review"
  - "golang"
  - "mcp"
  - "performance"
  - "sync"
  - "testing"
---

## Goal

Fix every proven defect from the July 2026 full CLI audit — five parallel golang-pro reviews over
`cmd/`, `internal/mcp/`, the supporting packages, the test suite, and the performance benchmarks.
Each finding carried file:line evidence: a traced failure scenario or a measured number.

## Status

Executed 2026-07-03 on branch `fix/full-audit-2026-07`.

All five phases landed as 16 commits. `go vet` is clean, `go test -race -count=1 ./...` is green
across all 13 packages, and `gofmt` is clean.

## Tasks

### Phase 1 — MCP security and integrity ✅ done

- `sanitizeError` closes all ~16 absolute-path leaks, including handler Go errors, which mcp-go transmits verbatim.
- `manifestStore` serializes manifest mutations through clone-and-swap under a mutex, and caches reads on (mtime, size).
- `guardWritablePath` adds document-only targets (`.md` plus `SkipFiles`), case-folded global matching, and `EvalSymlinks` containment for writes.
- `create_document` uses `O_CREATE|O_EXCL`; `update_document` writes atomically.
- `SplitDocument` surfaces YAML errors and strips the BOM, so `update_document` no longer erases the metadata of a broken document.
- A corrupt manifest is a uniform tool error; the search spec was amended to match.
- `relations_removed` counts only the deleted document's edges.
- The hook and MCP installers were rebuilt on `internal/jsonfile`: `RawMessage` round-trip, so user configs keep their fields; atomic writes; no-op second runs; abort when the `.bak` write fails; fixed panics on null sections; JSONC `.vscode/mcp.json` left untouched with manual instructions; opencode delegated to the shared writer.
- Amended documents: the global sources contract §5.4, the `search_documents` contract §3, and the ADR on backing up invalid configs.

### Phase 2 — sync ✅ done

- The manifest records only `resp.Accepted`; deletions apply only when the path is absent from `resp.Errors`.
- `ScanFiles` skips `global/` and the declared globals.
- `NewSyncClient` with a 120s timeout replaces the dead `NewAuthenticatedClient`.
- Server URLs get their trailing slash trimmed.
- The created-project id is surfaced in the Save-failure error.
- Dead `runSync` removed.
- The sync feature gate itself was left untouched.

### Phase 3 — durability ✅ done

- CRLF and BOM handling in `checkFrontmatter` (`cmd/status`).
- Atomic `config.Save`.
- Update extraction size check plus fsync.
- Non-zero exit on a failed update.
- A hookless `--agent` still receives the MCP config.
- `init` soft-fails on an unreachable server.
- Preserved tags keep their order.

### Phase 4 — performance ✅ done

- Added an mtime-keyed scan cache, a per-call relation index, a cached manifest store, ReadDir-based nearby hints, and a `list_documents` limit/offset envelope.
- Re-measured on a realistic corpus: list at 10K documents 234 → 36 ms (−85%); search-snippets at 10K 746 → 240 ms (−68%); `get_document` at 10K 38.6 ms → about 60 µs (O(stat)); search allocations at 10K 382 → 93 MB.
- List output is bounded: default 100, maximum 500.
- The numbers are recorded in the related read-path performance idea.

### Phase 5 — tests, dead code, consistency, docs ✅ done

- Coverage gaps closed: `checkFrontmatter` branches, `checkTagHygiene`, `parseMtimeAfter`, `atomicReplace` rollback, update cancellation, hooks auto-detect, copilot wiring, config usage, and concurrent `add_relation` under `-race`.
- Dead code removed: `ListProjects`, `GetProject`, `Project`, `ParseFrontmatter`, `ValidSyncTypes`, `ValidStatuses`, `ValidCategories`, `TypesByCategory`, unused `hookInput` fields, and dead branches.
- Consistency sweep: `ErrAlreadyReported` sentinel, stderr for errors, `cleanVersion("dev")`, MCP server version from the build, git context propagation, sorted deletions, types-filter validation, `archsync` aliases, import-group order, `errors.New`, kind constants, and `readErrorBody` with a `LimitReader`.
- Stale documents amended: naming-rule deviations pruned, the import order in the Go code quality rule fixed, the sync path-security rule aligned to the free-form layout, and the read-path idea re-measured.

### Deliberately left open

- `LocalDocument.Type` and `searchResult.Type` as `templates.DocumentType` (§G) — recorded as the remaining known deviation in the strict Go naming rule.
- Source weighting in search ranking (read-path idea candidate 4, ranking half) and per-source global budgets (candidate 6).
- Search-snippet CPU at 6–7K documents and above, about 150 ms per call, caused by the content-substring scan. Revisit through candidate 5 IF a corpus ever reaches that size.
- The duplicated `.archcore/ not found` guard in `@cmd/mcp.go`, in the outer `RunE` and in the inner function, kept intentionally: the inner one is test-pinned and the outer one covers auto-detect.

## Acceptance Criteria

Verified at execution time:

- No MCP tool result and no handler error contains an absolute filesystem path — sanitizer unit table, chmod-based per-tool tests, and the existing leak guards.
- Concurrent `add_relation` calls under `-race` lose no relations — integration test with N parallel adds.
- `update_document` and `remove_document` reject non-`.md` and `SkipFiles` targets; write paths pass `EvalSymlinks` containment; `Global/` is rejected in any case variant — guard unit table plus per-tool pinned messages.
- `archcore hooks install` run twice against a `settings.json` with foreign hooks and unknown fields leaves a byte-identical file after the first run — test-pinned.
- Sync updates the manifest only for accepted paths, and the payload never contains `global/` — test-pinned.
- `go test -race ./...` green; benchmarks re-run and recorded.

## Dependencies

- Phase 2 unblocks removal of the sync feature gate, recorded in the related ADR, once the server side is ready.
- The original audit findings stay in this document's git history, first revision, for traceability.
