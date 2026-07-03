---
title: "Read-Path Scan Performance: Scaling Risks and Optimization Directions"
status: draft
tags:
  - "mcp"
  - "performance"
  - "relations"
---

## Idea

Record the measured scaling behaviour of the MCP read path (`list_documents`,
`search_documents`, `get_document`) and a prioritised optimisation backlog, so
future performance work starts from data instead of guesses.

> **Status update (2026-07-03).** Candidates 1–3 below were implemented in the
> July 2026 audit follow-up; see the re-measurement table. The remaining open
> items are candidate 4 (source weighting in ranking — the `source_kind` field
> half is already shipped), candidate 5 (frontmatter-only reads — largely moot
> now that the scan cache removes repeat reads), and candidate 6 (per-source
> global budgets).

The read path is backed by a two-phase filesystem scan (`scanDocuments`,
@internal/mcp/tools/common.go). Since 2026-07 it has an **mtime+size-keyed
per-file cache** (@internal/mcp/tools/scan_cache.go): the walk still runs every
call (adds/removes detected by enumeration), but on a warm scan no file is
re-read or re-parsed. `get_document` and `search_documents` read the relation
manifest through a cached store (@internal/mcp/tools/manifest_store.go) keyed on
`.sync-state.json` (mtime, size), and search builds a per-call relation index
instead of a linear `RelationsFor` scan per matched document. `list_documents`
is paginated (default 100, max 500) and returns a `{documents, total, offset,
returned, truncated}` envelope.

## Value

- The earliest wall (list token output) arrived at ~500–1000 docs; the default
  list cap now bounds it, and `truncated` tells the agent to refine or page.
- The measurements separate three independent cost drivers (doc count, doc size,
  relation density) that hit different tools, so optimisation effort can be
  aimed precisely instead of "make it faster" broadly.
- Reproducible: two benchmark harnesses are committed under an env guard
  (excluded from CI), so any future change can be re-measured against the same
  baseline.

## Measurements

Apple Silicon, SSD, warm cache. Harnesses (kept under the `ARCHCORE_SCALING`
env guard + `-bench`, so plain `go test ./...` skips them):

- @internal/mcp/tools/scaling_bench_test.go — synthetic 1.8 KB bodies, no relations.
- @internal/mcp/tools/realistic_bench_test.go — corpora replicating this repo's
  real docs (avg 5.46 KB) at real relation density (~2 relations/doc).

Reproduce:

```
ARCHCORE_SCALING=1 go test ./internal/mcp/tools/ -run TestRealisticOutputSizes -v
go test ./internal/mcp/tools/ -bench BenchmarkRealisticReadTools -benchmem -benchtime=15x
```

### Baseline 2026-06-12 (pre-optimisation) — ms/call, realistic corpus

| N | list | search-snip | search-full | get |
|---|------|-------------|-------------|-----|
| 100 | 2.1 | 4.3 | 4.9 | 0.39 |
| 1000 | 20 | 46 | 51 | 3.7 |
| 3000 | 65 | 153 | 167 | 11 |
| 10000 | 228 | 712 | 769 | 37 |

Memory at N=10000: list 254 MB, search-snip 377 MB per call. `list` output:
~97.5 tokens/doc unbounded (97K tokens at N=1000, 974K at N=10000).

### Re-measured 2026-07-03 (scan cache + relation index + manifest store + list cap)

| N | list | search-snip | search-full | get |
|---|------|-------------|-------------|-----|
| 100 | 0.8 | 2.6 | 3.2 | 0.03 |
| 1000 | 3.3 | 24 | 29 | 0.03 |
| 3000 | 9.8 | 72 | 88 | 0.04 |
| 10000 | 36 | 240 | 293 | 0.06 |

Memory at N=10000: list 30 MB (−88%), search-snip 93 MB (−75%) per call.
`get_document` is now O(stat) — ~30–60 µs flat at every N (was O(R), 37 ms at
N=10000). `list` output is bounded by the default cap regardless of N.

Remaining search cost above ~5000 docs is dominated by the content-substring
scan itself (`strings.ToLower` over cached bodies) — candidate 5 territory if
it ever matters; at 5000 docs search-snip is ~115–120 ms/call.

## Risks (updated walls)

| Wall | Was | Now |
|------|-----|-----|
| Token output (list) | ~500–1000 docs (unbounded) | closed — default cap 100/max 500 + `truncated` signal |
| Search CPU (relations) | ~1000–3000 docs (O(d·N²)) | closed — per-call relation index; enrichment is O(R) |
| Scan CPU / RAM | ~3000–10000 docs | pushed out ~6×: warm scan is walk+stat; ~36 ms list @10000 |
| get (manifest) | ~10000 docs (O(R)) | closed — cached manifest store, O(stat) |
| Search content scan | — | new earliest CPU wall: ~150 ms/call around ~6–7K docs |

### Relation density

Density still multiplies the manifest size (load on cache miss) and the
per-call index build, but no longer multiplies per-matched-document work.

### Global multiplier

Globals still add to N on the scan axis for every consumer call (their files
are cached like locals, but `CheckGlobalDir` probes remain per call — required
by @.archcore/globals/global-sources.spec.md §6.1 fail-fast). A bloated shared
global still taxes every consumer; candidate 6 remains open.

### Relevance (orthogonal to speed)

- **`search_documents` ranks local and global together with no source weight**
  (`sortResults`: specificity → type priority → mtime). As globals grow, a
  global can occupy the top-N and push an authoritative local doc below the
  result limit. The `local-overrides-global` precedence
  (@.archcore/globals/local-overrides-global.rule.md) is a reading convention
  applied *after* the fact, not a ranking input. **Still open.**
- `searchResult` now carries `source_id`/`source_kind`/`global`/`read_only`
  (the disambiguation half of candidate 4 is shipped; only ranking weight
  remains).

## Possible Implementation (status)

1. **Scan cache keyed by mtime.** ✅ Shipped 2026-07 as
   @internal/mcp/tools/scan_cache.go — (mtime, size)-keyed per-file cache,
   mutex-protected, write-handler invalidation, amortised pruning, globals
   covered; settings.json is loaded once per request.
2. **`list_documents` cap + pagination.** ✅ Shipped 2026-07 — `limit` (default
   100, max 500) + `offset`, envelope with `total`/`truncated`.
3. **Manifest index + cache.** ✅ Shipped 2026-07 — cached manifest store
   (also serialises mutations) + per-call relation index in search.
4. **Source weighting in search ranking.** ◐ Half-shipped: `source_kind` et al.
   are in `searchResult`; ranking weight for local-over-global remains open.
   Touches @.archcore/mcp/search-documents.spec.md.
5. **Metadata-prefilter before reading bodies.** Largely moot with the scan
   cache (files are read once, then served from memory); revisit only if the
   content-scan wall (~6–7K docs) becomes real.
6. **Per-source global handling.** Open. Must preserve the global-mandatory
   invariant in @.archcore/globals/global-sources.spec.md.

## Notes

- Baseline recorded 2026-06-12; re-verified unchanged 2026-07-02 during the full
  CLI audit; re-measured 2026-07-03 after the optimisation work landed.
- Re-run the harnesses after any change to the scan, manifest, or
  search-ranking paths.
- The harnesses are intentionally kept in-tree as the reproduction baseline for
  this idea; remove them only together with this document.
- Related token-efficiency work on tool definitions / system prompt (a different
  axis) is in @.archcore/cli/mcp-token-optimization.idea.md.