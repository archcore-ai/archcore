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
future performance work starts from data instead of guesses. **No changes are
proposed now** — at the current corpus size (78 docs) nothing here is urgent.
This is a reference for when (if) the corpus grows enough to warrant the work.

The read path is backed by a two-phase filesystem scan (`scanDocuments`,
@internal/mcp/tools/common.go) with **no caching**: every `list_documents` /
`search_documents` call re-walks `.archcore/` and re-reads every document
(local + all declared globals) from disk, re-parses frontmatter, and re-marshals
JSON. `get_document` reads one file but loads and parses the **entire** relation
manifest (`.sync-state.json`) on every call. The costs grow with corpus size
along several axes that bite at *different* scales.

## Value

- The earliest wall (list token output) arrives at ~500–1000 docs — roughly
  6–13× the current corpus — so this is a "next 1–2 years of growth" concern, not
  hypothetical.
- The measurements separate three independent cost drivers (doc count, doc size,
  relation density) that hit different tools, so optimisation effort can be
  aimed precisely instead of "make it faster" broadly.
- Reproducible: two benchmark harnesses are committed under an env guard
  (excluded from CI), so any future change can be re-measured against the same
  baseline.

## Background — how the read path works today (cost drivers)

1. **No scan cache.** `scanDocuments` runs in full on every list/search. Nothing
   is memoised between calls; an agent doing 10 searches in a session pays the
   full scan 10×. (The OS page cache softens disk I/O, but the walk, allocation,
   frontmatter parse, and JSON marshal are redone every call.)
2. **`buildDoc` always reads the full file** (@internal/mcp/tools/common.go),
   even for metadata-only `list_documents`. The `includeContent` flag only
   controls whether the body string is *retained*, not whether the file is
   *read*. Disk I/O is therefore identical for list and search; only heap
   retention differs.
3. **Two-phase scan re-walks every global on every call** (phase 2 in
   `scanDocuments`). A large shared global imposes its full per-call cost on
   *every consuming project*, on every list/search — even if the agent never
   reads a global doc. See @.archcore/globals/global-sources.spec.md.
4. **`RelationsFor` is an O(R) linear scan** over all relations
   (@internal/sync/manifest.go), called **once per matched document** inside
   `search_documents` (before the result limit is applied). With relation
   density d, R ≈ d·N, so search relation-enrichment is **O(N·R) = O(d·N²)**.
5. **`LoadManifest` parses the whole manifest per call** in `get_document` and
   `search_documents`. So `get_document` is **O(R)**, not O(1) — it inherits the
   manifest-parse cost even though it reads only one document.
6. **`list_documents` has no limit.** It returns every matching document,
   unbounded, at ~95–100 tokens/doc. (`search_documents` is bounded: 50 results
   in snippets mode, 3 in full mode — see @.archcore/mcp/search-documents.spec.md.)

## Measurements

Apple Silicon, SSD, warm cache. Recorded 2026-06-12. Two harnesses (kept under
the `ARCHCORE_SCALING` env guard + `-bench`, so plain `go test ./...` skips them):

- @internal/mcp/tools/scaling_bench_test.go — synthetic 1.8 KB bodies, **no
  relations** (isolates the scan term).
- @internal/mcp/tools/realistic_bench_test.go — corpora built by replicating this
  repo's **real** `.archcore/` docs (78 docs, avg 5.46 KB, p50 4.2 KB, max
  23.6 KB) at **real relation density (~2 relations/doc)**.

Reproduce:

```
ARCHCORE_SCALING=1 go test ./internal/mcp/tools/ -run TestRealisticOutputSizes -v
go test ./internal/mcp/tools/ -bench BenchmarkRealisticReadTools -benchmem -benchtime=15x
```

### Realistic timing (ms/call) — real docs, ~2 relations/doc

| N | list | search-snip | search-full | get |
|---|------|-------------|-------------|-----|
| 10 | 0.7 | 0.8 | 0.7 | 0.10 |
| 100 | 2.1 | 4.3 | 4.9 | 0.39 |
| 300 | 5.9 | 13.2 | 14.6 | 1.1 |
| 1000 | 20 | 46 | 51 | 3.7 |
| 3000 | 65 | 153 | 167 | 11 |
| 10000 | 228 | 712 | 769 | 37 |

### Realistic memory (MB/call) and output tokens (≈ output bytes ÷ 4)

| N | list mem / **tokens** | snip mem / tokens | full mem / tokens | get mem / tokens |
|---|-----------------------|-------------------|-------------------|------------------|
| 100 | 2.5 / **9.8K** | 4.0 / 10K | 5.3 / ~2K | 0.25 / 0.6K |
| 1000 | 25 / **97K** | 38 / 9.4K | 52 / ~3K | 2.3 / 0.6K |
| 3000 | 78 / **292K** | 113 / 9.4K | 157 / ~2K | 7 / 0.6K |
| 10000 | 254 / **974K** | 377 / 9.4K | 524 / ~2K | 25 / 0.6K |

`list` ≈ 97.5 tokens/doc, **unbounded**. `search`/`get` token output is bounded
by result limits regardless of N.

