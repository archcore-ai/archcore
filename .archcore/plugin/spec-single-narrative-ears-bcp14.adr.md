---
title: "Spec — Single Narrative, Generalized Sections, EARS + BCP 14 Notation"
status: accepted
tags:
  - "architecture"
---

## Context

`spec` had been defined as the contract of a boundary that other code calls, which is narrower than the repository's own usage: several canonical specs — `agent-system`, `skills-system`, `plugin-architecture` — pin subsystem behavior with no external code consumer. A first fix introduced two switching profiles, contract and system, inside one type; the owner rejected it, because two narratives in one type is its own defect. The industry survey in `spec-format-industry-research.rnd`, with 23 of 25 claims verified, showed that no surveyed tool maintains two shapes of one artifact, that what Kiro and GitHub Spec Kit call a "spec" is a pre-code per-feature requirements bundle which Spec Kit's own maintainer calls a PRD — Archcore `prd` and `srs` territory — and that no surveyed tool defines a durable post-implementation behavior contract, which is exactly the niche Archcore's `spec` occupies.

## Decision

Adopted one `spec` type in one form: six fixed sections for every subject, boundary or feature — Purpose & Scope, Surface, Normative Behavior, Constraints & Invariants, Failure Behavior, Conformance — with numbered lines in Normative Behavior and Failure Behavior written in EARS clause order using BCP 14 keywords as the modal.

Section 2 generalizes the former Contract Surface and Composition & States pair. Section 4 takes plain BCP 14 statements, because EARS fits triggered behavior rather than limits. Section 5 generalizes the former Error Handling and Failure & Edge Behavior. Section 6 may close with one non-normative Given/When/Then block of 5 lines or fewer.

The notation is `WHEN <trigger>, the <subject> MUST <response>`, plus the WHILE, IF…THEN, and ubiquitous forms; MUST, SHOULD, and MAY are graded per RFC 2119, uppercase-only per RFC 8174, with MUST kept sparing per RFC 2119 §6. Three strict-EARS rules — an active obligated subject, one modal per line, and an explicit WHEN for an event response — are specified in `spec-contract.md` and softly enforced by `check-precision` checks 7b and 7c. The admission gate is: behavior others rely on right now goes to `spec`; what should we build and why, including stories, priorities, and metrics, goes to `prd` or `srs`.

## Alternatives Considered

1. **Two profiles inside one type** — implemented and reverted in the same session, because it put two narratives in one type and no surveyed tool ships more than one template per artifact.
2. **A narrow contract-only spec**, the pre-existing state — rejected because it leaves subsystem and feature behavior homeless and contradicts the repository's own canonical specs.
3. **Plain RFC 2119 lines with no EARS clauses** — rejected because it loses the forced trigger and state statement, which is exactly where LLM agents otherwise guess, while EARS carries peer-reviewed defect reduction.
4. **Pure EARS using `shall` only** — rejected because it loses the MUST, SHOULD, and MAY grading.

## Consequences

- Backward compatible: an existing `X MUST Y` line is a valid EARS ubiquitous sentence, so no migration is required, and the `Contract Surface` and `Error Handling` headings map one-to-one onto `Surface` and `Failure Behavior` on the next edit.
- The EARS and BCP 14 hybrid is a synthesis rather than a named standard, though each half is standardized; protocol RFCs combine the two informally in the same way.
- Tradeoff: the 80-line body cap has not been re-validated after EARS clause expansion on real specs, and several existing specs exceed it after conversion.
- Tradeoff: an external coherence gap remains — the MCP server's type label still reads "spec — Contract of a depended-on boundary" and needs the "normative behavior contract" wording in the next server release.
- The companion decision keeping feature-scale requirements in `prd` through a scope rule is recorded in `no-frd-type-prd-scope-rule.adr`.

## Superseded when

- Measurement on the converted corpus shows the 80-line cap unreachable for a majority of specs without losing normative content, which would call for a revised cap rather than decomposition.
- Numbered requirements are wired into `/archcore:review --drift`, which would make requirement identifiers load-bearing and may require stable IDs this decision does not define.
