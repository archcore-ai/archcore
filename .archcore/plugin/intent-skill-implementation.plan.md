---
title: "Intent Skill Implementation Plan — 4-Layer Migration"
status: accepted
tags:
  - "plugin"
  - "roadmap"
  - "skills"
---

## Goal

Migrate the Archcore Claude Plugin from a flat 27-skill surface to the 4-layer intent-based command hierarchy defined in `intent-based-skill-architecture.adr.md`. After this plan is complete, users see 8 primary intent commands, with track and type skills properly tiered.

> **Subsequent additions (post-plan)**: the `graph` intent skill was added later, bringing intent count to 9; the `context` and `bootstrap` intents were added after that (intent count → 11); the `verify` utility skill was added separately. Total skill directories at the 33-skill peak: 33 (9 intent + 6 track + 17 type + 1 utility). **Today (post `remove-document-type-skills.adr.md`): 18 skill directories** (11 intent + 6 track + 1 utility) — type skills were removed after evidence showed their per-type content was already duplicated in intent/track skills. See `component-registry.doc.md` for the current inventory.

## Tasks

### Phase 1: Create Intent Skills (Layer 1) — DONE

Created 7 intent skills following the 5-section structure: title+one-liner, When to Use, Routing Table, Execution, Result.

- [x] `skills/capture/SKILL.md` — NEW. Absorbs create wizard. Routes to adr/spec/doc/guide.
- [x] `skills/plan/SKILL.md` — REWRITE. Absorbs plan type skill. Routes to product-track or single plan.
- [x] `skills/decide/SKILL.md` — NEW. Creates adr or rfc (RFC branch added later); offers rule+guide follow-up.
- [x] `skills/standard/SKILL.md` — NEW. Routes to standard-track (adr → optional cpat → rule → guide; CPAT step added later).
- [x] `skills/review/SKILL.md` — REWRITE. Intent structure with routing table for scope.
- [x] `skills/status/SKILL.md` — REWRITE. Intent structure, compact dashboard.
- [x] `skills/help/SKILL.md` — NEW. Command guide.

### Phase 1b: Actualize Intent Skill — DONE

Added 8th intent skill after the Actualize System ADR and Specification were completed:

- [x] `skills/actualize/SKILL.md` — NEW. Detects stale docs via code drift, cascade, and temporal analysis.

### Phase 2: Remove Absorbed Skills — DONE

- [x] Deleted `skills/create/` directory.

### Phase 3: Update Track Skill Descriptions (Layer 2) — DONE

- [x] `skills/product-track/SKILL.md` — "Advanced — Create idea, PRD, and plan with full traceability."
- [x] `skills/sources-track/SKILL.md` — "Advanced — Create MRD, BRD, URD discovery documents."
- [x] `skills/iso-track/SKILL.md` — "Advanced — Create ISO 29148 requirements cascade (BRS → StRS → SyRS → SRS)."
- [x] `skills/architecture-track/SKILL.md` — "Advanced — Create ADR, spec, and plan for architectural design."
- [x] `skills/standard-track/SKILL.md` — "Advanced — Create ADR, rule, and guide to codify a standard." (Later extended with optional CPAT step.)
- [x] `skills/feature-track/SKILL.md` — "Advanced — Create PRD, spec, plan, and task-type for feature lifecycle."

### Phase 4: Update Type Skill Descriptions (Layer 3) — DONE

**Historical.** Type skills were later removed entirely (see `remove-document-type-skills.adr.md`). Tier-prefix work described below was relevant only while type skills existed.

**No prefix (high-frequency):** adr, prd, rule, guide, idea — kept as-is.

**"Expert —" prefix added:**
- [x] rfc, doc, spec, mrd, brd, urd, brs, strs, syrs, srs, task-type, cpat (12 skills)

### Phase 5: Trim Assistant Agent — DONE

- [x] Removed 18-type taxonomy and relation semantics from `agents/archcore-assistant.md`. Replaced with reference to MCP server instructions + focus areas (elicitation, composition, disambiguation, orchestration, relation patterns).

### Phase 6: Validate — DONE

- [x] All 8 intent skills existed with `disable-model-invocation: true` (note: this flag was later REMOVED by the Inverted Invocation Policy ADR — intent skills are now auto-invocable)
- [x] All 6 track descriptions start with "Advanced —"
- [x] All 12 non-high-freq type descriptions start with "Expert —" (moot after type-skill removal)
- [x] 5 high-frequency types (adr, prd, rule, guide, idea) keep original descriptions (moot after type-skill removal)
- [x] `skills/create/` removed
- [x] Agent trimmed — no duplicate taxonomy
- [x] Total at plan completion: 31 skill directories (8 intent + 6 track + 17 type)

## Acceptance Criteria

All met at plan completion. The `plan` type skill was absorbed into the `/archcore:plan` intent skill, bringing type skill count to 17 (from original 18). The `actualize` intent skill was added in Phase 1b, bringing intent count to 8. Total at completion: 31 = 8 + 6 + 17.

Note: Subsequent work added the `graph`, `context`, and `bootstrap` intent skills (intent count → 11) and the `verify` utility skill. At peak, total on disk was 33–34. Invocation flags were then re-tuned by the Inverted Invocation Policy ADR: intent and track skills lost `disable-model-invocation`; mainstream types gained it; niche types gained `user-invocable: false`. Finally, **type skills were removed entirely** by `remove-document-type-skills.adr.md` — current total on disk is 18 (11 intent + 6 track + 1 utility).

## Dependencies

- `intent-based-skill-architecture.adr.md` — the decision being implemented (structural decomposition still stands; Layer 3 has been collapsed) ✓
- `inverted-invocation-policy.adr.md` — supersedes the per-class invocation flags decided here (added after plan completion). Type-skill portion of the policy was later superseded by `remove-document-type-skills.adr.md`.
- `remove-document-type-skills.adr.md` — removes the entire Type Skill (Layer 3) surface.
- `skills-system.spec.md` — defines the intent/track skill structure (now 3-group hierarchy) ✓
- `commands-system.spec.md` — defines tier prefixes and discoverability rules ✓
- `plugin-architecture.spec.md` — defines the overall architecture (Layer 3 removed) ✓
- `actualize-system.adr.md` — decision for the actualize intent skill ✓
