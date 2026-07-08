---
title: "Spec — Single Narrative, Generalized Sections, EARS + BCP 14 Notation"
status: accepted
tags:
  - "architecture"
---

## Context

`spec` was defined as the contract of a boundary other code calls — narrower than our own usage: several canonical specs (`agent-system`, `skills-system`, `plugin-architecture`) pin subsystem behavior with no external code consumer. A first fix introduced two switching profiles (contract/system) inside one type, but two narratives in one type is its own defect, and the owner rejected it. An industry survey (spec-format-industry-research rnd, 23/25 claims verified) showed: no surveyed tool maintains two shapes of one artifact; what Kiro/Spec Kit call a "spec" is a pre-code per-feature requirements bundle (Spec Kit's maintainer calls it a PRD) — Archcore `prd`/`srs` territory; and no surveyed tool defines a durable post-implementation behavior contract, which is exactly the niche Archcore's `spec` occupies.

## Decision

One `spec` type, one form — six fixed sections for every subject (boundary or feature):

1. **Purpose & Scope** — subject + who depends on it
2. **Surface** — interface and/or parts, states, field-drivers, `@path`-referenced (generalizes the former Contract Surface / Composition & States pair)
3. **Normative Behavior** — numbered lines
4. **Constraints & Invariants** — plain BCP 14 (EARS fits triggered behavior, not limits)
5. **Failure Behavior** — error/edge conditions with observable outcome (generalizes Error Handling / Failure & Edge Behavior)
6. **Conformance** — MAY close with one ≤5-line non-normative Given/When/Then block

Notation for numbered lines in sections 3 and 5: EARS clause order with BCP 14 keywords as the modal — `WHEN <trigger>, the <subject> MUST <response>`, plus WHILE / IF…THEN / ubiquitous forms; MUST/SHOULD/MAY graded per RFC 2119, uppercase-only per RFC 8174, MUST kept sparing per RFC 2119 §6.

Admission gate: "behavior others rely on right now" → spec; "what should we build and why" (stories, priorities, metrics) → prd/srs.

## Alternatives

- **Two profiles in one type** — implemented and reverted same-session: two narratives in one type; no industry precedent (every surveyed tool ships one template per artifact).
- **Narrow contract-only spec** — the pre-existing state; leaves subsystem/feature behavior homeless and contradicts our own canonical specs.
- **Plain RFC 2119 lines (no EARS clauses)** — loses the forced trigger/state statement, exactly where LLM agents guess wrong; EARS carries peer-reviewed defect reduction.
- **Pure EARS (shall-only)** — loses MUST/SHOULD/MAY grading.

## Consequences

- Backward compatible: existing `X MUST Y` lines are valid EARS Ubiquitous sentences — no migration; `Contract Surface`/`Error Handling` headings map 1:1 to `Surface`/`Failure Behavior` on next edit.
- The EARS+BCP14 hybrid is a synthesis, not a named standard (each half is; protocol RFCs combine them informally).
- Open: re-validate the ≤80-line cap after EARS clause expansion on real specs; consider wiring numbered requirements into `/archcore:audit --drift`.
- External coherence gap: the MCP server's type label still reads "spec — Contract of a depended-on boundary"; update to "normative behavior contract" wording in the next server release.
- Companion decision: feature-scale requirements stay in `prd` via a scope rule — see [[no-frd-type-prd-scope-rule]].