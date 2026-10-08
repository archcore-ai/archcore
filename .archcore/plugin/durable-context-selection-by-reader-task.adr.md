---
title: "Durable Context Selection by Reader Task and Evidence — Signals Are Not Document Triggers"
status: accepted
tags:
  - "component:plugin"
  - "plugin"
  - "precision"
  - "skills"
---

## Context

On a plain branch review, `@plugin/plugins/archcore/skills/review/SKILL.md` checks a matched `plan` and then offers only a repeated-pattern capture, and `@plugin/plugins/archcore/skills/_shared/tracks/closeout.md` captures residue only when the plan or the verify and merge reports name it, with no `spec` allowed at that gate (observed in the shipped text on 2026-10-09, HEAD `b7cc60b`). A branch with no plan — code from another agent, or work that began without documents — therefore leaves no durable record beyond an edit shape repeated in two files. The candidate heuristics for a replacement — document every changed file, document whatever is complex, document only what repeats — each fail one of the cases in the companion `rnd`: the first produces prose that restates readable code, the second skips a three-line public contract while documenting a tested local algorithm, the third misses a single failure case by construction. Any new per-document question also competes with the 5-question ceiling in `@plugin/plugins/archcore/skills/_shared/elicitation-contract.md`, of which closeout already draws up to 4.

## Decision

`/archcore:review` selects documentation units by three tests — a named future reader task, direct evidence (code, tests, history, supplied MR material, or the user's answer), and an existing owner or owning type — applies the same tests with or without a `plan` and whatever agent wrote the code, treats complexity, implicitness, and uncertainty as signals to inspect rather than triggers to write, runs after `closeout.merge` and before `closeout.capture`, and asks one batched preview question per invocation for every write.

## Alternatives Considered

1. One document per changed file or module — rejected because `precision-over-coverage.adr` records that context volume lowered agent task success in 5 of 8 settings (arxiv 2510.21413), and a file is not a claim a reader looks for.
2. Complexity as the trigger — rejected because a tested local algorithm needs no owner while a three-line public contract with a consumer does; the trigger inverts the reader's need.
3. Repetition as the only trigger, the shipped `experience.detect` two-file rule — rejected because a single failure case that changes a Failure Behavior clause never repeats inside one diff, so the rule misses it by construction.
4. A retrospective `plan` for planless work — rejected because a `plan` declares a future Δ (tense invariant in `delta-routing-instruments.spec`) and leaves the corpus at discharge, so the write and the removal would bracket no surviving content.
5. A new document type for a one-case lesson — deferred because no review has yet produced a finding that a `spec` Failure Behavior clause, an `adr` consequence, or a procedure's Pitfalls cannot own; revisit after the review bench runs.
6. One confirmation question per proposed write — rejected because closeout already draws up to 4 of the 5-question ceiling (`plan-discharge-by-deletion.adr`), so three candidates would exhaust the ceiling before the removal confirmation.
7. Documenting the delivered part of an unfinished `plan` — deferred because it conflicts with `command-surface-v2.spec` behavior 40 and the plain-review entry rule in closeout, and the selection rule is testable without it.

## Consequences

Positive:

- A planless branch can produce an evidenced `spec`, `guide`, `doc`, or `scenario` from one review invocation. [expected]
- A review invocation spends at most 1 question on document writes, leaving the ceiling for verify, merge, and removal confirmations. [expected]
- The two-file repetition rule stops being the only gate for experience capture. [expected]
- The reason for the rule survives plan discharge in this record; the draft spec carries the obligations and the `rnd` carries the investigation.

Negative:

- A branch review adds an inspection pass over relied-on contracts beyond the changed lines; the added invocation time is unmeasured. [expected]
- `command-surface-v2.spec` behaviors 39–40 and `delta-routing-instruments.spec` behavior 23 change, together with the structural tests that pin them (`@plugin/test/structure/command-grammar.bats`, `@plugin/test/structure/loop-integrity.bats`).
- A batched preview can hide one weak unit among sound ones; partial authorization is the only guard.
- Drafts the selection creates wait for the next review's accept gate, so the draft count rises between reviews. [expected]

## Superseded when

- The review bench records more than 1 excess document or more than 1 missed contract per 5 reviewed branches (current: no bench data).
- Three real reviews produce findings that no existing type owns, or the kernel gains a type for a one-case lesson.
- The per-invocation question ceiling changes so that one confirmation per write fits inside it.
