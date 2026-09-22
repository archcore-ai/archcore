---
title: "Hybrid GraphRAG Indexing on FalkorDB: Asserted Spine + Extracted Domain, Federated Globals"
status: draft
tags:
  - "globals"
  - "graphrag"
  - "performance"
  - "relations"
  - "sync"
---

## Idea

Concretize `sync-design`'s server-side "GraphRAG indexing" on **FalkorDB** (openCypher + native vector/full-text indexes), settling the central question that design left open: **what in the graph is asserted (curated, trusted) vs inferred (LLM-extracted).** The design is **hybrid on two axes**:

- **Per document** — a deterministic *meta layer* (documents + the slim typed relation spine) plus an LLM-extracted *content/domain layer*, joined by provenance edges.
- **Across projects** — isolated per-project graphs (`graph_name = proj__<id>`) plus shared *canonical global graphs* (`global__<id>`, extracted once), stitched by server-**derived** boundary edges — not authored relations.

## Value

- Pure file-first: no semantics (substring only), no domain graph, 1-hop navigation, no cross-project; hits the read-path scan walls at ~3–10k docs.
- Pure GraphRAG (drop the spine, extract everything): discards the one hand-curated, typed, directed, git-reviewable signal and asks an LLM to re-derive it from prose — exactly the part that is *not* mechanically reconstructible (per `relation-write-path-graph-growth`). It also substitutes co-mention similarity for intentional typed relations, and loses relation reviewability in PR diffs.
- **Hybrid**: assert what is authored (spine + frontmatter), infer what is not (domain + semantics). Relations are cheap to store and expensive to reconstruct, so keeping the spine is economically correct. Unlocks semantic search, a domain graph, trusted multi-hop over the spine, and cross-project retrieval — and gives `read-path-scan-performance`'s half-done source-weighting a concrete home (the federated ranker). Preserves every existing invariant: one-way push, files = source of truth, `local-overrides-global`, globals read-only + un-relatable (authored), reviewable relations.

## Possible Implementation

**Two-node-population project graph** (`proj__<id>`), distinguished by label:

```
[META]   (:Doc {path,type,category,title,status,tags,source})
             --IMPLEMENTS|EXTENDS|DEPENDS_ON--> (:Doc)   // slim typed spine only
             | MENTIONS
[DOMAIN] (:Entity Payment) --PROCESSED_BY--> (:Entity Stripe)
             | REFERENCES
[CODE]   (:CodeSymbol PaymentService @internal/payments/stripe.go)
```

- **Meta layer (deterministic, zero-LLM)** from `.sync-state.json` + frontmatter. Feed only the *slim spine*: typed edges + cross-cluster `related`. Drop intra-directory `related` cliques — ~60% reconstructible from the tree (`relation-write-path-graph-growth` direction #4).
- **Content/domain layer (extracted)** from bodies. Deterministic-first: `@path`-refs + backticked identifiers → `:CodeSymbol`/`:File` with exact keys (reuses the existing path_ref parser, zero hallucination). LLM, schema-constrained per doc type → `:Entity` domain concepts + typed domain relations. Provenance: `(:Doc)-[:HAS_CHUNK]->(:Chunk)-[:MENTIONS {confidence,span}]->(:Entity)`; no entity without a MENTIONS back to source. Vector index on `:Chunk`, full-text on `:Doc`/`:Entity`.

**Multi-project topology**:

```
FalkorDB
├── proj__ecommerce   (spine + domain/code)
├── proj__billing
├── global__org-standards        (canonical, extracted once, read-only)
└── global__security-baseline
```

`graph_name` per project = tenancy boundary; bounds `finalize()` to one project. Each global extracted once, keyed by source id so multiple consumers dedup to one graph.

**Globals — authored vs derived edges** (reconciles the `no-relate-to-globals` invariant with graph-native linking):

- *Authored* archcore relations (user-written, both-sides-queryable in sync-state) still cannot target a read-only, consumer-agnostic global — unchanged.
- *Derived* server edges (computed, stored consumer-side, one-directional) can and should link local→global; the global never records its consumers, so read-only + consumer-agnostic holds. Boundary edges: `(:Project)-[:MOUNTS]->(:GlobalSource)` from settings.json; `(:Doc)-[:OVERRIDES]->(:Doc @global)` — `local-overrides-global` expressed as an edge (topic/slug match), not a post-hoc ranking convention; `(:Entity @local)-[:SAME_AS|COMPLIES_WITH]->(:Entity @global)` — domain stitch via entity resolution.
- Storage = **hybrid federation**: global stays canonical in `global__<id>` (extract once, updates propagate free); at project sync the server projects the *referenced subset* of global nodes as thin read-only stubs + the derived boundary edges into `proj__<id>`, so local→global is a real traversable edge locally while duplication stays bounded by the boundary, not the global size. This is selective, automatic graph-level vendoring (cf. `vendoring-a-global`).

**Sync / write path**: project push (existing created/modified/deleted delta) → `apply_changes` + `finalize` on `proj__<id>`, batched once per sync (`finalize` = O(project graph)). Global owner pushes once; consumers refresh only thin stubs. `.sync-state.json` remains source of truth and the non-GraphRAG fallback.

**Retrieval / search path** for a query scoped to project P: (1) resolve P's declared globals; (2) fan-out vector+graph over `proj__P` ∪ its `global__*`; (3) merge + cross-boundary entity resolution; (4) precedence local > global on conflict (via `OVERRIDES`), dedup among globals; (5) rank with source-weight (local boost → global) + status/recency; (6) LLM completion, every fact tagged `source_id`. Cross-project search = same federation over multiple `proj__*`, gated by authorization.

**Engine**: FalkorDB GraphRAG SDK as the engine (native, ontology-first — accepts the archcore ontology as a supplied `GraphSchema`; `apply_changes`/`finalize` matches the sync delta 1:1; multi-hop via Cypher). Borrow only the *Statement/Fact summarisation tier* idea from the AWS Labs graphrag-toolkit as the model for the content layer — do not adopt the toolkit wholesale (its traversal retriever is unsupported on FalkorDB, its structure-discovery duplicates the authored spine, its infra is heavier).

## Risks

- **Extraction ontology per project** is unknown a priori (differs e-commerce vs compiler vs medtech): open-world (noisy) vs supplied `GraphSchema` (per-project authoring) vs auto-detect-then-curate. Anti-noise levers: closed schema, type-conditioned extraction, deterministic code-symbols first, confidence + `status: accepted` gating.
- **Determinism / reviewability**: the extracted layer is non-deterministic (re-extraction may drift) and not code-reviewable like a sync-state edge — acceptable only because it is a *derived index*, not the source of truth.
- **Only the documented domain**: the domain graph covers what docs *say*, not the full system; it is partial, not an ERD. A fuller domain graph would require extracting from code too, which is out of archcore's scope.
- **Cross-boundary entity resolution** quality: exact for code symbols, fuzzy for concepts.
- **Tie-break among multiple mandatory globals** is undefined in archcore (no priority ordering) — needs a rule before federation can resolve global-vs-global conflicts.
- **Freshness lag** (push→ingest→finalize) vs instant local file reads; **`finalize()` = O(graph size)** cost to budget once per sync.
- **Infra**: FalkorDB + LLM + embedder is a real operational step beyond the current zero-dependency CLI; server-side only — the local file-first path stays the offline, deterministic default.