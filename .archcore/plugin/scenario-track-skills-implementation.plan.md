---
title: "Scenario Track Skills Implementation Plan"
status: accepted
tags:
  - "plugin"
  - "skills"
---

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

## Acceptance Criteria

- 3 new SKILL.md files exist at `skills/{architecture,standard,feature}-track/SKILL.md`
- Each has `disable-model-invocation: true` in frontmatter
- Each follows the Step 0-N structure from existing tracks (product-track as reference)
- Each creates documents exclusively via `mcp__archcore__create_document`
- Each adds relations via `mcp__archcore__add_relation` between created documents
- Each checks for existing documents to determine scope (skip already-created steps)
- skills-system.spec.md Track Skills table has 6 entries (3 existing + 3 new)

## Dependencies

- Existing track skills (product-track, sources-track, iso-track) as structural reference
- All referenced document-type skills (adr, spec, plan, rule, guide, prd, task-type) already exist
- skills-system.spec.md as the spec to update
