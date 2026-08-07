---
title: "Intent-Based Skill Architecture with 4-Layer Command Hierarchy"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Context

The plugin exposed **27 skills** in a flat namespace — 18 document-type skills, 6 track skills, and 3 workflow skills — and Claude Code surfaces every registered skill in one flat list, so a user saw 27 `archcore:*` entries with no way to know where to start. The naming reflected internal system structure — document types, ISO standards, tracks — rather than user intent: nobody thinks "I need a `strs`", they think "I need to formalize stakeholder requirements". `keep-document-type-skills.adr` had established the type skills as the domain-knowledge layer over the MCP primitives and `scenario-track-skills.idea` had added the track skills for engineering workflows; both were sound, and neither addressed the user-facing entry point.

The archcore MCP server already supplied 8 atomic CRUD tools with roughly 20 lines of type guidance in its server instructions, so the agent could pick a type and create a document unaided. What MCP did not supply was intent routing, multi-document orchestration, elicitation strategy, or progressive UX — leaving no layer to translate "plan this feature", "document this module", or "record this decision" into the right types, tracks, and relation chains. A user had to either know which of 18 types fit the situation or use a creation wizard that still demanded a type. The product spoke in infrastructure terms rather than user terms.

## Decision

Add a four-layer command hierarchy with **intent-based skills as the primary user-facing entry point**: 7 intent skills at Layer 1, the 6 track skills as advanced Layer 2, the 18 document-type skills as expert Layer 3, and the 8 MCP tools as unchanged Layer 4 infrastructure.

| Layer 1 skill | User intent | Routes to |
|---|---|---|
| `/archcore:capture` | "document this module or component" | adr, spec, doc, or guide, by context |
| `/archcore:plan` | "plan this feature or initiative" | product-track (idea → prd → plan) or a single plan |
| `/archcore:decide` | "record this decision" | adr, with an optional rule and guide follow-up |
| `/archcore:standard` | "make this a team standard" | standard-track (adr → rule → guide) |
| `/archcore:review` | "check documentation health" | the existing review logic |
| `/archcore:status` | "show dashboard" | the existing status logic |
| `/archcore:help` | "what can I do?" | layer navigation and onboarding |

Layer 2 and Layer 3 carry a tier signal in their `description` prefix — "Advanced —" and "Expert —" respectively — and Layer 4 stays non-user-facing, used by skills, agents, and the model directly.

**Five design principles govern an intent skill.** It carries an explicit routing table — a bounded decision tree mapping user input to a type or a track, with no open-ended "reason about what fits" instruction. It elicits minimally, asking one scope-confirmation question at the intent level and then content questions per document during creation. It is self-contained, because the host activates one skill at a time, so it inlines its creation recipes rather than delegating to a type skill at runtime. It is user-only, with `disable-model-invocation: true`, because auto-activating an orchestration flow from ambient context risks false positives. And it defaults to the minimum viable path, unlocking a larger flow only through a binary scope question, never defaulting to the largest.

**Naming.** `capture` rather than `document`, because "document" is ambiguous as verb and noun while "capture" is an unambiguous verb matching how users describe the action. `standard` rather than `standardize`, because brevity matters in a chat interface and the noun form is conventional, as with `git commit` and `git branch`. `plan` absorbs the existing `plan` type skill, resolving the name collision by making the intent skill the primary entry point.

**Progressive disclosure** happens inside a command through conversation rather than through command selection, because the host namespace is flat: a Layer 1 skill detects scope from its arguments, asks one clarifying question, and invokes the track or type logic internally without redirecting the user to another command, with `/archcore:help` as the navigation layer between tiers.

## Alternatives Considered

