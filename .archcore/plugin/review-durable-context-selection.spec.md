---
title: "Review Durable Context Selection — One Rule for Planned and Planless Changes"
status: accepted
tags:
  - "component:plugin"
  - "plugin"
  - "precision"
  - "skills"
---

## Purpose & Scope

This spec defines how `/archcore:review` selects durable project context from changed code: which claims get a document owner, what evidence each needs, and where the selection sits in the closeout sequence. Normative for the review skill (`@plugin/plugins/archcore/skills/review/SKILL.md`) and for the closeout, describe, decision, and experience instruments it calls. Consumed by developers and coding agents who read `.archcore/` before later changes.

The selection runs with or without a matching `plan`. `command-surface-v2.spec` behavior 39 routes a planless branch review through it, and `delta-routing-instruments.spec` behavior 23 admits a review-side `spec` through the describe instrument.

Out of scope: starting a review nobody invoked, editing source code, a new document type, a new command mode, and documenting the delivered part of a `plan` that stays open.

## Surface

- Inputs: the branch boundary from `@plugin/plugins/archcore/skills/_shared/branch-state.md`, the working tree, relevant code and tests, the scoped `.archcore/` documents, and MR material the user supplies.
- Outputs: a selection report, one batched write preview, the drafts the user authorizes, and the plan disposition when a plan matched.
- Documentation unit: one claim, or one related claim group, that answers a named future reader task. A changed file is not a unit.
- Selection report: per unit — the claim, its evidence, the owning document or type, and the reader task the owner serves; plus the omitted, deferred, and declined units.
- Position: with a plan, after `closeout.merge` and before `closeout.capture`; without a plan, after the bidirectional check and any actualize fixes, before the experience offer.

## Normative Behavior

1. WHEN reviewing changed code, the review skill MUST inspect relevant code, tests, scoped documents, and available history before selecting units.
2. WHEN a plan matches the work, the review skill MUST run the selection after `closeout.merge` and before `closeout.capture`.
3. WHEN no plan matches, the review skill MUST run the selection after the bidirectional check and any confirmed actualize fixes.
4. WHEN no plan matches, the review skill MUST derive units from the observed change without creating a retrospective `plan`.
5. WHEN a plan covers only part of the change, the review skill MUST select units from the remaining change without attributing it to that plan.
6. The review skill MUST select units by future reader task and durable claim, never by file count or diff hunk.
7. For each selected unit, the review skill MUST name the claim, the evidence, the owning document or type, and the reader task.
8. The review skill MUST treat complexity, implicitness, and uncertainty as signals to inspect, never as triggers to write.
9. WHEN a changed subject carries a relied-on contract, the review skill MUST inspect its surface, invariants, and failure behavior beyond the changed lines.
10. WHEN an existing document owns a selected claim, the review skill MUST propose a focused update instead of a new document.
11. WHEN an authorized unit is a contract with no owning document, the review skill MUST produce its `spec` through the describe instrument.
12. WHEN an authorized unit is a decision with direct evidence, the review skill MUST route it through the decision instrument.
13. WHEN an authorized unit is a procedure or an edit shape, the review skill MUST route it by the experience instrument's actor rule.
14. WHEN code and tests answer the reader task without a durable owner, the review skill MUST omit the unit and state why.
15. WHEN one observed case changes a contract, decision, or procedure, the review skill MUST route it to that owner without a repetition prerequisite.
16. WHEN no existing type owns a unit, the review skill MUST report the gap without forcing a `cpat` or `task-type`.
17. WHEN decision rationale lacks direct evidence, the review skill MUST NOT derive alternatives or reasons from the final code.
18. WHEN evidence conflicts with an accepted document, the review skill MUST report the conflict before treating the changed code as canonical.
19. BEFORE writing any unit, the review skill MUST show every proposed write in one preview naming target, claims, evidence, and status.
20. The review skill MUST ask for the preview's authorization as one question against the per-invocation ceiling.
21. WHEN the user authorizes part of the preview, the review skill MUST write only the authorized units.
22. BEFORE `closeout.capture` runs, the review skill MUST pass it the selection report, including the units still confined to the plan.
23. The review skill MUST report created, updated, omitted, deferred, and declined units separately.
24. The review skill MUST NOT copy source bodies, generated schemas, or file inventories where an `@path` reference answers the reader task.

## Constraints & Invariants

- Constraint: the visible palette, the modes, and the local git boundary stay unchanged; the selection is internal to a branch review.
- Constraint: the preview question draws from the 5-question ceiling of `@plugin/plugins/archcore/skills/_shared/elicitation-contract.md`; per-document acceptance and plan-removal confirmations keep their existing rules.
- Constraint: the preview is the capture offer for a unit routed to the experience instrument; `experience.offer` MUST NOT ask again for an authorized unit.
- Constraint: an instrument invoked for an authorized unit runs in callable mode with that unit's scope pre-filled.
- Constraint: a document the selection creates starts at `draft`; the same invocation MUST NOT offer its acceptance — the next review's accept gate does.
- Constraint: a completed plan keeps the accepted per-plan removal confirmation; a declined unit is reported and blocks no removal.
- Constraint: the `spec` content contract owns a review-created `spec`; `prd`, `plan`, and `adr` keep their content ownership.
- Constraint: a confirmed existing flow may produce a `scenario`; the selection MUST NOT produce a `journey`.
- Invariant: the origin of the code — a plan, another agent, no documents — changes neither the evidence rules nor the selection rules.
- Invariant: zero units is a valid result when the evidence supports no durable claim.
- Invariant: the review boundary is local git; an MR description or discussion counts as evidence only when supplied.

## Failure Behavior

1. IF evidence cannot settle a material claim, THEN the review skill MUST report the missing check.
2. IF evidence cannot settle a material claim, THEN the review skill MUST NOT write that claim as a fact.
3. IF a plan match is ambiguous, THEN the review skill MUST select from the whole change as planless and report the candidate plans.
4. IF a new flow has no run report and no user confirmation, THEN the review skill MUST label its examples unconfirmed.
5. IF the user declines the whole preview, THEN the review skill MUST write no unit.
6. IF the user declines a unit, THEN the review skill MUST record that unit as declined in the report.
7. IF the question ceiling is exhausted before the preview, THEN the review skill MUST report the unwritten selection with the ceiling as the reason.

## Conformance

The review skill conforms when the planned and planless paths satisfy behaviors 1–24, hold the invariants, and degrade per the failure rules. Regression coverage: `@plugin/test/structure/command-grammar.bats` and `@plugin/test/structure/loop-integrity.bats` pin the routing; `@plugin/test/behavioral/review-bench.sh` measures the selection on a live model against `@plugin/test/behavioral/fixtures/review-bench.tsv` — a simple public contract, a complex internal implementation, comprehensive capture, a focused update, a zero-document change, a mixed-origin diff, a one-case lesson, unknown rationale, and conflicting accepted canon.

Given a branch with no matching plan that adds a public API and an internal helper,
When the review runs,
Then the preview proposes one `spec` for the API, names the helper as omitted, and asks one question.
