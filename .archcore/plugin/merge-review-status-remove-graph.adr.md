---
title: "Merge /archcore:status into /archcore:review and Remove /archcore:graph"
status: accepted
---

## Context

After `remove-document-type-skills.adr.md` collapsed the per-type skill layer (Phase 7), the visible `/archcore:` palette stabilized at 18 commands: 11 intent + 6 track + 1 utility. Three of those intents — `status`, `review`, and `graph` — all sit in the "inspect documentation health" problem space. Concrete observations after running with that surface:

**1. `status` and `review` overlap structurally.** Both call `mcp__archcore__list_documents` and `mcp__archcore__list_relations` and operate on the full project. `status` formats four counting tables (by category / status / type / relation) plus a one-line issues summary. `review` produces those same totals as its "Overview" section, then extends with coverage gaps, staleness, orphans, and prioritized recommendations. The split is depth, not topic — a flag (`--deep`) expresses the depth difference more honestly than two separate intents that the model has to pick between.

**2. The split forces the model to disambiguate adjacent intents.** Every intent skill carries explicit "Activate when X. Do NOT activate for Y (use /archcore:other)." guidance per the Inverted Invocation Policy. Three skills mutually disambiguating on the same topic produces brittle routing — "show docs status" could plausibly land on either `status` (counts) or `review` (analysis), and several sibling skills (`actualize`, `bootstrap`, `context`) carry anti-trigger lines pointing away from both, multiplying the surface that has to stay consistent on every edit.

**3. `graph` is unused in practice.** The `/archcore:graph` intent produces a Mermaid flowchart of the relation graph. In sessions across 2026-04 to 2026-05 we observed ~zero invocations: the dashboard (`status`) and the audit (`review`) carried the analytical load, the actual visualization step was redundant once relation health and orphan lists were already surfaced inside `review`. Mermaid output also doesn't render in every host (Codex/CLI cases), so the value-per-token of the skill is lower than its description-cost on every session start.

## Decision

**Merge `status` into `review` as the default short mode. Remove `graph` entirely. Result: 9 intent skills (down from 11) and 16 total visible commands (down from 18).**

Concrete changes:

1. **`/archcore:review` becomes the single inspection skill** with two modes:
   - **Default short mode** — no arguments. Outputs the four counting tables + one-line issues summary that `status` produced. Project-wide; does not take filters.
   - **Deep mode** — triggered by `--deep` or any non-flag argument (category/tag/type filter). Outputs the current `review` body: Overview, Gaps, Staleness, Orphans, Actions. Filters apply only in deep mode.

2. **Routing rule** for `/archcore:review`: any non-flag argument routes to deep mode. Empty invocation routes to short mode. The model picks the mode from user phrasing ("dashboard" / "how many docs" → short; "audit" / "documentation gaps" → deep) — both phrasings now resolve to a single skill, eliminating the cross-skill disambiguation noise.

3. **`skills/status/` and `skills/graph/` deleted on disk.**

4. **Sibling-skill anti-trigger lines updated** in `skills/actualize/`, `skills/bootstrap/`, `skills/context/` — they no longer reference `/archcore:status` or `/archcore:graph`. Where they pointed at `status` for "quick counts", they now point at `/archcore:review` (with a note that the default mode is the short dashboard).

5. **Count invariants updated**:
   - `README.md`: 18 → 16, 11 intent → 9 intent.
   - `test/structure/skills.bats`: `>= 18` → `>= 16`.
   - `.archcore/plugin/skills-system.spec.md`, `commands-system.spec.md`, `plugin-architecture.spec.md`, `component-registry.doc.md`, `claude-plugin.prd.md`, `plugin-component-architecture.adr.md`, `inverted-invocation-policy.adr.md`, `precision-over-coverage.adr.md`, `development-roadmap.plan.md` — totals + intent inventory + visible palette diagrams.

6. **Help skill** (`skills/help/SKILL.md`) — Quick Start table loses the `status` and `graph` rows; the `review` row now describes both modes.

## Alternatives Considered

### Keep three separate inspection intents

**Rejected.** The original "one skill per output shape" framing produced the duplication described in Context #1. Maintaining three skills required mirroring the totals/relation-counting code path between `status` and `review` (already present today) and propagating disambiguation logic across at least five sibling skills. The cost was paid every time any one of them was edited.

### Merge `status` + `review` but keep `graph`

**Considered.** Graph is the most divergent of the three (it produces a Mermaid block, not a narrative or a counts table). However, observed near-zero invocation made it dead weight in the palette. The orphan list and relation-by-type counts that `graph` carried as a footer are already inside `review` (orphans section + relation counts in the dashboard). For users who actually want the Mermaid diagram, MCP `list_documents` + `list_relations` plus an ad-hoc Mermaid request remains a one-prompt path; keeping a skill for it is overhead for a feature few use.

### Keep `graph` but hide it via `disable-model-invocation: true`

**Rejected.** Same multi-host portability problem that motivated `remove-document-type-skills.adr.md` — `disable-model-invocation` works in Claude Code but is inconsistent in Cursor/Codex. Hiding the skill in only one host while it stays visible in others reintroduces cross-host divergence.

### Make `--deep` explicit-only (don't auto-route filters to deep)

**Rejected.** The dashboard (short mode) is project-wide by design and doesn't take a filter. If the user passes a filter, they want analysis of that scope, not "the dashboard but filtered." Routing any non-flag argument to deep mode keeps the surface intuitive and removes one error case ("filter passed but ignored").

## Consequences

### Positive

- Visible `/` palette: **16 commands** (9 intent + 6 track + 1 utility). Two fewer skills to load on every session start.
- One source of truth for "documentation health" — no more cross-skill disambiguation between `status` and `review`. The `--deep` flag expresses depth honestly.
- Sibling skills (`actualize`, `bootstrap`, `context`) carry shorter anti-trigger lines.
- The Mermaid-output skill that did not work uniformly across hosts is gone — one less host-fragility surface.

### Negative

- Users who memorized `/archcore:status` need to learn the new short-mode invocation of `/archcore:review`. The change is small (the skill description now leads with "show status / how many docs" triggers) and `/archcore:help` documents the merge.
- Users who want a Mermaid diagram of the graph lose the dedicated path. Workaround: ask the model to render Mermaid from `list_relations` output. Lost ergonomics, but observed usage was ~zero.
- The `review` SKILL.md gains a routing table for two modes — slightly larger than either of the two it replaced individually, but smaller than their sum.

### Supersedes

- Three rows of `inverted-invocation-policy.adr.md` (the `status`, `review`, `graph` intent rows in the post-supersession matrix). The intent/track/utility class rules from that ADR remain in force.
- Phase 7 acceptance criteria in `development-roadmap.plan.md` referencing 11 intent skills and 18 commands. Updated counts are recorded in a new Phase 8 entry.

### Constraints (new)

- `/archcore:review` MUST default to short mode when invoked without arguments.
- `/archcore:review --deep` and `/archcore:review <filter>` MUST route to the full audit.
- Short mode MUST NOT take a filter (the dashboard is project-wide by design).
- No new inspection-flavored intent SHOULD be added without first checking whether it can be a mode of `/archcore:review`.
