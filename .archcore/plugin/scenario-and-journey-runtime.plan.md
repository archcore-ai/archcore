---
title: "Scenario and Journey Runtime — Contracts, Illustrate Instrument, Evidence Checks, and the 0.8.4 Gate"
status: draft
tags:
  - "document-types"
  - "plugin"
  - "skills"
---

## Goal

Deliver the plugin portion of `concepts/scenario-and-journey-types` [global · archcore · read-only] in one release: two content contracts, the illustrate instrument with journey production, scenario evidence on the describe and closeout tracks, and the compatibility gate with the two `/archcore:document` entries. The linked `prd` owns outcomes; four linked specs own behavior.

Grounding used branch `dev`, HEAD `7b732c1`, clean working tree, on 2026-09-16. The engine half is CLI v0.8.4 (commit `2a8f6e4` in `../cli`); the global RFC and both investigations stay `draft` and are not imported into this repository. Implementation runs on branch `feat/scenario-journey-runtime`; the task-level plan is @docs/superpowers/plans/2026-09-16-scenario-journey-plugin-runtime.md.

## Declared Delta

| Field | Value |
|---|---|
| `creates` | `actor-subject-content-contracts`; `illustrate-instrument`; `scenario-evidence-in-describe-and-closeout`; `actor-subject-compatibility` |
| `modifies` | track catalog — `track-layer.spec`, verdict `spec-wrong` (the global RFC and CLI 0.8.4 are the newer upstream); instrument registry and the 21-type invariant — `delta-routing-instruments.spec`, verdict `spec-wrong`; `document` expert surface — `command-surface-v2.spec`, verdict `spec-wrong`; agent domain knowledge — `agent-system.spec`, verdict `spec-wrong`; type-count invariant — `delta-routing-conductor.spec`, verdict `spec-wrong`; reachability metric — `claude-plugin.prd`, verdict `spec-wrong`. All six edits were confirmed by the user on 2026-09-16 and applied. |
| `retires` | None |
| `decision` | None at routing; two choices settled in brainstorming on 2026-09-16 are recorded as `document-journey-callable-intent-entry.adr` and `actor-subject-compatibility-separate-file.adr` |
| `intent_gap` | Yes; no local document recorded the runtime intent before the linked `prd` |
| Π | `machine`: the global RFC, `concepts/bdd-document-type-evaluation`, `concepts/bdd-and-adjacent-practices`, the CLI contracts under `document-types/` in the `cli` repository, the research-vocabulary precedent in this repository; `user`: the two settled choices above |
| M | `stone`: accepted `track-layer.spec`, `delta-routing-instruments.spec`, `command-surface-v2.spec`, and `agent-system.spec` cover the zone |
| R | `external-contract`: the CLI version gate and the MCP type enum |
| Route | `umbrella`, base L, raised once to XL by `stone`; `external-contract` adds no further step at the cap |
| Instruments | intent, contract ×4, decompose; verdicts on the six `modifies` entries recorded above |

Four capabilities with four consumer sets justify the umbrella: composing skills rely on the contracts; the conductor relies on the instrument; the document and review skills rely on the evidence checks; every skill, agent, and host relies on the gate. The precedent in this repository for a vocabulary release edited the covering specs in place; this delta keeps those edits as confirmed tasks and states the new behavior in the four created specs.

## Tasks

### Phase 1 — content contracts and canon hooks

1. [x] Write `@plugins/archcore/skills/_shared/scenario-contract.md` per the content-contracts spec.
2. [x] Write `@plugins/archcore/skills/_shared/journey-contract.md` per the content-contracts spec.
3. [x] Edit `@plugins/archcore/skills/_shared/precision-rules.md`: add both types to the claim-recording list with the F6 profile.
4. [x] Edit `@plugins/archcore/skills/_shared/prd-contract.md`: add the three actor-subject rows.
5. [x] Extend the Conformance section in `@plugins/archcore/skills/_shared/spec-contract.md`.
6. [x] Draft one document of each type in a scratch project and record the hook findings here.

### Phase 2 — illustrate instrument on the sdd track

7. [x] Add the registry row, the package contribution, and the illustrate condition to `@plugins/archcore/skills/_shared/delta-routing.md`.
8. [x] Add the gate `sdd.illustrate` to `@plugins/archcore/skills/_shared/tracks/sdd.md`.
9. [x] Extend `sdd.require` with the journey production in `@plugins/archcore/skills/_shared/tracks/sdd.md`.
10. [x] Extend the `sdd.design` exit checks in `@plugins/archcore/skills/_shared/tracks/sdd.md`.
11. [x] Record the journey single-type exception in the Track notes of `@plugins/archcore/skills/_shared/tracks/sdd.md`.
12. [x] Extend the Ground step of `@plugins/archcore/skills/plan/SKILL.md` with the probe trigger.
13. [x] Add two routing traces under `@test/behavioral/fixtures/`.

### Phase 3 — scenario evidence on the describe and closeout tracks

