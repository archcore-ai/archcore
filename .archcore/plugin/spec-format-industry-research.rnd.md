---
title: "Spec Format in AI-Assisted Development — Industry Survey"
status: accepted
tags:
  - "architecture"
  - "vision"
---

## Goal

Determine the best format for Archcore's `spec` type by surveying how spec-driven development tools and current practice define and structure a "spec" — resolving the tension between one narrative (no profile switching), requirements staying in `prd`/`srs`, and alignment with industry perception of "spec".

## Questions

1. What does each tool call a "spec" — requirements, design, behavior contract, or a bundle?
2. What notation wins for normative lines — EARS, RFC 2119, Given/When/Then?
3. Does the "feature behavior" case belong in `spec` or in `prd`/`srs`?

## Approach

Deep-research workflow (2026-07-08): 5 search angles, 20 sources fetched, 99 claims extracted, top 25 adversarially verified with 3 independent votes each — 23 confirmed, 2 refuted. Primary sources preferred: kiro.dev docs, github/spec-kit templates, alistairmavin.com/ears, RFC 2119/8174, OpenAI Model Spec repo, martinfowler.com SDD series.

## Findings

- **Industry "spec" ≠ Archcore spec.** Kiro's spec is a per-feature bundle (requirements.md with user stories + EARS criteria, design.md, tasks.md); Spec Kit's spec.md is user scenarios + FR-### + success criteria, and its maintainer explicitly calls it a PRD — both are pre-code requirements artifacts in Archcore's `prd`/`srs` territory, with tech detail banned ("Avoid HOW") and pushed to plan/design artifacts.
- **No surveyed tool defines a durable post-implementation behavior contract.** Archcore's `spec` fills a gap, not a misalignment.
- **Sentence-level consensus:** structured natural language in versioned markdown (EARS at Kiro; MUST-keyed FR-### + Given/When/Then at Spec Kit; plain markdown for the OpenAI Model Spec), with testability and traceability as stated goals. Notation itself is contested — the three major artifacts each chose differently.
- **No tool maintains two shapes of one artifact** — one template, varying content.
- EARS: peer-reviewed defect reduction (Mavin et al. RE'09; Springer 2025), near-zero training cost; limits — unwieldy past ~3 preconditions, poor fit for NFRs (keep Constraints plain BCP 14). RFC 2119 §6: MUST used sparingly, only for interoperation/harm.
- Caveats: OpenSpec/Tessl/BMAD claims did not survive verification (base = 2 tools + standards); EARS benefit claims originate with its inventor; no controlled evidence yet on spec-format impact on LLM-agent accuracy.

## Recommendation

Single-narrative spec: keep the six-section spine with generalized **Surface** and **Failure Behavior** sections (no profiles), and adopt EARS clause order with BCP 14 keywords as the modal for numbered behavior lines — fully backward compatible since `X MUST Y` is a valid EARS Ubiquitous sentence. Keep feature-level breadth (stories, priorities, metrics) in `prd`/`srs` via an explicit routing gate.

## Next Action

Implemented in `skills/_shared/spec-contract.md` and the three spec-creating skills. Open: re-validate the 80-line cap on real specs after EARS clause expansion; consider tying numbered requirements to `/archcore:audit --drift`.