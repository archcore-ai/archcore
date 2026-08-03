---
title: "Collapse Skill Surface to 7 Skills — Merge Tracks and Inspection Modes"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Context

After the prior consolidation steps — `intent-based-skill-architecture.adr` establishing the four-layer model, `remove-document-type-skills.adr` collapsing Layer 3 to 18 visible skills, `merge-review-status-remove-graph.adr` merging status into review at 16, and `inverted-invocation-policy.adr` — the visible `/` palette stabilized at 16 skills: 9 intent, 6 track, and 1 utility. Three further frictions then surfaced in practice: the track skills duplicated plan-skill logic, the `actualize` skill was a one-mode peer to `review`, and `standard` and `verify` carried palette weight disproportionate to their surface.

## Observations

1. **Track skills duplicated plan-skill logic.** All six — `product-track`, `sources-track`, `iso-track`, `architecture-track`, `standard-track`, `feature-track` — orchestrated multi-document flows ending in or composing a `plan` document, and that orchestration already lived inside `skills/plan/SKILL.md`. Each track added little beyond a slightly different routing table and per-flow questions, making it a `plan` mode parameterized by which preceding documents to create.
2. **`actualize` was a one-mode peer to `review`.** Both loaded the same data through `list_documents` and `list_relations`, both produced findings tables, and both answered the same "is documentation healthy?" intent. `review` already had a default short mode and `--deep`, so adding `--drift` as a third mode is structurally identical to the earlier `status` into `review` merge.
3. **`standard` and `verify` were thin shims.** `standard` was a one-skill router into `standard-track`. `verify` was a maintenance-only utility that re-invoked `make verify`, which the shell already reaches directly.

## Decision

Collapse the visible `/` palette from 16 skills to **7** — `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help` — removing the track skills, `standard`, `verify`, and `actualize` as standalone skills and folding their behavior into the remaining intents.

| Skill | What it does | Absorbed |
|---|---|---|
| `init` | First-time onboarding — seed `.archcore/` with documents sized to the detected scale | renamed from `bootstrap` |
| `capture` | Document a module or component; routes to adr, spec, doc, or guide | unchanged |
| `decide` | Record a decision (ADR) or draft a proposal (RFC); optional rule and guide continuation | also covers the former `standard` continuation chain |
| `plan` | Plan a feature or initiative end to end; route to the product, sources, iso, or feature flow | absorbs all six former track skills as flow references under `skills/plan/references/` |
| `audit` | Documentation health: dashboard by default, `--deep` audit, or `--drift` detection | absorbs the former `review` short and deep modes and `actualize` as `--drift` |
| `context` | Surface rules and decisions for a code area or pickup | unchanged |
| `help` | Layer navigation and onboarding | unchanged |

The concrete migrations were: `skills/bootstrap/` renamed to `skills/init/` with its `lib/*.md` sub-references, and the command renamed accordingly, with the seeding behavior unchanged; `review` and `actualize` merged into `audit` with three modes via `[--deep] [--drift] [filter]`, the drift logic moving to `skills/audit/lib/drift-detection.md`; the six track skills becoming `skills/plan/references/{product-flow,sources-flow,iso-flow,feature-flow}.md`, with the `architecture-track` chain reachable through `decide` and `plan` and the `standard-track` chain through `decide`'s rule and guide continuation, whose optional CPAT step lives in that continuation logic; `standard` removed, because "establish a standard" lands on `decide` directly; `verify` removed with no replacement, since `make verify` from the repository root is the canonical integrity check; the Codex `commands/*.md` wrappers reduced to 7; and the count invariants updated across `README.md`, the structure tests, and the foundational `.archcore/` documents.

## Alternatives Considered

