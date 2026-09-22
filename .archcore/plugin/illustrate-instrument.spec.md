---
title: "Illustrate Instrument — sdd.illustrate, Journey Production at sdd.require, and the Example Check at sdd.design"
status: draft
tags:
  - "component:plugin"
  - "document-types"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines the plan-side production of the actor-subject types: the illustrate instrument that produces a `scenario` at `sdd.illustrate`, the `journey` production at `sdd.require`, the advisory example check at `sdd.design`, and the conductor's registry and package contribution for all three. Normative for the `plan` skill, the conductor contract, and the authors of `skills/_shared/tracks/sdd.md`. Content of the produced documents is the content-contracts spec; consumption on the document and review tracks is a separate spec; the engine gate is the compatibility spec.

## Surface

- Registry row in `@plugin/plugins/archcore/skills/_shared/delta-routing.md`: `illustrate` → `scenario` → `skills/_shared/tracks/sdd.md`, gate `sdd.illustrate` — once per capability.
- Package contribution in the same file's route table: the illustrate condition below adds one `scenario` per qualifying capability; the same condition makes `sdd.require` produce a `journey` beside the `prd`.
- Illustrate condition: the capability's Δ names a user-facing surface (a UI, a conversational skill, an operator-facing flow), or grounding finds `features/*.feature` or a BDD runner in the test-runner slot of `@plugin/plugins/archcore/skills/_shared/grounding/detect-stack.md`.
- Gates in `@plugin/plugins/archcore/skills/_shared/tracks/sdd.md`: `sdd.illustrate` (new), `sdd.require` (journey production), `sdd.design` (advisory example check). The gate records follow `@plugin/plugins/archcore/skills/_shared/gate-contract.md`.
- Track state: the `archcore:track` block of the scenario draft carries `gate: sdd.illustrate`, `route:`, and `delta:` per the conductor.

## Normative Behavior

1. The instrument registry MUST list `illustrate` producing `scenario` at `sdd.illustrate`.
2. WHEN a capability meets the illustrate condition, the conductor MUST add one `scenario` for that capability to the package.
3. WHEN a capability does not meet the illustrate condition, the conductor MUST NOT add a `scenario` for it.
4. The conductor MUST sequence `sdd.illustrate` after that capability's `sdd.design`.
5. WHEN `sdd.illustrate` opens and a `scenario` that `depends_on` the designed `spec` exists, the executing skill MUST skip the gate.
6. WHEN `sdd.illustrate` produces a scenario, the executing skill MUST add `depends_on` → the designed `spec`.
7. WHEN a `journey` on the topic exists at `sdd.illustrate`, the executing skill MUST add `implements` → that journey.
8. WHEN `sdd.illustrate` closes, the executing skill MUST verify that every clause number cited in Subject exists in the `spec`.
9. WHEN `sdd.illustrate` closes, the executing skill MUST verify that every Flows subsection carries an `Anchors:` line.
10. WHEN `sdd.illustrate` closes, the executing skill MUST verify the body cap or the applied split per the scenario contract.
11. The `sdd.illustrate` gate MUST declare a per-gate question maximum of 2, drawn from the shared ceiling in auto mode.
12. WHEN the illustrate condition holds at `sdd.require`, the executing skill MUST produce a `journey` beside the `prd`.
13. WHEN `sdd.require` produces a journey, the executing skill MUST add `related` → the `prd`.
14. WHEN `sdd.require` closed through the compression path, the executing skill MUST still produce the journey under the illustrate condition.
15. WHEN `sdd.design` closes, the executing skill MUST report each Normative Behavior clause that no example illustrates.
16. WHEN a `journey` exists and no `spec` covers the interaction, the conductor MUST NOT produce a `scenario`.
17. WHEN the compatibility probe returns other than `yes`, the conductor MUST drop the illustrate instrument from the package.
18. WHEN the conductor drops the illustrate instrument, the plan skill MUST report the required engine version once.
19. WHEN `sdd.illustrate` exits, the executing skill MUST remove the track state block from the scenario.

## Constraints & Invariants

- Constraint: `sdd.require` producing a `journey` beside the `prd` is a recorded exception to single-type production, on the pattern of `decision.cascade`; the conductor still owns the sequence.
- Constraint: the example check of behavior 15 counts an example in the `spec` Conformance block, in a `scenario` that `depends_on` the `spec`, or in a feature file the `spec` cites; it is advisory and closes no gate.
- Constraint: the instrument asks no question that the `spec`, the `prd`, the journey, or recorded clarifications answer (elicitation contract, "Ground first").
- Constraint: the illustrate condition reads grounding output and Δ only; the conductor never asks the user whether to illustrate.
- Constraint: a declined illustrate instrument follows the conductor's existing decline rule; this spec adds no second rule.
- Invariant: a `scenario` produced here illustrates exactly one `spec`; a capability whose scenario needs two specs is a capability boundary defect the conductor reports.
- Invariant: no edge runs from a `spec` to a `scenario`; a scenario edit that changes behavior enters a later `/archcore:plan` as a `modifies` delta with a verdict.

## Failure Behavior

1. IF a cited clause number is absent from the `spec`, THEN the executing skill MUST stop at `sdd.illustrate` and report the number.
2. IF the designed `spec` has no numbered Normative Behavior clause, THEN the executing skill MUST report that the scenario cites none and continue.
3. IF grounding cannot decide the illustrate condition, THEN the conductor MUST record it as a `user`-source Π need and ask within the ceiling.
4. IF no actor boundary is unambiguous for an over-cap scenario draft, THEN the executing skill MUST keep it whole and report the excess.

## Conformance

An implementation is conformant when the registry row and the package contribution exist, the three gate records follow the gate contract, and behaviors 1–19 hold on the routing bench. Regression coverage: `@plugin/test/structure/delta-routing.bats` (registry row and contract references), `@plugin/test/structure/track-goldens.bats` (gate record shape), and two new traces in `@plugin/test/behavioral/fixtures/` — a user-facing capability and a repository with `features/*.feature` [planned].

Given `creates` = 1 with a user-facing surface, When the conductor sequences contract → illustrate → decompose, Then the package holds one `spec`, one `scenario` with `depends_on` → the spec, and one `plan`.
