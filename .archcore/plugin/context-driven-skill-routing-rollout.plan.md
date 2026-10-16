---
title: "Roll Out Context-Driven Plan, Document, and Review"
status: draft
tags:
  - "architecture"
  - "concepts"
  - "skills"
---

## Goal

Deliver context-driven routing for /archcore:plan, /archcore:document, and /archcore:review after the companion RFC is accepted. The plugin's accepted local contracts stay authoritative until their amendments are accepted. The change edits the plugin runtime, its command wrappers, and its benches; it adds no kernel document type and keeps plan-removal consent. The mode words are retired without an alias or a deprecation window; a former mode word is topic text with no notice and no handling of its own (user decision, 2026-10-16).

The 2026-10-15 matrix bench on `claude-haiku-5-5` shaped the tasks: a context-routing prototype raised `plan` from 7 to 11 of 18 held-out calls, kept `document` at 12 of 18, and lowered `review` from 14 to 11 of 18 (77 cases, 3 repetitions per case; task 3 records the result).

## Tasks

### Phase 0 — Record the baseline

1. [x] Record the 2026-10-15 route, document, and review bench baseline in @.archcore/plugin/delta-routing-bench.doc.md.
2. [x] Correct the fixture counts in that document and in @plugin/test/behavioral/fixtures/routing-bench.tsv.
3. [x] Record the 2026-10-15 matrix, held-out, skill-selection, and delivery results in @.archcore/plugin/delta-routing-bench.doc.md.
4. [x] Add the matrix harness, its 59 cases, and the separate 18 held-out cases under @plugin/test/behavioral/.
5. [x] Add a `test-matrix-bench` target with `MATRIX_BENCH_MODEL` and `MATRIX_BENCH_REPS` to @plugin/Makefile.

### Phase 1 — Align the contracts

6. [x] Replace the ADR's route-bench argument: hosts passed no document mode word in 10 of 10 skill-bench calls.
7. [x] Correct the ADR's count of mode rows in skill-bench from five to eleven; source or remove its 1.17k token figure.
8. [x] Settle the leading `research` slug in the ADR: a leading `research` requests the type `research`; choice wording selects `rnd`.
9. [x] State in the ADR that the 5-question ceiling, with `deferred` and resume, replaces the expert budget.
10. [x] On ADR acceptance, reject @.archcore/plugin/command-entry-grammar.adr.md and @.archcore/plugin/delta-routing-model.idea.md; the `supersedes` relations exist.
11. [x] On RFC acceptance, reject @.archcore/plugin/pi-elicitation-driver.rfc.md with a `supersedes` relation from the RFC.
12. [x] Amend @.archcore/plugin/command-surface-v2.spec.md to free-text entry, no-argument continuation, and partial-plan capture in behavior 40; behavior 29 stays.
13. [x] Amend @.archcore/plugin/track-layer.spec.md to the signal order without an explicit mode and the context record announcement.
14. [x] Amend @.archcore/plugin/delta-routing-conductor.spec.md and @.archcore/plugin/delta-routing-instruments.spec.md to the context record and claim-level selection.
15. [x] Amend @.archcore/plugin/review-durable-context-selection.spec.md and reverse alternative 7 of @.archcore/plugin/durable-context-selection-by-reader-task.adr.md.
16. [x] Triage the documents in the table below, and amend the normative ones to the free-text grammar.
17. [x] Update the three drafts in the table below to the free-text grammar.

| Task | Documents |
|---|---|
| 16, accepted | @.archcore/plugin/elicitation-contract.spec.md, @.archcore/plugin/delta-routing-type-engagement.doc.md, @.archcore/plugin/bounded-elicitation.adr.md, @.archcore/plugin/four-command-palette.adr.md, @.archcore/plugin/plugin-architecture.spec.md, @.archcore/plugin/skill-file-structure.rule.md, @.archcore/plugin/component-registry.doc.md, @.archcore/plugin/plugin-testing.guide.md, @.archcore/plugin/actualize-system.spec.md, @.archcore/plugin/init-entry-and-assess.spec.md, @.archcore/plugin/research-runtime-category-and-evidence-entry.adr.md, @.archcore/plugin/plugin-v2.prd.md, @.archcore/plugin/delta-routing-compatibility.doc.md |
| 17, draft | @.archcore/plugin/track-handoffs.spec.md, @.archcore/plugin/plugin-usage-catalog.doc.md, @.archcore/plugin/command-entry-grammar.scenario.md |

### Phase 2 — Resolve context once