1. **Keep the tracks as standalone skills** — rejected because each track is a `plan` mode parameterized by which preceding documents to create, so keeping six standalone tracks forced the model to disambiguate "plan a feature" between `/archcore:plan`, `/archcore:product-track`, and `/archcore:feature-track` on every invocation, repeating the routing-overlap problem that `merge-review-status-remove-graph.adr` had already solved for inspection skills.
2. **Keep `actualize` and `review` as separate standalone intents** — rejected because both are pure analysis over the same data source, and a mode flag expresses the depth and topic distinction more honestly than three peer intents that all anti-trigger one another. This is the same argument that retired `status` and `graph`.
3. **Keep `standard` as a router into the `decide` cascade** — considered, because it would preserve the explicit "establish a standard" entry phrase, and rejected because `decide` already enumerates the "we decided" trigger plus continuation prompts for rule and guide; one more anti-trigger surface to maintain was not worth the explicitness.
4. **Keep `verify` because it surfaces a useful action** — rejected because it was always a one-line passthrough to `make verify`, and a palette entry for a passthrough costs more cognitive load than it saves.

## Consequences

- The visible palette drops to 7 commands from 16, and each skill maps to a distinct user intent, so no two skills anti-trigger each other.
- One source of truth per concern: `plan` for any forward-looking flow, `audit` for any health check, `decide` for any standards or decision cascade.
- Track flows stay reachable as references under `skills/plan/references/`, so adding a flow is a new markdown file rather than a new skill.
- Drift detection sits beside the other audit modes, which makes the modes easier to keep consistent.
- The session-start token budget shrinks from 16 skill descriptions to 7.
- Cross-host parity holds: every remaining skill is auto-invocable in Claude Code, Cursor, and Codex through the matching wrapper.
- Tradeoff: users who memorized `/archcore:bootstrap`, `/archcore:review`, `/archcore:actualize`, `/archcore:standard`, `/archcore:verify`, or any `*-track` invocation must learn the new mapping. `/archcore:help` documents the migration and the README explains the active surface.
- Tradeoff: track-flow discoverability shifts. A user who would have typed `/archcore:iso-track` now types `/archcore:plan` and either describes the cascade or passes `--iso`, which the plan routing table makes deterministic but which costs one extra step of phrasing.
- Tradeoff: the `plan` skill grows, holding the routing logic for all four flows. The references directory absorbs the per-flow content so `SKILL.md` stays inside its line budget.
- Tradeoff: `verify` was the only in-session integrity path. Its replacement runs from the repository root rather than the plugin root, because the `Makefile` stays at the repository root while the plugin lives in `plugins/archcore/` per `subdirectory-plugin-layout.adr`.
- This decision supersedes the Layer 2 track tier of `intent-based-skill-architecture.adr`, whose intent-versus-utility classification remains; the 16-command palette of `merge-review-status-remove-graph.adr`, whose intent-merge pattern is extended here; and the standalone `standard` and `verify` intents in `commands-system.spec`. The mainstream and niche distinction of `inverted-invocation-policy.adr` had already been retired by `remove-document-type-skills.adr`, and this change additionally retires the auto-invocable track tier.

## Constraints

1. The visible `/` palette MUST hold exactly 7 commands: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`.
2. An eighth skill MUST NOT be added without a new ADR.
3. `audit` MUST support three modes: default short, `--deep`, and `--drift`, with the drift protocol at `skills/audit/lib/drift-detection.md`.
4. `plan` MUST hold its per-flow logic in `skills/plan/references/*.md` rather than spawning a top-level skill.
5. `decide` MUST own the standard and decision cascade: ADR, then optional CPAT for a code-pattern change, then optional rule, then optional guide.
6. A skill MUST NOT carry `disable-model-invocation: true`. Where a utility need re-emerges, the author SHOULD prefer a `make` target or a CLI command over a hidden skill.

## Superseded when

- A measured session sample shows users routinely failing to reach a former track flow through `plan`, which would argue for restoring an explicit entry point.
- An eighth genuinely distinct user intent appears that no existing skill can absorb as a mode, which would require the new ADR that constraint 2 mandates.
