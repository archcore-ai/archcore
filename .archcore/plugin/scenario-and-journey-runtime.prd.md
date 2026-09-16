---
title: "Stored Scenarios and Journeys: Plugin Runtime Outcomes"
status: draft
tags:
  - "document-types"
  - "plugin"
  - "skills"
  - "vision"
---

## Vision

The plugin runtime produces, reads, and verifies the two actor-subject document types the engine ships in CLI v0.8.4 — `scenario` in knowledge and `journey` in vision — so a project keeps how a user moves through the system beside the `spec` that governs it, through the four commands it already has.

## Problem Statement

The engine registers 23 types since CLI v0.8.4 (commit `2a8f6e4` in the `cli` repository), but the plugin runtime knows 21. No content contract under `skills/_shared/` tells a composing skill what a `scenario` or a `journey` holds; no instrument on `/archcore:plan` produces either; `describe.read` ignores `features/*.feature`; `closeout.verify` cannot say which `spec` clauses lack an example; the `/archcore:document` argument hint omits both names; and no compatibility probe gates the two names on the engine release. A user on the current plugin gets a raw engine template with no routing, no relations, and no checks. The vocabulary, its boundaries, and the runtime obligations are recorded in `concepts/scenario-and-journey-types` and `concepts/bdd-document-type-evaluation` [global · archcore · read-only], both still `draft`.

## Goals and Success Metrics

- Content contracts under `skills/_shared/`: 2 new files (`scenario-contract.md`, `journey-contract.md`) at section parity with the CLI templates in @../cli/templates/templates.go; today 0.
- A `scenario` composed at `sdd.illustrate` from the contract reports 0 missing-section findings and 0 modal-in-step findings in the CLI post-write hook.
- Routing bench: 2 new traces (a user-facing capability, a repository with `features/*.feature`) resolve to a package that carries the illustrate instrument; the 40 existing traces keep their route. [assumption] Measured with `test/behavioral/route-bench.sh`.
- Compatibility: on CLI 0.8.3 the plugin writes 0 documents of either type and reports the required version once per invocation; on 0.8.4 the probe returns `yes`.
- Structure suite, unit suite, and the integration suite pinned to CLI 0.8.4 pass on the release commit.

## Requirements

1. An author obtains a `scenario` or a `journey` through a plugin track with the actor as the subject of every step and no modal in any step.
2. A capability planned through `/archcore:plan` that names a user-facing surface leaves the package with one `scenario` linked to its `spec`.
3. A team that keeps feature files sees them read as evidence when it documents existing behavior.
4. A closeout report tells the team which `spec` clauses on the branch have no example and which scenarios were confirmed.
5. On an engine below 0.8.4 the plugin writes neither type and names the required version.
6. The `/archcore:document` argument hint shows both names where a user can type them.
7. A `scenario` over its body cap is split by actor, never truncated.

## Out of Scope

The docs site, the "Archcore + Cucumber" integration recipe, the global shared-context updates on RFC acceptance, and the per-type body caps for the other 21 types (`concepts/size-caps-measured-on-a-real-corpus` [global · archcore · read-only]). Execution of any scenario: the runtime executes nothing.

## Dependencies

CLI v0.8.4 (tag on commit `2a8f6e4`) ships the engine half; the two CLI-side contracts are `scenario-and-journey-types.spec` and `scenario-and-journey-advisory-canon.spec` under `document-types/` in the `cli` repository. The tag is published as [CLI v0.8.4](https://github.com/archcore-ai/cli/releases/tag/v0.8.4), 2026-09-16, with checksums for six platform assets. Release of the plugin half waits for the global RFC to reach `accepted` (its adoption step 3); implementation proceeds before that on the user's instruction of 2026-09-16, as the engine half did on 2026-09-15.
