---
title: "Review Durable Context Selection: Bench, Canon Amendments, Routing, and Verification"
status: draft
tags:
  - "component:plugin"
  - "plugin"
  - "precision"
  - "skills"
---

## Goal

Make `/archcore:review` select and record durable project context from changed code, whether an Archcore plan exists or another agent produced the code — one selection rule, one batched preview question, no prose that repeats readable code. The draft `review-durable-context-selection.spec` owns the obligations, `durable-context-selection-by-reader-task.adr` owns the decision and its alternatives, and `review-durable-context-selection.rnd` owns the investigation and the case matrix; this plan owns implementation and verification.

Planning baseline: branch `dev`, HEAD `b7cc60b`, 2026-10-09. The shipped skill checks a matched plan and sends planless work straight to the repeated-pattern offer; `command-surface-v2.spec` behaviors 39–40 and `delta-routing-instruments.spec` behavior 23 still require that. This plan amends those accepted documents only with the user's per-document confirmation.

## Declared Delta

| Field | Planned change |
|---|---|
| `creates` | Review-side durable-context selection for branch changes, with and without a matching plan. |
| `modifies` | Branch review routing; closeout gate order and capture input; describe callable use; experience detection and offer; result reporting. |
| `retires` | No visible command, mode, document type, or accepted status. The two-file repetition threshold ends as the sole experience signal. |
| `decision` | Recorded in `durable-context-selection-by-reader-task.adr` (draft): reader task, evidence, and owner decide the document set; complexity, implicitness, and uncertainty are signals only; one batched preview question per invocation. |
| `intent_gap` | No new PRD: the requested outcome fits the review capability and its draft spec. |
| Π | `machine`: current skills, tests, accepted contracts, and the `rnd`; `user`: confirmation of each accepted-document amendment and of each preview during the bench. |
| M | `stone`: accepted command-surface, instrument, plan-discharge, status-transition, precision, and track-cap decisions cover the zone. |
| R | None identified; the visible command set and CLI document vocabulary remain unchanged. |
| Route | `capability`, base M, raised to L by `stone`. |
| Instruments | Decision recorded the rule; contract for the selection capability; decompose into this plan. |

## Decision Points

| Question | Settled on 2026-10-09 | Change required before a different implementation |
|---|---|---|
| How does review authorize a document write? | One batched preview per invocation, counted as one question against the ceiling; per-document acceptance and plan removal keep their current confirmations. | Amend the `adr` and the elicitation contract if per-write confirmations are wanted. |
| May a finished plan be removed when a proposed unit is declined? | Yes; the declined unit is reported and blocks nothing. | Amend `plan-discharge-by-deletion.adr` if declined material must block removal. |
| Where does a valuable one-case lesson go? | The existing contract, decision, or procedure when one honestly owns it; otherwise a reported ownership gap. | Add a document type only after the bench shows existing types cannot own a real finding. |
| May review document the delivered part of an unfinished plan? | No; out of scope for this plan (deferred alternative 7 in the `adr`). | A separate decision that reconciles it with `command-surface-v2.spec` behavior 40. |
| Which source can establish MR intent or test success? | Only supplied MR material and actual run reports; local git alone establishes neither. | A separate evidence integration if review must fetch these sources. |

## Tasks

### Phase 1 — prove the selection rule on a bench

1. [x] Build positive and negative review cases from the `rnd` case matrix in `@plugin/test/behavioral/fixtures/review-bench.tsv`.
2. [x] Label reader task, evidence, selected units, omissions, and gaps per case in `@plugin/test/behavioral/fixtures/review-bench.tsv`.
3. [x] Add the runner `@plugin/test/behavioral/review-bench.sh` with a key=value verdict line, following `@plugin/test/behavioral/gate-bench.sh`.
4. [x] Revise `@.archcore/plugin/review-durable-context-selection.spec.md` where a case exposes an ambiguous clause.

### Phase 2 — amend the accepted canon, one confirmation per document

5. [x] Amend `@.archcore/plugin/command-surface-v2.spec.md` behaviors 39–40 to route a planless review through the selection.
6. [x] Amend `@.archcore/plugin/delta-routing-instruments.spec.md` behavior 23 to admit a review-side `spec` through the describe instrument.
7. [x] Update register rows 19–21 in `@.archcore/plugin/track-handoffs.spec.md` for the selection edge.
8. [x] Check `@.archcore/plugin/track-layer.spec.md`, `@.archcore/plugin/plan-discharge-by-deletion.adr.md`, and `@.archcore/plugin/document-status-transitions.adr.md`; record no conflict or the conflict here.