18. [x] Define the read-only context record, its certainty and volume axes, the source ladder, and the read budget. @plugin/plugins/archcore/skills/_shared/gate-contract.md
19. [x] Write a scope answer into the first draft the invocation creates. @plugin/plugins/archcore/skills/_shared/elicitation-contract.md
20. [x] Amend the three skill files in place instead of replacing them. @plugin/plugins/archcore/skills/plan/SKILL.md, @plugin/plugins/archcore/skills/document/SKILL.md, @plugin/plugins/archcore/skills/review/SKILL.md
21. [x] Keep the review sentinel table, the document exit for a request without material, and the named-type path in those files.
22. [x] Add no-argument continuation and an evidenced start menu of at most four items to the three skills.
23. [x] Record declined review units and offers under `## Clarifications` of the retained plan or created draft. @plugin/plugins/archcore/skills/_shared/durable-context-selection.md
24. [x] Pin the kept guard rules in @plugin/test/structure/loop-integrity.bats.

### Phase 3 — Route documents by need

25. [x] Replace the package table, route names, size escalators, expert map, and `route:` announcement. @plugin/plugins/archcore/skills/_shared/delta-routing.md
26. [x] Read legacy `route:`, `delta:`, and `## Declared Delta` as a claim ledger on resume. @plugin/plugins/archcore/skills/_shared/tracks/closeout.md
27. [x] Open the decompose gate to a claim ledger of two or more dependent tasks without a `prd` or `spec`. @plugin/plugins/archcore/skills/_shared/tracks/sdd.md
28. [x] Select plan artifacts by claim ownership; engage only the requirement levels an obligation names. @plugin/plugins/archcore/skills/_shared/tracks/sdd.md, @plugin/plugins/archcore/skills/_shared/tracks/requirements-cascade.md
29. [x] Create an `adr` when the request names alternatives and reasons; add a `prd` only for an outcome beyond one contract.
30. [x] Create no `idea` beside a `prd` in one invocation.
31. [x] Select present-state owners first, and test whether a README or code comment already answers the reader. @plugin/plugins/archcore/skills/document/SKILL.md, @plugin/plugins/archcore/skills/_shared/tracks/describe.md
32. [x] Remove `doc` as the fallback type; a `doc` holds only reference data that a path cannot explain. @plugin/plugins/archcore/skills/_shared/tracks/describe.md
33. [x] Apply the actor rule in describe: an agent procedure becomes a `task-type`, a human procedure a `guide`.
34. [x] Add closing-test examples that separate `research` from `rnd`. @plugin/plugins/archcore/skills/_shared/tracks/research.md
35. [x] Separate open choices from settled decisions. @plugin/plugins/archcore/skills/_shared/tracks/decision.md
36. [x] Add an advisory exit check: a draft from describe, decision, or experience names a governed path or file kind.
37. [x] Update or merge an existing owner when the target file already returns five matched rows in file context.
38. [x] Replace the hints with `[topic]`, `[subject]`, and `[path, tag, or scope]` in the command wrappers and skill descriptions. @plugin/plugins/archcore/commands/
39. [x] Keep "investigate", "research", and "compare the alternatives" near the start of the plan description.
40. [x] Replace mode-word commands in the next-step mod and its tests. @plugin/plugins/archcore/hooks/next-step.tsx, @plugin/plugins/archcore/tests/next-step.test.ts
41. [x] Replace mode-word commands in the other runtime assets listed in the table below.
42. [x] Remove the retired-word notice; a former mode word is plain topic text (user decision, 2026-10-16).

| Task | Files |
|---|---|
| 41 | @plugin/plugins/archcore/skills/init/SKILL.md, @plugin/plugins/archcore/skills/init/lib/seed-compose.md, @plugin/plugins/archcore/skills/_shared/tracks/actualize.md, @plugin/plugins/archcore/skills/_shared/tracks/research.md, @plugin/plugins/archcore/skills/_shared/gate-contract.md, @plugin/plugins/archcore/skills/_shared/precision-rules.md |

### Phase 4 — Complete review and learning

43. [x] Capture the delivered units of a partially delivered plan by editing the closeout and selection files directly. @plugin/plugins/archcore/skills/_shared/tracks/closeout.md, @plugin/plugins/archcore/skills/_shared/durable-context-selection.md, @plugin/plugins/archcore/skills/review/SKILL.md
44. [x] Report each retained plan's missing work or proof and its next action. @plugin/plugins/archcore/skills/_shared/tracks/closeout.md
45. [x] Require a reusable trigger, mechanism, evidence, and limit for experience. @plugin/plugins/archcore/skills/_shared/tracks/experience.md, @plugin/plugins/archcore/skills/_shared/durable-context-selection.md
46. [x] Treat only a contradiction of accepted canon as an authority conflict; an extension updates the owner. @plugin/plugins/archcore/skills/_shared/verdict-contract.md
47. [x] Name the exit of each vision document at its producing gate, and flag closable vision drafts in the health report.

