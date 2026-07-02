---
title: "Fix Plan: Full CLI Audit Findings (July 2026)"
status: draft
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

Fix all proven defects from the July 2026 full audit of the CLI (5 parallel golang-pro reviews: cmd/, internal/mcp/, supporting packages, test suite, performance benchmarks). Every finding below has file:line evidence — a traced failure scenario or a measured number. Baseline at audit time: `go vet` clean, all tests pass, `-race` clean, coverage 73.8–100% per package (display/main 0%).

## Tasks

### Phase 1 — Security & data integrity in the MCP server (P0)

1. **Sanitize OS-error passthrough in all MCP tools** (violates `no-absolute-paths-in-mcp-errors.rule.md`; ~15 sites; mcp-go v0.49 provably transmits both `errorResult` strings and returned Go errors to the client):
   - CRITICAL: `add_relation.go:107,114`, `remove_relation.go:71,77`, `remove_document.go:89,95`, `init_project.go:62,84` — manifest/settings load/save errors embed absolute `*os.PathError` paths in tool results.
   - HIGH: `create_document.go:161,183`, `update_document.go:107,135`, `remove_document.go:76,82`, `get_document.go:62`, `list_documents.go:54`, `search_documents.go:228` — handler Go errors wrap `*os.PathError` (phase-2 global scan errors ARE sanitized; phase 1 and write paths are not).
   - Fix: one shared sanitizer in `common.go`; fixed messages without OS error text.
2. **Serialize manifest mutations** — `add_relation.go:105-116`, `remove_relation.go:69-79`, `remove_document.go:87-97` do unlocked load-modify-save of `.sync-state.json`; mcp-go stdio dispatches tools/call on a 5-goroutine worker pool → lost updates (parallel `add_relation` silently drops a relation while reporting `added: true`). Fix: package-level mutex. Also fix create TOCTOU (`create_document.go:169-183`) with `O_CREATE|O_EXCL`.
3. **Harden the write-path validator** (`common.go`):
   - Case-insensitive-FS bypass of globals read-only guard: `.archcore/Global/x.rule.md` passes `isReservedGlobalDir`/`matchGlobal` (`common.go:229-234,202-211`) and mutates `global/` on APFS/NTFS.
   - Symlink escape: write tools validate lexically only; read path already does `EvalSymlinks` containment (`common.go:442-455`), writes don't (`update_document.go:101,135`, `create_document.go:156-183`).
   - Non-document paths: `update_document`/`remove_document` accept `.archcore/settings.json` / `.sync-state.json` — update rewrites settings.json as markdown, bricking the project. Require `.md` + reject `templates.SkipFiles`.
4. **hooks/MCP installers must stop corrupting user configs** (`cmd/hooks*.go`, `internal/agents/mcp_helpers.go`):
   - CRITICAL: `cmd/hooks.go:184-217` round-trips the user's entire `.claude/settings.json` hooks section through typed structs, silently dropping unknown fields (e.g. `timeout` on non-archcore hooks) and rewrites the file even when already installed. Same class in cursor/copilot/gemini installers (`hooks_cursor.go:61-99`, `hooks_copilot.go:61-99`, `hooks_gemini_cli.go:64-90`).
   - HIGH: panic on `"hooks": null` in `.gemini/settings.json` (`hooks_gemini_cli.go:64-82`, nil-map assignment — reproduced).
   - HIGH: all four installers write live config files non-atomically (`os.WriteFile` truncate-then-write) — kill mid-write empties the user's `.claude/settings.json`.
   - MEDIUM: `.bak` backup write errors swallowed before overwriting the corrupted original (`cmd/hooks.go:174` et al., `internal/agents/mcp_helpers.go:37-39`); JSONC `.vscode/mcp.json` treated as corrupt and silently replaced.
   - Fix: extract shared `readJSONConfigWithBackup` + `writeJSONConfigAtomic` helpers with `json.RawMessage` round-tripping; abort if `.bak` fails; skip write when unchanged. This also removes the ~240-line duplication across the four installers.

