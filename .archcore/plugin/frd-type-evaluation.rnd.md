---
title: "FRD Type Evaluation — Feature-Scale Requirements"
status: accepted
tags:
  - "architecture"
  - "vision"
---

## Goal

Test the hypothesis that `prd` is broad and that a smaller `frd` — a feature requirements document — would fit a smaller feature, as critically as possible, against the null hypothesis that a new `frd` type is not justified.

## Questions

1. Is FRD an established artifact in any requirements standard or named practice?
2. How do organizations and AI spec tools document a small feature today, as opposed to a full product?
3. What would `frd` overlap with in Archcore's existing taxonomy, and what new failure modes would it introduce?

## Approach

Deep-research workflow on 2026-07-08 across 5 angles, one of them dedicated to skepticism and term legitimacy. It fetched 19 sources, extracted 89 claims, and verified 25 with 3 adversarial votes each: 20 were confirmed unanimously and 5 refuted. Primary sources were the IEEE 830 and ISO 29148 standard pages, the IIBA Business Analysis Standard, the SVPG essays by Cagan, Shape Up, Linear, kiro.dev, and github/spec-kit.

## Findings

- **FRD exists in no standard.** IEEE 830's artifact is the SRS, in which functional requirements are a section. Its successor ISO 29148 defines exactly the BRS, StRS, SyRS, and SRS cascade that Archcore already carries. BABOK treats functional requirements as a classification rather than a document, and names no FRD, PRD, or BRD artifact.
- **No stable PRD-versus-FRD boundary exists to route on.** Cagan states as a primary source that the PRD, MRD, BRD, and FSD distinctions "merged and morphed and lost many of their original distinctions". The historical FRD is the *Functional* Requirements Document — the waterfall functional-spec role that `spec` and `srs` already occupy.
- **Industry handles a small feature by varying the scope and process weight of one artifact, never by adding a size-based type.** Kiro's Quick Plan keeps the same three files and drops the approval gates. Linear applies a scope threshold to one spec of 1–2 pages and produces no artifact below it. Shape Up bounds a pitch by appetite, which is the structural inverse of an FRD.
- **Overlap analysis leaves zero residual sections.** Against `prd` the overlap is roughly 85–90%, where the delta is size alone; against `spec` roughly 50–60%; against `srs` roughly 40%, since a feature-scoped `srs` *is* the ISO-track equivalent; against `plan` roughly 30%; against `urd` roughly 25–30%; and against `idea` roughly 15%.
- **Adding the type would introduce four new failure modes:** routing hesitation between `prd` and `frd`, because feature size is continuous; confusion with the ISO cascade; split-brain requirements once a feature outgrows its `frd`; and the permanent token cost of one more type in every agent's routing context.
- **Honest counterpoint.** Kiro and Spec Kit unify stories, requirements, and acceptance criteria in one per-feature file where Archcore composes `prd → spec → plan`. That is a flow-composition question rather than a type gap.

## Recommendation

Do not add `frd`, at high confidence on 20 of 20 unanimous claims. Encode granularity instead as a scope rule on `prd`: size never changes type; a `prd` may cover a whole product or a single feature; a feature-scoped `prd` compresses to a target of 40 lines or fewer rather than switching types; and a product-level `prd` links its feature-level children through relations. Add a feature-flow compression note stating that the weight varies and the types do not.

## Next Action

Implemented. The scope rule was added to `skills/plan/references/product-flow.md` and `skills/plan/references/feature-flow.md`. Revisit only if dogfooding shows agents producing product-weight PRDs for small features despite the rule.
