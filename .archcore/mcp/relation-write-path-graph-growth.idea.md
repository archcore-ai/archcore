---
title: "Relation Write Path and Graph Growth: Scaling Structure and Optimization Directions"
status: draft
tags:
  - "component:cli"
  - "mcp"
  - "performance"
  - "relations"
---

## Idea

Record the scaling behaviour of the relation **write path** and the observed
graph-growth law, measured on a real corpus, so relation-storage work starts
from data instead of guesses. The read path (`list_documents` / `search_documents` / `get_document`) is a separate axis and already optimised.

Relations are stored as a flat JSON array in `.sync-state.json`
(@.archcore/relations/local-relations-in-sync-state.adr.md). Every mutation goes
through `manifestStore.mutate` (@cli/internal/mcp/tools/manifest_store.go): it deep-
clones the whole manifest, runs the op, and calls `SaveManifest`
(@cli/internal/sync/manifest.go), which re-serialises and rewrites the **entire**
file. `AddRelation` also linearly scans all relations to dedup. Because
relations are created by hand rather than auto-linked
(@.archcore/relations/no-auto-relations-on-create-document.adr.md), the dominant
usage pattern is bulk clique-building — which is exactly the pattern this write
model handles worst.

## Value

- Separates the write axis (mutation cost, graph growth) from the already-
  optimised read axis, so effort can be aimed at the part that is still naive.
- Gives the growth law an actual shape (`R ≈ ½·Σ kᵢ²` over leaf-directory sizes)
  instead of the pessimistic O(N²)-in-doc-count assumption, so the real
  controlling variable — max leaf-directory size — is named.
- Surfaces that ~59% of stored edges duplicate the filesystem tree, which is
  both a storage cost and a relevance dilution.

## Measurements

Real corpus: the `.archcore/` of one production monorepo — **153 docs, 513 directed
relations (506 undirected edges), `.sync-state.json` 81 KB**. Snapshot
2026-07-09. Graph-structure figures below are measured; the write-amplification
projection is modeled (linear extrapolation of the measured 162 bytes/relation).

| Metric | Value |
|--------|-------|
| Average vertex degree | 6.71 |
| Graph density | 4.35% of the N² ceiling (506 / 11 628) |
| Orphans (docs with no relation) | 0 |
| `related` vs typed edges | 469 / 44 (91% / 9%) |
| Edges within one directory | 66% |
| Max degree node | `auth/popup/architecture.doc.md` — 23 |

**Graph structure — relations form near-complete cliques inside leaf
directories** (three leaf folders are at 100% fill; the largest, `auth/popup`, holds 19 docs and 99 of 171 possible edges).

The decisive number: if *every* leaf directory were a full clique the graph
would have **497** undirected edges; it actually has **506**. To within 2%, the
relation graph is "each leaf folder is a clique" plus a handful of cross-folder
edges — i.e. it is almost entirely reconstructible from the directory tree.

**Growth law.** `R ≈ ½·Σ kᵢ²` over leaf-directory sizes `kᵢ` — NOT O(N²) in
total doc count:

- many small bounded folders → `Σkᵢ² ∝ N` → **linear** (healthy; here
  `Σk²/N = 7.5`, the effective average-degree ceiling)
- one folder that keeps accreting → its `k²` term dominates → **quadratic
  locally** (`auth/popup` at 19 docs already carries up to 171 edges; at 40
  docs ≈ 780 edges from one folder)

The controlling variable is **max leaf-directory size, not N**. There is no
guard: no per-node degree cap and no "relate to cluster" primitive.

**Edge signal.** 59% of edges are intra-directory `related` — they restate the
directory co-location the filesystem already encodes. Only 41% (cross-directory
or typed) carry graph signal beyond the tree.

## Risks

**Per-mutation cost is O(R) with a full-file rewrite.** Building `m` relations
into a manifest of current size `R₀` costs `Σ O(R₀+i) = O(m·R₀ + m²)` —
quadratic write amplification. Clique-building is the dominant usage pattern, so
the quadratic case is the common case, not the corner case.

| R | Manifest size | One `add_relation` rewrites |
|---|---------------|-----------------------------|
| 513 (measured) | 81 KB | 81 KB |
| 5,000 | ~790 KB | ~790 KB |
| 50,000 (`maxManifestRelations`) | ~7.9 MB | **~7.9 MB for a single edge** |

Each `add_relation` also does 2× `ReadDocumentContent` (endpoint existence) +
`loadGlobalsFailClosed` per call.

**`list_relations` has a cap but no pagination.** Since 2026-11-07
(`ffbb7645`) an unfiltered listing stops at 300 relations in manifest order and
returns `{total, truncated, relations}` (@cli/internal/mcp/tools/list_relations.go).
A listing filtered by `path` stays complete. No `offset` exists, so the
relations after the first 300 are reachable only one document at a time.

## Possible Implementation (priority order)

1. **Paginate `list_relations`** — the cap and `total` / `truncated` shipped;
   add `offset` and `returned` to mirror the `list_documents` envelope, so an
   audit can read the whole graph without one call per document.
2. **O(1) dedup in `AddRelation`** — a key set instead of the O(R) scan in
   @cli/internal/sync/manifest.go. Removes one quadratic factor of bulk builds.
3. **Batch `add_relations` (plural)** — one clone + one `SaveManifest` per
   batch. Kills write amplification for clique-building, which is how cliques
   are actually created.
4. **Derive same-folder adjacency from the tree** instead of storing it; reserve
   explicit relations for cross-cluster typed edges. Drops ~60% of edges and
   bends the growth law back toward linear. Touches the reading model, not just
   storage — design-level.
5. **Soft advisory** (doctor / tool response) when a leaf-directory clique or a
   node's degree crosses a threshold, before one folder pulls the graph
   quadratic.

## Notes

- Snapshot taken 2026-07-09 from that production monorepo's
  `.archcore/.sync-state.json`; re-run the analysis after any change to the
  relation storage or mutation path.
- The write-amplification rows are modeled, not benchmarked — a
  `SaveManifest` / `add_relation` write bench in the style of the read-path
  harnesses (@cli/internal/mcp/tools/realistic_bench_test.go) would turn them into
  measurements.
- Tracked as GitHub issue archcore-ai/cli#26.
