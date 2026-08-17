---
title: "Plan Discharge Implementation — the closeout.discharge Gate and the Legacy Sweep"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Goal

Give the closeout track gates that dispose of a completed `plan` instead of reporting a candidate it cannot act on, and clear the completed plans parked in `status: rejected` and `status: accepted`. Scope covers the plugin only — the kernel enum stays at three values.

## Tasks

### Phase 1 — Gate machinery

1. Add a `closeout.capture` gate that routes a plan's residue to the instrument owning that type. — @plugins/archcore/skills/_shared/tracks/closeout.md
2. Add a `closeout.discharge` gate that removes the plan and produces no document. — @plugins/archcore/skills/_shared/tracks/closeout.md
3. Write discharge's first blocking precondition: every plan task and acceptance criterion carries a `fulfilled` verdict. — @plugins/archcore/skills/_shared/tracks/closeout.md
4. Write the second precondition: the plan file is absent from the branch boundary's uncommitted block. — @plugins/archcore/skills/_shared/branch-state.md
5. Restrict the decision instrument at capture to its standard cascade, barring a new `spec` or `plan`. — @plugins/archcore/skills/_shared/tracks/closeout.md
6. Replace the Discharge report section's `archived` paragraph with the plan-only deletion rule. — @plugins/archcore/skills/_shared/tracks/closeout.md
7. Update the track-notes gate order and `closeout.accept`'s `Next:` field. — @plugins/archcore/skills/_shared/tracks/closeout.md

### Phase 2 — Contract alignment

8. Rewrite behaviors 19 through 26 of the delta-routing instruments spec; retire the `archived` precondition for `plan`.
9. Add both invariants to that spec: `plan` is the only removed type; capture creates no type of its own.
10. Update the review skill's write affinity and Result sections for the two new gates. — @plugins/archcore/skills/review/SKILL.md
11. Regenerate the closeout golden fixture and confirm the diff carries only the intended gate changes. — @test/fixtures/goldens/closeout.golden

### Phase 3 — Legacy sweep

12. Discharge the plans holding `status: rejected` that clear both preconditions, confirming each removal.
13. Discharge the plans holding `status: accepted`, confirming each removal.
14. Verify no relation names a deleted plan as source or target. — @.archcore/.sync-state.json

### Phase 4 — Release

15. Bump the plugin version in the four host manifests. — @plugins/archcore/.claude-plugin/plugin.json

## Acceptance Criteria

1. The closeout track file carries `closeout.capture` and `closeout.discharge`, each declaring `skip_when` and tagging every exit check `blocking` or `advisory`.
2. `closeout.discharge` declares `Produces: none` — disposal creates nothing.
3. A run over a plan holding one unfulfilled task stops before any `remove_document` call and names the failed precondition.
4. A run over an uncommitted plan stops before any `remove_document` call and names the failed precondition.
5. A declined capture still leaves a fulfilled plan removable.
6. `list_documents(types=["plan"])` returns only plans whose work is unfinished.
7. No string `archived` remains in the closeout track file outside the kernel-absence statement.
8. `list_relations` returns no edge whose source or target is a removed plan.
9. `make verify` passes.

## Dependencies

- `remove_document` in the shipped CLI — already present and already clearing both relation directions, so no version gate applies (`@internal/mcp/tools/remove_document.go` in the CLI repository).
- The branch boundary resolution used by the committed-file precondition — @plugins/archcore/skills/_shared/branch-state.md.
- The decision instrument, callable from `review` per the track-layer spec, supplies every capture type beyond `task-type` and `guide`.
- Cross-repo follow-up, outside this plan's delivery scope: the CLI's `remove_document` description still instructs "A plan is abandoned → change status to rejected", so an agent acting outside these gates keeps receiving the superseded guidance.

## Declared Delta

- `creates`: none — the discharge behavior falls under specs that already exist.
- `modifies`: closeout discharge, covered by the delta-routing instruments spec.
- `retires`: report-only discharge blocked on the absent `archived` value.
- `decision`: plan discharge by deletion, recorded as an ADR in this route.
- `intent_gap`: no — the lifecycle intent already sits in accepted canon.
- Route rationale: `amendment` at size M. Base label S from the `modifies` and `retires` rows; raised one step because the zone is `stone` and R carries `irreversibility`. The `actualize` track needs no edit — its temporal rule is correct as written, and its findings clear with the sweep rather than with a rule change.
- Design revision during implementation: the single capture-and-dispose gate split into `closeout.capture` → `closeout.discharge`, and capture became a routing step rather than a type menu, so `rule`, `cpat`, and `adr` residue reaches the decision instrument that already owns those types.
