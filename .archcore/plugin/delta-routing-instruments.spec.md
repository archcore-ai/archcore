---
title: "Delta-Routing Instruments — Producer Layer, Unknown Engagement, and Lifecycle Sequences"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec defines the instrument layer — the single-type producers the conductor invokes — and the lifecycle sequences that keep their gate order. Normative for track-file authors under `@plugin/plugins/archcore/skills/_shared/tracks/` and for the `plan`, `document`, and `review` skills that execute instruments. Out of scope: claim-level selection (the conductor spec), gate record mechanics (`@plugin/plugins/archcore/skills/_shared/gate-contract.md`), and per-type content contracts.

## Surface

Instrument registry — instrument → produced type → current carrier:

- concept → `idea` — `sdd.frame`.
- intent → `prd`, plus `journey` under the illustrate condition — `sdd.require`.
- contract → `spec` — `sdd.design`.
- illustrate → `scenario` — `sdd.illustrate`, once per capability after that capability's `sdd.design`.
- decompose → `plan` — `sdd.decompose`.
- runbook → `guide` — package member, composed from the route's operational and verification tasks.
- decision → `adr`, `rfc` — the decision track gates.
- research → `research` or `rnd` — the research track gates; gather optionally produces `evidence`, and one supplied external material enters gather directly.
- spike → timeboxed `rnd` holding Goal, Questions, and Findings only.
- describe → `spec`, `doc`, `guide`, `task-type`, `scenario` — the describe track gates.
- acquisition → `mrd`, `brd`, `urd` — the requirements-cascade acquisition gates.
- iso links → `brs`, `strs`, `syrs`, `srs` — the requirements-cascade iso gates.

Lifecycle sequences: closeout (`closeout.verify` → `closeout.merge` → `closeout.accept` → `closeout.capture` → `closeout.discharge`), actualize, experience. Shared verdict vocabulary: `code-wrong`, `spec-wrong`, `ok` — consumed by `amend` claims, actualize, and closeout. Callable mode: an instrument entry whose scope the calling skill pre-fills, so the instrument runs scope-question-free.

## Normative Behavior

1. An instrument MUST keep the gate record template of `@plugin/plugins/archcore/skills/_shared/gate-contract.md`.
2. An instrument MUST NOT chain into another instrument through its `Next:` field.
3. The conductor MUST own every cross-instrument sequence.
4. WHEN an unknown is a `fact` the repository, canon, or git answers, the executing skill MUST compose without a question and cite the artifact.
5. WHEN an unknown is a `choice` the user owns, the executing skill MUST interview within the ceiling of `@plugin/plugins/archcore/skills/_shared/elicitation-contract.md`.
6. WHEN an unknown is a `fact` about the outside world, the conductor MUST engage the research instrument.
7. WHEN an unknown is a `choice` with competing options and missing evidence, the conductor MUST engage the decision instrument.
8. WHEN an unknown is a `feasibility` question, the conductor MUST engage a spike.
9. WHEN the calling skill pre-fills an instrument's scope, the instrument MUST skip its own scope elicitation.
10. A spike `rnd` MUST hold only the sections Goal, Questions, and Findings.
11. Spike code MUST NOT merge into the mainline.
12. WHEN a spike resolves its question, the conductor MUST re-run selection with the revised claims.
13. WHEN two or more unknowns need research, a spike, or a decision, the conductor MUST compose one instrument per unknown kind.
14. WHILE composing under high uncertainty, the conductor MUST order instruments cheapest kill shot first: conversation, research, spike, formal document.
15. WHEN an applicable `task-type` or `cpat` covers a technical unknown, the executing skill MUST treat only that unknown as answered by the precedent.
16. WHEN a selection includes a `guide`, the executing skill MUST state the reader and the step actor in the draft.
17. WHEN a repeatable procedure surfaces for capture, the executing skill MUST route it by actor — human to `guide`, agent to `task-type`.
18. WHEN closeout verifies a plan carrying a claim ledger under `## Declared Delta`, the review skill MUST reconcile that ledger against the branch diff.
19. WHEN a scoped document's unique information is absorbed elsewhere, the review skill MAY offer discharge for that document.
20. WHEN offering discharge, the review skill MUST obtain the per-document confirmation before any removal or status change.
21. WHEN a durable-context unit names a settled standard or choice with evidenced reasons, the review skill MUST route it through the decision instrument.
22. WHILE capturing at closeout, the review skill MUST record each selected unit as created, updated, omitted, deferred, or declined.
23. WHILE capturing at closeout, the review skill MUST NOT create a `plan`.
24. WHEN a completed `plan` clears its discharge preconditions, the review skill MUST remove that plan with `remove_document`.
25. IF a plan task carries a verdict other than fulfilled, THEN the review skill MUST NOT remove that plan.
26. WHEN a completed plan carries uncommitted changes, the review skill MUST apply the same discharge preconditions as for a committed plan.
27. WHILE the `archived` status is absent from the kernel, the review skill MUST NOT discharge a `prd`, an `idea`, or an `rnd`.
28. WHEN a capability meets the illustrate condition, the conductor MUST engage the illustrate instrument after that capability's contract instrument.
29. WHEN verifying completion, the review skill MUST inspect staged, unstaged, and untracked files in the scoped working tree.
30. WHEN closeout retains a scoped plan, the review skill MUST report its remaining work or blocking reason.
31. WHILE executing closeout, the review skill MUST NOT stage files.
32. WHILE executing closeout, the review skill MUST NOT create commits.
33. BEFORE requesting plan removal confirmation, the review skill MUST disclose whether git history preserves the plan's current content.
34. WHEN `sdd.design` reviews a draft spec, the executing skill MUST report material omissions, contradictions, and unsupported claims against applicable grounding.
35. WHILE an unanswered user-owned choice changes a new spec's normative behavior, the executing skill MUST keep `sdd.design` open before decomposition.
36. WHILE the selection has instruments left, the conductor MUST keep the state block on the newest draft.
37. WHEN the architecture cascade creates a `spec`, the decision instrument MUST apply the `sdd.design` user-owned choice check before creating the `plan`.
38. WHEN a matched plan is partially delivered, the review skill MUST capture its delivered units through the owning instruments.
39. WHILE a matched plan is partially delivered, the review skill MUST NOT offer its removal.

