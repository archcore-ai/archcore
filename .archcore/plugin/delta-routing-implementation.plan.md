---
title: "Delta-Routing Implementation — Conductor as Primary Routing for /archcore:plan"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Goal

Make delta routing (ΔΠMR) the primary routing logic of `/archcore:plan`, per the conductor and instrument-layer specs, then align the user-facing documentation and stand up the behavioral routing harness.

## Declared Delta

- creates: [conductor contract asset, capability-granularity contract, verdict contract, runbook gate, spike gate, guide content contract, behavioral routing harness]
- modifies: [plan-skill routing, sdd gate sequencing, requirements-cascade engagement, describe and actualize entries, closeout verify, experience offer, gate-contract state block, README routing narrative]
- retires: [the fixed routing table inside the plan skill]
- decision: the ΔΠMR model acceptance this cycle
- intent_gap: no — the specs record the intent
- Route rationale: umbrella, size XL — a multi-capability change in a stone zone (the accepted track-layer documents cover the touched zone).

## Tasks

Phase 1 — conductor core (completed 2026-08-15):

1. Author the conductor contract @plugins/archcore/skills/_shared/delta-routing.md
2. Author @plugins/archcore/skills/_shared/capability-granularity.md
3. Rewrite the Route and Execute steps in @plugins/archcore/skills/plan/SKILL.md
4. Extend the state block in @plugins/archcore/skills/_shared/gate-contract.md with route and delta fields

Phase 2 — instruments (completed 2026-08-15):

5. Rework @plugins/archcore/skills/_shared/tracks/sdd.md — conductor-owned sequencing, per-capability contract gate, new sdd.runbook gate
6. Add the spike gate to @plugins/archcore/skills/_shared/tracks/research.md
7. Add callable mode to @plugins/archcore/skills/_shared/tracks/describe.md
8. Reframe @plugins/archcore/skills/_shared/tracks/requirements-cascade.md as the acquisition instrument and iso links
9. Extract @plugins/archcore/skills/_shared/verdict-contract.md from @plugins/archcore/skills/_shared/tracks/actualize.md and reference it back
10. Add Δ reconciliation and the discharge report to @plugins/archcore/skills/_shared/tracks/closeout.md
11. Add the human-actor guide option to @plugins/archcore/skills/_shared/tracks/experience.md
12. Align @plugins/archcore/skills/document/SKILL.md and @plugins/archcore/skills/review/SKILL.md with the command-tense split

Phase 3 — tests (completed 2026-08-15; make test 520/520 after the verification round added 19 pin tests):

13. Regenerate track goldens via @test/helpers/extract-gates.sh
14. Update the state-block pin and add conductor structure tests under @test/structure/
15. Run make test to green

Phase 4 — documentation and the guide contract (completed 2026-08-15; suite at 521/521):

16. Rework the "Inside the commands" section of @README.md into conductor vocabulary: computed route, null route, instruments.
17. Update @README.md lines 9 and 21 — gated tracks as default become computed routes over instruments.
18. Regenerate @3-commands.png as conductor → instruments flow (pairs with the README diagram issue #18).
19. Author @plugins/archcore/skills/_shared/guide-contract.md — reader-and-task line, actor-explicit steps, `task-type` boundary.
20. Reference guide-contract.md from every guide-producing exit check.

Task 18 landed as an SVG source rendered through headless Chrome at 2x (3604×4068), replacing the stale "default → sdd" chip with the conductor row. Task 20 landed in four producers — @plugins/archcore/skills/_shared/tracks/sdd.md, @plugins/archcore/skills/_shared/tracks/describe.md, @plugins/archcore/skills/_shared/tracks/experience.md, and @plugins/archcore/skills/_shared/tracks/decision.md (the standard cascade, a fourth producer found during wiring) — pinned by a structure test.

Phase 5 — behavioral routing harness (completed 2026-08-15):

21. Convert the 40 bench traces into harness fixtures: task text in, expected route, package, label out.
22. Build the LLM-in-the-loop runner that scores route announcements against the fixtures
23. Wire the harness as a non-CI make target; record first-run results back into the bench document

The fixtures landed as 42 rows in @test/behavioral/fixtures/routing-bench.tsv; the runner is @test/behavioral/route-bench.sh behind `make test-routing-bench`. First run: 42 of 42 route names match; three announcements carried field-level inconsistencies — recorded in the bench document as the runner's next scoring dimension.

Phase 6 — deferred decision (recorded 2026-08-15):

24. Record the phase-2 decision on Π replacing budget-from-vagueness via the decision instrument, before touching @plugins/archcore/skills/_shared/elicitation-contract.md.

The proposal is recorded as the open rfc "Π Profile as the Elicitation Driver"; the elicitation contract stays unchanged until the rfc resolves.

## Acceptance Criteria

- `make test` passes (521 structural tests as of phase 4).
- Each behavior of the conductor and instrument-layer specs maps to one carrying instruction in a runtime asset (verified 2026-08-15, second-round conformance sweep).
- The pinned rows of @test/fixtures/routing/fixtures.tsv hold unchanged — gate names and entry budgets.
- No forbidden-lexicon term enters a changed file.
- @README.md carries no claim that gated tracks are the default routing (met 2026-08-15, text and diagram).
- guide-contract.md is referenced by its guide-producing consumers (met 2026-08-15 — four producers, pinned by a structure test).
- The behavioral harness reproduces at least the 7 verified traces of the 12-trace dry-run (met 2026-08-15 — first run reproduced 42 of 42 route names).

## Dependencies

- The conductor and instrument-layer specs (draft, this cycle).
- bats-core for the structural suite.
- An LLM runner (host session or API) for phase 5 — outside bats.
- The `archived` status rfc blocks only discharge transitions; no phase above waited on the CLI.