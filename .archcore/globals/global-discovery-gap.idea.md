---
title: "Global Source Discovery Gap: Read-Tool Cutoffs, Ranking Skew, and a Silent Session Context"
status: accepted
tags:
  - "globals"
  - "integrations"
  - "mcp"
---

## Idea

Declared global sources mount correctly but stay undiscoverable: `list_documents` cuts them by walk order, `search_documents` ranks them by signals unrelated to relevance, the SessionStart context reports zero bytes about a healthy source, and the MCP server instructions never mention that globals exist. A user report confirmed the effect: an agent working against a data-rich global did not pull expected org context. The proposal has four parts: a zero-read `GLOBALS` summary in the SessionStart context, source-aware paging in `list_documents`, ranking fixes in `search_documents`, and a globals paragraph in the server instructions.

## Value

- The failure is silent. `list_documents` reports `truncated: true` without saying that the cut removed one source entirely — the agent sees a plausible page and stops (@internal/mcp/tools/list_documents.go).
- The agent cannot compensate. The session context and the server instructions carry no evidence that a global corpus exists, so the agent has no reason to page deeper or to search for org vocabulary (@cmd/hooks_common.go, @internal/mcp/server.go).
- `local-overrides-global.rule` assumes the agent sees both documents and applies precedence. Precedence never runs on a document that never surfaces (@.archcore/globals/local-overrides-global.rule.md).

## Measurements

Measured 2026-08-14 with an in-process harness over `HandleListDocuments` and `HandleSearchDocuments`: a synthetic primary plus a sibling global source, ~1.2 KB bodies, mixed document types, a unique needle term placed controllably (10% of bodies, 2 titles per corpus).

### list_documents: visibility is a function of local corpus size only

The scan returns one flat list — local first, then each global (@internal/docs/scan.go, `scanAll`). The handler slices `filtered[offset:end]` with no sort (@internal/mcp/tools/list_documents.go, `listDefaultLimit` 100, `listMaxLimit` 500).

| Local docs | Default call | `limit=500` |
| --- | --- | --- |
| < 100 | globals visible | globals visible |
| 100–499 | 0 globals | globals visible past the local block |
| >= 500 | 0 globals | 0 globals — unreachable at any limit |

Content placement changed nothing: with 500 local docs the default page held 0 of 500 global docs whether the needle sat in local, global, or both. The only recovery is a manual `offset >= local count`, which the agent has no way to derive.

### search_documents: matching works, ranking keys do not

All rows 500 local / 500 global unless noted. "Top-50" is the default snippets page; "full" is `mode=full` with its default limit 3.

| Scenario | Global share of matches | Top-50 globals / first rank | Full-mode globals |
| --- | --- | --- | --- |
| Needle only in global | 52 of 52 | 50 / #1 | 3 of 3 |
| Needle in both | 52 of 104 | 23 / #2 | 1 of 3 |
| Global has no title hits | 50 of 102 | 23 / #11 | 0 of 3 |
| Global freshly cloned (mtime today) | 52 of 104 | 27 / #1 | 2 of 3 |
| Global is doc-heavy | 52 of 104 | 14 / #3 | 1 of 3 |

- The core user scenario — content exists only in the global — ranks perfectly. The matching path is healthy.
- Specificity is binary: 3 for a title hit, 1 for a body hit (@internal/mcp/tools/search_documents.go, `extractContentMatch`, since removed). Removing title hits dropped the first global from #2 to #11 and emptied full mode.
- Ties break on `typePriority` then `mtime` (@internal/mcp/tools/search_documents.go, `sortResults`). A vendored global's mtime is its clone date, not a relevance signal: the same content scored 14, 23, or 27 top-50 slots depending on document types and clone recency.
- Match frequency carries no weight: `extractContentMatch` returns on the first occurrence, so a document with 30 hits ranks equal to one with 1.

### SessionStart and server instructions: zero bytes about a healthy global

- `buildSessionContext` scans local only and uses `InspectGlobals` solely for fatal and empty warnings (@cmd/hooks_common.go). On this repository the context reported `CORPUS: 102 documents` while the mounted corpus held 144.
- `countGlobalDocs` already walks every source and counts documents by filename, with no file reads; the count is discarded for a healthy source (@internal/docs/inspect.go).
- `mcpServerInstructions` contains no mention of global sources (@internal/mcp/server.go).

