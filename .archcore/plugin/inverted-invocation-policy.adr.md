---
title: "Inverted Invocation Policy — Intent Auto-Invoked, Mainstream Types Expert-Only, Niche Types Hidden"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Context

The intent-based skill architecture (`intent-based-skill-architecture.adr.md`) established four layers but configured `disable-model-invocation: true` on intent and track skills — making them user-only — while leaving all 18 document-type skills model-invocable.

In practice this inverted the routing intent of the architecture:

- When a user said "record the decision to use PostgreSQL", Claude auto-invoked `/archcore:adr` directly because the type skill was model-invocable. The intent layer (`/archcore:decide`) never ran.
- The duplicate check (`list_documents` before `create_document`), relation-suggestion, rule+guide follow-up, and contextual disambiguation — all built into `decide`, `capture`, `plan`, `standard` — were bypassed.
- The intent layer was architecturally clean but operationally dead. Users had to explicitly type `/archcore:decide` to benefit from it; very few did.

Two further facts became actionable since the original ADR:

1. Claude Code's SKILL.md frontmatter now exposes `user-invocable: false` — a flag that hides a skill from the `/` menu while keeping its description in the model's context. This unlocks a configuration that was not possible before.
2. Cognitive-load analysis showed that 7 of the 18 document-type skills (`mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs`) are niche — required for specific discovery and ISO 29148 workflows, but irrelevant to 90%+ of users. They occupied prominent slots in `/` autocomplete despite rarely being useful directly.

## Decision

Invert the invocation policy across the skill catalog.

### New matrix

| Layer             | Skills                                                                                     | `disable-model-invocation` | `user-invocable`    | In `/` menu | Model auto-invokes                |
| ----------------- | ------------------------------------------------------------------------------------------ | -------------------------- | ------------------- | ----------- | --------------------------------- |
| Intent            | capture, plan, decide, standard, review, status, actualize, graph, help                    | — (removed)                | default (`true`)    | ✓           | ✓                                 |
| Track             | product-track, architecture-track, standard-track, feature-track, sources-track, iso-track | — (removed)                | default (`true`)    | ✓           | ✓                                 |
| Type — mainstream | adr, prd, rfc, rule, guide, doc, spec, idea, task-type, cpat                               | **`true`**                 | default (`true`)    | ✓           | ✗                                 |
| Type — niche      | mrd, brd, urd, brs, strs, syrs, srs                                                        | — (default)                | **`false`**         | ✗           | ✓ (typically via track orchestration) |
| Utility           | verify                                                                                     | `true` (unchanged)         | default (`true`)    | ✓           | ✗                                 |

Note: `graph` was added to the intent layer after the initial inversion. It follows the same auto-invocable contract (no flags), bringing intent count to 9 and visible `/` menu to 26.

### Rationale per class

- **Intent and track skills become auto-invocable** so the model routes user intent through them. Their descriptions carry explicit "Activate when X. Do NOT activate for Y (use /archcore:other)." guidance as the routing signal.
- **Mainstream type skills become expert-only** via `disable-model-invocation: true`. This removes their descriptions from the model's initial context (token savings) and forces all auto-invocation through the intent layer. Users who know exactly what they want can still `/archcore:adr <topic>`.
- **Niche type skills become hidden** via `user-invocable: false`. The model still sees their descriptions (needed for track orchestration — `iso-track` internally invokes `brs/strs/syrs/srs`, `sources-track` invokes `mrd/brd/urd`), but users do not see them in `/` autocomplete. The visible `/` menu is now 26 commands (9 intent + 6 track + 10 mainstream type + 1 utility).
- **Utility (`verify`) stays user-only** — it is a maintenance skill for plugin developers, not for end users, and should not auto-activate.

## Alternatives Considered

### Keep the status-quo user-only intent/track policy

Rejected. The intent layer is the primary UX promise of the plugin ("describe what you need, the system picks the type"), and it was operationally bypassed. Keeping the old policy would require users to memorize intent commands — negating the promise.

### Remove type skills entirely, route everything through intent

Rejected. Power users value `/archcore:adr <topic>` as a single-purpose shortcut. Removing it would lose a productive path without a UX gain, since expert users already know the type they want.

### Make niche types user-hidden AND model-hidden (`disable-model-invocation: true`)

Rejected. If the model cannot see `brs/strs/syrs/srs` descriptions, `iso-track` has no programmatic way to invoke them. Track skills would have to embed full ISO cascade logic inline, ballooning their size and coupling invocation logic to content.

### Split niche types into a separate sub-plugin

Deferred. A future `archcore-iso` sub-plugin is a valid option, but it requires marketplace fragmentation (two installs instead of one) and cross-plugin orchestration. `user-invocable: false` achieves the cognitive-load goal without that complexity.

## Consequences

### Positive

- Intent routing is load-bearing — duplicate checks, relation suggestions, and multi-document follow-up execute for auto-invoked flows, not just explicit `/` invocations.
- Visible `/` menu went from 32 to 25 commands at the time of the inversion, then to 26 after the `graph` intent skill was added — concentrated on what newcomers should see first.
- Model's initial context no longer carries 10 mainstream type-skill descriptions — token savings on every session start and more budget for intent descriptions to be precise.
- Cognitive load is stratified: newcomers see 9 intent + 6 tracks + 10 mainstream types + 1 utility = 26; ISO/discovery specialists reach niche types via tracks or direct MCP tools.

### Negative

- Supersedes principle 4 ("User-only invocation") of `intent-based-skill-architecture.adr.md`. That principle is explicitly reversed here; the 4-layer structural decomposition stands, only the invocation wiring flips.
- Intent skill descriptions become the single source of routing truth. Imprecise descriptions lead to mis-routing. Mitigated by the description-rewrite enforcing the "Activate when X. Do NOT activate for Y." format.
- Niche types are harder to discover directly. `/archcore:help` must document the track-based access path, and the niche types' `SKILL.md` still exist so tracks can invoke them.

### Constraints

- Intent and track skill descriptions MUST explicitly enumerate trigger phrases and anti-triggers (use /archcore:other references).
- Mainstream type skills (adr, prd, rfc, rule, guide, doc, spec, idea, task-type, cpat) MUST carry `disable-model-invocation: true`.
- Niche type skills (mrd, brd, urd, brs, strs, syrs, srs) MUST carry `user-invocable: false`.
- Tracks that orchestrate niche types (`sources-track`, `iso-track`) MUST remain auto-invocable so users can reach niche types via natural-language requests.
- `/archcore:help` MUST document the niche-type access path (tracks or MCP tools).
