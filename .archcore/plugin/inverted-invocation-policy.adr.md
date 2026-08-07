---
title: "Inverted Invocation Policy — Intent Auto-Invoked, Mainstream Types Expert-Only, Niche Types Hidden"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

**Surface note.** The matrices below describe intermediate states. The current palette is four auto-invocable commands (`init`, `plan`, `document`, `review`), fixed by `four-command-palette.adr`, with a non-palette gated track layer per `track-layer.spec`, which retired the track tier and the `verify` utility. What survives from this decision is its core rule: a skill that routes user intent is auto-invocable, and its description carries explicit trigger and anti-trigger phrasing as the routing signal.

## Context

`intent-based-skill-architecture.adr` established four layers but set `disable-model-invocation: true` on the intent and track skills, making them user-only, while leaving all 18 document-type skills model-invocable. That configuration inverted the architecture's own routing intent: when a user said "record the decision to use PostgreSQL", the model auto-invoked the type skill directly, so the intent layer never ran and its duplicate check, relation suggestion, rule and guide follow-up, and contextual disambiguation were all bypassed. The intent layer was architecturally clean and operationally dead — a user had to type the intent command explicitly to benefit from it, and very few did.

Two further facts became actionable after that original record. Claude Code's `SKILL.md` frontmatter began exposing `user-invocable: false`, a flag that hides a skill from the `/` menu while keeping its description in the model's context, unlocking a configuration that had not been possible before. And cognitive-load analysis showed that 7 of the 18 document-type skills — `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs` — are niche, required for specific discovery and ISO 29148 workflows but irrelevant to more than 90% of users, while occupying prominent slots in `/` autocomplete.

## Decision

Invert the invocation policy across the skill catalog: a skill that routes user intent is auto-invocable and carries no invocation-restricting flag, while a maintenance-only utility stays user-only through `disable-model-invocation: true`.

| Layer | Skills | `disable-model-invocation` | `user-invocable` | In `/` menu | Model auto-invokes |
| --- | --- | --- | --- | --- | --- |
| Intent | bootstrap, capture, plan, decide, standard, review, actualize, help, context | — (removed) | default (`true`) | ✓ | ✓ |
| Track | product-track, architecture-track, standard-track, feature-track, sources-track, iso-track | — (removed) | default (`true`) | ✓ | ✓ |
| Utility | verify | `true` (unchanged) | default (`true`) | ✓ | ✗ |

The type-skill rows of the original matrix — mainstream types carrying `disable-model-invocation: true` and niche types carrying `user-invocable: false` — are historical. `remove-document-type-skills.adr` deleted those skills from disk and moved their per-type elicitation inline into the intent and track skills, citing content duplication, multi-host flag inconsistency, and cognitive load. `merge-review-status-remove-graph.adr` then merged `status` into `review` and removed `graph`, and is the source of truth for the intent inventory of that period.

Intent and track skills are auto-invocable so that the model routes user intent through them, with their descriptions carrying explicit `Activate when X. Do NOT activate for Y.` guidance as the routing signal. The `verify` utility stays user-only, because it is a maintenance skill for plugin developers rather than for end users and should not auto-activate. The post-merge visible menu was 9 intent, 6 track, and 1 utility, totaling 16 commands, with no hidden surface.

## Alternatives Considered

1. **Keep the status-quo user-only intent and track policy** — rejected because the intent layer is the plugin's primary UX promise, that a user describes what they need and the system picks the type, and it was being bypassed operationally; keeping the old policy would require users to memorize intent commands, which negates the promise.
2. **Remove type skills entirely and route everything through intent** — originally rejected as a loss of the productive path for power users, and later adopted by `remove-document-type-skills.adr` once evidence showed that type-skill content was already duplicated inline, that the invocation flags were not portable across Cursor and Codex, and that every document type stayed reachable through an intent or track skill or a direct MCP call.
3. **Make niche types both user-hidden and model-hidden with `disable-model-invocation: true`** — rejected at the time, because a model that could not see the `brs`, `strs`, `syrs`, and `srs` descriptions left `iso-track` no programmatic way to invoke them. The concern became moot once type skills were removed and tracks inlined the per-type elicitation directly.
4. **Split the niche types into a separate sub-plugin** — deferred, then superseded by the removal of type skills, since niche types became reachable as track steps and directly through MCP.

## Consequences

- Intent routing became load-bearing: duplicate checks, relation suggestions, and multi-document follow-up execute for auto-invoked flows rather than only for explicit `/` invocations.
- The visible menu went from 32 entries to 25 at the inversion, to 26 after `graph` was added, to 18 after type skills were removed, and to 16 after `status` merged into `review` and `graph` was removed.
- The model's initial context stopped carrying per-type-skill descriptions, saving tokens on every session start and leaving more budget for precise intent descriptions.
- Cross-host parity improved: the intent, track, and utility policy relies only on the absence of a flag or on `disable-model-invocation: true`, both of which behave consistently in Claude Code, and the more brittle `user-invocable: false` field — unsupported in Cursor and Codex — is no longer relied on.
- Tradeoff: this supersedes principle 4, "user-only invocation", of `intent-based-skill-architecture.adr`, and that record's four-layer decomposition was reduced to three effective layers plus MCP primitives once type skills were removed.
- Tradeoff: intent and track skill descriptions became the single source of routing truth, so an imprecise description causes mis-routing. The mandated `Activate when X. Do NOT activate for Y.` format is the mitigation.

## Constraints

1. An intent skill description MUST enumerate its trigger phrases.
2. An intent skill description MUST enumerate its anti-triggers, naming the skill to use instead.
3. A utility skill MUST carry `disable-model-invocation: true`.
4. A track skill MUST remain auto-invocable, so a user reaches a multi-document flow through natural language.
5. Command descriptions and CLI help MUST document direct-MCP access (help removed under v2).

Three constraints from the original record no longer apply, because type skills no longer exist: that a mainstream type skill carry `disable-model-invocation: true`, that a niche type skill carry `user-invocable: false`, and that a track orchestrating niche types remain auto-invocable to reach them. Constraint 4 survives in its own right, and `skill-surface-collapse.adr` later retired the track tier entirely.

## Superseded when

- A host ships a portable, standardized invocation-control field that every supported host honors, which would reopen the hidden-surface options rejected here.
- Routing measurement shows the trigger and anti-trigger format failing to disambiguate adjacent intents, which would call for a different routing signal.