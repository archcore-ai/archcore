---
title: "Bootstrap (Now Init) Scale Modes — Small / Medium / Large with Tracked-Context Targets"
status: rejected
tags:
  - "onboarding"
  - "plugin"
  - "roadmap"
  - "skills"
---

**Outcome (2026-05-15).** The plan was executed. The skill shipped as `skills/init/`, renamed from `skills/bootstrap/` per `skill-surface-collapse.adr`, with its lib files at `skills/_shared/grounding/detect-*.md` and the command at `/archcore:init`. Read every reference to the old names below as the new ones. The three-mode detection logic is preserved as designed.

**Update (2026-07-01).** The premise below that large mode cannot meaningfully seed per-domain artifacts in one pass is reversed. Real-world run evidence — a 773-module, 24-domain repository seeded only 3 specs and 4 domain data-models at day one — showed the opposite failure: seeding too little per domain rather than too much. Large mode now scales its hotspot-spec budget with the domain selection, applying a per-selected-domain floor of at least 1 spec filled to a depth-scaled cap by repo-wide rank — 2 per domain with a minimum of 6 and a cap of 12 at `light`, 3 per domain with a minimum of 10 and a cap of 24 at `standard`, and 4 per domain with a minimum of 14 and a cap of 40 at `deep` — and seeds a data-model doc for **every** schema-bearing domain regardless of selection. `SKILL.md` and `skills/_shared/grounding/detect-hotspots.md` hold the current numbers. The per-domain re-run described below still exists as a narrower top-up mechanism, now on top of a substantive day-one seed rather than instead of one. The target-context and steady-state lists from the M2 tasks onward predate the Tier-2 confirmed-synthesis rewrite in `magic-first-day-init.adr` and are historical context rather than the current contract.

## Goal

`/archcore:init` produced two artifacts derived from manifests alone. On anything beyond a narrow single-domain library that is insufficient: the output says nothing an agent could not read from the manifest in 30 seconds. On a multi-domain application it collapses into noise, because no single init output usefully covers a 50-module monolith.

The fix is to **branch init by repository scale** — small, medium, large — seeding the artifacts each scale calls for, and to document per-scale **context targets** so that one-time seeding is understood as the first step of a longer capture trajectory rather than the endpoint.

In scope: language-agnostic core signals, with no per-language parser and the per-language data held in tables under the skill's lib directory; extending the parent onboarding plan rather than replacing it; and defining scale detection, the per-scale flow, and the per-scale target list.

Out of scope: automatic capture of everything in the target lists, since init seeds only its designated artifacts and the rest are tracked as work for `capture`, `decide`, or `plan`; cross-project syncing of target lists; and import-graph or call-graph analysis, which is deferred because it needs per-language parsers.

## Scale detection

Three cheap, language-agnostic signals are computed in one pass. **`domain_count`** counts top-level subdirectories under the conventional roots, excluding the conventional utility names, where a subdirectory counts only if it holds at least 2 source files above 50 lines. **`module_count`** counts source files above 100 lines, excluding tests, vendored dependencies, generated code, and lockfiles. **`entry_point_count`** counts files matching the entry-point patterns, including language main files, command directories, `bin/`, and files carrying HTTP-route decorators.

| Mode | Rule |
|---|---|
| **Small** — a library or single-domain repository | `domain_count ≤ 1` and `module_count ≤ 15` |
| **Medium** — a focused service, framework kit, or mid-sized SDK | `domain_count ≤ 2` and `15 < module_count ≤ 40` |
| **Large** — a multi-domain application, monorepo, or modular monolith | `domain_count ≥ 3` or `module_count > 40` |

The thresholds live in `skills/_shared/grounding/detect-scale.md` as editable data. Init prints the detected mode in its opening line together with the override hint.

## Target context per mode

Each item below is marked **[seed]** when init writes it directly, **[propose]** when init detects candidates and surfaces a to-capture list the user triggers per candidate, or **[organic]** when init is uninvolved and the document is added as the work touches that area.

