---
title: "Intent-Based Skill Architecture with 4-Layer Command Hierarchy"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Context

The Archcore Claude Plugin currently exposes **27 skills** in a flat namespace:
- 18 document-type skills (`/archcore:adr`, `/archcore:strs`, `/archcore:cpat`, etc.)
- 6 track skills (`/archcore:product-track`, `/archcore:iso-track`, etc.)
- 3 workflow skills (`/archcore:create`, `/archcore:review`, `/archcore:status`)

Claude Code surfaces all registered skills in a flat list. Users see 27 `archcore:*` entries and don't know where to start. The naming reflects **internal system structure** (document types, ISO standards, tracks) rather than **user intent**. Nobody thinks "I need a strs" — they think "I need to formalize stakeholder requirements."

The existing `keep-document-type-skills.adr.md` established that type skills should remain as the domain knowledge layer over MCP primitives. The `scenario-track-skills.idea.md` added track skills for engineering workflows. Both decisions were correct — but neither addressed the user-facing entry point problem.

### What MCP already provides

The archcore MCP server gives 8 atomic CRUD tools with type guidance in its server instructions (~20 lines). The agent can already pick types and create documents independently. MCP handles the data layer. But MCP does NOT provide: intent routing, multi-document orchestration, elicitation strategy, or progressive UX.

### The gap

There is no layer that translates user intent ("plan this feature", "document this module", "record this decision") into the correct document types, tracks, and relation chains. Users must either:
1. Know which of 18 types fits their situation, or
2. Use the `/archcore:create` wizard which still requires type selection

This is the core UX problem: the product speaks in infrastructure terms, not user terms.

## Decision

**Add a 4-layer command hierarchy with intent-based skills as the primary user-facing entry point.**

### Layer 1 — Intent API (PRIMARY, 7 registered skills)

| Skill | User intent | Routes to |
|---|---|---|
| `/archcore:capture` | "document this module/component" | Routes to adr, spec, doc, guide based on context |
| `/archcore:plan` | "plan this feature/initiative" | Routes to product-track (idea→prd→plan) or single plan |
| `/archcore:decide` | "record this decision" | Creates adr, offers rule+guide follow-up |
| `/archcore:standard` | "make this a team standard" | Routes to standard-track (adr→rule→guide) |
| `/archcore:review` | "check documentation health" | Existing review logic |
| `/archcore:status` | "show dashboard" | Existing status logic |
| `/archcore:help` | "what can I do?" | Layer navigation guide, onboarding |

### Layer 2 — Domain Flows (ADVANCED, registered with "Advanced" prefix in description)

6 track skills: product-track, sources-track, iso-track, architecture-track, standard-track, feature-track. For users who know which multi-document flow they need.

### Layer 3 — Typed Artifacts (EXPERT, registered with "Expert" prefix in description)

18 document-type skills. For power users who know exactly which document type they need. Remain model-invoked (Claude auto-activates from context).

### Layer 4 — MCP Primitives (INFRASTRUCTURE, unchanged)

8 CRUD tools. Not user-facing. Used by skills, agents, and Claude directly.

### Intent skill design principles

1. **Explicit routing tables** — each intent skill contains a bounded decision tree that maps user input to specific document types or tracks. No open-ended "reason about what fits" instructions.

2. **Minimal elicitation** — one scope-confirmation question at the intent level ("full feature plan or just a plan document?"), then content questions per-document during creation (inherited from track/type skill patterns).

3. **Self-contained** — Claude Code activates one skill at a time, so intent skills contain inline creation recipes (question + sections + create_document + add_relation per type) without delegating to type skills at runtime.

4. **User-only invocation** — all intent skills use `disable-model-invocation: true`. Auto-activation of orchestration flows from ambient context is too risky for false positives. Type skills remain model-invoked.

5. **Default to minimum path** — `/archcore:plan` defaults to the smallest complete unit (product-track: idea→prd→plan). Binary scope question unlocks larger flows. Never default to the largest flow.

### Naming decisions

- **`/archcore:capture`** instead of `/archcore:document` — "document" is ambiguous as verb/noun. "Capture" is an unambiguous verb matching how users describe the action ("capture a decision", "capture the architecture").
- **`/archcore:standard`** instead of `/archcore:standardize` — brevity matters in a chat interface. Noun form is conventional (git uses `commit`, `branch`).
- **`/archcore:plan`** absorbs the existing `plan` type skill — the intent version routes to either a single plan document or the full product-track, resolving the name collision by making the intent skill the primary entry point.

### Progressive disclosure mechanism

In Claude Code's flat skill namespace, progressive disclosure happens **inside commands via conversation**, not via command selection:
- Layer 1 skills detect scope from arguments and ask one clarifying question
- They internally invoke track/type logic without redirecting users to different commands
- `/archcore:help` serves as the navigation layer between tiers

Description fields carry the tier signal: Layer 1 descriptions start normally, Layer 2 descriptions prefix "Advanced —", Layer 3 descriptions prefix "Expert —".

## Alternatives Considered

### Keep flat 27-command structure with better descriptions

Tried to fix discoverability through description quality alone. Rejected: 27 entries in a flat list overwhelm regardless of description quality. The cognitive load is in the count, not the labels.

### Remove type and track skills entirely, keep only intent skills

Would lose the domain knowledge layer that `keep-document-type-skills.adr.md` established. Type skills serve model-invocation (Claude auto-activates `adr` when user says "record this decision") — this is valuable and would be lost. Track skills serve expert users who know exactly which flow they need.

### Add layer-level skills (/archcore:vision, /archcore:knowledge)

Layers are internal classification, not user mental models. Nobody thinks "I need a knowledge document." Still requires type selection within the layer (knowledge has 6 types), offering minimal friction reduction.

### Move intent routing into MCP (suggest_type tool)

Type suggestion is classification over natural language — belongs in the prompt/skill layer. A `suggest_type` MCP tool would make the server stateful, add unnecessary round-trips, and couple domain knowledge into a primitive CRUD server. Intent routing is prompt work, not data work.

## Consequences

### Positive

- Users see 7 intent-based entry points instead of 27 infrastructure-named skills
- Entry points match user mental models ("plan", "decide", "capture") not system taxonomy ("strs", "iso-track")
- Full power of 18 types and 6 tracks preserved for advanced/expert users
- MCP layer stays stable (no changes to 8 primitive tools)
- Type skills continue to serve model-invocation use case
- Clean layered architecture: each layer has a distinct role without duplication

### Negative

- Intent skills must be self-contained (include inline creation recipes), adding maintenance surface
- `/archcore:plan` name collision requires absorbing the existing `plan` type skill into the intent version
- Layer 2-3 skills are less discoverable in the flat picker (mitigated by description prefixes and `/archcore:help`)
- New intent skill structure (5 sections) differs from type skill structure (7 sections) — two patterns to maintain

### Constraints

- Intent skills MUST use `disable-model-invocation: true`
- Intent skills MUST contain explicit routing tables, not open-ended reasoning instructions
- Intent skills MUST default to minimum viable path, offer expansion via scope question
- Each intent skill MUST be self-contained with inline creation recipes per document type
- The existing `plan` type skill MUST be absorbed into the `/archcore:plan` intent skill
- Layer 2-3 skill descriptions MUST include tier prefixes ("Advanced —" / "Expert —")