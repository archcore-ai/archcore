---
title: "Scenario-Based Track Skills for Common Workflows"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

**Outcome (2026-05-15).** The idea was implemented and then superseded by `skill-surface-collapse.adr`. The three proposed scenario tracks shipped, bringing the track tier to 6 skills, and the entire tier was subsequently removed: the flow logic moved into `skills/plan/references/*-flow.md` for the four cascades that still made sense as `plan` modes, and into `skills/decide/references/continuations.md` for the ADR-driven standard and architecture cascades. The scenario-track framing survives as the routing model inside `plan` and `decide`, without a dedicated skill per scenario.

## Idea

Expand the track skills beyond the three requirements tracks — product, sources, and ISO — with **scenario-based tracks** that orchestrate multi-document creation for common engineering workflows.

The alternative under consideration was layer-level commands such as `/archcore:vision` and `/archcore:knowledge`, which would have been an awkward middle ground between the type-specific skills and the creation wizard. Scenario tracks instead reflect how engineers describe a documentation task.

| Track | Flow | Use case |
|-------|------|----------|
| `architecture-track` | adr → spec → plan | Design an architectural decision from rationale through contract to implementation |
| `standard-track` | adr → rule → guide | Establish a decision as a team standard with how-to instructions |
| `feature-track` | prd → spec → plan → task-type | Take a feature from requirements through specification to repeatable implementation |

Each track follows the existing pattern: sequential creation, focused questions at each step, and automatic `add_relation` calls between the documents.

## Value

**Why not layer-level skills.** The layers — vision, knowledge, experience — are Archcore's internal classification rather than the user's mental model: nobody thinks "I need a knowledge document", they think "I made a decision" or "I need to set a standard". A layer skill would still require type selection inside the layer, since knowledge alone holds 6 types, so it reduces little friction over the wizard. And model invocation on a vague description would conflict with the specific type skills.

**Why scenario tracks.** They match real use cases — design the architecture, establish a standard, plan a feature end to end. They create chains of related documents with proper relations, which a single-type skill cannot do. They encode domain expertise about which types naturally follow each other. And they save time: three or four related documents in one workflow rather than each created by hand.

**Where the flows live now**, after the collapse:

| Original track | Current home |
|---|---|
| `product-track` | `skills/plan/references/product-flow.md` |
| `sources-track` | `skills/plan/references/sources-flow.md` |
| `iso-track` | `skills/plan/references/iso-flow.md` |
| `feature-track` | `skills/plan/references/feature-flow.md` |
| `architecture-track` (adr → spec → plan) | `skills/decide/references/continuations.md`, and also reachable by starting in `plan` against an existing ADR |
| `standard-track` (adr → optional cpat → rule → guide) | `skills/decide/references/continuations.md` |

## Possible Implementation

1. Create `skills/architecture-track/SKILL.md`, `skills/standard-track/SKILL.md`, and `skills/feature-track/SKILL.md`.
2. Follow the existing track skill structure in each.
3. Set `disable-model-invocation: true` on all three, so the user initiates explicitly.
4. Keep the tracks free of document-type skill content: a track defines the flow and the relation chain only.
5. Register the new tracks in the skills specification.

All five steps were executed, and the tier was then retired per the outcome note above.

## Risks

- **Track proliferation.** Too many tracks overwhelm the user, so the plan was to start at 6 total and evaluate before adding more. Validated in the outcome: 6 was too many, and the whole tier collapsed.
- **Overlap with the existing tracks.** The `plan` step of `architecture-track` overlaps with the `plan` step of `product-track`. That overlap was the core observation that drove the later collapse.
- **Maintenance cost.** Each track is another file to maintain, which the move to references under one skill reduced.
- **Scope creep per track.** Optional steps and conditional branches were to be resisted. The constraint inverted in the final form: a reference can be richer, because there is only one entry point.
