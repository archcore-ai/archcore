---
title: "Scenario-Based Track Skills for Common Workflows"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

> **Outcome (2026-05-15):** Idea implemented and then **superseded** by `skill-surface-collapse.adr.md`. The 3 proposed scenario tracks shipped (architecture-track, standard-track, feature-track), bringing the track tier to 6 skills. The entire track tier was subsequently removed — the flow logic moved into `skills/plan/references/*-flow.md` (for the four cascades that still made sense as `plan` modes) and `skills/decide/references/continuations.md` (for the ADR-driven standard and architecture cascades). The "scenario tracks" framing is preserved as the routing model inside `plan` and `decide`, just without a dedicated skill per scenario.

## Idea

Expand the track skills system beyond the existing 3 requirements tracks (product-track, sources-track, iso-track) with new **scenario-based tracks** that orchestrate multi-document creation for common engineering workflows.

Instead of adding layer-level commands (`/archcore:vision`, `/archcore:knowledge`, `/archcore:experience`) — which would be an awkward middle ground between the type-specific skills and the create wizard — invest in scenario tracks that reflect how engineers actually think about documentation tasks.

## Value

### Why not layer-level skills

- Layers (vision/knowledge/experience) are Archcore's internal classification, not the user's mental model. Nobody thinks "I need a knowledge document" — they think "I made a decision" or "I need to set a standard."
- Layer skills still require type selection within the layer (e.g., knowledge has 6 types), offering minimal friction reduction over the create wizard.
- Model invocation with vague descriptions ("user wants a knowledge document") would conflict with specific type skills.

### Why scenario tracks

- They match real **use cases**: "design the architecture", "establish a standard", "plan a feature end-to-end."
- They create **chains of related documents** with proper relations — something single-type skills can't do.
- They encode **domain expertise** about which document types naturally follow each other.
- They save significant time: 3-4 documents with relations in one workflow vs. creating each manually.

### Proposed tracks

| Track | Flow | Use case |
|-------|------|----------|
| `architecture-track` | adr → spec → plan | Design an architectural decision from rationale through contract to implementation |
| `standard-track` | adr → rule → guide | Establish a decision as a team standard with how-to instructions |
| `feature-track` | prd → spec → plan → task-type | Take a feature from requirements through specification to repeatable implementation |

Each track follows the same pattern as existing tracks: sequential creation, focused questions at each step, automatic `add_relation` calls between documents.

### Where the flows live now

After `skill-surface-collapse.adr.md`:

| Original track | Current home |
|---|---|
| `product-track` | `skills/plan/references/product-flow.md` |
| `sources-track` | `skills/plan/references/sources-flow.md` |
| `iso-track` | `skills/plan/references/iso-flow.md` |
| `feature-track` | `skills/plan/references/feature-flow.md` |
| `architecture-track` (adr → spec → plan) | `skills/decide/references/continuations.md` (spec + plan continuations) — also reachable by starting in `plan` against an existing ADR |
| `standard-track` (adr → optional cpat → rule → guide) | `skills/decide/references/continuations.md` (rule + guide + cpat continuations) |

## Possible Implementation

1. Create `skills/architecture-track/SKILL.md`, `skills/standard-track/SKILL.md`, `skills/feature-track/SKILL.md`
2. Each follows the existing track skill structure
3. All tracks use `disable-model-invocation: true` — user initiates explicitly
4. Track skills do NOT duplicate document-type skill content — they define the flow and relation chain only
5. Update skills-system.spec.md to register the new tracks

(All five steps were executed; then the entire tier was retired per the Outcome note at the top.)

## Risks and Constraints

- **Track proliferation**: Too many tracks can overwhelm users. Start with 3 new tracks (6 total) and evaluate before adding more. — *Eventually validated: 6 tracks were too many, and the entire tier was collapsed.*
- **Overlap with existing tracks**: architecture-track's `plan` step overlaps with product-track's `plan`. — *This overlap was the core observation that drove the later collapse.*
- **Maintenance cost**: Each track is another file to maintain. — *Reduced by moving content into references that live under one skill instead of N skills.*
- **Scope creep per track**: Resist adding optional steps or conditional branches. — *Constraint inverted in the final form: the references can be richer because there's only one entry point.*
