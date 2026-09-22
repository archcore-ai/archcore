---
title: "Init Import Mode — Target-Driven Conversion, Staged by a Plan, No Import Marks"
status: draft
tags:
  - "architecture"
  - "commands"
  - "component:plugin"
  - "onboarding"
  - "plugin"
  - "skills"
---

## Context

On 2026-09-20 the import inside `/archcore:init` probes 13 agent-file paths and copies what it finds: an aggregate file becomes a one-line link stub, a modular rule file becomes a verbatim body, and every result carries the tags `imported` and `source:<slug>`, a pointer line, and the `imported-` filename prefix (@plugins/archcore/skills/init/lib/agent-files.md, @plugins/archcore/skills/_shared/grounding/extract-routing.md). Decision records, contributor docs, published docs sites, and git history are not read. `magic-first-day-init.adr` places this import in the main flow as one preview line and rejects staging with resume, and @plugin/plugins/archcore/skills/init/SKILL.md holds 442 lines against the 300-line maximum of `skill-file-structure.rule`. The user asked for a mode that migrates the whole authored context of a repository into native documents, with no mark that a document came from an import.

## Decision

`/archcore:init` takes the form `[import|refresh] [path or domain]`, and its `import` mode converts authored sources of five discovery levels into native documents by target-driven conversion — knowledge units clustered by topic, one document per cluster composed under its type contract — staged by an import `plan` document with one confirm per wave, finished by a separate `retire` gate for agent-instruction files and by deletion of the plan.

This decision replaces the "Imports" paragraph of `magic-first-day-init.adr` and limits that ADR's rejection of staging (alternative 6) to the code seed. The seed keeps one preview and one confirm. A plain init runs host wiring first, then an assessment gate that measures the authored backlog; the extractive facts are created in every route, and modules that authored sources cover leave the hotspot pool. `--depth` and `--scale` leave the signature and stay as the preview toggles `depth:` and `scale:`; `domain <slug>` becomes the subject of `refresh`.

## Alternatives Considered

1. **Source-driven conversion (file → blocks → one document per block), an upgrade of `extract-routing.md`** — rejected because the corpus then copies the structure of the sources, and two sources on one topic produce two documents.
2. **Import in one confirm and one pass, no plan document** — rejected because a target document merges spans of several sources and takes its name from the topic, so without `source:` tags an interrupted run cannot tell which spans are converted and a second run creates duplicates.
3. **Keep provenance tags and the pointer line** — rejected because the user requires a corpus with no import marks, and the import plan carries the source-to-target map while the work is open.
4. **Edit the source files in the same confirm as the conversion** — rejected because the user would approve the removal of authored text before seeing the converted documents.
5. **Keep `--depth` and `--scale` as flags beside the new mode** — rejected because init always stops at a confirm gate, where the preview already shows each depth's cost, so a flag adds a second place for the same choice.

## Consequences

### Enabled

- Discovery rises from 1 level (13 paths) to 5 levels: agent instructions, decision records, contributor docs, internal pages of a published docs site, git history.
- A converted corpus holds 0 documents with an import mark after the plan is removed per `plan-discharge-by-deletion.adr`.
- [expected] An interrupted import above 40 target documents resumes from the plan and creates 0 duplicate documents.
- `init/SKILL.md` drops from 442 lines to 300 or fewer, with the flows in `init/lib/` and one track file.

### Costs and limits

- Rewriting an authored rule under a type contract can shift its meaning; one changed modal changes agent behavior. The `verify` gate compares each normative statement with its source span, and every document is created as `draft`. [expected] The residual risk stays material for `rule`.
- Import takes more than one confirm when the plan grows: a source found during conversion is appended as `proposed` and waits for the end-of-wave confirm, which keeps the guarantee that no document is created unseen.
- The coverage set that narrows the hotspot pool is an estimate from path mentions in unread sources, so a plain init can budget one spec too many or too few per covered module.
- Breaking change for recorded invocations: `init domain <slug>`, `--depth=<tier>`, `--scale=<mode>`.
- The size-tier thresholds (8 and 40 target documents) are [assumption] until the behavioral bench calibrates them.

## Superseded when

- The behavioral bench shows a resumed import creating 1 or more duplicate documents on a supported model.
- The fidelity check reports a changed modal in more than 1 of 20 converted `rule` documents.
- Users decline the `retire` gate in more than half of recorded runs, which would show that source files are kept on purpose.
