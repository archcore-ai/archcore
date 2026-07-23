---
title: "No FRD Type — PRD Scope Rule Instead"
status: accepted
tags:
  - "architecture"
---

## Context

Hypothesis under review: "prd is broad; for smaller features an frd (feature requirements document) would fit". A critical investigation (frd-type-evaluation rnd; 20/20 claims verified unanimously) found: FRD exists in no standard (IEEE 830's artifact is the SRS; ISO 29148 defines exactly the brs→strs→syrs→srs cascade we already have; BABOK names no FRD/PRD/BRD artifacts); Cagan confirms the PRD/MRD/BRD/FSD distinctions collapsed — there is no stable boundary an agent could route on; and every surveyed practice (Kiro Quick Plan, Linear's scope threshold, Shape Up's appetite) handles small features by varying scope and process weight of one artifact, never by a size-based type. Overlap of frd's candidate content vs existing types leaves zero residual sections (prd ~85-90%, spec ~50-60%, srs ~40%, plan ~30%).

## Decision

Do not add an `frd` type. Instead, encode granularity as a scope rule on `prd`:

- A prd covers one unit of product decision — a whole product OR a single feature.
- **Size never changes type**: a feature-scoped prd uses the same four sections, compressed (target ≤ 40 lines).
- A product-level prd links its feature-scoped prds via relations.
- Feature flow may compress for small, well-understood features (short prd, or straight to spec when the "why" is recorded upstream) — vary the weight, never the types.

Routing stays content-kind-based, never size-based: why/what-outcome → `prd`; normative behavior → `spec`; execution → `plan`.

## Alternatives

- **Add `frd`** — no standards basis, ~zero residual content, and introduces prd-vs-frd routing hesitation, ISO-cascade confusion, split-brain requirements on feature growth, and a permanent token tax on every agent's routing context — against the deliberate surface-collapse direction.
- **Unified per-feature doc (à la Kiro requirements.md)** — the strongest counterargument; rejected as a flow-composition question: prd→spec→plan already composes it, and merging would blur the vision/knowledge boundary.

## Consequences

- Scope rule lives in `skills/plan/references/product-flow.md` and `feature-flow.md` (PRD steps); compression note in feature-flow Step 2.
- Revisit trigger: dogfooding shows agents producing product-weight prds for small features despite the rule, or a strict two-tier hierarchy is ever adopted where product prds may not carry feature requirements.
- Companion decision: the behavior side of the boundary is governed by the spec gate — see [[spec-single-narrative-ears-bcp14]].