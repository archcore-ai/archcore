---
title: "Review on the Default Branch Scopes to Unpushed and Uncommitted Work, and the Health Dashboard Is Removed"
status: accepted
tags:
  - "commands"
  - "component:plugin"
  - "plugin"
  - "skills"
---

## Context

On 2026-10-16 a trace of `/archcore:review` on `main` with an uncommitted edit to `shop/delivery.py` showed that `@plugin/plugins/archcore/skills/_shared/branch-state.md` stopped at step 4 with the sentinel `on-default-branch` before step 8 read `git status --porcelain`, so the edit was never in scope and the skill printed the project health dashboard instead of a review. The user reported the dashboard as unused and the behavior as a defect. Behavior 3 of the Failure Behavior section in `command-surface-v2.spec` required that dashboard.

## Decision

Adopted on 2026-10-16: `branch-state.md` resolves the default branch's base as `@{upstream}` when one exists and `HEAD` otherwise, so a default-branch review scopes the unpushed commits plus the uncommitted changes; the `on-default-branch` sentinel and the project health dashboard are removed from `@plugin/plugins/archcore/skills/review/SKILL.md`, and `empty-diff` asks for a path, topic, or ref like every other sentinel.

## Alternatives Considered

1. Keep the sentinel and the dashboard, add a hint to pass a path — rejected because the hint still hides the uncommitted work the user asked to review, and the dashboard carried counts no documented reader task consumed.
2. Scope a default-branch review to `git status --porcelain` only, ignoring unpushed commits — rejected because a local commit on `main` is as unreviewed as an uncommitted edit; `@{upstream}` captures both for one extra git probe.
3. Keep a reduced health line on `empty-diff` — rejected because a review with nothing to review has nothing to say, and `archcore status` already reports structure counts in the CLI.

## Consequences

Positive:

- A review on `main` or `master` with local work reviews that work; the demo trace on 2026-10-16 resolved a one-file scope where the previous contract resolved a sentinel.
- One fewer sentinel and one fewer routing branch in the review skill; the structure suite dropped the dashboard assertions and passes at 392 tests.

Negative:

- A default-branch review without an upstream sees only uncommitted changes; local commits on a branch with no remote stay out of scope. [expected]
- Health counts are no longer available from the plugin; users who want them run `archcore status`.
- The review argument hint shows example phrases of the free text; `skill-file-structure.rule` item 20 was amended on 2026-10-17 to allow plain-language examples while still forbidding type slugs, bare route names, and retired mode words.

## Superseded when

- A reader task for a health dashboard is documented, with the counts it consumes.
- `branch-state.md` gains a user-supplied base ref, which makes the upstream rule one option among several.
