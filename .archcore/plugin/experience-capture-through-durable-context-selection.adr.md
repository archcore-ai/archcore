---
title: "The Experience Track Is Absorbed into the Durable-Context Selection, Which Captures Every Type"
status: draft
tags:
  - "component:plugin"
  - "document-types"
  - "plugin"
  - "skills"
---

## Context

A headless `/archcore:review` on 2026-10-16 over a six-file branch routed a reasoned cross-file practice change (`shop/errors.py`, `ShopError` base) to a `rule` through the durable-context selection, then printed `experience: none — no detection signal`, because the experience track in `skills/_shared/tracks/experience.md` skipped every pattern the selection had already listed. Two procedures extracted lessons from one diff, the second one almost always skipped, and the visible report line described only the second. The selection already wrote `spec`, `doc`, `guide`, `scenario`, `adr`, `rule`, `cpat`, and `task-type` through the owning instruments, so the experience track added no type the selection could not produce.

## Decision

Adopted on 2026-10-16: the experience track (`experience.detect`, `experience.offer`) is deleted; its transfer test, actor rule, incident-failure rule, and body requirements for `cpat`, `task-type`, and procedural `guide` move into `@plugin/plugins/archcore/skills/_shared/durable-context-selection.md`, capture wording runs that selection over the named scope, and the selection report ends with one line `captured: <types>` or `captured: none — <reason>`.

## Alternatives Considered

1. Rename the experience track to `retro` and let it produce any type — rejected because it duplicates the selection under a third name and breaks the one-term-per-concept rule of `AGENTS.md`; the user's premise that capture was bound to `cpat` and `task-type` held only for the residual track, not for the selection.
2. Keep both procedures and fix the report line only — rejected because the detect gate's `skip_when` fires whenever the selection has run, which is every branch review, so the track stays dead code with two gates, a golden, and bench contracts to maintain.
3. Add new experience types for one-case lessons — deferred by `durable-context-selection-by-reader-task.adr` until the review bench shows a lesson no `spec` Failure Behavior clause, `adr` consequence, or procedure Pitfalls can own; the 2026-10-16 trace produced no such case.

## Consequences

Positive:

- One capture procedure in review; the track catalog goes from eight tracks and 35 gates to seven tracks and 33 gates.
- The report line covers every captured type, so a `rule` extracted from a practice change counts as captured instead of reporting `none`.
- The structure suite passes at 392 tests after deleting the track, its golden, and its purity entry.

Negative:

- The `experience` category keeps two types while its capture has no dedicated track; a reader may look for an experience track and find only the selection. [expected]
- The matrix bench field `experience=` keeps its name as an outcome measure; its 23 fixtures in `matrix-experience.tsv` are not re-labeled. [expected]

## Superseded when

- The review bench records a pattern the selection omitted that a separate detection pass would have captured.
- A lesson appears that no existing type owns, which reopens the deferred new-type alternative.