### Phase 3 — implement the selection

9. [x] Add the selection procedure as the shared file `@plugin/plugins/archcore/skills/_shared/durable-context-selection.md`.
10. [x] Route the planned and planless paths through that procedure in `@plugin/plugins/archcore/skills/review/SKILL.md`.
11. [x] Insert the selection after `closeout.merge` and before `closeout.capture` in `@plugin/plugins/archcore/skills/_shared/tracks/closeout.md`, with `closeout.capture` consuming the selection report.
12. [x] Add the one-question preview rule beside the confirmation `[assumption]` in `@plugin/plugins/archcore/skills/_shared/tracks/closeout.md`.
13. [x] Call `@plugin/plugins/archcore/skills/_shared/tracks/describe.md` in callable mode with review-scoped evidence and the duplicate check.
14. [x] Route evidenced decisions through `@plugin/plugins/archcore/skills/_shared/tracks/decision.md` without inventing rationale.
15. [x] Broaden `@plugin/plugins/archcore/skills/_shared/tracks/experience.md` beyond the two-file edit shape; the preview replaces `experience.offer` for an authorized unit.
16. [x] Add the selection report groups to the Result section of `@plugin/plugins/archcore/skills/review/SKILL.md`.
17. [x] Keep `journey` off the review path in `@plugin/plugins/archcore/skills/review/SKILL.md`.

### Phase 4 — guard against missing and excess documents

18. [x] Pin the changed gate structure in `@plugin/test/fixtures/goldens/` after inspecting each generated diff.
19. [x] Update the review routing checks in `@plugin/test/structure/command-grammar.bats` and `@plugin/test/structure/loop-integrity.bats`.
20. [x] Run `make -C plugin test` and the review bench; record misses and excess per case in `@.archcore/plugin/review-durable-context-selection.rnd.md`.

### Implementation record, 2026-10-09

- Task 4: the bench exposed no ambiguous clause; the spec is unchanged.
- Task 5: behaviors 8 and 39 amended; behavior 40 kept, because a partly fulfilled plan stays out of scope.
- Task 8: no conflict with `document-status-transitions.adr`. Two stale statements remain in accepted text, left for the user: the `plan-discharge-by-deletion.adr` consequence that capture adds up to 4 questions (now 1), and the `track-layer.spec` catalog entry "detect repeated pattern" for experience.
- Task 14: `@plugin/plugins/archcore/skills/_shared/tracks/decision.md` needed no change; the evidence rule lives in the selection procedure.
- Task 20: `make -C plugin test` passed — 695 bats and 357 mod tests, 0 failures. Review bench on Claude: 12 pass, 0 fail.

## Acceptance Criteria

1. A branch with no matching plan can produce an evidenced `spec`, `guide`, `doc`, or `scenario` without a retrospective plan.
2. A branch with a complete plan captures a missing contract before the plan is offered for removal.
3. A review invocation spends at most one question on document writes; acceptance and removal confirmations keep their current rules.
4. A comprehensive case covers every material contract surface and distinct owner without copying implementation bodies.
5. A selective case updates only affected claims; an obvious internal refactor creates no document.
6. One material failure case routes to its existing owner without a two-file repetition prerequisite.
7. Unseen MR discussion, missing decision reasons, and unrun tests do not become asserted facts or accepted statuses.
8. Every proposed unit names its reader task, evidence, owner, and the reason the owner helps the reader.
9. No new command mode or document type appears, and a partly fulfilled plan keeps its current behavior.
10. The structure and unit suites pass, and the review bench shows no unexplained missing or extra document per case.

## Dependencies

The draft selection spec defines the target behavior; the `adr` records the rule and its rejected alternatives; the `rnd` supplies the case matrix for Phase 1. The accepted precision-over-coverage and architect-voice decisions require concise, evidenced artifacts. The accepted plan-discharge and status-transition decisions retain their confirmation rules. The 300-line track cap (`@plugin/test/structure/track-file-cap.bats`) keeps the procedure out of `@plugin/plugins/archcore/skills/_shared/tracks/closeout.md`, which holds 268 lines at the baseline. The review skill's local git boundary does not provide MR discussion; any use of it requires a supplied source.

Phase 1 runs before any skill text changes and can revise the draft spec. Phase 2 requires confirmation of each accepted-document amendment. Phase 3 depends on the confirmed canon and the Phase 1 cases. Phase 4 verifies the resulting routes; the pre-merge review of the implementation branch is the ordinary `/archcore:review`, not a task here. A PR or CI trigger for reviews that users never run is outside this plan.
