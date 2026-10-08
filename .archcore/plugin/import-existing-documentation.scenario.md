---
title: "Importing Existing Documentation With /archcore:init import"
status: draft
tags:
  - "actor:plugin-user"
  - "commands"
  - "component:plugin"
  - "plugin"
---

## Subject

A repository with agent instructions, ADR folders, and contributor docs moves that knowledge into native `.archcore/` documents. Cross-spec scenario; illustrates `command-surface-v2.spec` clause 37; `init-entry-and-assess.spec` clauses 1, 7, 19; `import-conversion-and-staging.spec` clauses 1, 3, 5, 7, 8, 9, 10, 11, 16, 19, 20. Plugin users read it before migrating; the import bench depends on its examples.

## Actors

| Actor | Who they are | What they want |
|---|---|---|
| Maintainer | a user who owns `CLAUDE.md`, `AGENTS.md`, and `docs/adr/` | one canon without duplicated instructions |

## Flows

### Maintainer

Anchors: @plugin/plugins/archcore/skills/init/SKILL.md, @plugin/plugins/archcore/skills/_shared/tracks/import.md, @plugin/plugins/archcore/skills/_shared/grounding/convert-routing.md, @plugin/test/behavioral/fixtures/import-bench.tsv

1. Maintainer types `/archcore:init import`; the skill assesses all five discovery levels.
2. Maintainer reads the assessment line; it names the target estimate and the tier.
3. Maintainer reads the preview; each target names its type and its source spans.
4. Maintainer confirms once; the skill creates the import plan and converts wave 1.
5. Maintainer types `/archcore:init import` again; the skill resumes at the first open row.
6. Maintainer reads the conflicts list; each entry names the source and the contradicting code.
7. Maintainer confirms the retire step; the skill removes only covered sections from agent files.
8. Maintainer reads the closing report; the skill deletes the import plan after the last row.

Extensions:

- 2a. The tier is `S`; one preview and one confirm convert everything, with no plan document.
- 4a. The tier is `M`; every wave runs in this session, with no second command.
- 2b. No authored source exists; the skill prints the no-source report and converts nothing.
- 5a. Conversion found a new source; its rows wait as `proposed` until Maintainer confirms them.
- 7a. Maintainer declines retire; the agent files stay unchanged.

## Examples

Background: a repository with `CLAUDE.md` (6 sections), `AGENTS.md`, `docs/adr/` (14 files), and an empty `.archcore/`.

### A small import

Illustrates: `import-conversion-and-staging.spec` 5, 7, 9.
Given the assessment counts 6 targets, tier `S`.
When Maintainer confirms the preview.
Then Maintainer sees 6 draft documents with no `imported` tag and no import plan.

### A large import in waves

Illustrates: `init-entry-and-assess.spec` 7; `import-conversion-and-staging.spec` 10, 11, 16.
Given the assessment counts 52 targets, tier `L`.
When Maintainer confirms the preview.
Then Maintainer sees an import plan, wave 1 converted, and `/archcore:init import` named to continue.

### A medium import in one session

Illustrates: `import-conversion-and-staging.spec` 10.
Given the assessment counts 22 targets, tier `M`.
When Maintainer confirms the preview.
Then Maintainer sees an import plan and every wave converted in this session.

### Kinds become types

Illustrates: `import-conversion-and-staging.spec` 1, 3.
Given `CLAUDE.md` holds a naming rule and `docs/adr/0007-postgres.md` holds a decision.
When the import converts both.
Then Maintainer sees one `rule` and one `adr`, each clustered by topic.

### Retiring covered instructions

Illustrates: `import-conversion-and-staging.spec` 19, 20.
Given every row is done, and 4 of 6 `CLAUDE.md` sections are covered.
When Maintainer confirms retire.
Then Maintainer sees 2 sections left in `CLAUDE.md`, and the import plan deleted.

## Open Questions

- The examples are unconfirmed: no run report or confirmation records them.
- The tier thresholds 8 and 40 are marked `[assumption]` in `import-conversion-and-staging.spec`.
