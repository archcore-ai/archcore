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

### New matrix (current — post-supersession)

| Layer     | Skills                                                                            | `disable-model-invocation` | `user-invocable` | In `/` menu | Model auto-invokes |
| --------- | --------------------------------------------------------------------------------- | -------------------------- | ---------------- | ----------- | ------------------ |
| Intent    | bootstrap, capture, plan, decide, standard, review, actualize, help, context      | — (removed)                | default (`true`) | ✓           | ✓                  |
| Track     | product-track, architecture-track, standard-track, feature-track, sources-track, iso-track | — (removed)                | default (`true`) | ✓           | ✓                  |
| Utility   | verify                                                                            | `true` (unchanged)         | default (`true`) | ✓           | ✗                  |

**Historical matrix (type-skill rows, now superseded by `remove-document-type-skills.adr.md`):**

| Layer             | Skills                                                       | `disable-model-invocation` | `user-invocable` |
| ----------------- | ------------------------------------------------------------ | -------------------------- | ---------------- |
| Type — mainstream | adr, prd, rfc, rule, guide, doc, spec, idea, task-type, cpat | **`true`**                 | default          |
| Type — niche      | mrd, brd, urd, brs, strs, syrs, srs                          | — (default)                | **`false`**      |

Type skills no longer exist on disk. Their per-type elicitation moved inline into intent and track skills. See `remove-document-type-skills.adr.md` for the removal rationale (content duplication with intents/tracks; multi-host flag inconsistency; cognitive-load reduction).

The `status` and `graph` intents were merged into `review` and removed respectively per `merge-review-status-remove-graph.adr.md` — that ADR is the source of truth for the current intent inventory.

### Rationale per remaining class

- **Intent and track skills are auto-invocable** so the model routes user intent through them. Their descriptions carry explicit "Activate when X. Do NOT activate for Y (use /archcore:other)." guidance as the routing signal.
- **Utility (`verify`) stays user-only** — it is a maintenance skill for plugin developers, not for end users, and should not auto-activate.

Post-merge visible `/` menu: 9 intent + 6 track + 1 utility = **16 commands**. No hidden surface.

## Alternatives Considered

### Keep the status-quo user-only intent/track policy

Rejected. The intent layer is the primary UX promise of the plugin ("describe what you need, the system picks the type"), and it was operationally bypassed. Keeping the old policy would require users to memorize intent commands — negating the promise.

### Remove type skills entirely, route everything through intent (original alternative — now adopted)

Originally rejected as "loss of productive path for power users". **This alternative was later adopted** via `remove-document-type-skills.adr.md` after evidence showed that (a) type-skill content was already duplicated inline in intent/track skills, (b) the `disable-model-invocation` / `user-invocable` flags were not portable across Cursor and Codex, and (c) every document type remained reachable through intent/track skills or direct MCP calls.

### Make niche types user-hidden AND model-hidden (`disable-model-invocation: true`)

Rejected at the time. If the model could not see `brs/strs/syrs/srs` descriptions, `iso-track` had no programmatic way to invoke them. This concern became moot when type skills were removed entirely — tracks now inline per-type elicitation directly.

### Split niche types into a separate sub-plugin

Deferred. Superseded by the decision to remove type skills entirely — niche types are reachable via `/archcore:iso-track` and `/archcore:sources-track`, and directly via MCP.

## Consequences

### Positive

- Intent routing is load-bearing — duplicate checks, relation suggestions, and multi-document follow-up execute for auto-invoked flows, not just explicit `/` invocations.
- Visible `/` menu went from 32 to 25 at the time of the inversion, to 26 after the `graph` intent was added, to **18** after type skills were removed (`remove-document-type-skills.adr.md`), and to **16** after `status` was merged into `review` and `graph` was removed (`merge-review-status-remove-graph.adr.md`).
- Model's initial context no longer carries per-type-skill descriptions — token savings on every session start and more budget for intent descriptions to be precise.
- Cross-host parity: the intent/track/utility policy uses only "no flag" / `disable-model-invocation: true`, both of which work consistently in Claude Code. The more brittle `user-invocable: false` field (not supported in Cursor/Codex) is no longer relied upon because type skills have been removed.

### Negative

- Supersedes principle 4 ("User-only invocation") of `intent-based-skill-architecture.adr.md`. The 4-layer structural decomposition from that ADR has also been reduced to 3 effective layers (intent, track, utility) + MCP primitives after type skills were removed.
- Intent and track skill descriptions become the single source of routing truth. Imprecise descriptions lead to mis-routing. Mitigated by the description-rewrite enforcing the "Activate when X. Do NOT activate for Y." format.

### Constraints

- Intent and track skill descriptions MUST explicitly enumerate trigger phrases and anti-triggers (use /archcore:other references).
- Utility skills MUST carry `disable-model-invocation: true`.
- Tracks MUST remain auto-invocable so users can reach multi-document flows via natural-language requests.
- `/archcore:help` MUST document direct-MCP access for any document type (since there is no type-skill surface).

**Superseded constraints** (no longer apply — see `remove-document-type-skills.adr.md`):

- ~~Mainstream type skills MUST carry `disable-model-invocation: true`.~~ Type skills no longer exist.
- ~~Niche type skills MUST carry `user-invocable: false`.~~ Type skills no longer exist.
- ~~Tracks that orchestrate niche types MUST remain auto-invocable so users can reach niche types via natural-language requests.~~ Tracks now inline per-type elicitation; niche types are reached as track steps, not via orchestration of separate skills.