## Constraints & Invariants

- Invariant: command tenses — `plan` declares future claims, `document` records the present state, `review` reconciles delivered claims.
- Invariant: instruments produce only types supported by the engine gate; the vocabulary releases expose 23 types on CLI 0.8.4, gated by `@plugin/plugins/archcore/skills/_shared/research-compatibility.md` and `@plugin/plugins/archcore/skills/_shared/actor-subject-compatibility.md`.
- Invariant: `plan` is the only type any track removes at closeout.
- Invariant: capture at closeout owns no document type — every document it creates comes from the instrument it routes to; a `spec` comes only through the describe instrument.
- Constraint: the decision instrument's `decision.cascade` gate creates its cascade documents (`rule`, `guide`, `spec`, `plan`, `cpat`) inside the instrument — a recorded exception to single-type production.
- Constraint: the intent instrument's `journey` beside the `prd` under the illustrate condition is a second recorded exception to single-type production; the document skill does not reach that gate.
- Constraint: research gather may create evidence and relations inside the instrument; the shared gate contract defines its pending-write checkpoint exception.
- Constraint: at `closeout.capture` the decision instrument runs its standard cascade only; the architecture cascade is out of scope there.
- Constraint: the acquisition instrument engages only for a market, business, or user source question that the request or a gate names, never by default.
- Constraint: iso links engage only for the levels a named traceability obligation requires, never from a risk keyword.
- Constraint: elicitation ceilings and budget mechanics stay per the elicitation contract, unchanged.
- Constraint: discharge defaults per type — `spec` and `adr` stay canon; a completed `plan` is removed at `closeout.discharge` after `closeout.capture` routes its durable units to the owning instruments; a `prd` holds until its success metrics verify; an `idea` discharges after every document that implements it is accepted; a spike `rnd` keeps only its Findings section.

## Failure Behavior

1. IF a produced draft fails a blocking exit check, THEN the conductor MUST stop the selection at that instrument.
2. IF an instrument's upstream product is missing, THEN the conductor MUST invoke the producing instrument first.
3. IF the user declines an offered instrument, THEN the conductor MUST record the decline under `## Clarifications` of the newest draft and continue.
4. IF a settled decision surfaces inside any instrument, THEN the executing skill MUST route it through the decision instrument before the track exits.
5. IF `remove_document` fails at `closeout.discharge`, THEN the review skill MUST report the failure and leave the plan in place.
6. IF a capture unit is declined, THEN the review skill MUST still offer removal for a plan that clears its discharge preconditions.

## Conformance

An implementation is conformant when behaviors 1–39 hold across the matrix bench cases, the invariants hold on every invocation, and the failure rules produce the stated outcomes. Non-normative example: Given two `create spec` claims and one `choice` unknown with competing options, When the conductor sequences decision → contract → contract → decompose, Then no instrument's `Next:` field fires and the produced `plan` records the sequence.
