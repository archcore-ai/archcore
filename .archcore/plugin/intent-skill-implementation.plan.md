---
title: "Intent Skill Implementation Plan — 4-Layer Migration"
status: accepted
tags:
  - "plugin"
  - "roadmap"
  - "skills"
---

## Goal

Migrate the Archcore Claude Plugin from a flat 27-skill surface to the 4-layer intent-based command hierarchy defined in `intent-based-skill-architecture.adr.md`. After this plan is complete, users see 7 primary intent commands, with track and type skills properly tiered.

## Tasks

### Phase 1: Create Intent Skills (Layer 1) — DONE

Created 7 intent skills following the 5-section structure: title+one-liner, When to Use, Routing Table, Execution, Result.

- [x] `skills/capture/SKILL.md` — NEW. Absorbs create wizard. Routes to adr/spec/doc/guide.
- [x] `skills/plan/SKILL.md` — REWRITE. Absorbs plan type skill. Routes to product-track or single plan.
- [x] `skills/decide/SKILL.md` — NEW. Creates adr, offers rule+guide follow-up.
- [x] `skills/standard/SKILL.md` — NEW. Routes to standard-track (adr→rule→guide).
- [x] `skills/review/SKILL.md` — REWRITE. Intent structure with routing table for scope.
- [x] `skills/status/SKILL.md` — REWRITE. Intent structure, compact dashboard.
- [x] `skills/help/SKILL.md` — NEW. 3-tier command guide.

### Phase 2: Remove Absorbed Skills — DONE

- [x] Deleted `skills/create/` directory.

### Phase 3: Update Track Skill Descriptions (Layer 2) — DONE

- [x] `skills/product-track/SKILL.md` — "Advanced — Create idea, PRD, and plan with full traceability."
- [x] `skills/sources-track/SKILL.md` — "Advanced — Create MRD, BRD, URD discovery documents."
- [x] `skills/iso-track/SKILL.md` — "Advanced — Create ISO 29148 requirements cascade (BRS → StRS → SyRS → SRS)."
- [x] `skills/architecture-track/SKILL.md` — "Advanced — Create ADR, spec, and plan for architectural design."
- [x] `skills/standard-track/SKILL.md` — "Advanced — Create ADR, rule, and guide to codify a standard."
- [x] `skills/feature-track/SKILL.md` — "Advanced — Create PRD, spec, plan, and task-type for feature lifecycle."

### Phase 4: Update Type Skill Descriptions (Layer 3) — DONE

**No prefix (high-frequency):** adr, prd, rule, guide, idea — kept as-is.

**"Expert —" prefix added:**
- [x] rfc, doc, spec, mrd, brd, urd, brs, strs, syrs, srs, task-type, cpat (12 skills)

### Phase 5: Trim Assistant Agent — DONE

- [x] Removed 18-type taxonomy and relation semantics from `agents/archcore-assistant.md`. Replaced with reference to MCP server instructions + focus areas (elicitation, composition, disambiguation, orchestration, relation patterns).

### Phase 6: Validate — DONE

- [x] All 7 intent skills exist with `disable-model-invocation: true`
- [x] All 6 track descriptions start with "Advanced —"
- [x] All 12 non-high-freq type descriptions start with "Expert —"
- [x] 5 high-frequency types (adr, prd, rule, guide, idea) keep original descriptions
- [x] `skills/create/` removed
- [x] Agent trimmed — no duplicate taxonomy
- [x] Total: 30 skill directories (7 intent + 6 track + 17 type)

## Acceptance Criteria

All met. The `plan` type skill was absorbed into the `/archcore:plan` intent skill, bringing type skill count to 17 (from original 18). Total 30 = 7 + 6 + 17.

## Dependencies

- `intent-based-skill-architecture.adr.md` — the decision being implemented ✓
- `skills-system.spec.md` — defines the 5-section intent skill structure ✓
- `commands-system.spec.md` — defines tier prefixes and discoverability rules ✓
- `plugin-architecture.spec.md` — defines the 4-layer model ✓