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

Fix all proven defects from the July 2026 full audit of the CLI (5 parallel golang-pro reviews: cmd/, internal/mcp/, supporting packages, test suite, performance benchmarks). Every finding had file:line evidence — a traced failure scenario or a measured number.

## Status — EXECUTED 2026-07-03 (branch `fix/full-audit-2026-07`)

All five phases landed as 16 commits; `go vet` clean, `go test -race -count=1 ./...` green across all 13 packages, `gofmt` clean.

**Phase 1 (MCP security/integrity)** — done: sanitizeError closes all ~16 absolute-path leaks (incl. handler Go errors, which mcp-go transmits verbatim); manifestStore serializes manifest mutations (clone-and-swap under mutex) and caches reads on (mtime, size); guardWritablePath adds document-only targets (.md + SkipFiles), case-folded global matching, and EvalSymlinks containment for writes; create uses O_CREATE|O_EXCL, update writes atomically; SplitDocument surfaces YAML errors + strips BOM (update_document no longer erases metadata of broken docs); corrupt-manifest is a uniform tool error (search spec amended); relations_removed counts only the deleted doc's edges. Hook/MCP installers rebuilt on internal/jsonfile: RawMessage round-trip (no more field stripping from user configs), atomic writes, no-op second runs, abort when .bak fails, null-section panics fixed, JSONC .vscode/mcp.json left untouched with manual instructions, opencode delegated to the shared writer. Specs/ADRs amended: global-sources.spec §5.4, search-documents.spec §3, backup-invalid-configs.adr.

**Phase 2 (sync)** — done: manifest records only resp.Accepted (deletions only when not in resp.Errors); ScanFiles skips global/ and declared globals; NewSyncClient (120s) replaces the dead NewAuthenticatedClient; trailing-slash trim on server URLs; created-project id surfaced in the Save-failure error; dead runSync removed. Still gated — the gate itself is untouched.

**Phase 3 (durability)** — done: CRLF/BOM in cmd/status checkFrontmatter; atomic config.Save; update extraction size check + fsync; non-zero exit on failed update; hookless --agent gets MCP config; init soft-fails on unreachable server; preserved tags no longer reordered.

**Phase 4 (performance)** — done: mtime-keyed scan cache, per-call relation index, cached manifest store, ReadDir-based nearby hints, list_documents limit/offset envelope. Re-measured (realistic corpus): list @10K 234→36 ms (−85%), search-snip @10K 746→240 ms (−68%), get_document @10K 38.6 ms→~60 µs (O(stat)), search allocations @10K 382→93 MB; list output bounded (default 100/max 500). Numbers recorded in read-path-scan-performance.idea.md.

**Phase 5 (tests/dead code/consistency/docs)** — done: coverage gaps closed (checkFrontmatter branches, checkTagHygiene, parseMtimeAfter, atomicReplace rollback, update cancellation, hooks auto-detect, copilot wiring, config usage, concurrent add_relation under -race); dead code removed (ListProjects/GetProject/Project, ParseFrontmatter, ValidSyncTypes, ValidStatuses/ValidCategories/TypesByCategory, unused hookInput fields, dead branches); consistency sweep (ErrAlreadyReported sentinel, stderr for errors, cleanVersion("dev"), MCP server version from build, git ctx, sorted deletions, types-filter validation, archsync aliases, import-group order, errors.New, kind constants, readErrorBody LimitReader); stale docs amended (naming rule deviations pruned, go-code-quality import order fixed, sync-path-security aligned to free-form, read-path idea re-measured).

### Deliberately left open

- `LocalDocument.Type` / `searchResult.Type` as `templates.DocumentType` (§G) — recorded as the remaining known deviation in strict-go-naming-conventions.rule.md.
- Source weighting in search ranking (read-path idea candidate 4, ranking half) and per-source global budgets (candidate 6).
- search-snip CPU at ≥6–7K docs (~150 ms/call) — content-substring scan; revisit via candidate 5 if corpora ever reach that size.
- The duplicated `.archcore/ not found` guard in cmd/mcp.go (outer RunE + inner function) kept intentionally: the inner is test-pinned, the outer covers auto-detect.

## Acceptance Criteria — verified

- No MCP tool result or handler error contains an absolute filesystem path (sanitizer unit table + chmod-based per-tool tests + existing leak guards).
- Concurrent add_relation calls under -race lose no relations (integration test, N parallel adds).
- update/remove reject non-.md and SkipFiles targets; write paths pass EvalSymlinks containment; Global/ rejected in any case variant (guard unit table + per-tool pinned messages).
- `archcore hooks install` twice on a settings.json with foreign hooks and unknown fields → byte-identical file after the first run (test-pinned).
- Sync updates the manifest only for accepted paths; payload never contains global/ (test-pinned).
- `go test -race ./...` green; benchmarks re-run and recorded.

## Dependencies

- Phase 2 unblocks removing the sync feature gate (temporarily-disable-sync.adr.md) whenever the server side is ready.
- The original audit findings remain in this document's git history (first revision) for traceability.