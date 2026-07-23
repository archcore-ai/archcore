---
title: "Scenario Track Skills Implementation Plan"
status: accepted
tags:
  - "plugin"
  - "skills"
---

> **Outcome (2026-05-15):** Plan executed (3 track skills shipped) and then **superseded** by `skill-surface-collapse.adr.md`. All 6 track skills (the original 3 product/sources/iso plus the 3 added here — architecture-track, standard-track, feature-track) were removed and their flow content moved into `skills/plan/references/{product,sources,iso,feature}-flow.md` (with continuation logic for the architecture and standard chains living under `skills/decide/references/continuations.md`). The flows are reachable via `/archcore:plan --<flow>` or natural language; the standalone `/archcore:<flow>-track` commands are gone.

## Goal

Implement 3 new scenario-based track skills — `architecture-track`, `standard-track`, `feature-track` — following the established track skill pattern. Update the skills-system spec to register them.

## Tasks

### Phase 1: architecture-track (adr → spec → plan)

- [x] Create `skills/architecture-track/SKILL.md`
- [x] Flow: adr → spec → plan
- [x] Relations: spec `implements` adr, plan `implements` spec
- [x] Questions per step:
  - adr: "What decision was made? What alternatives were considered?"
  - spec: "What is the contract surface? What are the constraints?"
  - plan: "What are the implementation phases? Dependencies?"

### Phase 2: standard-track (adr → rule → guide)

- [x] Create `skills/standard-track/SKILL.md`
- [x] Flow: adr → rule → guide
- [x] Relations: rule `implements` adr, guide `related` rule
- [x] Questions per step:
  - adr: "What decision was made? Why this approach?"
  - rule: "What are the mandatory behaviors? How to enforce?"
  - guide: "What steps should developers follow? Common pitfalls?"

### Phase 3: feature-track (prd → spec → plan → task-type)

- [x] Create `skills/feature-track/SKILL.md`
- [x] Flow: prd → spec → plan → task-type
- [x] Relations: spec `implements` prd, plan `implements` spec, task-type `related` plan
- [x] Questions per step:
  - prd: "What problem does this solve? Success metrics?"
  - spec: "What is the technical contract? API surface?"
  - plan: "What phases? What are blockers?"
  - task-type: "What's the recurring pattern? Key steps?"

### Phase 4: Spec update and validation

- [x] Update skills-system.spec.md — add 3 new tracks to the Track Skills table
- [x] Verify each track works end-to-end via `/archcore:<track-name> <topic>`
- [x] Ensure no type-level guidance duplication — tracks define flow only

### Phase 5: Migration (post `skill-surface-collapse.adr.md`)

- [x] Delete all 6 track skill directories (`skills/{architecture,feature,iso,product,sources,standard}-track/`).
- [x] Move flow content into `skills/plan/references/{feature,iso,product,sources}-flow.md`.
- [x] Move ADR-driven continuation content (the standard-track and architecture-track flows that previously linked ADR → spec/rule/guide) into `skills/decide/references/continuations.md`.
- [x] Delete the corresponding `commands/<track>.md` wrappers.
- [x] Update `skills-system.spec.md` to describe the 7-skill surface (down from 16).

## Acceptance Criteria

Original criteria — all met at plan completion (pre-collapse):

- 3 new SKILL.md files exist at `skills/{architecture,standard,feature}-track/SKILL.md`
- Each follows the Step 0-N structure from existing tracks
- Each creates documents exclusively via `mcp__archcore__create_document`
- Each adds relations via `mcp__archcore__add_relation` between created documents
- Each checks for existing documents to determine scope
- skills-system.spec.md Track Skills table has 6 entries

Current state (post-collapse):

- The 6 track SKILL.md files no longer exist.
- Their flow content lives under `skills/plan/references/` (4 files) and `skills/decide/references/continuations.md`.
- `skills-system.spec.md` documents the 7-skill surface; no track tier.

## Dependencies

- Existing track skills (product-track, sources-track, iso-track) as structural reference — also retired in the collapse.
- All referenced document-type skills (adr, spec, plan, rule, guide, prd, task-type) — also retired by `remove-document-type-skills.adr.md`; their elicitation now lives inline in capture / decide / plan.
- `skills-system.spec.md` — current authoritative spec.
- `skill-surface-collapse.adr.md` — the decision that retired the track tier.
