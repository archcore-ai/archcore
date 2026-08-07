---
title: "Elicitation Contract — Bounded User Interview"
status: accepted
tags:
  - "plugin"
  - "precision"
  - "skills"
---

## Purpose & Scope

This spec defines the bounded user interview: when a skill asks the user questions, how many, in what form, and where answers persist. Normative for every command and track gate in the plugin; shipped as prose in `skills/_shared/elicitation-contract.md`. Out of scope: per-gate question lists (owned by track files) and per-type coverage taxonomies (owned by content contracts).

## Surface

- Trigger signals: absent or vague arguments; a coverage scan returning Missing on a material category; two or more viable alternatives on one decision; code-versus-document conflict.
- Materiality filter: architecture, data model, task decomposition, tests, UX behavior, operations, compliance (per Spec Kit `/clarify`, https://github.com/github/spec-kit).
- Question form: one interrogative; a one-line "why it matters"; a recommended option stated first; 2–4 options plus a free answer plus "you decide".
- Budget: ceiling of 5 questions per invocation in auto mode; expert invocation raises per-gate budgets up to the track's declared maximum.
- Write-back targets: `## Clarifications` session log in the draft artifact; `[assumption]` markers; the track state block.

## Normative Behavior

1. Before asking a question, the skill MUST attempt to answer it from `.archcore/` search, the codebase, git history, and prior clarifications.
2. The skill MUST drop any candidate question that fails the materiality filter.
3. WHEN candidate questions exceed the budget, the skill MUST rank them by impact times uncertainty and ask only the top ones.
4. The skill MUST state a recommended answer with every question.
5. WHILE the host offers no question widget, the skill MUST present exactly one question per message.
6. WHEN the host offers a question widget, the skill MAY batch up to 4 related questions in one call.
7. A subagent MUST NOT conduct an interview.
8. WHEN the user answers "you decide" or "I don't know", the skill MUST adopt the recommended answer and mark it `[assumption]` in the artifact.
9. IF the user delegates twice in a row, THEN the skill MUST end the interview and finish on recorded assumptions.
10. WHEN a gate closes, the skill MUST write accepted answers under `## Clarifications` in one `update_document` call.
11. WHEN the budget is exhausted, the skill MUST record remaining material questions under `Deferred` with a one-line reason each.
12. IF no candidate question passes the materiality filter, THEN the skill MUST skip the interview and state that no material ambiguity was found.
13. The skill MUST NOT count a re-asked disambiguation against the budget.

## Constraints & Invariants

- Constraint: the widget-versus-prose switch lives only in this contract; a skill file MUST NOT contain host-conditional text.
- Constraint: a hook MUST NOT ask the user a question.
- Constraint: a hook MAY inject a notice of unresolved assumptions.
- Constraint: auto mode MUST NOT exceed the 5-question ceiling regardless of how many gates run.
- Invariant: grounding (behavior 1) precedes every question on every host.

## Failure Behavior

1. IF the user interrupts the interview, THEN the skill MUST proceed on recorded answers and mark unresolved material items `[assumption]`.
2. IF the user rejects the recommended answer without giving an alternative, THEN the skill MUST re-ask once with the option list.
3. IF the draft artifact for write-back does not exist, THEN the skill MUST create the draft via `create_document` before writing clarifications.

## Conformance

An interview implementation is conformant when it satisfies behaviors 1–13, holds all invariants, and degrades per the failure rules.