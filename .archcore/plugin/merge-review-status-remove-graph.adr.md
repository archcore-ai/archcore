---
title: "Merge /archcore:status into /archcore:review and Remove /archcore:graph"
status: accepted
tags:
  - "architecture"
  - "commands"
  - "plugin"
  - "skills"
---

**Naming note.** This decision merged `status` into a skill then called `/archcore:review`. `skill-surface-collapse.adr` later renamed that skill to `/archcore:audit` and removed the track tier, so the counts recorded here — 16 visible commands, 9 intent skills — describe the surface at the time of this decision, not the current four-command palette. The merge itself, and the short-versus-deep mode split, survived the rename intact and now live as the default dashboard and `--deep` modes of `/archcore:review` — `audit` itself was later reabsorbed into `review` by `four-command-palette.adr`.

## Context

After `remove-document-type-skills.adr` collapsed the per-type skill layer, the visible `/archcore:` palette stabilized at 18 commands — 11 intent, 6 track, 1 utility — of which three intents (`status`, `review`, `graph`) all sat in the "inspect documentation health" problem space. Running with that surface produced three concrete observations: the two analysis skills overlapped structurally, the split forced the model to disambiguate adjacent intents on every session, and the third skill went effectively unused.

## Observations

1. **`status` and `review` overlap structurally.** Both call `list_documents` and `list_relations` and operate on the full project. `status` formats four counting tables — by category, status, type, and relation — plus a one-line issues summary; `review` produces those same totals as its Overview section and then extends with coverage gaps, staleness, orphans, and prioritized recommendations. The split is depth rather than topic, and a `--deep` flag expresses depth more honestly than two intents the model must choose between.
2. **The split forced cross-skill disambiguation.** Every intent skill carries explicit `Activate when X. Do NOT activate for Y.` guidance, and three skills mutually disambiguating on one topic produces brittle routing: "show docs status" could plausibly land on either. Several sibling skills — `actualize`, `bootstrap`, `context` — carried anti-trigger lines pointing away from both, multiplying the surface that had to stay consistent on every edit.
3. **`graph` was unused in practice.** The skill produced a Mermaid flowchart of the relation graph, and sessions from 2026-04 through 2026-05 showed approximately zero invocations: the dashboard and the audit carried the analytical load, and the visualization step was redundant once relation health and orphan lists already appeared inside the audit. Mermaid output also does not render in every host, including Codex CLI cases, so the value per token was lower than the skill's description cost on every session start.

## Decision

Merge `status` into the inspection skill as its default short mode, and remove `graph` entirely — taking the palette from 18 commands and 11 intent skills to 16 and 9.

The inspection skill gains two modes. Its **default short mode**, invoked with no arguments, outputs the four counting tables and the one-line issues summary that `status` produced, project-wide and without filters. Its **deep mode**, triggered by `--deep` or by any non-flag argument used as a category, tag, or type filter, outputs the full audit body: Overview, Gaps, Staleness, Orphans, Actions. The routing rule is that any non-flag argument routes to deep mode and an empty invocation routes to short mode, so both user phrasings — "dashboard" or "how many docs" for short, "audit" or "documentation gaps" for deep — resolve inside one skill.

`skills/status/` and `skills/graph/` were deleted from disk; sibling anti-trigger lines in `skills/actualize/`, `skills/bootstrap/`, and `skills/context/` were repointed at the merged skill with a note that its default mode is the short dashboard; the help skill's Quick Start table lost the two rows and gained a two-mode description; and the count invariants were updated across `README.md`, `@test/structure/skills.bats`, and the affected `.archcore/` documents.

## Alternatives Considered

1. **Keep three separate inspection intents** — rejected because the "one skill per output shape" framing produced the duplication in observation 1: maintaining three skills required mirroring the totals and relation-counting path between two of them and propagating disambiguation logic across at least five siblings, a cost paid on every edit to any one of them.
2. **Merge `status` and `review` but keep `graph`** — considered seriously, because `graph` is the most divergent of the three, producing a Mermaid block rather than a narrative or a counts table. Rejected because observed near-zero invocation made it dead weight, and because the orphan list and relation-by-type counts it carried as a footer already appear inside the audit. A user who wants the diagram still has a one-prompt path through `list_documents` plus `list_relations` and an ad-hoc Mermaid request.
3. **Keep `graph` but hide it behind `disable-model-invocation: true`** — rejected because of the multi-host portability problem that motivated `remove-document-type-skills.adr`: the flag works in Claude Code but behaves inconsistently in Cursor and Codex, so hiding the skill on one host while it stays visible on others reintroduces cross-host divergence.
4. **Make `--deep` explicit-only, so a filter does not auto-route to deep mode** — rejected because the dashboard is project-wide by design and takes no filter, so a user passing a filter wants analysis of that scope rather than a filtered dashboard. Auto-routing removes the "filter passed but ignored" error case.

## Consequences

- Two fewer skills load on every session start.
- One source of truth for documentation health, with no cross-skill disambiguation and a flag that expresses depth honestly.
- Sibling skills carry shorter anti-trigger lines.
- The Mermaid-output skill that did not work uniformly across hosts is gone, removing one host-fragility surface.
- Tradeoff: users who memorized `/archcore:status` had to learn the short-mode invocation of the merged skill. The skill description leads with the "show status" and "how many docs" triggers, and the help skill documents the merge.
- Tradeoff: users who want a Mermaid diagram of the graph lose the dedicated path and fall back to asking the model to render Mermaid from `list_relations` output. Observed usage was approximately zero.
- Tradeoff: the merged `SKILL.md` gains a two-mode routing table, making it larger than either predecessor individually though smaller than their sum.
- This decision supersedes the `status`, `review`, and `graph` intent rows of `inverted-invocation-policy.adr`, while that record's intent, track, and utility class rules remain in force, and it supersedes the Phase 7 acceptance criteria in `development-roadmap.plan` that referenced 11 intent skills and 18 commands.

## Constraints

1. The inspection skill MUST default to short mode when invoked with no argument.
2. The inspection skill MUST route `--deep` to the full audit.
3. The inspection skill MUST route a non-flag filter argument to the full audit.
4. Short mode MUST NOT accept a filter, because the dashboard is project-wide by design.
5. A new inspection-flavored intent SHOULD NOT be added before checking whether it can be a mode of the existing inspection skill.

## Superseded when

- A measured session sample shows users invoking deep mode with a filter more often than the project-wide dashboard, which would argue for reversing the default.
- Mermaid rendering becomes uniform across all supported hosts, which would reopen the case for a dedicated graph surface.