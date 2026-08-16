---
title: "Delta-Routing Instruments — Producer Layer, Π Engagement, and Lifecycle Sequences"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines the instrument layer — the single-type producers the conductor invokes — and the lifecycle sequences that keep their gate order. Normative for track-file authors under `@plugins/archcore/skills/_shared/tracks/` and for the `plan`, `document`, and `review` skills that execute instruments. Out of scope: route computation (the conductor spec), gate record mechanics (`@plugins/archcore/skills/_shared/gate-contract.md`), and per-type content contracts.

## Surface

Instrument registry — instrument → produced type → current carrier:

- concept → `idea` — `sdd.frame`.
- intent → `prd` — `sdd.require`.
- contract → `spec` — `sdd.design`.
- decompose → `plan` — `sdd.decompose`.
- runbook → `guide` — package member, composed from the route's operational and verification tasks.
- decision → `adr`, `rfc` — the decision track gates.
- research → `rnd` — the research track gates.
- spike → timeboxed `rnd` holding Goal, Questions, and Findings only.
- describe → `spec`, `doc`, `guide` — the describe track gates.
- acquisition → `mrd`, `brd`, `urd` — the requirements-cascade sources gates.
- iso links → `brs`, `strs`, `syrs`, `srs` — the requirements-cascade iso gates.

Lifecycle sequences (gate order unchanged): closeout, actualize, experience. Shared verdict vocabulary: `code-wrong`, `spec-wrong`, `ok` — consumed by the amendment route, actualize, and closeout. Callable mode: an instrument entry whose scope the calling skill pre-fills, so the instrument runs scope-question-free.

## Normative Behavior

1. An instrument MUST keep the gate record template of `@plugins/archcore/skills/_shared/gate-contract.md`.
2. An instrument MUST NOT chain into another instrument through its `Next:` field.
3. The conductor MUST own every cross-instrument sequence.
4. WHEN a Π need's source is `machine`, the executing skill MUST compose without a question and cite the grounding artifact.
5. WHEN a Π need's source is `user`, the executing skill MUST interview within the ceiling of `@plugins/archcore/skills/_shared/elicitation-contract.md`.
6. WHEN a Π need's source is `world`, the conductor MUST engage the research instrument.
7. WHEN a Π need's source is `undecided`, the conductor MUST engage the decision instrument.
8. WHEN a Π need's source is `empirical`, the conductor MUST engage a spike.
9. WHEN the calling skill pre-fills an instrument's scope, the instrument MUST skip its own scope elicitation.
10. A spike `rnd` MUST hold only the sections Goal, Questions, and Findings.
11. Spike code MUST NOT merge into the mainline.
12. WHEN a spike resolves its question, the conductor MUST re-enter routing with the revised Δ.
13. WHEN two or more Π needs name `world`, `undecided`, or `empirical`, the conductor MUST compose one instrument per uncertainty kind.
14. WHILE composing under high uncertainty, the conductor MUST order instruments cheapest kill shot first: conversation, research, spike, formal document.
15. WHEN grounding surfaces a matching `task-type` or `cpat`, the executing skill MUST de-escalate the covered Π needs from `user` to `machine`.
16. WHEN a package includes a `guide`, the executing skill MUST state the reader and the step actor in the draft.
17. WHEN a repeatable procedure surfaces for capture, the executing skill MUST route it by actor — human to `guide`, agent to `task-type`.
18. WHEN closeout verifies a plan carrying a declared Δ, the review skill MUST reconcile that Δ against the branch diff.
19. WHEN a scoped document's unique information is absorbed elsewhere, the review skill MAY offer discharge for that document.
20. WHEN offering discharge, the review skill MUST obtain the per-document confirmation before any status change.
21. WHILE the `archived` status is absent from the kernel, the review skill MUST NOT apply a discharge transition.

## Constraints & Invariants

- Invariant: command tenses — `plan` declares future Δ, `document` records the present state, `review` reconciles past Δ.
- Invariant: instruments produce only the 19 shipped document types.
- Constraint: the decision instrument's `decision.cascade` gate creates its cascade documents (`rule`, `guide`, `spec`, `plan`, `cpat`) inside the instrument — a recorded exception to single-type production.
- Constraint: the acquisition instrument engages on a product-scale `intent_gap` or an expert invocation, never by default.
- Constraint: iso links engage per flagged capability, never as a whole-initiative mode.
- Constraint: elicitation ceilings and budget mechanics stay per the elicitation contract, unchanged.
- Constraint: discharge defaults per type — `spec` and `adr` stay canon; a completed `plan` discharges into `task-type` capture for an agent-actor procedure, into `guide` capture for a human-actor procedure, plus history; a `prd` holds until its success metrics verify; an `idea` discharges after every document that implements it is accepted; a spike `rnd` keeps only its Findings section.

## Failure Behavior

1. IF a produced draft fails a blocking exit check, THEN the conductor MUST stop the route at that instrument.
2. IF an instrument's upstream product is missing, THEN the conductor MUST invoke the producing instrument first.
3. IF the user declines an offered instrument, THEN the conductor MUST record the decline in the route rationale and continue the remaining sequence.
4. IF a settled decision surfaces inside any instrument, THEN the executing skill MUST route it through the decision instrument before the track exits.

## Conformance

An implementation is conformant when behaviors 1–21 hold across the 40 recorded bench traces, the invariants hold on every invocation, and the failure rules produce the stated outcomes. Non-normative example: Given `creates` = 2 with one `undecided` need, When the conductor sequences decision → contract → contract → decompose, Then no instrument's `Next:` field fires and the produced `plan` records the sequence.