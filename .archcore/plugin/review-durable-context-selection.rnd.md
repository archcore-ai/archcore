---
title: "Review Durable Context Selection — Shipped Behavior, Gaps, Evidence Model, and Case Matrix"
status: draft
tags:
  - "component:plugin"
  - "plugin"
  - "precision"
  - "skills"
---

## Goal

Decide how `/archcore:review` selects the durable context a later agent needs after a change — with a matching `plan`, with a partly matching one, or with none — and bound the resulting document set so the corpus gains contracts, reasons, and procedures rather than prose that restates readable code.

## Questions

1. What does the shipped review do for a planned, a partly planned, and a planless change, and where are the gaps?
2. Which signal in a change justifies a document, and which type owns it?
3. Which evidence source establishes which kind of claim?
4. How does the selection avoid one Markdown file per changed file?
5. Which cases does a verification bench need to cover?

## Approach

Read the shipped skill and track files, the accepted specs and decisions that constrain them, and the structural tests at HEAD `b7cc60b` on 2026-10-09. No live-model run; the case matrix in F6 is the input for a later bench. Sources consulted: `@plugin/plugins/archcore/skills/review/SKILL.md`, `@plugin/plugins/archcore/skills/_shared/tracks/closeout.md`, `@plugin/plugins/archcore/skills/_shared/tracks/experience.md`, `@plugin/plugins/archcore/skills/_shared/tracks/describe.md`, `@plugin/plugins/archcore/skills/_shared/spec-contract.md`, `@plugin/plugins/archcore/skills/_shared/precision-rules.md`, `@plugin/plugins/archcore/skills/_shared/elicitation-contract.md`, `@plugin/test/structure/`, and the accepted `command-surface-v2.spec`, `delta-routing-instruments.spec`, `prd-spec-plan-content-ownership.adr`, `plan-discharge-by-deletion.adr`, and `document-status-transitions.adr`.

## Findings

### F1 — Shipped behavior and gaps (Q1)

| Situation | Shipped behavior | Gap |
|---|---|---|
| A plan matches the diff | Verify, merge, accept, capture named residue, remove the plan on confirmation. | Capture takes only residue the plan or reports name, forbids a `spec`, and runs no general selection of durable facts. |
| Closeout creates a new document | Accept runs before capture, so the new document stays `draft`. | The report has to show that status; nothing offers acceptance in the same run. |
| A plan matches but a task is unfulfilled | Closeout stops after verify and names the remaining work. | Delivered behavior stays undocumented until the whole plan completes (deferred, see the companion `adr`). |
| Code changed, no plan | Bidirectional check, then the repeated-pattern offer. | No path from the diff to documents about new behavior or decisions. |
| No plan and no documents | Empty corpus; the review proceeds on code. | No path to the first documents from one branch. |
| Code from another agent | Reviewed as any change. | No rule states that origin proves neither intent nor rationale. |
| One complex case, no repetition | Fails the two-file threshold in `experience.detect`. | A single exception, failure, or limit does not reach durable context. |
| Review never invoked | The skill receives no event. | Out of scope: needs a PR or CI trigger, not a skill change. |

### F2 — Signal in the change → what a later agent misses → owner (Q2)

| Signal | What the next agent misses | Owner |
|---|---|---|
| External interface, state, rule priority, or failure behavior | A contract condition invisible on one success path. | Update the covering `spec`; a new independent surface gets its own `spec`. |
| A choice between options with a known reason | Why the alternative lost and when the choice expires. | `adr`; never derived from the final code alone. |
| A repeatable operation with an order that matters | Prerequisites, the hazardous step, verification, recovery. | `guide` for a human actor; `task-type` for an agent actor. |
| A deliberately changed code practice | Before, after, scope, reason. | `cpat` when the lesson transfers beyond one edit. |
| An observable user flow with examples | Branches and the result for the actor. | `scenario` beside the covering `spec`, once examples are confirmed. |
| A registry or reference data looked up on its own | Where the list lives and how to read it. | `doc`, when an `@path` alone does not answer the task. |
| One unexpected failure or limit | The condition under which the old picture was wrong. | A Failure Behavior clause, an `adr` consequence, or a procedure's Pitfalls; no invented repeated pattern. |
| A local implementation that code and tests explain | Nothing durable. | No document; the report names the omission. |

Four questions decide each candidate: which future reader task the owner serves; whether code, tests, history, MR material, or the user confirms it; whether the claim survives an ordinary refactor; which existing document or type owns exactly that claim. A missing answer to the first or second question drops the candidate. A simple public contract can still need a `spec`; a complex internal implementation can need nothing.

### F3 — Evidence boundaries (Q3)

| Source | Establishes | Does not establish on its own |
|---|---|---|
| Code and diff | The implemented branch, interface, state, error handling. | That the behavior was intended, agreed, or verified in use. |
| Tests and a run report | The examples checked and the result of that run. | Full contract coverage or the product goal. |
| Plan and `Declared Delta` | The original intent and the promised scope. | That the work is done or that undeclared code is wrong. |
| PR/MR description, discussion, commits | The stated motivation and the trace of a choice, when recorded. | The truth of an unverified statement or agreement with every conclusion. |
| Accepted documents | The current contract and project constraints. | That the changed code still satisfies them. |
| The user's confirmation | Intent, status, and plan removal. | A test result that never ran. |

