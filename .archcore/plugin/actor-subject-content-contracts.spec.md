---
title: "Scenario and Journey Content Contracts — Runtime Canon for Actor-Subject Documents"
status: draft
tags:
  - "component:plugin"
  - "document-types"
  - "plugin"
  - "precision"
  - "skills"
---

## Purpose & Scope

This spec defines the two content contracts the plugin ships for the actor-subject types — `scenario-contract.md` and `journey-contract.md` — and the canon hooks that reach the composing skill before it writes: rule 7 of the precision rules, the ownership table of the prd contract, and the Conformance note of the spec contract. Normative for every skill that composes either type (`plan` at `sdd.illustrate` and `sdd.require`, `document` on the describe track) and for contract authors. The gates that produce the documents are a separate spec; the engine's own checks are the CLI-side contract `scenario-and-journey-advisory-canon.spec` in the `cli` repository. Out of scope: execution of a scenario, and the docs site.

## Surface

- Contract files: `@plugin/plugins/archcore/skills/_shared/scenario-contract.md` and `@plugin/plugins/archcore/skills/_shared/journey-contract.md`, with the section set of `@plugin/plugins/archcore/skills/_shared/spec-contract.md`: What it is, routing gate, When NOT to write, Mandatory sections, Notation, Body cap with an "Over the cap" section, Status, Forbidden in the body, Enforcement, Rationale, Examples.
- Canon hooks: `@plugin/plugins/archcore/skills/_shared/precision-rules.md` rules 6 and 7; `@plugin/plugins/archcore/skills/_shared/prd-contract.md` content-kind ownership table; `@plugin/plugins/archcore/skills/_shared/spec-contract.md` Conformance section.
- Engine canon the contracts mirror: `@cli/templates/precision.go` (`RequiredSections`, `ActorStepSections`, `MaxBodyLines`), `@cli/templates/templates.go` (`generateScenarioTemplate`, `generateJourneyTemplate`).
- Line format F6, actor-subject step: `<Actor> <action>; <system> <observable response>.` or `Given|When|Then|And|But <observation>.`
- Spec-owned headings: `Surface`, `Normative Behavior`, `Failure Behavior`.

## Normative Behavior

1. The scenario contract MUST require the sections Subject, Actors, Flows, Examples, and Open Questions, in that order.
2. The journey contract MUST require the sections Intent, Actors, Journeys, and Open Questions, in that order.
3. Each contract MUST state the F6 step form: actor subject, one action or observation, 20 words or fewer, no modal.
4. Each contract MUST carry an "Over the cap" section that names the actor as the split sub-surface.
5. The scenario contract MUST name the `spec` clause set as the second split boundary after the actor.
6. WHEN composing a Flows subsection, the composing skill MUST open it with an `Anchors:` line of `@path` references.
7. WHEN composing a scenario, the composing skill MUST cite the illustrated `spec` clauses by number in Subject.
8. WHEN composing an Examples entry, the composing skill MUST give it a title, an `Illustrates:` line, and unfenced Given/When/Then lines.
9. WHEN composing a journey, the composing skill MUST open Intent with the header `In order to <goal> / As a <actor> / I want <outcome>`.
10. WHEN composing either type, the composing skill MUST fill the Actors table with the columns Actor, Who they are, What they want.
11. WHEN composing either type, the composing skill MUST NOT write a BCP 14 modal in a step.
12. WHEN a draft exceeds 120 body lines, the composing skill MUST apply the contract's over-the-cap remedies in order.
13. WHEN a scenario takes over a journey's flow, the composing skill MUST edit the journey down to intent in the same gate close.
14. Rule 7 of the precision rules MUST list `scenario` and `journey` as claim-recording types with the F6 step profile.
15. Rule 6 of the precision rules MUST list `scenario` and `journey` under the architect-voice default.
16. The prd-contract ownership table MUST carry the three actor-subject content-kind rows of the global RFC.
17. The spec contract's Conformance section MUST send an example past its five-line allowance to a linked `scenario`.
18. Each contract MUST name the tag conventions `actor:<type>`, `component:<name>`, and `nfr:<concern>`.
19. Each contract MUST state the routing test against its pair: a covering `spec` exists, `scenario`; none exists, `journey`.
20. WHEN composing a step under Flows or Journeys, the composing skill MUST open it with the actor's name as the Actors table spells it.

## Constraints & Invariants

- Invariant: the contracts mirror the engine canon; WHEN a contract and the CLI hook disagree, the CLI is right and the contract is corrected (precision-rules preamble).
- Invariant: the body cap is 120 lines for both types, counted as the `spec` cap is counted — headings and blank lines included.
- Invariant: a step in Flows, Journeys, or Examples is not a code block; the Examples section holds unfenced lines, and rule 6's code-block guidance does not admit a fenced example.
- Constraint: the engine matches a step's opening words against the first column of the Actors table, case-insensitively and literally; a leading article (`The beginner starts…`) fails that match, so a step opens with the bare actor name (`Beginner starts…`). Observed against CLI 0.8.4 on 2026-09-16.
- Constraint: status meanings are authoring conventions — an accepted `journey` records team agreement on the wanted interaction; an accepted `scenario` records a reader's confirmation against the running system; the engine verifies neither.
- Constraint: a contract carries no section that enumerates other `.archcore/` documents (precision rule 5); relation conventions are stated in prose, edges live in the graph.
- Constraint: neither contract prescribes a test runner, a discovery technique, or a feature-file layout.

## Failure Behavior

1. IF a Flows subsection carries no `Anchors:` line at gate close, THEN the executing skill MUST report the subsection as an advisory finding.
2. IF a draft of either type carries a spec-owned heading, THEN the composing skill MUST route that content to the covering `spec`.
3. IF a journey draft carries data-bearing examples, THEN the composing skill MUST route them to a `scenario` or mark the draft for the routing test.
4. IF no split boundary is unambiguous for an over-cap draft, THEN the composing skill MUST keep the document whole and report the excess.
5. IF the engine predates CLI 0.8.4, THEN the compatibility spec governs and the composing skill writes neither type.

## Conformance

An implementation is conformant when both contract files exist with the sections behaviors 1–5 and 18–19 require, the canon hooks of behaviors 14–17 are present, and a draft composed from either contract satisfies behaviors 6–13 and 20 and the failure rules. Regression coverage: `@plugin/test/structure/actor-subject-contracts.bats` pins the files and the canon hooks; the CLI hook over a composed draft pins the mechanical half — on 2026-09-16 a scenario and a journey drafted from the contracts' examples reported 0 findings on CLI 0.8.4.

Given a capability with a designed `spec`, When the composing skill drafts a scenario from the contract, Then every Flows subsection opens with `Anchors:` and no step carries a modal.
