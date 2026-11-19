---
title: "Proof-Tracked Plans — a Checked Task Carries Its Proof, and Only the Frontier Phase Holds Tasks"
status: draft
tags:
  - "architecture"
  - "component:cli"
  - "component:plugin"
  - "plugin"
  - "skills"
---

## Context

A field audit of 94 plans across 19 repositories found that about 70% of the living plans older than 14 days had their work wholly or partly landed while the plan stayed open, and that 89 of about 101 removed plans left through bulk corpus audits rather than through closeout. In 1,069 field sessions, a resumed session opened the plan 2 times out of 28 "continue" requests and rebuilt the state from git instead, because the plan template in `@cli/templates/templates.go` records tasks as numbered steps with no progress state. OpenSpec, spec-kit, and Kiro track progress with checkboxes that the implementing agent ticks without evidence, and each has open issues about a ticked task whose work was not done. A trial on one public ticket found that a plan that names a check per task and records a proof per tick was ranked first by one of two blind model reviewers, and every one of its 6 proofs failed when the task's code was reverted. [assumption] One ticket in one small repository does not establish the effect on a large corpus.

## Decision

Adopted the proof-tracked plan form of `@plugin/plugins/archcore/skills/_shared/plan-contract.md`: a plan whose `## Tasks` holds a `[ ] T<n>` task details only the frontier phase, each task names its files, covered spec clauses, and one check, a checked task counts as done only with a `proof: <check> @ <commit sha>` line, a spec specified ahead of code carries a check map that names checks only, and a plan without a `[ ] T<n>` task keeps the legacy behavior in every skill and hook.

## Alternatives Considered

1. A new plan section layout with Frontier, Next, Done, and Decisions headings — rejected because the CLI precision scan requires Goal, Tasks, and Acceptance Criteria (`@cli/templates/precision.go`), and the content-kind ownership table assigns the reason for a choice to `adr`, not to `plan`.
2. Stable clause identifiers such as `NEW-4` in a new spec section — rejected because the precision scan grades only numbered Normative Behavior clauses, so a renamed section would drop every clause out of the word and modal checks; the clause number serves as the identifier.
3. Removing a checked task from the plan at each phase change — rejected because the trial lost one proof in that collapse and because removal renumbers the tasks that other documents cite.
4. A revert probe run by a hook on every commit — rejected because the probe runs the project's test suite, which can reach networks, databases, or paid services, and because a hook must stay advisory (`document-status-transitions.adr`).
5. Converting existing plans to the new form — rejected because a legacy plan carries no proof to record, and an automatic rewrite of a user's corpus changes documents nobody asked to change.

## Consequences

- A plan with a `[ ] T<n>` task in Tasks shows its frontier and next task in the session-start context; a legacy plan shows nothing new. [expected]
- `closeout.verify` judges a checked task without a valid proof as unfulfilled, so `closeout.discharge` keeps that plan in the corpus.
- Recognition needs `T<n>` after the checkbox, so an existing plan with `1. [x] …` tasks stays legacy.
- The precision scan is unchanged and counts `[`, `]`, and `T<n>` as words; the contract caps a task line at 17 words after `T<n>`, and the blank line after each task line keeps its detail list out of the count on every CLI version.
- A task is checked only after its change is committed; the closeout track still neither stages files nor creates commits.
- The proof commit lives only in the plan; the spec check map names the check, so the sha has one owner.
- A CLI older than this change renders no progress line; every rule still holds through the skills, so the plugin needs no version gate.
- A plan created through `create_document` with empty content still receives the legacy template and stays a legacy plan.
- The revert probe stays an opt-in step in a temporary copy of the repository, a clone or a worktree, after the user confirms it.
- The proof form is pinned: `proof: <check> @ <commit sha>`, with nothing after the sha. The session-start `PLAN PROGRESS` header names the form, and the CLI counts a proof in another form as a claim. In trial runs without the skill, the implementing agent wrote `<sha>; <note>`.
- Leaving this form costs a plugin revert and a CLI release that drops `PLAN PROGRESS`. No corpus migration is needed: a proof-tracked plan reads as a numbered task list to a legacy skill, and a released CLI keeps the header only for plans with a `[ ] T<n>` task.

## Superseded when

- A field corpus shows that checked tasks with proofs still fail closeout verification in more than 1 of 10 plans.
- The CLI gains a machine check that runs the proof itself without a user confirmation.
