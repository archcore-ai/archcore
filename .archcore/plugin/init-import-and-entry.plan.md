---
title: "Init Rework — Split the Skill, Add the Assessment Gate, Build the Import Track"
status: draft
tags:
  - "commands"
  - "component:plugin"
  - "onboarding"
  - "plugin"
  - "skills"
---

## Goal

Ship `/archcore:init [import|refresh] [path or domain]`: a skill file inside the 300-line maximum, an assessment gate that divides the fill, and an import track that replaces the link-and-copy import.

## Tasks

### Phase 1 — Split the skill without changing seed behavior

1. Move phases A (detect) and the announce step into `init/lib/seed-detect.md`. Source: @plugin/plugins/archcore/skills/init/SKILL.md.
2. Move phases B–E and the closing messages into `init/lib/seed-compose.md`. Source: @plugin/plugins/archcore/skills/init/SKILL.md.
3. Move the CLI pre-flight and the host-wiring cascade into `init/lib/host-wiring.md`. Source: @plugin/plugins/archcore/skills/init/SKILL.md.
4. Reduce `init/SKILL.md` to arguments, when to use, routing table, execution order, and result.
5. Point the string assertions at `init/SKILL.md` plus `init/lib/*.md`. Target: @plugin/test/structure/init-skill.bats.
6. Add a structure test for the 300-line and 200-line maxima on `init`. Target: @plugin/test/structure/init-skill.bats.

### Phase 2 — New entry form

7. Set the hint `[import|refresh] [path or domain]` in @plugin/plugins/archcore/skills/init/SKILL.md and @plugin/plugins/archcore/commands/init.md.
8. Replace the `domain <slug>` mode with the `refresh` subject in the seed files and in @plugin/plugins/archcore/skills/_shared/grounding/detect-domains.md, @plugin/plugins/archcore/skills/_shared/grounding/detect-hotspots.md, @plugin/plugins/archcore/skills/_shared/grounding/detect-scale.md, @plugin/plugins/archcore/skills/_shared/grounding/detect-cross-cutting.md, @plugin/plugins/archcore/skills/init/lib/compose-overview.md.
9. Replace the `--depth` and `--scale` settings with the preview toggles `depth:` and `scale:`.
10. Add the retired-form notice for `domain`, `--depth`, `--scale`.
11. Update the hint table, the flag test, and the retired-form grep. Target: @plugin/test/structure/command-grammar.bats.
12. Update the init rows. Target: @plugin/test/fixtures/routing/fixtures.tsv.

### Phase 3 — Discovery and the assessment gate

13. Write the five-level catalog with verdicts and skip classes in `init/lib/sources.md`; delete @plugins/archcore/skills/init/lib/agent-files.md.
14. Add the assessment gate to `init/SKILL.md`: outputs, tier, coverage set, and the route each tier takes.
15. Apply the coverage set to pool eligibility in `init/lib/seed-detect.md`; label it as an estimate in the preview.

### Phase 4 — Import track

16. Write `skills/_shared/tracks/import.md` with eight gates under @plugin/plugins/archcore/skills/_shared/gate-contract.md.
17. Write `skills/_shared/grounding/convert-routing.md`; delete @plugins/archcore/skills/_shared/grounding/extract-routing.md.
18. Remove the import marks from the seed files: tags, pointer line, filename prefix, `imported/` directory, umbrella document, `has_imports`.
19. Update the file list. Target: @plugin/test/structure/v2-purity.bats.
20. Add structure tests: eight gates present, no import mark in any init asset, plan tag `import-plan` named once.
21. Add `test/behavioral/import-bench.sh` with six fixtures and a Makefile target. Target: @plugin/Makefile.

### Phase 5 — Records and user-facing text

22. Add the update note to `magic-first-day-init.adr`; set the init row and the flag sentence in `command-entry-grammar.adr`.
23. Change the init clauses of `command-surface-v2.spec` to the new form; report the edit to the user for acceptance.
24. Update the init row of `component-registry.doc` and risk row 3 of `delta-routing-compatibility.doc`.
25. Update the init rows. Target: @plugin/README.md.
26. Run `make test` and `make lint`; fix every failure.

## Acceptance Criteria

- `make test` passes with the new and the changed structure tests.
- `wc -l` reports 300 or fewer for `init/SKILL.md`, 200 or fewer for each `init/lib/*.md`, 300 or fewer for `tracks/import.md`.
- `grep -rn "imported-\|source:<slug>\|Imported from" plugins/archcore/skills` returns no line.
- The argument hint is byte-identical in `init/SKILL.md` and `commands/init.md`.
- The import bench runs; its result is recorded, pass or fail, in the closing report.

## Dependencies

- `init-import-mode.adr` stays the recorded decision; a rejection of it stops phases 3–5.
- Phase 1 precedes every other phase, because phases 2–4 edit the split files.
- The import bench needs a model API key at run time; no task depends on its result.
- No CLI change: `remove_document`, `create_document`, and `update_document` exist in the shipped server.

## Declared Delta

- Entry: expert invocation `sdd` — intent, contract per capability, decompose; the decision instrument recorded `init-import-mode.adr` first.
- creates: `init-entry-and-assess`, `authored-source-discovery`, `import-conversion-and-staging`.
- modifies: the init clauses of `command-surface-v2.spec` — verdict spec-wrong: the spec follows the recorded decision.
- retires: agent-file import by link stub and verbatim copy (`agent-files.md`, `extract-routing.md`).
- decision: `init-import-mode.adr`.
- intent_gap: yes — recorded in `init-import-and-entry.prd`.
- M: stone (`magic-first-day-init.adr` and `command-surface-v2.spec` are accepted). R: none.
