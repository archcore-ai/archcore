---
title: "Collapse Skill Surface to 7 Skills — Merge Tracks and Inspection Modes"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Context

After the prior consolidation steps — `intent-based-skill-architecture.adr.md` (4-layer model), `remove-document-type-skills.adr.md` (collapse Layer 3 → 18 visible skills), `merge-review-status-remove-graph.adr.md` (merge status into review → 16 visible skills), and `inverted-invocation-policy.adr.md` — the visible `/` palette stabilized at 16 skills: 9 intent + 6 track + 1 utility. In practice three further frictions surfaced:

**1. Track skills duplicated plan-skill logic.** All six track skills (`product-track`, `sources-track`, `iso-track`, `architecture-track`, `standard-track`, `feature-track`) orchestrated multi-document flows that ended in (or composed) a `plan` document. The orchestration logic was already inside `skills/plan/SKILL.md`; the track skills mostly added a slightly different routing table and per-flow questions. Each track-flow was effectively a `plan` mode parameterized by which preceding documents to create.

**2. The `actualize` skill was a one-mode peer to `review`.** Both intents loaded the same data (`list_documents`, `list_relations`), both produced findings tables, and both were called from the same "is documentation healthy?" intent space. `review` already had two modes (default short, `--deep`); adding `--drift` as a third mode is structurally identical to how the `status` → `review` merge already worked.

**3. `standard` and `verify` carried weight disproportionate to their surface.** `standard` was a one-skill router into `standard-track`. `verify` was a maintenance-only utility that re-invoked `make verify` — accessible directly via the shell. Both were palette real estate for thin shims.

## Decision

**Collapse the visible `/` palette from 16 skills to 7. Track skills, standard, verify, and actualize are removed as standalone skills; their behavior is folded into the remaining intents.**

### Final surface (7 skills, 7 commands)

| Skill | What it does | Absorbed |
|---|---|---|
| `init` | First-time onboarding — seed `.archcore/` with scale-appropriate docs | renamed from `bootstrap` |
| `capture` | Document a module/component → routes to adr/spec/doc/guide | unchanged |
| `decide` | Record a decision (ADR) or draft a proposal (RFC); optional rule+guide continuation | also covers former `standard` (continuation chain → adr → optional rule → guide) |
| `plan` | Plan a feature or initiative end-to-end; route to product/sources/iso/feature flow as needed | absorbs all six former track skills via flow references under `skills/plan/references/` |
| `audit` | Documentation health: dashboard (default), deep audit (`--deep`), or drift detection (`--drift`) | absorbs former `review` (short + deep) and `actualize` (`--drift`) |
| `context` | Surface rules/decisions for a code area or pickup | unchanged |
| `help` | Layer navigation and onboarding | unchanged |

### Concrete migrations

1. **`bootstrap` → `init`.** Skill directory `skills/bootstrap/` renamed to `skills/init/` (all `lib/*.md` sub-references move with it). Command `/archcore:bootstrap` renamed to `/archcore:init`. The seeding behavior is unchanged.

2. **`review` + `actualize` → `audit`.** The new `audit` skill carries three modes via `[--deep] [--drift] [filter]`. Drift-detection logic moves to `skills/audit/lib/drift-detection.md` (formerly `skills/actualize/SKILL.md`). Short and deep modes preserve the behavior from the merged-`review` skill.

3. **Six track skills → `plan` references.** `skills/plan/references/{product-flow,sources-flow,iso-flow,feature-flow}.md` hold the per-flow content. The `architecture-track` ADR→spec→plan chain is reachable through `decide` (which already offers spec+plan continuation) and `plan` (which can implement an existing ADR). The `standard-track` ADR→(cpat)→rule→guide chain is reachable through `decide` with its rule+guide continuation; the optional CPAT step lives in `decide`'s continuation logic.

4. **`standard` removed.** Its only behavior was routing into `standard-track`. Users say "establish a standard" → `decide` picks up the ADR + rule + guide cascade directly.

5. **`verify` removed.** No replacement skill — `make verify` from the plugin root is the canonical way to run plugin integrity checks. Removing the skill recovers a palette slot for the more common audit invocation.

6. **Codex `commands/*.md` wrappers updated**: 7 wrappers (`init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`). Old wrappers (`bootstrap`, `review`, `actualize`, `standard`, `verify`, six `*-track`) deleted. Wrappers remain thin host-adapter shims that delegate to `skills/<name>/SKILL.md`.

