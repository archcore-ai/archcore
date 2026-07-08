---
title: "FRD Type Evaluation — Feature-Scale Requirements"
status: accepted
tags:
  - "architecture"
  - "vision"
---

## Goal

Test the hypothesis "prd is broad; smaller features would fit an frd (feature requirements document)" — maximally critically, with the null hypothesis "a new frd type is NOT justified".

## Questions

1. Is FRD an established artifact in any requirements standard or named practice?
2. How do modern orgs and AI spec tools document small features vs full products?
3. What would frd overlap with in Archcore's existing taxonomy, and what new failure modes would it introduce?

## Approach

Deep-research workflow (2026-07-08): 5 angles including a dedicated skeptical/term-legitimacy angle, 19 sources, 89 claims extracted, 25 verified with 3 adversarial votes — 20 confirmed unanimously (3-0), 5 refuted. Primary sources: IEEE 830 / ISO 29148 standard pages, IIBA Business Analysis Standard, SVPG (Cagan) essays, Shape Up, Linear, kiro.dev, github/spec-kit.

## Findings

- **FRD exists in no standard.** IEEE 830's artifact is the SRS (functional requirements are a section inside it); its successor ISO 29148 defines exactly BRS/StRS/SyRS/SRS — the cascade Archcore already has. BABOK treats "functional requirements" as a classification, not a document, and names no FRD/PRD/BRD artifacts.
- **No stable PRD-vs-FRD boundary to route on.** Cagan (primary source): PRD/MRD/BRD/FSD distinctions "merged and morphed and lost many of their original distinctions". Historical "FRD" = *Functional* Requirements Document — the waterfall functional-spec role that `spec`/`srs` already occupy.
- **Industry solves "small feature" by varying scope and process weight of ONE artifact, never by a size-based type**: Kiro Quick Plan (same three files, no approval gates), Linear (scope threshold on one 1-2 page spec; below it, no artifact), Shape Up (pitch bounded by appetite — the structural inverse of an FRD).
- **Overlap analysis — zero residual sections**: vs prd ~85-90% (delta is size only), vs spec ~50-60%, vs srs ~40% (a feature-scoped srs IS the ISO-track "frd"), vs plan ~30%, vs urd ~25-30%, vs idea ~15%.
- **New failure modes if added**: prd-vs-frd routing hesitation (feature size is continuous), ISO-cascade confusion, split-brain requirements when a feature outgrows its frd, permanent token cost of one more type in every agent's routing context.
- Honest counterpoint: Kiro/Spec Kit unify stories + requirements + acceptance criteria in one per-feature file where Archcore composes prd→spec→plan — a flow-composition question, not a type gap.

## Recommendation

Do not add `frd` (high confidence, 20/20 unanimous claims). Instead: a scope rule on `prd` — size never changes type; a prd may cover a whole product or a single feature; compress (target ≤ 40 lines) rather than switch types; product-level prd links feature prds via relations — plus a feature-flow compression note ("vary the weight, never the types").

## Next Action

Implemented: scope rule added to `skills/plan/references/product-flow.md` and `feature-flow.md`. Revisit only if dogfooding shows agents producing product-weight prds for small features despite the rule.