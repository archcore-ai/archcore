---
title: "Scenario-Based Track Skills for Common Workflows"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

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

## Possible Implementation

1. Create `skills/architecture-track/SKILL.md`, `skills/standard-track/SKILL.md`, `skills/feature-track/SKILL.md`
2. Each follows the existing track skill structure (Step 0: Verify MCP → Step 1: Check existing → Step 2: Determine scope → Steps 3-N: Create documents → Final: Cross-relate)
3. All tracks use `disable-model-invocation: true` — user initiates explicitly
4. Track skills do NOT duplicate document-type skill content — they define the flow and relation chain only
5. Update skills-system.spec.md to register the new tracks

## Risks and Constraints

- **Track proliferation**: Too many tracks can overwhelm users. Start with 3 new tracks (6 total) and evaluate before adding more.
- **Overlap with existing tracks**: architecture-track's `plan` step overlaps with product-track's `plan`. Tracks should have clear entry points and distinct use cases.
- **Maintenance cost**: Each track is another file to maintain. Keep tracks lean — flow definition only, no type-level guidance duplication.
- **Scope creep per track**: Resist adding optional steps or conditional branches. Each track should have a fixed, predictable flow of 3-4 documents.
