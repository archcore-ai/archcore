---
title: "Spec Format in AI-Assisted Development — Industry Survey"
status: accepted
tags:
  - "architecture"
  - "vision"
---

## Goal

Determine the best format for Archcore's `spec` type by surveying how spec-driven development tools and current practice define and structure a "spec", resolving the tension between keeping one narrative with no profile switching, keeping requirements in `prd` and `srs`, and aligning with the industry perception of the word.

## Questions

1. What does each tool call a "spec" — requirements, design, a behavior contract, or a bundle?
2. Which notation wins for a normative line: EARS, RFC 2119, or Given/When/Then?
3. Does the feature-behavior case belong in `spec`, or in `prd` and `srs`?

## Approach

Deep-research workflow on 2026-07-08 across 5 search angles. It fetched 20 sources, extracted 99 claims, and adversarially verified the top 25 with 3 independent votes each: 23 confirmed and 2 refuted. Primary sources were preferred — the kiro.dev documentation, the github/spec-kit templates, alistairmavin.com/ears, RFC 2119 and RFC 8174, the OpenAI Model Spec repository, and the martinfowler.com SDD series.

## Findings

- **The industry "spec" is not the Archcore spec.** Kiro's spec is a per-feature bundle of `requirements.md` with user stories and EARS criteria, `design.md`, and `tasks.md`. Spec Kit's `spec.md` is user scenarios, `FR-###` items, and success criteria, and its maintainer calls it a PRD explicitly. Both are pre-code requirements artifacts in Archcore's `prd` and `srs` territory, with technical detail banned and pushed into a plan or design artifact.
- **No surveyed tool defines a durable post-implementation behavior contract.** Archcore's `spec` fills a gap rather than misaligning with the field.
- **Sentence-level consensus is structured natural language in versioned markdown**, with testability and traceability as the stated goals — EARS at Kiro, MUST-keyed `FR-###` plus Given/When/Then at Spec Kit, and plain markdown for the OpenAI Model Spec. The notation itself is contested: each of the three major artifacts chose differently.
- **No tool maintains two shapes of one artifact.** One template, varying content.
- **EARS carries peer-reviewed defect reduction** (Mavin et al., RE'09; Springer 2025) at near-zero training cost. Its limits are that it becomes unwieldy past roughly 3 preconditions and fits non-functional requirements poorly, which is why constraints stay plain BCP 14. RFC 2119 §6 requires MUST to be used sparingly, only for interoperation or to prevent harm.
- **Caveats.** The OpenSpec, Tessl, and BMAD claims did not survive verification, so the evidence base is two tools plus the standards. The EARS benefit claims originate with its inventor. No controlled evidence yet measures spec-format impact on LLM-agent accuracy.

## Recommendation

Adopt a single-narrative spec: keep the six-section spine with generalized **Surface** and **Failure Behavior** sections and no profiles, and adopt EARS clause order with BCP 14 keywords as the modal for each numbered behavior line. The change is fully backward compatible, because `X MUST Y` is a valid EARS ubiquitous sentence. Keep feature-level breadth — stories, priorities, and metrics — in `prd` and `srs` behind an explicit routing gate.

## Next Action

Implemented in `skills/_shared/spec-contract.md` and in the three spec-creating skills. Two items stay open: re-validate the 80-line cap against real specs after EARS clause expansion, and consider tying numbered requirements into `/archcore:review --drift`.
