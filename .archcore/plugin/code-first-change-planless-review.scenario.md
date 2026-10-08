---
title: "Code-First Change — Durable Context From a Branch With No Plan"
status: draft
tags:
  - "actor:plugin-user"
  - "commands"
  - "component:plugin"
  - "plugin"
---

## Subject

A branch written without an Archcore plan — by the user or by another agent — reaches `/archcore:review`, which selects the durable context a later reader needs. Cross-spec scenario; illustrates `command-surface-v2.spec` clause 39 and `review-durable-context-selection.spec` clauses 3, 4, 6, 7, 8, 11, 14, 17, 19, 20, 21, 23 and Failure Behavior 6. Plugin users read it to know what review writes without a plan; the review bench depends on its examples.

## Actors

| Actor | Who they are | What they want |
|---|---|---|
| Developer | a user who merges a branch with no Archcore plan | the contracts of the change recorded, nothing the code already says |
| Agent | a coding agent outside Archcore that wrote part of the branch | no requirement to follow an Archcore flow |

## Flows

### Developer

Anchors: @plugin/plugins/archcore/skills/review/SKILL.md, @plugin/plugins/archcore/skills/_shared/durable-context-selection.md, @plugin/plugins/archcore/skills/_shared/tracks/describe.md, @plugin/plugins/archcore/skills/_shared/tracks/experience.md, @plugin/test/behavioral/fixtures/review-bench.tsv

1. Developer types `/archcore:review` on the branch; the skill finds no matching plan.
2. Developer reads the bidirectional check; conflicts carry `spec-wrong`, `code-wrong`, or `ok`.
3. Developer reads the selection; each unit names its claim, evidence, owner, and reader task.
4. Developer reads the omitted units; each names why code and tests already answer it.
5. Developer answers one preview question; the skill writes only the authorized units as drafts.
6. Developer reads the report; units are grouped created, updated, omitted, deferred, and declined.

Extensions:

- 3a. A choice has no recorded reason; the skill reports an open gap and proposes no `adr`.
- 3b. No unit passes the tests; the skill writes nothing and asks no question.
- 5a. Developer authorizes a subset; the declined units appear in the report as declined.

### Agent

Anchors: @plugin/plugins/archcore/skills/_shared/branch-state.md

1. Agent commits code on the branch; no Archcore command runs.
2. Agent leaves no MR description; the review uses local git and the code as evidence.

## Examples

Background: a repository with an initialized `.archcore/`, the plugin from branch `dev`, a branch with no matching `plan`.

### An API, a procedure, and a helper

Illustrates: `review-durable-context-selection.spec` 6, 7, 11, 14, 19, 20.
Given the branch adds `GET /v1/reports`, `scripts/rotate-keys.sh`, and an internal `slugify` helper.
When Developer types `/archcore:review`.
Then Developer sees a preview with one `spec` and one `guide`, asked as one question.
And Developer sees `slugify` listed as omitted.

### A refactor with passing tests

Illustrates: `review-durable-context-selection.spec` 8, 14.
Given the branch renames three private functions, and the CI report shows tests passing.
When Developer types `/archcore:review`.
Then Developer sees no preview and no question.

### A library switch with no reason recorded

Illustrates: `review-durable-context-selection.spec` 17.
Given the branch replaces `node-fetch` with `undici`, and the commit message reads "switch http lib".
When Developer types `/archcore:review`.
Then Developer sees the reason reported as an open gap, and no `adr` or `cpat` draft.

### Declining the preview

Illustrates: `review-durable-context-selection.spec` 21, 23; Failure Behavior 6.
Given the preview proposes a `spec` and a `guide`.
When Developer authorizes the `spec` only.
Then Developer sees one `spec` draft, and the `guide` listed as declined.

## Open Questions

- Examples 1–3 match review bench rows `planless-api-procedure-helper`, `planless-refactor`, and `unknown-rationale` (Claude, 2026-10-09); example 4 is unconfirmed.
- The flow ships after plugin 0.10.14; the installed release goes from the check straight to the experience offer.