### Phase 5 — Prove outcomes and release

48. [x] Replace route-name scoring with the matrix bench as the routing guard. @plugin/test/behavioral/route-bench.sh, @plugin/test/behavioral/fixtures/routing-bench.tsv
49. [x] Add review-bench rows: delivered endpoint in an open plan, unfinished plan, spec extension. @plugin/test/behavioral/fixtures/review-bench.tsv
50. [x] Update the pinned tests in the table below with Phase 3 in one change, so `make test` stays green.
51. [x] Extend the retired-form grep to the retired mode words. @plugin/test/structure/command-grammar.bats
52. [x] Run the matrix bench with 3 repetitions and read a change only against the held-out set.
53. [x] Run skill-bench on the new descriptions with 2 repetitions. @plugin/test/behavioral/skill-bench.sh
54. [x] List the retired mode words and their plain-language forms in the guidance. @README.md, @plugin/README.md, @plugin/docs/release.md

### Phase 6 — Re-measure on an untuned set

55. [x] Have a second author write a fresh held-out set; the 2026-10-16 contract fixes answered held-out cases H03, H11, and H15, so that set is tuned. @plugin/test/behavioral/fixtures/matrix-holdout-fresh.tsv
56. [x] Run the matrix bench on `claude-haiku-5-5`, 3 repetitions, on the fresh held-out set and the own cases; the bench loads `gate-contract.md` for `plan` and `review` since 2026-10-16. @plugin/test/behavioral/matrix-bench.py
57. [x] Record input tokens, output tokens, and latency per call against the Phase 0 baseline, as step 6 of the RFC migration requires. @.archcore/plugin/delta-routing-bench.doc.md

| Task | Pinned tests and fixtures |
|---|---|
| 50 | @plugin/test/structure/command-grammar.bats, @plugin/test/structure/delta-routing.bats, @plugin/test/structure/loop-integrity.bats, @plugin/test/structure/trigger-routing.bats, @plugin/test/fixtures/routing/fixtures.tsv, @plugin/test/structure/agent-contracts.bats, @plugin/test/structure/research-track.bats, @plugin/test/structure/actor-subject-compat.bats, @plugin/test/behavioral/skill-bench.sh, @plugin/test/behavioral/fixtures/skill-bench.tsv, @plugin/test/behavioral/document-bench.sh, @plugin/test/behavioral/fixtures/document-bench.tsv |

## Acceptance Criteria

- The three command hints take optional free text; no leading word is a mode; a retired mode word gets no notice and no handling of its own.
- A no-argument invocation resumes one matching draft, shows an evidenced menu of at most four items, or asks one question; a diff alone produces no plan.
- The context record separates claims, authority, unknowns with their sources, hazards, and conflicts; no report prints a route name or a size label.
- A non-capability migration receives a plan without a spec; a security issue without a traceability obligation receives no requirement documents.
- The delivered contract of a partially delivered plan is captured, the plan stays, and no run offers its removal.
- An extension of an accepted spec updates that spec; a contradiction stays a conflict until the user decides.
- An agent procedure becomes a `task-type`, a human procedure with a hazardous step a `guide`, and `doc` is never a fallback.
- Every `adr`, `rule`, `cpat`, `spec`, `guide`, `doc`, `task-type`, or `scenario` that the describe, decision, or experience instrument creates passes the advisory governed-path check of task 36. A `spec` that `plan` creates for code not yet written may name no path.
- On the held-out set on `claude-haiku-5-5`, `plan` scores at least 11, `document` at least 12, and `review` at least 14 of 18 calls.
- At most 10 of the 59 own cases change outcome across repetitions, the current-contract count of 2026-10-15.
- skill-bench selects the right skill in 46 of 46 calls on the new descriptions.
- document-bench does not regress against the same-model Phase 0 baseline: 27 of 27 on `claude-fable-5-1`, 22 of 27 on `claude-haiku-5-5`.
- `make test` passes on dev after the Phase 3 change.

## Verification Cases