7. **Count invariants updated** in `README.md`, structure tests (`test/structure/skills.bats`, `test/structure/codex-plugin.bats`), and `.archcore/plugin/` foundational docs (`commands-system.spec.md`, `skills-system.spec.md`, `component-registry.doc.md`, `plugin-architecture.spec.md`, `claude-plugin.prd.md`, `plugin-component-architecture.adr.md`).

## Alternatives Considered

### Keep tracks as standalone skills

**Rejected.** Each track is effectively a `plan` mode parameterized by which preceding documents to create. Keeping six standalone tracks meant the model had to disambiguate "plan a feature" between `/archcore:plan`, `/archcore:product-track`, and `/archcore:feature-track` on every invocation — repeating the routing-overlap problem that `merge-review-status-remove-graph.adr.md` already solved for inspection skills.

### Keep `actualize` as a standalone intent, keep `review` as a standalone intent

**Rejected.** Both are pure analysis skills over the same data source. The `--mode` flag (default / `--deep` / `--drift`) expresses the depth/topic distinction more honestly than three peer intents that all anti-trigger each other. Same argument that retired `status` and `graph` as peers of `review`.

### Keep `standard` as a router into the `decide` ADR + rule + guide chain

**Considered.** Would preserve the explicit "establish a standard" entry phrase. **Rejected** because `decide` already enumerates "Activate when user says 'we decided'" plus continuation prompts for rule and guide; one more anti-trigger surface to maintain wasn't worth the explicitness gain.

### Keep `verify` because it surfaces a useful action

**Rejected.** `verify` was always thin (forward to `make verify`). Plugin developers can run `make verify` directly. Keeping a palette entry for a one-line passthrough cost more cognitive load than it saved.

## Consequences

### Positive

- Visible `/` palette: **7 commands** (down from 16). Each skill maps to a clearly distinct user intent — no two skills anti-trigger each other.
- One source of truth per concern: `plan` for any forward-looking flow, `audit` for any health check, `decide` for any standards/decision cascade.
- Track flows are still reachable, but as references under `skills/plan/references/` rather than top-level skills. Adding a new flow is a new markdown file, not a new skill.
- Drift detection lives next to the other audit modes — easier to keep modes consistent.
- Session-start token budget shrinks: 7 skill descriptions instead of 16.
- Cross-host parity is maintained: every remaining skill is auto-invocable in Claude Code, Cursor, and Codex via the matching `commands/*.md` wrapper.

### Negative

- Users who memorized `/archcore:bootstrap`, `/archcore:review`, `/archcore:actualize`, `/archcore:standard`, `/archcore:verify`, or any `*-track` invocation need to learn the new mapping. `/archcore:help` documents the migration; the README explains the active surface.
- Track-flow discoverability shifts: users who would have typed `/archcore:iso-track` now type `/archcore:plan` and either describe the cascade or pass `--iso`. The plan-skill routing table makes this deterministic, but the action requires one extra step of phrasing for users who used to invoke the track directly.
- The `plan` skill grows: it now holds the routing logic for all four flows previously split across tracks. The references directory absorbs the per-flow content so the SKILL.md itself stays under the 200-line budget.

### Supersedes

- The Layer 2 "track skills" tier in `intent-based-skill-architecture.adr.md`. The intent-vs-utility classification remains; the per-track stratification is removed.
- The 16-command palette in `merge-review-status-remove-graph.adr.md`. The intent-merge pattern from that ADR is extended here to absorb `actualize` into `audit`.
- The mainstream/niche distinction in `inverted-invocation-policy.adr.md` was already retired by `remove-document-type-skills.adr.md`; this change additionally retires the auto-invocable track tier.
- The standalone `standard` and `verify` intents from `commands-system.spec.md` — both are removed.

### Constraints (new)

- The visible `/` palette MUST be exactly 7 commands: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`. Adding an eighth skill requires a new ADR.
- `audit` MUST support three modes: default short, `--deep`, `--drift`. The drift protocol lives at `skills/audit/lib/drift-detection.md`.
- `plan` MUST hold per-flow logic in `skills/plan/references/*.md` rather than spawning new top-level skills. Adding a flow is a new reference file.
- `decide` MUST own the standard/decision cascade: ADR → optional CPAT (for code-pattern changes) → optional rule → optional guide.
- No skill MAY have `disable-model-invocation: true` going forward — every remaining skill is intent-class and auto-invocable. If a utility need re-emerges, prefer a `make` target or CLI command over reintroducing a hidden skill.