## Possible Implementation

1. **SessionStart `GLOBALS` block, zero-read.** One line per healthy source: id, document count, per-category counts, and top-level directory names — all derivable from filenames and directory names inside the walk `countGlobalDocs` already performs. Example: `- archcore — 42 docs (knowledge 40, vision 1, experience 1) · concepts/ 14, product/ 14`. Ceilings: 8 sources, 6 directories per source, truncation named (@.archcore/code-quality/bounded-and-deterministic-output.rule.md).
2. **`list_documents` source awareness.** A `by_source` count map in the response envelope, so a truncation that removes a whole source becomes visible; a `source` filter parameter; a per-source page quota so the default page always carries every mounted source.
3. **`search_documents` ranking.** A reserved global slot in `mode=full`; intermediate specificity between title and body hits (heading hits, frequency); neutralize the `mtime` tiebreak for global documents, whose mtime is a clone artifact.
4. **Server instructions.** `buildInstructions` appends a short globals paragraph when `config.ReadGlobals(baseDir)` returns a non-empty list: what a global is, that `source_kind` marks it, and that local documents take precedence.

## Cost Analysis: Many Local x Many Global

The hook is a short-lived process; `sharedScanCache` is in-process (@internal/docs/cache.go), so any content read of the global corpus repeats on every session start.

| Option | Data source | Extra I/O per session start | Tokens | Scaling |
| --- | --- | --- | --- | --- |
| id + count | `GlobalInspection.Docs`, already computed | none | ~15–25 per source | O(sources) |
| + category split | filename-derived, same walk | none | +~10 per source | O(sources) |
| + top-level dirs | dirname-derived, same walk | none | +~10–20 per source, capped | O(sources) |
| global tags | frontmatter of every global doc | full read of the global corpus (5000 docs ≈ 10 MB + YAML each start) | +~30–60 | rejected — O(global corpus) |
| global doc titles | same reads + selection heuristic | competes with the 24-line recap budget (@cmd/hooks_common.go, `maxRecapDocs`) | — | rejected — O(global corpus) |

The filename-derived summary is constant in both corpus sizes and linear only in the source count, which the user declares by hand. Content-derived summaries do not scale and stay behind the MCP read tools — which is what makes the `list_documents` quota (item 2) part of the same fix rather than a separate one.

## Risks

- `global-sources.spec` held the invariant that the SessionStart context operates on local documents only; amended 2026-08-14 together with `session-start-context.spec` and `globals-are-read-only-everywhere.rule` (see Outcome).
- `TestBuildSessionContext_ScansTheCorpusOnce` counts corpus walks; the summary must reuse the `InspectGlobals` walk, not add a `docs.Scan` (@cmd/hook_scan_budget_test.go).
- Every static line in the session context is paid on every session start; the block must stay inside the ceilings above (@.archcore/cli/mcp-token-optimization.idea.md).
- The pagination-with-globals interaction had no test coverage, which is why the cutoff shipped unnoticed; closed by @internal/mcp/tools/list_interleave_test.go.

## Outcome

Implemented 2026-08-14, all four parts:

1. `GLOBALS` block — contract in @.archcore/globals/session-globals-disclosure.spec.md, implementation in @cmd/hooks_common.go, tests in @cmd/hooks_globals_block_test.go.
2–3. Read-tool recall — decided in @.archcore/mcp/global-recall-guarantees.rfc.md, contract amended in @.archcore/mcp/search-documents.spec.md, tests in @internal/mcp/tools/search_recall_test.go and @internal/mcp/tools/list_interleave_test.go.
4. Server instructions paragraph — @internal/mcp/server.go `buildInstructions`.

Post-fix rerun of the harness: with 500 local docs the default `list_documents` page carries the proportional global share (0 → 50 rows at 500/500); the no-title-hit scenario keeps one full-mode global slot (0 → 1); the clone-recency ranking skew is gone (27 → 23 top-50 slots, equal to the aged mount).