### What the synthetic-vs-realistic delta reveals

- **Relations make search superlinear.** Synthetic (R=0) search-snip at N=10000 =
  228 ms; realistic (R≈20000) = 712 ms — the +484 ms is pure O(d·N²) relation
  enrichment. The ratio grows with N (2.2× at N=1000 → 3.1× at N=10000),
  confirming the relation term overtakes the linear scan as N rises.
- **`get` is not flat.** Synthetic get ≈ 20 µs flat; realistic get = 37 ms at
  N=10000 (140K allocs, 25 MB) — entirely the `LoadManifest` parse of 20000
  relations. get is O(R).
- **`list` is doc-count-bound, not size-bound.** Real docs are 3× larger than
  synthetic, yet list time barely moved (213 → 228 ms at N=10000): list reads
  bodies but marshals only metadata, so per-doc fixed overhead dominates. The
  list **token** wall is therefore driven purely by doc *count*, not doc size.

## Risks

### Walls by tier

| Wall | Bites around | Cause |
|------|--------------|-------|
| **Token output (list)** | **~500–1000 docs** | 97.5 tok/doc, no cap; N≈2000 fills a 200K context window in one call; N≥3000 (~292K) cannot fit |
| **Search CPU (relations)** | **~1000–3000 docs** | O(d·N²) relation enrichment overtakes the scan; 153 ms @3000 → 0.7 s @10000 |
| **Scan CPU / RAM** | **~3000–10000 docs** | 65→228 ms and 78→254 MB per call, × no cache × repeated calls |
| **get (manifest)** | low until ~10000 | O(R) `LoadManifest`; 37 ms @10000, worse as density rises |

### Relation density is a second, independent driver

Knowledge graphs densify as they grow. The measurements assume ~2 relations/doc
(this repo: 78 docs → 160 relations). At 5–10 relations/doc, search and get on
N=10000 would be ~5× heavier: both the O(d·N²) search term and the O(d·N)
manifest-load term scale with density. Density hits search/get exactly where list
stays linear.

### Global multiplier

Globals add to N on the scan and list-token axes identically, for every consumer,
every call. A 2000-doc shared global puts even a 10-local-doc project into
"~3000-tier" list behaviour (~150K tokens per list call). The cost of a bloated
shared global is paid by all consumers, silently.

### Relevance (orthogonal to speed)

- **`search_documents` ranks local and global together with no source weight**
  (`sortResults`: specificity → type priority → mtime). As globals grow, a global
  can occupy the top-N and push an authoritative local doc below the result limit
  (50 snippets / 3 full). The `local-overrides-global` precedence
  (@.archcore/globals/local-overrides-global.rule.md) is a reading convention
  applied *after* the fact, not a ranking input.
- **`searchResult` carries no `source_kind`/`source_id`/`read_only`** (unlike
  `LocalDocument` in list/get). In search output the agent can only infer
  global-ness from the path shape — disambiguation degrades in exactly the tool
  that ranks, precisely as globals grow.

## Possible Implementation (candidates, not commitments)

Ordered roughly by impact-per-effort:

1. **Scan cache keyed by mtime.** Memoise `scanDocuments`; invalidate per-file on
   mtime change. Removes the re-read-everything cost from repeated list/search
   calls in a session — the single biggest win at every tier. (Watch: globals
   live outside the project; the cache must stat them too.)
2. **`list_documents` cap + pagination.** The only tool with unbounded token
   output. A default cap (with `limit`/offset, or a "refine your filter" signal)
   closes the earliest-arriving wall. Pair with a nudge toward filtered
   list / search.
3. **Manifest index + cache.** Replace linear `RelationsFor` with a prebuilt
   `map[path][]Relation` (outgoing + incoming), and cache the parsed manifest
   (invalidate on `.sync-state.json` mtime). Kills the O(d·N²) search term and the
   O(R) get cost in one move.
4. **Source weighting in search ranking + `source_kind` in `searchResult`.** Rank
   local above global (or at least expose the tag so an agent/judge can). Fixes
   both relevance risks; independent of the speed work. Touches
   @.archcore/mcp/search-documents.spec.md.
5. **Metadata-prefilter before reading bodies.** For queries filtering on
   type/status/mtime/tags, decide inclusion from frontmatter without retaining
   bodies (partly done via `needsBody`, but the file is still fully read — a
   frontmatter-only read for metadata queries would cut I/O and memory).
6. **Per-source global handling.** Skip globals for local-only queries, or give
   globals a separate budget so a bloated shared global can't dominate a
   consumer's every call. Must preserve the global-mandatory invariant in
   @.archcore/globals/global-sources.spec.md.

## Notes

- Measurements are point-in-time against the code at the recording commit. Re-run
  the harnesses after any change to the scan, manifest, or search-ranking paths.
- The harnesses are intentionally kept in-tree as the reproduction baseline for
  this idea; remove them only together with this document.
- Related token-efficiency work on tool definitions / system prompt (a different
  axis) is in @.archcore/cli/mcp-token-optimization.idea.md.
