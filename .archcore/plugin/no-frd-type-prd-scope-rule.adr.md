---
title: "No FRD Type — PRD Scope Rule Instead"
status: accepted
tags:
  - "architecture"
---

## Context

A hypothesis under review held that `prd` is broad, and that a smaller `frd` (feature requirements document) would fit feature-scale work better. The critical investigation recorded in `frd-type-evaluation.rnd`, where 20 of 20 claims verified unanimously, found that FRD exists in no standard — IEEE 830's artifact is the SRS, ISO 29148 defines exactly the `brs → strs → syrs → srs` cascade Archcore already carries, and BABOK names no FRD, PRD, or BRD artifact — and that Cagan confirms the PRD/MRD/BRD/FSD distinctions have collapsed, leaving no stable boundary an agent could route on. Every surveyed practice (Kiro Quick Plan, Linear's scope threshold, Shape Up's appetite) handles a small feature by varying the scope and process weight of one artifact rather than by adding a size-based type. Overlapping `frd` candidate content against the existing types left zero residual sections: roughly 85–90% covered by `prd`, 50–60% by `spec`, 40% by `srs`, and 30% by `plan`.

## Decision

Add no `frd` type, and encode granularity instead as a scope rule on `prd`: one `prd` covers one unit of product decision — a whole product or a single feature — with a feature-scoped `prd` using the same four sections compressed to a target of 40 lines or fewer, a product-level `prd` linking its feature-scoped children through relations, and routing that stays content-kind-based (why and what-outcome → `prd`, normative behavior → `spec`, execution → `plan`) rather than size-based.

## Alternatives Considered

1. **Add an `frd` type** — rejected because it has no standards basis and roughly zero residual content, and because it would introduce `prd`-versus-`frd` routing hesitation, ISO-cascade confusion, split-brain requirements as a feature grows, and a permanent token tax on every agent's routing context, all against the deliberate surface-collapse direction.
2. **A unified per-feature document, along the lines of Kiro's `requirements.md`** — rejected as a flow-composition question rather than a type question: `prd → spec → plan` already composes that document, and merging them would blur the vision and knowledge boundary. This was the strongest counterargument considered.

## Consequences

- The feature flow may compress for a small, well-understood feature — a short `prd`, or straight to `spec` when the "why" is already recorded upstream — so the weight varies while the types stay fixed.
- The scope rule and the compression path live in `plugins/archcore/skills/_shared/prd-contract.md`, and the `sdd.require` gate in `plugins/archcore/skills/_shared/tracks/sdd.md` references both. Their original homes, `skills/plan/references/product-flow.md` and `skills/plan/references/feature-flow.md`, were deleted at the v2 track cutover, which carried the 40-line target into `sdd.require` and dropped the compression path; `prd-spec-plan-content-ownership.adr` restored it.
- Tradeoff: an author working at feature scale must compress a `prd` deliberately, because nothing in the type system signals the smaller scope. [assumption] Whether authors do this without prompting has not been measured.
- The behavior side of the same boundary is governed by the spec admission gate recorded in `spec-single-narrative-ears-bcp14.adr`, and the statement-level split across `prd`, `spec`, and `plan` by the content-kind ownership table recorded in `prd-spec-plan-content-ownership.adr`.

## Superseded when

- Dogfooding shows agents producing product-weight PRDs for small features despite the scope rule.
- A strict two-tier hierarchy is adopted in which a product-level `prd` may not carry feature requirements, which would leave feature requirements without a home.