14. [x] Extend `describe.read` and the type heuristics in `@plugins/archcore/skills/_shared/tracks/describe.md`.
15. [x] Extend `describe.draft` Produces in `@plugins/archcore/skills/_shared/tracks/describe.md`.
16. [x] Extend the `closeout.verify` exit checks in `@plugins/archcore/skills/_shared/tracks/closeout.md`.
17. [x] Extend the `closeout.accept` offer text in `@plugins/archcore/skills/_shared/tracks/closeout.md`.
18. [x] Extend the closeout scope filter in `@plugins/archcore/skills/review/SKILL.md`.

### Phase 4 — compatibility gate, command entries, agents

19. [x] Write `@plugins/archcore/skills/_shared/actor-subject-compatibility.md` per the compatibility spec.
20. [x] Extend the argument hint and the expert form in `@plugins/archcore/skills/document/SKILL.md`.
21. [x] Extend the Ground step of `@plugins/archcore/skills/document/SKILL.md` with the probe trigger.
22. [x] Edit `@plugins/archcore/agents/archcore-assistant.md` and its Codex and Copilot siblings to name both types and the probe.
23. [x] Raise the pinned integration CLI to 0.8.4 in `@Makefile` and under `@.github/workflows/`.
24. [x] Write `@test/integration/actor-subject-vocabulary.bats` on the pattern of `@test/integration/research-vocabulary.bats`.
25. [x] Write `@test/structure/actor-subject-contracts.bats` and `@test/structure/actor-subject-compat.bats`.

### Phase 5 — canon updates under confirmation

26. [x] After confirmation, extend the catalog line of `track-layer.spec` with the illustrate gate and the two consuming surfaces.
27. [x] After confirmation, add the registry entry and the 23-type invariant to `delta-routing-instruments.spec`.
28. [x] After confirmation, add both names to the `document` expert names in `command-surface-v2.spec`.
29. [x] After confirmation, update the type lists and the probe duty in `agent-system.spec`.
30. [x] Add the two type rows and the 23-type count to `delta-routing-type-engagement.doc`.
31. [x] Add the 0.8.4 containment row to `delta-routing-compatibility.doc`.
32. [x] Search both trees for the old type count and fix every hit that describes the current registry.

### Phase 6 — verification and release handoff

33. [x] Run `make all` and `make test-integration` with CLI 0.8.4 on PATH; record the result here.
34. [x] Run `make test-routing-bench` on the two new traces and the 40 existing ones; record the result.
35. [x] Remove the `illustrate` registry row, confirm the structure suite fails, and restore it by editing.
36. [x] Bump the plugin version across the host manifests per the recorded pattern.
37. [ ] Confirm the global RFC reached `accepted` before tagging the plugin release.

## Acceptance Criteria

1. The four linked specs' conformance sections pass against the final implementation commit.
2. A scenario and a journey composed from the contracts report only the placeholder-body finding in the CLI hook.
3. The two new routing traces resolve to a package that carries the illustrate instrument; the 40 existing traces keep their route.
4. On CLI 0.8.3 the plugin writes neither type and reports the required version once; on 0.8.4 the probe returns `yes`.
5. The structure, unit, and integration suites pass with the pinned CLI at 0.8.4.
6. Implementation review reconciles all four `creates` entries and all six `modifies` verdicts through `/archcore:review`.

## Dependencies

The engine half is complete in CLI v0.8.4 (`2a8f6e4`); the CLI contracts `scenario-and-journey-types.spec` and `scenario-and-journey-advisory-canon.spec` under `document-types/` in the `cli` repository are the interface this plan implements against. Phase 1 precedes phases 2 and 3; phase 4 can run beside them; phase 5 waits for the user's per-document confirmation; phase 6 verifies all five. Release waits for the global RFC status (adoption step 3) and for the user's decision on whether the plugin ships the vocabulary before the docs site and the Cucumber recipe. The `v0.8.4` tag is published as [CLI v0.8.4](https://github.com/archcore-ai/cli/releases/tag/v0.8.4), 2026-09-16.

Verification record, 2026-09-16, on branch `feat/scenario-journey-runtime`: `make all` green — lint clean, 593 structure and unit tests; `make test-integration` against a CLI 0.8.4 built from commit `2a8f6e4` — 16 tests green, 4 of them new. Scratch drafts (task 6): a scenario written from the contract's example reported 0 findings; a journey first reported "step opens with no actor" because the CLI matches the literal actor name from the Actors table, so rule 1 of both contracts and their examples now require the literal name with no leading article; after that fix the journey reported 0 findings. Routing bench on the two new traces only (`ROUTE_BENCH_FIXTURES` with rows 43 and 44): 2 pass, 0 fail; both announcements name `illustrate (sdd.illustrate)`; the 40 existing traces were not re-run in this session. Negative check: deleting the `illustrate` registry row failed `test/structure/delta-routing.bats`; restoring it by editing passed 28 of 28. Line counts after the change: `sdd.md` 277, `closeout.md` 235 — both over the 200-line track-file constraint of `plugin-architecture.spec`, which they exceeded before this work (214 and 226). Deviations from the task text: rule 7 of the precision rules lists the new types after `cpat` because `test/structure/research-track.bats` pins the phrase `research`, `evidence`, `cpat`; `plugins/archcore/commands/document.md` also carries the argument hint and was updated for hint parity; two more accepted documents needed the type-count edit (`delta-routing-conductor.spec`, `claude-plugin.prd`), confirmed with the other four.
