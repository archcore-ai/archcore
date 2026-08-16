---
title: "Π Profile as the Elicitation Driver — Replacing Budget-from-Vagueness"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Summary

Drive interviews from the conductor's Π gap profile — the recorded `user`-source information needs — instead of the vagueness-triggered per-gate budgets, keeping the 5-question per-invocation ceiling as the safety bound. This is the deferred phase-2 item of the delta-routing rollout, recorded here before any edit to the elicitation contract.

## Motivation

Two overlapping systems now decide questioning: Π says *what* is missing and *who* holds it, while the elicitation contract's triggers and per-gate budgets say *how many* questions vagueness licenses. The conductor already derives the need list at Route time with a citation duty on `machine` claims; the budget mechanism re-detects the same gaps later, per gate, from wording heuristics. One driver would remove the duplicated detection and make every asked question traceable to a recorded need.

## Detailed Design

- The `user`-source entries of Π become the question queue for the whole invocation; each question names the need it resolves.
- The impact × uncertainty ranking and the 5-question auto-mode ceiling stay unchanged as the bound.
- Per-gate `budget` knobs remain only as expert-invocation caps; auto-mode gates stop deriving questions from their own trigger wording and consume the queue instead.
- Answered needs de-escalate to `machine` in the state block; `deferred` keeps unfunded needs, as today.
- Rollout gate: the behavioral routing harness runs before and after the switch; a drop in trace reproduction blocks the change.

## Drawbacks

- Gate-level trigger prose and budget values are pinned by goldens and routing fixtures — the switch reworks those pins across every track file.
- A mis-derived Π starves a needed question that today's per-gate trigger would still catch; the ceiling caps the damage but does not restore the question.
- The elicitation contract is shared by all four commands; `document` and `review` do not run the conductor, so they would keep the old triggers — two questioning modes in one plugin until a follow-up unifies them.

## Alternatives

- Status quo — Π engages instruments, budgets drive question counts; duplicated detection but zero migration cost.
- Π as a ranking input only — budgets keep licensing questions, Π reorders them; smaller change, keeps both systems alive.