**Small mode — a library or single-domain repository**, typically one cohesive public surface. The stack rule and the run guide are [seed]. Three hotspot capture candidates, the top modules by combined source and test lines, are [propose]. An ADR per deliberate non-trivial dependency, typically 2 to 4, is [organic], though init can surface the question in its propose list. A hotspot spec per critical module, typically 1 to 3, and a rule per cross-cutting convention, typically 1 to 2, are [organic]. A task-type per repeating extension pattern, typically 1 to 2, is [propose]. Expected steady state: 5 to 10 documents.

**Medium mode — a focused service, framework kit, or mid-sized SDK**, typically one domain with a non-trivial internal surface and few entry points. The stack rule, the run guide, and an entry-point inventory doc are [seed]. Three to five hotspot capture candidates and one cross-cutting rule candidate are [propose]. An ADR per architectural decision, typically 3 to 6; a spec per hotspot module, typically 3 to 6; a rule per cross-cutting concern, typically 2 to 4; and a task-type for the top 2 to 3 change patterns are [organic]. Expected steady state: 12 to 20 documents.

**Large mode — a multi-domain application, monorepo, or modular monolith**, typically three or more cohesive domains with many entry points. The strategy recorded here was to seed a map of the shape and then run narrow per-domain passes; the 2026-07-01 update above scales the day-one seed instead. The workspace stack rule, the monorepo-aware run guide, a top-level map doc, and an entry-point inventory doc are [seed]. A domain selection dialog runs interactively during init, asking which of the detected domains the user is working on. The per-selected-domain hotspot proposal is [propose]. An ADR per domain-level decision, typically 2 to 5 per domain; a spec per hotspot module, typically 2 to 5 per selected domain; a spec per cohesive domain boundary, typically one per neighboring pair; the repo-wide cross-cutting rules, typically 4 to 8, whose candidates init flags; and a task-type for the top 3 to 5 change patterns are all [organic]. Expected steady state: 20 to 40 or more documents, which is acceptable because the map plus the per-domain tag lets session hooks and command grounding scope their queries (formerly `/archcore:context`, removed under v2).

## Per-mode flow

| Step | Small | Medium | Large |
|---|---|---|---|
| 1. Detect scale | ✓ | ✓ | ✓ |
| 2. Announce mode + override hint | ✓ | ✓ | ✓ |
| 3. Stack rule (seed) | ✓ | ✓ | ✓ (workspace) |
| 4. Run guide (seed) | ✓ | ✓ | ✓ (monorepo) |
| 5. Top-level map (seed) | — | — | ✓ |
| 6. Entry-point inventory (seed) | — | ✓ | ✓ |
| 7. Hotspot capture candidates (propose) | ✓ (3) | ✓ (3–5) | per selected domain (3–5) |
| 8. Cross-cutting rule candidate (propose) | — | ✓ (1) | — (deferred to the per-domain pass) |
| 9. Domain selection dialog | — | — | ✓ |
| 10. Opt-in agent-file import | ✓ | ✓ | ✓ |
| 11. Closing message with tracked-context todos | ✓ | ✓ | ✓ |

The closing message in every mode lists the over-time targets not yet created, so the user knows what the tracked context aims for and can drive captures at their own pace.

## Common infrastructure

Six language-agnostic signal modules live under `skills/_shared/grounding/`: `detect-scale.md` holding the thresholds, the override semantics, and the utility-directory exclusions; `detect-domains.md` holding the conventional roots, the exclusion list, and the cohesion rule; `detect-modules.md` holding the per-language source-extension allowlist plus the test-file and generated-code patterns to exclude; `detect-entry-points.md` holding the language-independent patterns and the per-language additions; `detect-hotspots.md` holding the ranking formula and the optional git-activity weighting; and `detect-cross-cutting.md` holding the repeated-pattern heuristics.

## MVP slice status (2026-04-24)

