---
title: "Init Entry and Assessment Gate — Signature, Run Order, and Division of the Fill"
status: draft
tags:
  - "commands"
  - "onboarding"
  - "plugin"
  - "skills"
---

## Purpose & Scope

This spec governs the entry of `/archcore:init`: the argument form, the order of a plain run, the assessment gate, and how that gate divides the fill between authored sources and code. Dependents: plugin users, the Codex command wrapper (@plugins/archcore/commands/init.md), the hint test (@test/structure/command-grammar.bats), and the import track, which starts from the gate's output. Out of scope: source discovery, conversion and staging (separate specs), and the detection catalogs of the code seed.

## Surface

- Argument hint: `[import|refresh] [path or domain]` in @plugins/archcore/skills/init/SKILL.md.
- Modes: none (plain init), `import`, `refresh`.
- Subject: a repository path for `import`; a repository path or a domain slug for `refresh`.
- Preview toggles: `depth:light|standard|deep`, `scale:small|medium|large`.
- Assessment output: `targets_est`, `tier` (`none`, `S`, `M`, `L`), `coverage_set` (hotspot modules named by authored sources), `levels_found`, and — on a plain init or a refresh — `deeper_present`. The gate is `import.assess` in @plugins/archcore/skills/_shared/tracks/import.md; its measures are in @plugins/archcore/skills/init/lib/sources.md.
- No authored source: the assessment result in which `levels_found` is empty — no file was found, or every found source carries the verdict `skip`.
- Routes: plain init, `refresh`, `import`, import-only, empty, and no-source (`import` with no authored source).
- Flow files: @plugins/archcore/skills/init/lib/host-wiring.md, @plugins/archcore/skills/init/lib/seed-detect.md, @plugins/archcore/skills/init/lib/seed-compose.md.

## Normative Behavior

1. WHEN the first argument word is `import` or `refresh`, the init skill MUST select that mode.
2. WHEN the first word is any other text, the init skill MUST start a plain init.
3. WHEN a plain init starts and host wiring is absent, the init skill MUST plan the host wiring before it runs the assessment gate.
4. The assessment gate MUST measure authored sources from paths, byte sizes, heading counts, and path mentions, without reading a full file body.
5. WHEN the assessment gate sizes `CLAUDE.md`, `AGENTS.md`, or `GEMINI.md`, the gate MUST exclude every archcore managed block.
6. A plain init MUST assess discovery levels L1 and L2 only.
7. WHEN the mode is `import`, the assessment gate MUST assess all five discovery levels.
8. The init skill MUST compose the extractive facts in every route that finds a manifest or source code.
9. WHEN `coverage_set` names a hotspot module, the init skill MUST remove that module from the eligible hotspot pool.
10. The preview MUST label `coverage_set` as an estimate.
11. WHEN `tier` is `S` on a plain init, the init skill MUST list each conversion target in the same preview as the seed.
12. WHEN `tier` is `M` or `L` on a plain init, the confirm MUST create the seed, create the import plan, and run wave 1.
13. WHEN a plain init leaves waves open, the closing message MUST name `/archcore:init import` as the continuation.
14. WHEN the mode is `refresh` and the subject is an existing path, the init skill MUST scope the top-up to that path.
15. WHEN the mode is `refresh` and the subject matches a detected domain slug, the init skill MUST run the single-domain pass for that domain.
16. The init skill MUST accept `depth:` and `scale:` only as preview toggles.
17. WHEN the mode is `import`, the preview MUST NOT offer a `depth:` toggle.
18. WHEN `levels_found` is empty, the assessment gate MUST return `targets_est` 0 and the tier `none`.
19. The init skill MUST print the assessment line in every run, including a run that finds no authored source.
20. A plain init MUST list levels L3 and L4 for presence only, without a size, a heading count, or a verdict.
21. The assessment gate MUST keep `deeper_present` out of `targets_est`, `tier`, and `coverage_set`.
22. WHEN a plain init finds no authored source and `deeper_present` is non-empty, the closing message MUST name `/archcore:init import`.
23. WHEN a plain init finds no file at any level, the closing message MUST name `/archcore:document` as the entry for unwritten conventions.
24. WHEN the user names a skipped path after a no-source report, the init skill MUST enter the import track at `import.triage` with that verdict changed.
25. The assessment gate MUST add one target to `targets_est` for each L4 `reference` site record.
26. WHEN an L4 `reference` site record exists, the assessment gate MUST include L4 in `levels_found`.

## Constraints & Invariants

- Constraint: `init/SKILL.md` MUST NOT exceed 300 lines, and each file under `init/lib/` MUST NOT exceed 200 lines (`skill-file-structure.rule`, items 15 and 16).
- Constraint: the argument hint MUST NOT carry a `--` flag; init has no setting that a user fixes before the preview.
- Invariant: a plain init shows one preview and takes one confirm.
- Invariant: no `create_document`, `add_relation`, or host-wiring write fires before a confirm.
- Invariant: the seed budget formula `max(floor, round(rate × pool_size))` is unchanged; only pool eligibility changes.
- Invariant: an empty assessment result never lowers a skip class and never licenses a body read; zero targets is a valid result.

## Failure Behavior

1. IF the first word is `domain` or a retired flag, THEN the init skill MUST start a plain init.
2. IF a `refresh` subject matches neither a path nor a domain slug, THEN the init skill MUST ask one question that lists the detected domains.
3. IF a plain init or a refresh finds no authored source, THEN the init skill MUST run the code seed with an unchanged hotspot pool.
4. IF no manifest, source code, or authored source exists, THEN the init skill MUST offer host wiring only.
5. IF an open import plan exists when a plain init starts, THEN the init skill MUST report the plan and name `/archcore:init import`.
6. IF authored sources exist without a manifest or source code, THEN the init skill MUST run the import track without the code seed.
7. IF the `import` mode finds no authored source, THEN the init skill MUST print the no-source report.
8. IF the `import` mode finds no authored source, THEN the init skill MUST fire no gated operation.
9. IF the `import` mode finds no authored source, THEN the init skill MUST NOT compose the code seed.
10. IF the no-source report lists a `skip` source, THEN the report MUST name that source's skip class.
11. IF an `import` path subject names no existing path, THEN the init skill MUST report the missing path and stop.
12. WHEN an invocation uses a retired form, the init skill MUST name the current argument form.

## Conformance

An implementation conforms when it satisfies behaviors 1–26, holds the four invariants, stays inside both constraints, and follows the twelve failure rules. Regression coverage: @test/structure/init-skill.bats, @test/structure/command-grammar.bats, and @test/behavioral/import-bench.sh (route, size tier, triage verdict on a live model; fixtures 13–15 cover the no-source result; fixtures 16–18 cover site reference counts).

Given one L4 site with only end-user pages and no other authored source
When the user runs `/archcore:init import`
Then assessment reports one target, tier `S`, and L4 in `levels_found`.
