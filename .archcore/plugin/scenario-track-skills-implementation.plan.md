---
title: "Scenario Track Skills Implementation Plan"
status: accepted
tags:
  - "plugin"
  - "skills"
---

**Outcome (2026-05-15).** The plan was executed, shipping three track skills, and was then superseded by `skill-surface-collapse.adr`. All six track skills — the original product, sources, and ISO tracks plus the three added here — were removed, and their flow content moved into `skills/plan/references/{product,sources,iso,feature}-flow.md`, with the continuation logic for the architecture and standard chains moving into `skills/decide/references/continuations.md`. The flows are reachable through `/archcore:plan --<flow>` or natural language; the standalone track commands are gone.

## Goal

Implement three scenario-based track skills — `architecture-track`, `standard-track`, and `feature-track` — following the established track pattern, and register them in the skills specification.

## Tasks

**Phase 1 — architecture-track, running adr → spec → plan.**

- [x] Create `skills/architecture-track/SKILL.md`.
- [x] Implement the flow adr → spec → plan.
- [x] Add the relations: spec `implements` adr, and plan `implements` spec.
- [x] Ask per step: for the adr, what decision was made and what alternatives were considered; for the spec, what the contract surface and constraints are; for the plan, what the implementation phases and dependencies are.

**Phase 2 — standard-track, running adr → rule → guide.**

- [x] Create `skills/standard-track/SKILL.md`.
- [x] Implement the flow adr → rule → guide.
- [x] Add the relations: rule `implements` adr, and guide `related` rule.
- [x] Ask per step: for the adr, what decision was made and why this approach; for the rule, what the mandatory behaviors are and how to enforce them; for the guide, what steps a developer follows and what the common pitfalls are.

**Phase 3 — feature-track, running prd → spec → plan → task-type.**

- [x] Create `skills/feature-track/SKILL.md`.
- [x] Implement the flow prd → spec → plan → task-type.
- [x] Add the relations: spec `implements` prd, plan `implements` spec, and task-type `related` plan.
- [x] Ask per step: for the prd, what problem this solves and what the success metrics are; for the spec, what the technical contract and API surface are; for the plan, what the phases and blockers are; for the task-type, what the recurring pattern and its key steps are.

**Phase 4 — specification update and validation.**

- [x] Add the three new tracks to the track table in the skills specification.
- [x] Verify each track end to end by invoking it with a topic.
- [x] Confirm that no type-level guidance is duplicated, because a track defines the flow only.

**Phase 5 — migration, after the surface collapse.**

- [x] Delete all six track skill directories.
- [x] Move the flow content into the four `skills/plan/references/*-flow.md` files.
- [x] Move the ADR-driven continuation content — the standard and architecture flows linking an ADR to a spec, rule, or guide — into `skills/decide/references/continuations.md`.
- [x] Delete the corresponding command wrappers.
- [x] Update the skills specification to describe the 7-skill surface.

## Acceptance Criteria

The original criteria, all met at plan completion before the collapse:

- Three `SKILL.md` files exist at `skills/{architecture,standard,feature}-track/SKILL.md`.
- Each follows the step structure of the existing tracks.
- Each creates documents exclusively through `mcp__archcore__create_document`.
- Each adds relations through `mcp__archcore__add_relation` between the documents it creates.
- Each checks for an existing document to determine scope.
- The track table in the skills specification holds 6 entries.

The current state after the collapse:

- No track `SKILL.md` file exists.
- The flow content lives across four files under `skills/plan/references/` plus the continuations reference under `skills/decide/references/`.
- The skills specification documents the 7-skill surface and no track tier.

## Dependencies

- The existing product, sources, and ISO track skills served as the structural reference, and were also retired in the collapse.
- The referenced document-type skills were retired by `remove-document-type-skills.adr`, and their elicitation now lives inline inside `capture`, `decide`, and `plan`.
- `skills-system.spec` is the current authoritative specification.
- `skill-surface-collapse.adr` is the decision that retired the track tier.
