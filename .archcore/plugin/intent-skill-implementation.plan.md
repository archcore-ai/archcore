---
title: "Intent Skill Implementation Plan — 4-Layer Migration"
status: accepted
tags:
  - "plugin"
  - "roadmap"
  - "skills"
---

**Outcome (2026-05-15).** The plan was executed in stages, and the four-layer model it implemented was simplified twice: first by `remove-document-type-skills.adr`, which removed Layer 3 and left 18 skills, then by `skill-surface-collapse.adr`, which collapsed Layer 2 into the surviving intents and left 7. The goal of migrating from a flat surface to a tiered hierarchy was met and then deliberately re-flattened, once the tiering created more friction than it removed. The current surface is `init`, `capture`, `decide`, `plan`, `audit`, `context`, and `help`.

## Goal

Migrate the plugin from a flat 27-skill surface to the four-layer intent-based command hierarchy defined in `intent-based-skill-architecture.adr`. On completion, users see 8 primary intent commands, with the track and type skills tiered beneath them.

**Surface history.** After the plan, the `graph` intent was added, taking the intent count to 9, then `context` and `bootstrap` took it to 11, and the `verify` utility was added separately, peaking at 33 skill directories — 9 intent, 6 track, 17 type, 1 utility. Removing the type skills left 18 directories. Merging `status` into `review` and removing `graph` left 16. The final collapse left **7**: `bootstrap` renamed to `init`; `review` and `actualize` merged into `audit` with its `--deep` and `--drift` modes; every track folded into `plan` as a reference; `standard` folded into the `decide` continuation chain; and `verify` removed in favor of `make verify`. `component-registry.doc` holds the current inventory.

## Tasks

**Phase 1 — create the intent skills.** Done. Seven intent skills were created in the five-section structure of title with one-liner, When to Use, Routing Table, Execution, and Result.

- [x] `skills/capture/SKILL.md`, new, absorbing the create wizard and routing to adr, spec, doc, or guide.
- [x] `skills/plan/SKILL.md`, rewritten to absorb the plan type skill and, after the collapse, to route to per-flow references.
- [x] `skills/decide/SKILL.md`, new, creating an adr or rfc and offering the rule and guide follow-up.
- [x] `skills/standard/SKILL.md`, new, routing into the standard track. Later merged into `decide`.
- [x] `skills/review/SKILL.md`, rewritten. Later merged into `audit`.
- [x] `skills/status/SKILL.md`, rewritten. Later merged into `review`, then into `audit`.
- [x] `skills/help/SKILL.md`, new, carrying the command guide.

**Phase 1b — the actualize intent skill.** Done, added once the actualize decision and specification were complete.

- [x] `skills/actualize/SKILL.md`, new, detecting stale documents. Later merged into `audit --drift`.

**Phase 2 — remove the absorbed skills.** Done.

- [x] Delete the `skills/create/` directory.

**Phase 3 — update the track descriptions.** Done, then undone. The track skills received the "Advanced —" prefix as planned, and all six were later removed entirely, with their flow content moving to the four `skills/plan/references/*-flow.md` files plus the continuation logic under `skills/decide/references/`.

**Phase 4 — update the type descriptions.** Done, then undone. This is historical: the type skills were later removed entirely, so the tier-prefix work applied only while they existed.

**Phase 5 — trim the assistant agent.** Done.

- [x] Remove the 18-type taxonomy and the relation semantics from the assistant definition, replacing them with a reference to the MCP server instructions plus focus areas.

**Phase 6 — validate.** Done.

- [x] Every intent skill carried `disable-model-invocation: true`. That flag was later removed by `inverted-invocation-policy.adr`, so intent skills now auto-invoke, a policy `skill-surface-collapse.adr` reaffirms.
- [x] Every track description began with "Advanced —", which became moot after the track removal.
- [x] Every non-high-frequency type description began with "Expert —", which became moot after the type-skill removal.
- [x] `skills/create/` was removed.
- [x] The agent was trimmed, with no duplicate taxonomy.
- [x] The total at plan completion was 31 skill directories: 8 intent, 6 track, 17 type.

## Acceptance Criteria

All criteria were met at plan completion. The plan type skill was absorbed into the plan intent skill, the actualize intent skill was added in Phase 1b, and the total at completion was 31.

Subsequent work added the `graph`, `context`, and `bootstrap` intents plus the `verify` utility, peaking at 33 to 34 directories on disk, after which the invocation flags were re-tuned. The surface then consolidated in three further steps: the type skills were removed entirely, leaving 18; `status` merged into `review` and `graph` was removed, leaving 16; and the final collapse left 7.

## Dependencies

- `intent-based-skill-architecture.adr` — the decision this plan implements. Its structural decomposition still stands, though Layers 2 and 3 have both been collapsed.
- `inverted-invocation-policy.adr` — superseded the per-class invocation flags decided here, and was added after plan completion.
- `remove-document-type-skills.adr` — removed the entire type-skill layer.
- `merge-review-status-remove-graph.adr` — merged `status` into `review` and removed `graph`.
- `skill-surface-collapse.adr` — the final consolidation to 7 skills.
- `skills-system.spec`, `commands-system.spec`, and `plugin-architecture.spec` — define the current skill structure, the visible command surface, and the overall architecture.
- `actualize-system.adr` — the decision behind the actualize intent, later folded into `audit`.