| Case | Expected gate outcome |
|---|---|
| Plan with no arguments and one matching draft | Resume the earliest incomplete gate without repeating answered questions. |
| Plan with no arguments and only an unrelated dirty branch | Ask for the desired future outcome; create no plan from the diff. |
| Internal library upgrade with a coordinated rollout | Create a task plan without a new spec or adr. |
| Security repair without formal traceability | Require the evidenced security check; create no requirement documents. |
| Public contract change with an existing covering spec | Propose a focused amendment to that spec. |
| Code-first public contract with no covering spec | Review proposes an evidenced spec draft through one batched preview. |
| Partially delivered plan with a delivered endpoint | Capture the endpoint contract, retain the plan, offer no removal. |
| Code newer than accepted canon and contradicting it | Report an authority conflict before any spec-wrong verdict. |
| Code extending an accepted spec without contradicting it | Update that spec's owner clause. |
| One incident with a narrow fix | Omit generalized experience unless transfer scope and limits are evidenced. |
| Review where git resolves no scope | Ask for a path, topic, or ref; show no health report. |
| Document request with no report and no material | Write nothing and name the plan command. |
| Lone "should we X instead of Y" with no facts and no reviewer | One question offers `adr`, `rfc`, or investigation; "you decide" writes nothing and names the plan command. |
| "Should we X" with a named reviewer or a team disagreement | The same question recommends an `rfc`. |
| Release process with manual tagging and a rollback | Create a `guide`, not a `doc`. |
| Repeated agent procedure with an evidenced step order | Create a `task-type`. |
| Licence facts for every dataset, no pending choice | Create a `research` document, not an `rnd`. |
| Settled choice with alternatives named in the request | Create an `adr` and a plan. |
| Investigation request that names no command | The host invokes plan. |
| Plan request that leads with `research` and names candidates | Create a `research` document; `rnd` only when the wording asks to choose. |

## Dependencies

- The user accepted the companion RFC and the ADR on 2026-10-16; the Phase 1 amendments to the local mode, conductor, and review contracts landed the same day.
- The user settled tasks 8 and 9 on 2026-10-15; the ADR records both choices.
- Status after Phase 6 on 2026-10-16, from @.archcore/plugin/delta-routing-bench.doc.md: on the fresh held-out set on `claude-haiku-5-5`, `plan` scores 16, `document` 12, and `review` 18 of 18 calls, so the held-out criterion holds. skill-bench is 46 of 46, and document-bench is 23 of 27 against the baseline of 22. Open: 17 of 59 own cases change outcome across repetitions on Haiku, above the limit of 10; Sonnet has 1. F12 is settled by `open-choice-offers-four-exits.adr` (draft): an open choice offers three exits, and a lone question without evidence recommends investigation.
- The Phase 0 results of 2026-10-15, recorded in @.archcore/plugin/delta-routing-bench.doc.md, are the reference for Phase 5. The benches run by hand on `claude-haiku-5-5`; a larger model only confirms a Haiku failure. review-bench runs with `REVIEW_BENCH_REPS=3`.
- The held-out set stays separate from the own cases; only an author who did not write the contract under test extends it.
- Task 18 sets no numeric read limit: the stop rule of the Context resolution section in @plugin/plugins/archcore/skills/_shared/gate-contract.md bounds reads; the user accepted this default on 2026-10-16. The matrix bench `reads` field is self-reported.
- [assumption] The 2026-10-15 prototype is a proxy: one 168-line contract that replaced the three skill files; the edited runtime can score differently.
- The file-context ordering and bare-name work is a separate CLI plan; this plan does not wait for it.
- The plugin repository is on branch dev with unrelated working-tree changes; preserve them during implementation.
- The CLI stays the store and retrieval engine; the plugin runtime owns routing and document-filling gates.

## Declared Delta

- Route under the current conductor: amendment, size M; this is a transitional planning label, not the proposed public interface.
- Creates: no new kernel capability or document type.
- Modifies: plan, document, and review entry behavior; artifact selection; review capture, closeout reporting, and decision persistence; the routing guard bench.
- Retires: public mode words for the three commands; the ΔΠMR route table with its route names and size escalators; the expert-invocation budget; route-name scoring as the routing guard.
- Decision: the companion RFC's acceptance remains unresolved; the leading `research` slug and the 5-question ceiling were settled on 2026-10-15 and recorded in the ADR draft.
- Intent gap: none; the RFC records the proposed outcome.
- Π: machine for code paths and the 2026-10-15 measurements; user answers of 2026-10-15 for tasks 8 and 9; user confirmation for the RFC and ADR status transitions.
- M: stone, because accepted local command and conductor contracts cover the touched behavior.
- R: external-contract, because command invocation grammar changes for users and host wrappers.
- Verdict: the covering specs match the current runtime (`ok`); they change only through the Phase 1 amendments after RFC acceptance.
