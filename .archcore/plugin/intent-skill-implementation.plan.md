---
title: "Intent Skill Implementation Plan — 4-Layer Migration"
status: accepted
tags:
  - "plugin"
  - "roadmap"
  - "skills"
---

> **Outcome (2026-05-15):** Plan executed in stages. The 4-layer model proposed here was simplified twice: first by `remove-document-type-skills.adr.md` (Layer 3 removed → 18 skills), then by `skill-surface-collapse.adr.md` (Layer 2 collapsed into the surviving intents → 7 skills total). The "Migrate from a flat surface to a tiered hierarchy" goal was met and then deliberately re-flattened once the tiering created more friction than it removed. Current surface is `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`.

## Goal

Migrate the Archcore Plugin from a flat 27-skill surface to the 4-layer intent-based command hierarchy defined in `intent-based-skill-architecture.adr.md`. After this plan is complete, users see 8 primary intent commands, with track and type skills properly tiered.

> **Subsequent additions (post-plan)**: the `graph` intent skill was added later, bringing intent count to 9; the `context` and `bootstrap` intents were added after that (intent count → 11); the `verify` utility skill was added separately. Total skill directories at the 33-skill peak: 33 (9 intent + 6 track + 17 type + 1 utility).
> **Post `remove-document-type-skills.adr.md`**: 18 skill directories (11 intent + 6 track + 1 utility) — type skills were removed after evidence showed their per-type content was already duplicated in intent/track skills.
> **Post `merge-review-status-remove-graph.adr.md`**: 16 skill directories (9 intent + 6 track + 1 utility) — `status` merged into `review`, `graph` removed.
> **Post `skill-surface-collapse.adr.md` (current)**: **7 skill directories** total. `bootstrap` renamed to `init`. `review` + `actualize` merged into `audit` (with `--deep` and `--drift` modes). All track skills folded into `plan` as references under `skills/plan/references/`. `standard` folded into `decide`'s continuation chain. `verify` removed in favor of `make verify`.

See `component-registry.doc.md` for the current inventory.

## Tasks

### Phase 1: Create Intent Skills (Layer 1) — DONE

Created 7 intent skills following the 5-section structure: title+one-liner, When to Use, Routing Table, Execution, Result.

- [x] `skills/capture/SKILL.md` — NEW. Absorbs create wizard. Routes to adr/spec/doc/guide.
- [x] `skills/plan/SKILL.md` — REWRITE. Absorbs plan type skill. Routes to per-flow references (post `skill-surface-collapse.adr.md`).
- [x] `skills/decide/SKILL.md` — NEW. Creates adr or rfc; offers rule+guide follow-up.
- [x] `skills/standard/SKILL.md` — NEW. Routes to standard-track. **Later merged into `decide`** per `skill-surface-collapse.adr.md`.
- [x] `skills/review/SKILL.md` — REWRITE. **Later merged into `audit`** per `skill-surface-collapse.adr.md`.
- [x] `skills/status/SKILL.md` — REWRITE. **Later merged into `review`** per `merge-review-status-remove-graph.adr.md`, then into `audit`.
- [x] `skills/help/SKILL.md` — NEW. Command guide.

### Phase 1b: Actualize Intent Skill — DONE

Added after the Actualize System ADR and Specification were completed:

- [x] `skills/actualize/SKILL.md` — NEW. Detects stale docs. **Later merged into `audit --drift`** per `skill-surface-collapse.adr.md`.

### Phase 2: Remove Absorbed Skills — DONE

- [x] Deleted `skills/create/` directory.

### Phase 3: Update Track Skill Descriptions (Layer 2) — DONE, THEN UNDONE

Track skills received the "Advanced —" prefix as planned. **Later, all 6 track skills were removed entirely** per `skill-surface-collapse.adr.md` and their flow content moved to `skills/plan/references/{product,sources,iso,feature}-flow.md` plus continuation logic in `skills/decide/references/continuations.md`.

### Phase 4: Update Type Skill Descriptions (Layer 3) — DONE, THEN UNDONE

**Historical.** Type skills were later removed entirely (see `remove-document-type-skills.adr.md`). Tier-prefix work described below was relevant only while type skills existed.

### Phase 5: Trim Assistant Agent — DONE

- [x] Removed 18-type taxonomy and relation semantics from `agents/archcore-assistant.md`. Replaced with reference to MCP server instructions + focus areas.

### Phase 6: Validate — DONE

- [x] All intent skills existed with `disable-model-invocation: true` (note: this flag was later REMOVED by the Inverted Invocation Policy ADR — intent skills are now auto-invocable; the policy is reaffirmed by `skill-surface-collapse.adr.md`).
- [x] All track descriptions started with "Advanced —" (moot after track removal).
- [x] All non-high-freq type descriptions started with "Expert —" (moot after type-skill removal).
- [x] `skills/create/` removed.
- [x] Agent trimmed — no duplicate taxonomy.
- [x] Total at plan completion: 31 skill directories (8 intent + 6 track + 17 type). Current: 7.

## Acceptance Criteria

All met at plan completion. The `plan` type skill was absorbed into the `/archcore:plan` intent skill. The `actualize` intent skill was added in Phase 1b. Total at completion: 31 = 8 + 6 + 17.

Note: Subsequent work added the `graph`, `context`, and `bootstrap` intent skills and the `verify` utility skill. At peak, total on disk was 33–34. Invocation flags were then re-tuned by the Inverted Invocation Policy ADR.

**Then** the surface was consolidated in three further steps:

1. **Type skills removed entirely** by `remove-document-type-skills.adr.md` → 18 skills.
2. **`status` merged into `review`, `graph` removed** by `merge-review-status-remove-graph.adr.md` → 16 skills.
3. **Tracks collapsed into `plan` references; `review`+`actualize` merged into `audit`; `bootstrap` renamed to `init`; `standard` folded into `decide`; `verify` removed** by `skill-surface-collapse.adr.md` → **7 skills (current)**.

## Dependencies

- `intent-based-skill-architecture.adr.md` — the decision being implemented (structural decomposition still stands; Layer 2 and Layer 3 have both been collapsed) ✓
- `inverted-invocation-policy.adr.md` — superseded the per-class invocation flags decided here (added after plan completion).
- `remove-document-type-skills.adr.md` — removed the entire Type Skill (Layer 3) surface.
- `merge-review-status-remove-graph.adr.md` — merged `status` into `review`, removed `graph`.
- `skill-surface-collapse.adr.md` — final consolidation to 7 skills.
- `skills-system.spec.md` — defines the current skill structure ✓
- `commands-system.spec.md` — defines the visible command surface ✓
- `plugin-architecture.spec.md` — defines the overall architecture ✓
- `actualize-system.adr.md` — decision for the actualize intent skill (later folded into `audit`) ✓
