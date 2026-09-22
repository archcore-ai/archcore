---
title: "Init Rework — Import of Authored Context and a Shorter Entry"
status: draft
tags:
  - "commands"
  - "component:plugin"
  - "onboarding"
  - "plugin"
  - "skills"
---

## Vision

`/archcore:init` moves a repository's authored context — agent instructions, decision records, contributor docs — into native `.archcore/` documents, so the agent reads one typed corpus instead of scattered files. The command keeps one short entry form, `[import|refresh] [path or domain]`.

## Problem Statement

A team that adopts Archcore in an existing repository keeps its knowledge in `CLAUDE.md`, `AGENTS.md`, rule folders, ADR folders, and `docs/`. Today init reads 13 agent-file paths and leaves a link stub or a verbatim copy tagged `imported` (@plugins/archcore/skills/init/lib/agent-files.md). The team then carries two copies of each convention, the agent receives the same text twice, and decision records stay outside the relation graph. The plugin maintainer carries a second cost: @plugin/plugins/archcore/skills/init/SKILL.md holds 442 lines, 142 above the maximum the other three skills meet.

## Goals and Success Metrics

- Discovery levels read by import: 5 (today: 1).
- Documents with an import mark in a converted corpus, after the plan is removed: 0 (today: every imported document).
- Duplicate documents created by a resumed import: 0 on the bench fixture above the large threshold.
- `init/SKILL.md` body: 300 lines or fewer (today: 442); each file under `init/lib/`: 200 lines or fewer.
- Entries and settings in the init argument hint: 2 modes and 1 subject (today: 2 modes and 2 flags).
- [assumption] Behavioral bench: 6 of 6 import fixtures pass on the supported model.

## Requirements

1. A user who runs a plain init in a repository without host wiring gets the wiring before any content question.
2. A plain init tells the user how much authored context the repository holds before it composes anything.
3. The extractive facts reach the corpus in every route, whatever the authored backlog.
4. A module that authored sources already describe does not also receive a synthesized spec.
5. An import finds authored context beyond agent files: decision records, contributor docs, internal pages of a published docs site, and recoverable git history.
6. Pages written for end users of a published docs site stay out of the corpus; the corpus records where that site lives.
7. Each converted document reads as a native document of its type and carries no mark of its origin.
8. A statement that contradicts the code, or a second source, reaches the user as a conflict instead of reaching the corpus.
9. No document is created before the user has seen it in a plan and confirmed it.
10. A large import leaves a plan the user can resume in a later session without duplicates.
11. Source files change only after the user has seen the converted documents and confirmed the removal, and only agent-instruction files change.
12. A contributor finds each init flow in its own file within the repository's skill size limits.

## Out of Scope

- Wikis, issue trackers, and other stores outside the repository.
- Code comments as a source.
- A non-interactive init; every route stops at a confirm.