The first implementation pass delivered the lib files and a rewritten `SKILL.md` with full mode branching. It covered M1 through M6 plus part of M8, writing the six detector data files with language coverage for TypeScript and JavaScript, Python, and Go, plus stub entries for Rust, Java, Kotlin, Ruby, PHP, C#, Swift, Scala, and Elixir. Validation was manual across three real repositories. The specification updates of M8 were deferred until the behavior stabilized.

## Tasks

**M1 — mode detection, blocking.**
- [x] Implement the three signal counters.
- [x] Implement threshold classification and override-flag parsing.
- [x] Print the detected mode and the override hint in the opening line.
- [ ] Add the unit fixtures for a small library, a medium single-app service, and a large monorepo. Deferred to M7.

**M2 — medium-mode additions.**
- [x] Generate the entry-point inventory document.
- [x] Detect the cross-cutting rule candidate.

**M3 — large-mode additions.**
- [x] Generate the top-level map document.
- [x] Generate the monorepo-aware entry-point inventory.
- [x] Add the domain selection dialog.
- [x] Add the per-domain hotspot candidate pass.

**M4 — the shared hotspot proposal.**
- [x] Add the ranking module.
- [x] Add the presentation step.
- [x] Keep `/archcore:capture` un-invoked automatically (now `/archcore:document`; capture absorbed by document under v2).

**M5 — the signal libraries.**
- [x] Write the six detector files.

**M6 — the closing message.**
- [x] List the organic target items not yet seeded, per mode.
- [x] Add per-mode templates with specific calls to action.

**M7 — fixtures and tests. Deferred.**
- [ ] Add the small, medium, and large fixtures.
- [ ] Add the small Python and small Go fixtures.
- [ ] Add the mode assertions in `test/structure/init-modes.bats`.

**M8 — documentation. Partial.**
- [x] Update `SKILL.md` with the mode-branch flow.
- [ ] Cross-reference this plan from the parent onboarding plan. Deferred.
- [ ] Add a scale-modes subsection to the init entry in `commands-system.spec`. Deferred.

## Acceptance Criteria

1. On the small fixture, init produces the stack rule, the run guide, and a three-candidate capture list.
2. On the medium fixture, init produces the stack rule, the run guide, the entry-point inventory, a three-to-five-candidate capture list, and one cross-cutting rule candidate.
3. On the large fixture, init produces the workspace stack rule, the monorepo run guide, the top-level map, and the entry-point inventory, then asks for domain selection, then produces the per-domain capture candidates.
4. Mode detection classifies every fixture correctly, and the override flag forces the chosen mode.
5. The closing message in each mode cites the target-context outlook.
6. Every seeded artifact is idempotent on re-run.
7. Language coverage is validated on the TypeScript, Python, and Go small fixtures.
8. Neither the SessionStart nudge nor the agent-file import regresses.

## Dependencies

- The parent plan `zero-content-onboarding-implementation.plan`, which this assumes has already implemented the stack rule and the run guide.
- The source idea `zero-content-onboarding.idea`.
- No CLI release dependency, since everything is reachable through the existing MCP surface.
- The existing `document` and `plan` commands (capture and decide absorbed by document under v2), which the propose lists route to.
- Optionally, the `git` CLI for hotspot weighting.

## Risks

- **Mode misclassification**, because the thresholds are heuristic. Mitigated by always printing the detected mode and the override flag.
- **Domain detection noise.** Mitigated by the exclusion list in the domain detector.
- **Per-language bias in the entry-point patterns.** Mitigated by the data-driven lookup table.
- **Domain-dialog fatigue in large mode.** Mitigated by capping the presented domains at 5, ranked by activity and size.
- **Hand-off cost for the capture candidates.** Mitigated by presenting them as a to-do list in the closing message rather than auto-invoking.
- **Staleness of the target-context lists.** Mitigated by treating this plan as living.
- **Over-seeding.** Mitigated by threshold tuning, the override flag, and the idempotent skip.
