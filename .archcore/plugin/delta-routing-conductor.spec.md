---
title: "Delta-Routing Conductor — Route Computation Contract for /archcore:plan"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines the conductor — the route computation that replaces the fixed routing table in Step 2 of `@plugins/archcore/skills/plan/SKILL.md`. Normative for the `plan` skill. Consumed by the `document` and `review` skills, which share the Δ vocabulary, and by the behavioral routing tests of rollout phase 1. Out of scope: instrument internals and Π-to-instrument engagement (the instrument-layer spec), gate execution (`@plugins/archcore/skills/_shared/gate-contract.md`), closeout discharge mechanics, and the `archived` status value.

## Surface

Controlled vocabulary — one term per concept, no synonyms:

- canon — the accepted document graph in `.archcore/`.
- capability — one behavior an external consumer relies on, recordable as one `spec` within the caps of `@plugins/archcore/skills/_shared/spec-contract.md`.
- zone — the canon documents and code areas one request touches.
- Δ (canon delta) — `creates`, `modifies`, `retires` (capability lists), `decision` (settled-choice delta), `intent_gap` (product intent absent from the canon).
- Π (gap profile) — one source per information need: `machine`, `user`, `world`, `undecided`, `empirical`.
- M (maturity) — one value per zone: `pencil` or `stone`.
- R (risk flags) — `external-contract`, `data-migration`, `security-compliance`, `irreversibility`, `multi-team`.
- operational procedure — a human-performed sequence the delta introduces: install, migrate, operate, or verify steps.
- route — the package composition: `null`, `decision`, `amendment`, `capability`, `umbrella`.
- size label — `S` (null, decision, amendment), `M` (capability), `L` (umbrella, or capability raised), `XL` (umbrella raised); derived, never an input choice; capped at `XL`.
- route announcement — one report line: the route, the size label, and the Δ, Π, M, R values that produced them.
- expert invocation — the user names a route or an instrument, bypassing computation.
- state carrier — the `archcore:track` block per the gate contract, extended with `route:` and `delta:` fields.

## Normative Behavior

1. WHEN grounding completes, the conductor MUST derive Δ, Π, M, and R before any question and any document creation.
2. WHEN `creates`, `modifies`, and `retires` are empty, `decision` is none, and `intent_gap` is absent, the conductor MUST take the `null` route.
3. WHILE the route is `null`, the conductor MUST create no document beyond engaged instrument outputs.
4. WHEN only `decision` is non-empty, the conductor MUST route it to the decision instrument.
5. WHEN `modifies` names a capability, the conductor MUST route the change as an `amendment` through the code-wrong/spec-wrong verdict.
6. WHEN `creates` holds one capability, the conductor MUST assemble one `spec` plus one `plan`.
7. WHEN `creates` holds one capability and `intent_gap` names goals or metrics beyond that capability's purpose, the conductor MUST add one `prd`.
8. WHEN `creates` holds two or more capabilities, the conductor MUST assemble one umbrella `prd`, one `spec` per capability, and one `plan`.
9. WHEN a capability's delta introduces an operational procedure, the conductor MUST add one `guide` to the package.
10. WHEN R carries `data-migration`, the conductor SHOULD add a migration runbook `guide` to the package.
11. The conductor MUST report the route announcement before invoking an instrument.
12. WHEN a Π need names a non-`machine` source, the conductor MUST engage the matching instrument per the instrument-layer engagement rules.
13. WHEN M is `stone` or R is non-empty, the conductor MUST raise the size label one step and name the raising flag in the announcement.
14. WHEN R carries `security-compliance`, the conductor MUST escalate each flagged capability into iso links.
15. The conductor MUST NOT escalate an unflagged capability.
16. WHEN the user issues an expert invocation, the conductor MUST execute the named path without computation.
17. The conductor MUST NOT ask the user to choose a route or a size label.
18. WHEN decomposition surfaces a capability outside the declared Δ, the conductor MUST revise Δ and re-announce before continuing.
19. WHEN a route produces a `plan`, the conductor MUST record the declared Δ and the route rationale in that `plan`.
20. WHEN the intent gap fits one capability's purpose, the conductor MUST record it in that capability's `spec`.
21. WHEN a `decision` or `amendment` route's implementation spans two or more tasks, the conductor MUST add one `plan` through the decompose instrument.
22. WHEN iso links engage, the conductor MUST raise the size label one additional step.

## Constraints & Invariants

- Invariant: every produced document carries one of the 19 shipped types and a status in `draft`/`accepted`/`rejected` — the MCP tool schemas enum exactly these values.
- Invariant: Steps 1, 3, 5, and 6 of `@plugins/archcore/skills/plan/SKILL.md` keep their purpose and order; the conductor replaces Steps 2 and 4, and touches the other steps only where a numbered behavior of this spec requires it.
- Constraint: WHEN two or more of behaviors 4–8 match, the conductor composes the union of their packages and announces the highest route — `umbrella` over `capability` over `amendment` over `decision`.
- Constraint: expert aliases `sdd`, `sources`, `iso`, and `research` stay valid; each maps to one computed-era path.
- Constraint: capability granularity binds through the granularity contract under `_shared/` — a phase-1 exit condition, not a follow-up.
- Constraint: `retires` entries route to closeout discharge; the conductor performs no status transition.

## Failure Behavior

1. IF grounding cannot settle a Δ field, THEN the conductor MUST record it as a `user`-source Π need instead of guessing.
2. IF the touched zone carries an unresolved staleness flag, THEN the conductor MUST mark the affected Δ entries `[assumption]` and continue.
3. IF `modifies` names a capability no spec covers, THEN the conductor MUST engage the describe instrument in callable mode first.
4. IF the question ceiling exhausts before `user` needs resolve, THEN the conductor MUST record the remainder under `deferred` in the state block.
5. IF a resumed state block lacks the `route:` field, THEN the conductor MUST recompute the route from the recorded gate and clarifications.
6. IF a computed route contradicts a resumed draft's recorded route, THEN the conductor MUST surface both and ask one confirmation question.

## Conformance

An implementation is conformant when behaviors 1–22 hold over the 40 recorded bench traces, the invariants hold on every run, and the failure rules produce the stated outcomes. Non-normative example: Given "fix the Safari button overflow", When grounding finds empty Δ lists and no `intent_gap`, Then the conductor announces `route: null` and creates no document.