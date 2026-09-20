---
title: "Import Conversion and Staging — Target Documents, Waves, Verify, Retire, Discharge"
status: draft
tags:
  - "onboarding"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec governs how the import track of `/archcore:init` turns triaged sources into `.archcore/` documents: conversion, the import plan and its waves, verification, the retirement of source text, and the discharge of the plan. Dependents: users who run `/archcore:init import`, a plain init that hands over tiers `M` and `L`, and the `review` skill, which reads the created documents as native ones. Out of scope: how sources are found and triaged (discovery spec) and the code seed.

## Surface

- Track file: `skills/_shared/tracks/import.md`, gates `assess`, `discover`, `triage`, `plan`, `convert`, `verify`, `retire`, `discharge`.
- Routing file: `skills/_shared/grounding/convert-routing.md`. The file replaces @plugins/archcore/skills/_shared/grounding/extract-routing.md.
- Knowledge unit fields: `kind` (rule, decision, proposal, procedure, contract, reference, intent), `topic`, `scope_paths`, `source_spans`.
- Import plan: one `plan` document tagged `import-plan`; each row holds a target document, its source spans, its wave, and a state (`proposed`, `confirmed`, `done`, `dropped`).
- Tiers: `S` ≤ 8 target documents; `M` 9–40; `L` > 40. [assumption] Thresholds await bench calibration.

## Normative Behavior

1. The import track MUST cluster knowledge units from all sources by topic before it names a target document.
2. The import track MUST compose one target document per cluster under the content contract of its type.
3. The import track MUST map unit kinds to types: rule→`rule`, decision→`adr`, proposal→`rfc`, procedure→`guide`, contract→`spec`, reference→`doc`, intent→`idea`.
4. WHEN a cluster lacks content for a mandatory section, the import track MUST choose a type whose contract the content satisfies.
5. The import track MUST create every document with `status: draft`.
6. The import track MUST place each document in a directory named for its domain or topic.
7. A created document MUST NOT carry an `imported` tag, a `source:` tag, a pointer line, or an `imported-` filename prefix.
8. WHEN a statement contradicts the code or another source, the import track MUST add it to the conflicts list.
9. WHEN `tier` is `S`, the import track MUST run one preview and one confirm without a plan document.
10. WHEN `tier` is `M` or `L`, the confirm MUST create the import plan before the first target document.
11. WHEN `tier` is `L`, the import track MUST stop after wave 1 and report the open waves.
12. The import track MUST order rows by kind — rule, decision, contract, procedure, reference — then by `last_change`.
13. WHEN a wave ends with an import plan, the import track MUST persist its row states in one `update_document` call.
14. WHEN conversion finds a new in-repository source, the import track MUST append its rows with the state `proposed`.
15. The import track MUST NOT convert a `proposed` row before the user confirms that row at a checkpoint.
16. WHEN an import plan has open rows, the import track MUST resume at the first `confirmed` or `proposed` row.
17. The `verify` gate MUST compare each normative statement of a target `rule` or `adr` with its source span.
18. The `retire` gate MUST show, per agent-instruction file, the sections the created documents cover.
19. WHEN the user confirms `retire`, the import track MUST remove only the covered sections of agent-instruction files.
20. WHEN the last row closes and an import plan exists, the import track MUST delete that plan with `remove_document`.
21. WHEN `tier` is `S`, the import track MUST keep row results in session memory through closing.
22. WHEN `tier` is `S`, the import track MUST NOT persist bookkeeping in a plan, state block, or side file.
23. WHEN a resumed plan holds `proposed` rows, the import track MUST open their checkpoint before conversion.
24. WHEN the user confirms proposed rows at a checkpoint, the import track MUST convert those rows in the current wave.
25. WHEN a target conversion completes, the import track MUST mark its row `done`.

## Constraints & Invariants

- Constraint: no target document exceeds 200 lines, and a `spec` stays within the 120-line cap of `spec-contract.md`; an over-cap cluster splits by sub-topic.
- Constraint: `retire` MUST NOT edit `README`, files under `docs/`, or any L2–L4 source; those files have readers outside the agent.
- Invariant: no document is created that the user has not seen in a confirmed plan or preview.
- Invariant: the persistent source-to-target map lives only in the import plan. Tier `S` retains the map only in the current session.
- Invariant: the archcore managed block survives `retire` unchanged.
- Invariant: a body never enumerates other `.archcore/` documents; links go through `add_relation`.

## Failure Behavior

1. IF the fidelity comparison finds a changed modal or a dropped condition, THEN the `verify` gate MUST report the document and keep it `draft`.
2. IF the git tree holds uncommitted changes to a file `retire` would edit, THEN the `retire` gate MUST exclude that file.
3. IF a file joins others through an `@import` chain the track cannot resolve, THEN the `retire` gate MUST exclude that file.
4. IF a resumed plan names a source path that no longer exists, THEN the import track MUST set the row to `dropped` and report it.
5. IF `create_document` fails for one row, THEN the import track MUST leave the row `confirmed` and continue with the next row.
6. IF the user cancels at a confirm, THEN the import track MUST stop further conversion.
7. IF a checkpoint is cancelled with a plan present, THEN the import track MUST save finished rows as `done` in its wave update.
8. IF a checkpoint is cancelled with a plan present, THEN the import track MUST retain unresolved rows as `proposed`.
9. IF a tier `S` run is interrupted, THEN the import track MUST restart at assessment.
10. WHEN every plan row is closed, the import track MUST resume at the first unfinished closing gate.

## Conformance

An implementation conforms when it satisfies behaviors 1–25, holds the four invariants and both constraints, and follows the ten failure rules. Regression checks: @test/structure/init-skill.bats and @test/fixtures/goldens/import.golden.

Given a resumed plan with only `done`, `dropped`, and `proposed` rows
When the user confirms the proposed rows
Then the import converts those rows in the current wave.