1. **Keep the flat 27-command structure and fix discoverability through better descriptions** — rejected because 27 entries in a flat list overwhelm regardless of description quality; the cognitive load is in the count rather than the labels.
2. **Remove the type and track skills entirely and keep only intent skills** — rejected at the time because it would lose the domain-knowledge layer that `keep-document-type-skills.adr` established, along with the model-invocation path where the host auto-activates a type skill from user phrasing, and the expert path that track skills serve. This alternative was later partially adopted by `remove-document-type-skills.adr` and fully adopted by `skill-surface-collapse.adr`; see the addendum.
3. **Add layer-level skills such as `/archcore:vision` and `/archcore:knowledge`** — rejected because the layers are internal classification rather than a user mental model; nobody thinks "I need a knowledge document", and type selection would still be required inside the layer, since knowledge alone holds 6 types.
4. **Move intent routing into MCP as a `suggest_type` tool** — rejected because type suggestion is classification over natural language and belongs in the prompt layer; such a tool would make the server stateful, add round-trips, and couple domain knowledge into a primitive CRUD server. Intent routing is prompt work rather than data work.

## Consequences

- Users see 7 intent-based entry points instead of 27 infrastructure-named skills, and those entry points match user mental models rather than the system taxonomy.
- The full power of the 18 types and 6 tracks stays available to advanced and expert users.
- The MCP layer stays stable, with no change to the 8 primitive tools, and the type skills continue to serve the model-invocation case.
- Each layer has a distinct role, so the architecture is layered without duplication.
- Tradeoff: an intent skill must be self-contained with inline creation recipes, which adds maintenance surface.
- Tradeoff: the `/archcore:plan` name collision requires absorbing the existing `plan` type skill.
- Tradeoff: Layer 2 and Layer 3 skills are less discoverable in a flat picker, mitigated by the description prefixes and by `/archcore:help`.
- Tradeoff: the intent skill structure of 5 sections differs from the type skill structure of 7, leaving two patterns to maintain.

## Constraints

1. An intent skill MUST carry `disable-model-invocation: true`.
2. An intent skill MUST contain an explicit routing table rather than an open-ended reasoning instruction.
3. An intent skill MUST default to the minimum viable path.
4. An intent skill MUST offer expansion through a scope question.
5. An intent skill MUST be self-contained, with an inline creation recipe per document type it produces.
6. The existing `plan` type skill MUST be absorbed into the `/archcore:plan` intent skill.
7. A Layer 2 or Layer 3 description MUST carry its tier prefix.

## Addendum — post-decision evolution

**Inverted invocation policy.** Principle 4 and constraint 1 — user-only invocation for intent skills — were reversed by `inverted-invocation-policy.adr`. Intent and track skills became auto-invocable with no flag. The four-layer structural decomposition stood; only the invocation wiring flipped.

**Layer 3 collapse.** `remove-document-type-skills.adr` removed the type-skill layer, inlining its per-type elicitation into the intent and track skills. The runtime layering became intent skills, track skills, the `verify` utility, and the MCP primitives — the last now acting as the universal document-creation primitive for every Archcore type with no type-skill mediation.

**Inspection consolidation.** `merge-review-status-remove-graph.adr` merged `status` into the default short mode of `review` and removed `graph`, taking the intent count from 11 to 9 and the visible total from 18 to 16.

**Final collapse to a single tier.** `skill-surface-collapse.adr` flattened the rest: all 6 track skills were deleted, with their flow content moved to `skills/plan/references/<flow>.md` and `skills/decide/references/continuations.md`; `actualize` became `/archcore:audit --drift`; `bootstrap` was renamed `init`; `standard` was removed, since its cascade is reachable through `decide`; and `verify` was removed, with `make verify` as the canonical integrity check.

**Naming reversal.** `four-command-palette.adr` explicitly reverses the capture-over-document naming choice recorded above: `capture` is retired, and `document` becomes the primary verb, absorbing `decide` as well.

The current surface is four commands: `init`, `plan`, `document`, `review`, per `four-command-palette.adr` (capture and decide absorbed by document, audit by review; context and help removed). The intent-based framing of Layer 1 is fully in force, and the tiered structure is gone. The naming decisions, design principles 1, 2, 3, and 5, and the progressive-disclosure mechanism all remain valid in the final form; only the layered packaging was retired.

## Superseded when

- A fifth genuinely distinct user intent appears that no existing command can absorb as a mode or track, which `four-command-palette.adr` requires a new ADR to admit.
- A host stops surfacing skills in a flat namespace and offers real hierarchical grouping, which would reopen the tiered packaging this decision introduced and its successors retired.