The review boundary is local git (`@plugin/plugins/archcore/skills/_shared/branch-state.md`); MR material counts only when supplied. The origin of the code is not a condition for recording. `code-wrong` and `spec-wrong` apply only where a document claim exists; a missing document is a candidate, not a conflict.

### F4 — Document economy (Q4)

Order of preference: leave unrecorded → cite code or a test → update the owner → create one missing document → split only by independent surfaces. No count is a target or a limit: one branch can yield zero documents, several independent contracts can yield several `spec`s. A `spec` keeps obligations, invariants, and failures; an `adr` keeps the verified reason; a procedure keeps the step the next executor would otherwise guess; each cites `@path`. A `journey` stays off this path: one implemented branch cannot turn an observation into an intended future path without confirmation; a discovered product goal is reported as an open question for `/archcore:plan`.

### F5 — Constraints found while grooming the drafts (2026-10-09)

- The elicitation ceiling is 5 questions per invocation and closeout already draws up to 4; one confirmation per write does not fit. One batched preview question does.
- `@plugin/plugins/archcore/skills/_shared/tracks/closeout.md` holds 268 of the 300-line track cap, so the selection procedure lives in a separate shared file.
- The selection runs after `closeout.merge` and before `closeout.capture`; `closeout.capture` searches for nothing of its own today, so the selection report becomes its input. Drafts created after `closeout.accept` wait for the next review's accept gate.
- Behavioral fixtures follow `<name>-bench.tsv` with a `<name>-bench.sh` runner in `@plugin/test/behavioral/`; a fixture file without a runner is dead.

### F6 — Case matrix for the bench (Q5)

| Case | Expected result |
|---|---|
| A branch implements a plan and adds behavior with a consumer; no `spec` exists. | One `spec` proposed for the confirmed contract; the plan closes after its confirmations. |
| A branch implements a plan; an existing `spec` covers the behavior. | A focused update; no second `spec`. |
| No plan, no documents; the branch adds an API, a procedure, and an internal helper. | `spec` for the API, `guide` for the procedure; the helper stays in code. |
| No plan; only an obvious refactor with tests. | No document; the report names the reason. |
| A simple public contract is readable in code but has no owner. | A `spec` that cites the code without restating it. |
| A complex internal algorithm, fully local, covered by tests. | No document on the ground of complexity alone. |
| Part of the branch matches a plan; another agent added the rest. | Declared Δ verified separately; durable units selected from the unplanned part. |
| One failure revealed a new recovery condition. | The Failure Behavior clause or procedure is updated; no repetition required. |
| Code diverges from an accepted `spec`; the MR description claims a new intent. | The conflict and its uncertainty are reported; the code is not declared canon. |
| A plan is partly fulfilled, with no unplanned change. | The plan stays; the delivered part is not documented in this run (deferred). |
| Only code exists; the reason for a technology choice is unknown. | Verifiable behavior is described; the reason stays open; no `adr` with invented motivation. |
| Code from another agent with no tests and no MR description. | Observed structure is separated from unconfirmed behavior; a new `spec` is not accepted as verified. |

Bench runs, 2026-10-09, Claude host (`@plugin/test/behavioral/review-bench.sh`):

- Current contracts, 3 repetitions: 35 of 36 pass. The failure was `unknown-rationale` creating a `cpat` with no recorded reason for the library switch.
- Fix: a `cpat` now needs an evidenced reason, and the experience offer skips a pattern the selection omitted. The three affected rows then passed 12 of 12.
- Sensitivity, pre-change contracts at HEAD `b7cc60b`: 6 of 12 fail. Four misses — `plan-missing-spec`, `planless-api-procedure-helper`, `simple-public-contract`, `mixed-origin` — and one excess, `unknown-rationale`, so those rows guard the change; `other-agent-no-tests` also fails as a miss.
- The six rows that pass on both versions — `plan-covered-spec`, `planless-refactor`, `complex-internal`, `one-case-failure`, `canon-conflict`, `partial-plan` — guard against a regression into excess documents; the old contracts never over-documented them.
- In `canon-conflict` the model updates the accepted spec after the user confirms the supplied MR intent; the row pins only `created=none`.
- Not measured: the codex and copilot hosts.

Two error kinds bound the bench: a miss — a contract or lesson the next agent needed and did not get — and an excess — a record of what the code already says. No numeric threshold exists yet: the route has produced no data.

## Recommendation

Proceed. Adopt the three-test selection rule (reader task, evidence, owner), one batched preview question, selection after `closeout.merge`, and one rule for planned and planless changes — supported by F1 through F5. Refine before implementation: the case matrix in F6 becomes the review bench before any skill text changes. Defer: documenting the delivered part of an unfinished plan, and a new type for a one-case lesson, until the bench shows the existing types cannot own a real finding.

## Next Action

No new `/archcore:plan` run is needed: `review-durable-context-selection.plan` decomposes this recommendation, and `durable-context-selection-by-reader-task.adr` records the decision.
