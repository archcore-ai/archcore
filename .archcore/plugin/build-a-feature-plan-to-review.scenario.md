---
title: "Building a Feature — From /archcore:plan to Plan Removal at /archcore:review"
status: draft
tags:
  - "actor:plugin-user"
  - "commands"
  - "component:plugin"
  - "plugin"
---

## Subject

The full build cycle of the plugin: plan a feature, implement it, review the branch, close the plan. Cross-spec scenario; illustrates `delta-routing-conductor.spec` clauses 1, 6, 8, 11, 19; `command-surface-v2.spec` clauses 6, 38, 40; `delta-routing-instruments.spec` clauses 18, 24, 25, 33; `review-durable-context-selection.spec` clauses 2, 19, 20. Plugin users read it to learn the main path; the routing and review benches depend on its examples.

## Actors

| Actor | Who they are | What they want |
|---|---|---|
| Developer | a user who builds a feature with a coding agent | documents that match the shipped code, with few questions |
| Agent | the coding agent that implements the plan's tasks | the decisions and contracts that constrain the files it edits |

## Flows

### Developer

Anchors: @plugin/plugins/archcore/skills/plan/SKILL.md, @plugin/plugins/archcore/skills/_shared/delta-routing.md, @plugin/plugins/archcore/skills/_shared/tracks/sdd.md, @plugin/plugins/archcore/skills/review/SKILL.md, @plugin/plugins/archcore/skills/_shared/tracks/closeout.md

1. Developer types `/archcore:plan <feature>`; the plan skill searches `.archcore/`, git state, and code.
2. Developer reads the route announcement; it names the route, the size label, and the Δ.
3. Developer answers at most the open user-owned choices; the skill drafts a `spec` and a `plan`.
4. Developer asks Agent to implement the plan's tasks on a branch.
5. Developer types `/archcore:review`; the review skill compares changed code and documents in both directions.
6. Developer reads per-task verdicts; the closeout track judges every task and acceptance criterion.
7. Developer confirms the `spec` update and its acceptance; the skill changes the status to accepted.
8. Developer answers the durable-context preview once; the skill writes the authorized drafts.
9. Developer confirms the plan removal; the skill states whether git preserves the plan, then removes it.

Extensions:

- 2a. The request creates two capabilities; the skill drafts an umbrella `prd`, one `spec` each, and one `plan`.
- 3a. A choice stays unanswered; the gate stops and names the resume command.
- 6a. A task is unfulfilled; the plan stays, and the report names the remaining work.
- 8a. Developer declines the preview; nothing is written, and the plan removal is still offered.

### Agent

Anchors: @plugin/plugins/archcore/bin/pre-tool-use, @plugin/plugins/archcore/bin/session-start

1. Agent starts a session; the session-start hook lists open drafts and recent decisions.
2. Agent edits a file the `spec` cites; the pre-tool-use hook names the documents that claim it.
3. Agent writes `.archcore/` through MCP only; a direct file write is denied.

## Examples

Background: a repository `shop-api` with an initialized `.archcore/`, CLI 0.10 on PATH, the plugin from branch `dev`.

### Planning one capability

Illustrates: `delta-routing-conductor.spec` 1, 6, 11, 19.
Given no `spec` covers CSV export of orders.
When Developer types `/archcore:plan CSV export for orders`.
Then Developer sees `route: capability (size M)` announced before any document.
And Developer sees drafts `orders-csv-export.spec` and `orders-csv-export.plan` with a Declared Delta.

### Planning two capabilities

Illustrates: `delta-routing-conductor.spec` 8.
Given no `spec` covers export or scheduled delivery.
When Developer types `/archcore:plan CSV export and scheduled email delivery`.
Then Developer sees an umbrella `prd`, two `spec` drafts, and one `plan`.

### Reviewing a fulfilled plan

Illustrates: `command-surface-v2.spec` 6, 38; `delta-routing-instruments.spec` 18, 24, 33.
Given every task of `orders-csv-export.plan` is done and committed on branch `feat/csv`.
When Developer types `/archcore:review`.
Then Developer sees the Declared Delta confirmed against the diff.
And Developer sees a removal offer that states the commit preserving the plan.

### Reviewing an unfinished plan

Illustrates: `command-surface-v2.spec` 40; `delta-routing-instruments.spec` 25.
Given 2 of 4 tasks of `orders-csv-export.plan` have no change on the branch.
When Developer types `/archcore:review`.
Then Developer sees the plan retained with the 2 remaining tasks named.

### A contract the plan did not declare

Illustrates: `review-durable-context-selection.spec` 2, 19, 20.
Given the branch also adds a 429 response the plan did not declare, and no `spec` covers it.
When Developer types `/archcore:review`.
Then Developer sees one preview proposing a `spec` update, asked as one question.

## Open Questions

- The examples are unconfirmed: no run report covers them except the review bench row `plan-missing-spec` (Claude, 2026-10-09).
- Clauses 2, 19, 20 of `review-durable-context-selection.spec` ship after plugin 0.10.14; the installed release does not show that preview.