### Phase 2 — Sync correctness (before removing the feature gate)

5. `cmd/sync.go:236-247`: manifest records hashes for ALL diff entries even when the server returns HTTP 207 with per-file `resp.Errors` — rejected files are never retried (violates `sync-manifest-update-on-success-only.rule.md`; `resp.Accepted` never consulted). Confirmed independently by three reviewers; the missing tests (mock `err`/`Errors` fields never exercised) mask it.
6. `internal/sync/hash.go:44`: `ScanFiles` doesn't skip reserved `global/` (uses `WalkArchcoreFiles`, not `WalkArchcoreFilesSkipping`) — vendored read-only globals get pushed to the server as local docs.
7. `internal/api/client.go:49-56`: sync uses the 10s-total-timeout client; the 30s `NewAuthenticatedClient` "for sync" has zero production callers. Large first sync on slow uplink can never succeed. Also: `BaseURL` doesn't trim trailing slash of user-provided `archcore_url` (`//api/v1` → 404 on strict routers).
8. `cmd/sync.go:86-104`: `runSync` is dead code (gate returns "coming soon"); delete or wire when un-gating.

### Phase 3 — Durability & parser robustness

9. `templates/templates.go:222,231`: `SplitDocument` swallows `yaml.Unmarshal` errors → malformed frontmatter silently yields empty metadata; via `update_document.go:111-135` a status-only update then permanently erases title/tags. Also strip UTF-8 BOM (`templates.go:212` — BOM'd files lose all frontmatter). Fail the update when delimiters exist but YAML fails.
10. `internal/config/config.go:374-388`: `Save` is non-atomic (crash → truncated settings.json, every command fails). Mirror `SaveManifest` tmp+rename. Same for MCP document writes (`update_document.go:135`, `create_document.go:182`).
11. `internal/update/update.go:430,465`: extraction silently truncates at exactly 50 MB (`io.ReadAll(io.LimitReader(tr, max))` with no exceed check; checksum verified on the compressed archive only) → installs a broken binary over the working one. Use `limit+1` + error, as the download path already does (301-308). Also fsync before rename (`update.go:488,514`).
12. `cmd/status.go:165`: `checkFrontmatter` requires `"---\n"` — CRLF documents falsely reported "missing YAML frontmatter" while every other surface normalizes CRLF. Normalize + add the failing test.
13. `cmd/hooks_gemini_cli.go` + `cmd/update.go:42-67`: `archcore update` exits 0 on failed update (scripts can't detect); return the error. `cmd/hooks.go:109-116`: `--agent <hookless>` skips MCP install while auto-detect path installs it — unify.
14. `cmd/init.go:93-99,142-146`: server-unreachable aborts remaining init steps, contradicting the documented soft-fail convention (latent until cloud/on-prem init). Make it warn-and-continue.
15. Inconsistent corrupt-manifest handling: `list_relations.go:31-37` / `get_document.go:72` silently report "no relations"; `add_relation` errors; `search_documents` logs to stderr. Pick one (tool error) everywhere.

### Phase 4 — Performance (trigger: repos approaching ~500–1000 docs)

Measured (Apple M5, realistic corpus ~5.5 KB/doc, 2 rel/doc): today at 80 docs everything is <7 ms — NOT urgent. Walls, in arrival order:
16. `list_documents` unbounded output: 391 KB ≈ 98K tokens @1000 docs, 1.96 MB @5000. Add default cap + `limit`/offset. **Earliest wall, cheapest fix.**
17. Scan cache: every list/search/hook call re-reads + re-parses every doc (`buildDoc` `os.ReadFile` unconditional; scan = 45% of search CPU @5000; 26–257 MB allocs/call). mtime-keyed in-process cache; include globals (a 2000-doc global drags every consumer to 2000-doc cost); load settings once per request (currently 2–4× per call).
18. Relation enrichment O(d·N²): `search_documents.go:344` calls linear `RelationsFor` per matched doc pre-limit (+508 ms @10k vs control). Build a `map[path][]Relation` index once per call (~15 lines).
19. `get_document.go:72` is O(R): full `LoadManifest` parse per call (25 µs → 38.6 ms @10k). Manifest cache keyed on `.sync-state.json` mtime.
20. `create_document.go:213-234`: `populateNearbyDocuments` full-tree-reads every doc body to list ≤5 siblings — use `os.ReadDir` on the one directory.
Sync path and CLI startup measured healthy (sync 162 ms @5000 first push; startup 4 ms; hook 6 ms @80 docs, crosses 150 ms only at ~7–8k docs via fix 17).

### Phase 5 — Tests, dead code, consistency polish

21. Test gaps (HIGH first): `doSync` error/partial-failure paths (pins fix #5); `checkFrontmatter` malformed-YAML branches + CRLF; `parseMtimeAfter` "24h" branch + overflow guards; `atomicReplace` rename-failure rollback; update-package cancellation/connection-refused; hooks auto-detect path; one concurrent `add_relation` integration test under `-race` (pins fix #2); copilot `WriteMCPConfig` wiring.
22. Dead code: `runSync`, `NewAuthenticatedClient`/`applyAuth`, `ListProjects`/`GetProject`, `sync.ParseFrontmatter`, `config.ValidSyncTypes`, `templates.ValidStatuses`/`ValidCategories`/`TypesByCategory`, `cmd/mcp.go:121-123` unreachable re-check, unused `hookInput` fields.
23. Consistency: import-group order unified package-wide in cmd/ (9 files diverge, and `go-code-quality.rule.md`'s own example contradicts its list — fix the doc too); `archsync` alias in `cmd/status.go` + `hooks_common.go`; `errors.New` for constant messages; `readErrorBody` → `io.LimitReader` (client.go:76-83); deterministic order for deleted-files listing (diff.go:54-61); typed enums for `Type`/`SourceKind`/searchMode/sortMode in tools (§G); validate `types` filter values like sibling filters; session-start header duplicate line + missing `search_documents` in tool list (hooks_common.go:49-51); `cleanVersion("dev")` → "vdev"; double-printed "N issue(s) found" in status/doctor; recognized errors printed to stdout (main.go:37); tag reorder on non-tag updates (update_document.go:126-130); `relations_removed` over-count (remove_document.go:92); MCP server version hardcoded "1.0.0" (server.go:187).
24. Stale docs to amend: `strict-go-naming-conventions.rule.md` "Known existing deviations" — all three (SyncType/DocStatus/Category) are RESOLVED; `sync-path-security.rule.md` still mandates category-prefix validation contradicting the free-form ADR; `read-path-scan-performance.idea.md` re-verified 2026-07-02, numbers still hold.

## Acceptance Criteria

- No MCP tool result or handler error contains an absolute filesystem path (grep + the existing leak-guard tests extended to OS-failure paths).
- Concurrent `add_relation` calls under `-race` lose no relations.
- `update_document`/`remove_document` reject non-`.md` and `SkipFiles` targets; write paths pass an `EvalSymlinks` containment check; `Global/` (any case) rejected.
- `archcore hooks install` run twice on a settings.json with foreign hooks + unknown fields is byte-identical after the first run.
- Sync (when un-gated): manifest updates only for `resp.Accepted`; `global/` never in payload.
- `go test -race ./...` stays clean; new tests from item 21 in place.
- Benchmarks re-run after Phase 4: search @5000 docs < 100 ms, list output bounded.

## Dependencies

- Phase 2 blocks removing the sync feature gate (`temporarily-disable-sync.adr.md`).
- Phase 4 items 17–19 build on the risks recorded in `read-path-scan-performance.idea.md` (verified current).
- Fix #4's shared helpers are the single place where the hooks-family Critical/High findings (field-stripping, atomicity, backup, panic) all land — do them together.
