---
title: "Delta-Routing Conductor — Claim-Level Selection Contract for /archcore:plan"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines the conductor — the claim-level selection the `plan` skill runs after shared context resolution, in @plugin/plugins/archcore/skills/_shared/delta-routing.md. Normative for the `plan` skill. Consumed by the `document` and `review` skills, which share the context record of @plugin/plugins/archcore/skills/_shared/gate-contract.md, and by the matrix bench @plugin/test/behavioral/matrix-bench.py. Out of scope: instrument internals (the instrument-layer spec), gate execution, closeout discharge mechanics, and the `archived` status value.

## Surface

Controlled vocabulary — one term per concept, no synonyms:

- canon — the accepted document graph in `.archcore/`, local and global.
- capability — one behavior an external consumer relies on, recordable as one `spec` within the caps of `@plugin/plugins/archcore/skills/_shared/spec-contract.md`.
- context record — the read-only result of context resolution: intent, scope, work time, certainty, volume, claims, authority, unknowns, hazards, conflicts, candidates.
- claim — one unit of change: subject and consumer, baseline owner, proposed claim, evidence, verification, and one action `create`, `amend`, `retire`, or `none`.
- claim ledger — the claims of one invocation; a produced `plan` records it under `## Declared Delta`.
- unknown — one open question with kind (`fact`, `choice`, `feasibility`, `acceptance`), evidence state, candidate sources, and next action.
- hazard — trigger → consequence → required check → owner of the check.
- document-need test — the future reader task, missing content kind, owner, evidence, and reason no existing document holds the claim.
- selection line — one report line: instruments in order, owners, hazard checks, and open unknowns.
- state carrier — the `archcore:track` block per the gate contract, extended with the `claims:` field.
- Retired: route names, size labels, the Δ, Π, M, and R fields, and the expert invocation map.

## Normative Behavior

1. WHEN context resolution completes, the conductor MUST run the document-need test on every claim before any question and any document creation.
2. WHEN a claim fails the document-need test, the conductor MUST create no document for it.
3. WHEN every claim fails the document-need test, the conductor MUST name the next action instead of a document.
4. WHEN accepted canon contradicts a claim, the conductor MUST report the authority conflict before any verdict or write.
5. WHEN a claim extends an accepted `spec` without contradicting it, the conductor MUST amend that `spec`.
6. WHEN a claim names depended-on behavior that no `spec` covers, the conductor MUST select one `spec` through the contract instrument.
7. WHEN an outcome spans more than one contract, the conductor MUST select one `prd`.
8. The conductor MUST NOT select an `idea` beside a `prd` in one invocation.
9. WHEN the claims hold two or more dependent tasks, the conductor MUST select one `plan` through the decompose instrument.
10. WHEN the request names alternatives and reasons for a settled choice, the conductor MUST select an `adr` through the decision instrument.
11. WHEN a claim introduces a repeatable human operation, the conductor MUST select one `guide` through the runbook instrument.
12. WHEN a hazard is evidenced, the conductor MUST name its required check in the selection line.
13. The conductor MUST engage iso links only for the levels a named traceability obligation requires.
14. WHEN an unknown needs research, a spike, or a decision, the conductor MUST engage the matching instrument per the instrument-layer spec.
15. The conductor MUST report the selection line before invoking an instrument.
16. The conductor MUST NOT print a route name or a size label.
17. The conductor MUST NOT ask the user to choose an instrument.
18. WHEN decomposition surfaces a claim outside the ledger, the conductor MUST add it and re-report the selection line before continuing.
19. WHEN the selection produces a `plan`, the conductor MUST record the claim ledger in that `plan` under `## Declared Delta`.
20. WHEN a capability meets the illustrate condition, the conductor MUST add one `scenario` for that capability.
21. WHEN the actor-subject probe returns other than `yes`, the conductor MUST drop the illustrate instrument and report the required version once.
22. WHEN a resumed block carries legacy `route:` and `delta:` fields, the conductor MUST read `delta:` as the claim ledger.

## Constraints & Invariants

- Invariant: every produced document carries one of the 23 types supported on CLI 0.8.4 — 21 on CLI 0.8.3, before the actor-subject vocabulary — or a legacy type below its engine gate, and a status in `draft`/`accepted`/`rejected`.
- Invariant: a `plan` needs no `spec` or `prd`; a non-capability migration with a fixed task order receives a `plan` alone.
- Invariant: a security hazard engages its required check; it engages requirement documents only under a named traceability obligation.
- Constraint: a document-need result names a reader, a consumer, or an observable use in the request, code, or graph.
- Constraint: claims that share one owner and one verification merge into one claim.
- Constraint: capability granularity binds through the granularity contract under `_shared/`.
- Constraint: `retire` claims route to closeout discharge; the conductor performs no status transition.
- Constraint: the illustrate condition reads the claim and grounding only; the conductor never asks whether to illustrate.

## Failure Behavior

1. IF context resolution cannot settle a claim, THEN the conductor MUST record it as an unknown instead of guessing.
2. IF a claim's owner carries an unresolved staleness flag, THEN the conductor MUST mark the dependent claims `[assumption]` and continue.
3. IF an `amend` claim names a capability no spec covers, THEN the conductor MUST engage the describe instrument in callable mode first.
4. IF the question ceiling exhausts before `choice` unknowns resolve, THEN the conductor MUST record the remainder under `deferred`.
5. IF a resumed state block lacks `claims:` and `delta:`, THEN the conductor MUST recompute the claims from the recorded gate and clarifications.
6. IF recomputed claims contradict the recorded ones, THEN the conductor MUST surface both and ask one confirmation question.
7. IF grounding cannot decide the illustrate condition, THEN the conductor MUST record it as a `choice` unknown.

## Conformance

An implementation is conformant when behaviors 1–22 hold over the matrix bench cases, the invariants hold on every run, and the failure rules produce the stated outcomes. Non-normative example: Given "fix the Safari button overflow", When context resolution finds one `none` claim and no hazard, Then the conductor reports `selection: none` with the next action "implement directly" and creates no document.
