---
title: "User-Invoked Skills — Tiered Command Surface Specification"
status: draft
tags:
  - "commands"
  - "plugin"
---

## Purpose

Define the contract for user-invoked skills: their tier classification, discoverability, naming, argument handling, and behavior when users invoke them via slash commands.

Note: Claude Code has merged commands into skills. The `commands/` directory is legacy. All user-invoked workflows use `skills/<name>/SKILL.md`.

## Scope

This specification covers the user-invoked surface of the plugin: how users discover, invoke, and interact with the 3 tiers of user-facing skills (intent, track, type). It does not cover model-invoked behavior or MCP tools.

## Authority

This specification is the authoritative reference for user-invoked skill behavior and discoverability in the plugin. The Skills System Specification defines internal skill structure; this spec defines the external-facing contract.

## Subject

### Tiered Command Surface

User-invoked skills are organized into three tiers with decreasing prominence:

```
┌──────────────────────────────────────────────────────┐
│  TIER 1 — PRIMARY (7 skills)                         │
│  What most users see and use daily                   │
│                                                      │
│  /archcore:capture   "document this"                 │
│  /archcore:plan      "plan this feature"             │
│  /archcore:decide    "record this decision"          │
│  /archcore:standard  "make this a standard"          │
│  /archcore:review    "check docs health"             │
│  /archcore:status    "show dashboard"                │
│  /archcore:help      "what can I do?"                │
├──────────────────────────────────────────────────────┤
│  TIER 2 — ADVANCED (6 skills)                        │
│  For users who know which multi-doc flow they need   │
│                                                      │
│  /archcore:product-track    idea → prd → plan        │
│  /archcore:sources-track    mrd → brd → urd          │
│  /archcore:iso-track        brs → strs → syrs → srs │
│  /archcore:architecture-track  adr → spec → plan     │
│  /archcore:standard-track   adr → rule → guide       │
│  /archcore:feature-track    prd → spec → plan → tt   │
├──────────────────────────────────────────────────────┤
│  TIER 3 — EXPERT (18 skills)                         │
│  For power users who know the exact document type    │
│                                                      │
│  /archcore:adr  /archcore:rfc  /archcore:rule  ...   │
│  /archcore:prd  /archcore:mrd  /archcore:strs  ...   │
└──────────────────────────────────────────────────────┘
```

### Tier 1 — Primary Commands

Primary commands are intent-based. The user describes what they want to do, and the command routes to the correct document types and flows.

| Command | Description (in skill picker) | Argument | Behavior |
|---|---|---|---|
| `/archcore:capture` | Capture documentation for a module, component, or topic | `[topic]` | Routes to adr/spec/doc/guide based on context |
| `/archcore:plan` | Plan a feature or initiative end-to-end | `[topic]` | Routes to product-track or single plan |
| `/archcore:decide` | Record an architectural or technical decision | `[topic]` | Creates adr, offers rule+guide follow-up |
| `/archcore:standard` | Establish a team standard from a decision | `[topic]` | Routes to standard-track (adr→rule→guide) |
| `/archcore:review` | Review documentation for gaps, staleness, and issues | `[category or tag]` | Produces actionable findings |
| `/archcore:status` | Show documentation dashboard | — | Compact counts and coverage |
| `/archcore:help` | Guide to Archcore commands and capabilities | — | Layer navigation, onboarding |

### Tier 2 — Advanced Commands

Advanced commands are track-based. The user explicitly chooses a multi-document flow.

| Command | Description (in skill picker) |
|---|---|
| `/archcore:product-track` | Advanced — Create idea, PRD, and plan with full traceability |
| `/archcore:sources-track` | Advanced — Create MRD, BRD, URD discovery documents |
| `/archcore:iso-track` | Advanced — Create ISO 29148 requirements cascade (BRS→StRS→SyRS→SRS) |
| `/archcore:architecture-track` | Advanced — Create ADR, spec, and plan for architectural design |
| `/archcore:standard-track` | Advanced — Create ADR, rule, and guide to codify a standard |
| `/archcore:feature-track` | Advanced — Create PRD, spec, plan, and task-type for feature lifecycle |

### Tier 3 — Expert Commands

Expert commands create a single document of a specific type. Also activated automatically by Claude when conversation context matches (model invocation).

| Command | Description (in skill picker) |
|---|---|
| `/archcore:adr` | Record an architectural decision with context and alternatives |
| `/archcore:rfc` | Expert — Propose a change for team review |
| `/archcore:rule` | Expert — Define a mandatory team standard |
| `/archcore:guide` | Expert — Write step-by-step instructions |
| `/archcore:doc` | Expert — Create reference material |
| `/archcore:spec` | Expert — Define a normative technical contract |
| `/archcore:prd` | Create product requirements with goals and metrics |
| `/archcore:idea` | Expert — Explore a product or technical concept |
| `/archcore:plan` | Create an implementation plan with phased tasks |
| `/archcore:mrd` | Expert — Document market requirements and landscape |
| `/archcore:brd` | Expert — Document business requirements and ROI |
| `/archcore:urd` | Expert — Document user requirements and personas |
| `/archcore:brs` | Expert — Formalize business requirements (ISO 29148) |
| `/archcore:strs` | Expert — Formalize stakeholder requirements (ISO 29148) |
| `/archcore:syrs` | Expert — Formalize system requirements (ISO 29148) |
| `/archcore:srs` | Expert — Formalize software requirements (ISO 29148) |
| `/archcore:task-type` | Expert — Document a recurring task pattern |
| `/archcore:cpat` | Expert — Document a code pattern change |

Note: High-frequency types (adr, prd, rule, guide, plan) omit the "Expert —" prefix for cleaner display and better model invocation matching.

## Contract Surface

### Naming Conventions

- All commands use the `archcore:` plugin prefix
- Tier 1 commands use **action verbs or clear nouns**: capture, plan, decide, standard, review, status, help
- Tier 2 commands use **`<domain>-track`** pattern: product-track, iso-track, etc.
- Tier 3 commands use **Archcore type identifiers**: adr, prd, spec, etc.
- No sub-namespaces (no `archcore:track:iso` or `archcore:type:strs`) — Claude Code uses a single colon as plugin separator

### Argument Handling

All commands accept an optional `[topic]` argument:

- **With argument**: `/archcore:plan auth-redesign` — the topic is passed as `$ARGUMENTS`, skill uses it to scope the work and check for duplicates
- **Without argument**: `/archcore:plan` — skill asks an initial question to establish topic/scope

Intent skills (Tier 1) treat the argument as a **description of intent**, not a document slug. Track and type skills (Tiers 2-3) treat it as a **topic identifier**.

### Discoverability

Claude Code shows all registered skills in a flat list. The tiered hierarchy is communicated through:

1. **Description prefixes** — "Advanced —" for Tier 2, "Expert —" for Tier 3 (except high-frequency types)
2. **`/archcore:help`** — dedicated skill that explains the tier structure and guides users to the right command
3. **Natural conversation** — when a user asks "what can I do with Archcore?", Claude uses the help skill to present the tier 1 commands first

The `/archcore:help` output structure:

```
## Quick Start (most users start here)
/archcore:capture  — document a module or component
/archcore:plan     — plan a feature end-to-end
/archcore:decide   — record a technical decision
/archcore:standard — establish a team standard
/archcore:review   — check documentation health
/archcore:status   — show dashboard

## Advanced (multi-document flows)
/archcore:product-track, /archcore:architecture-track, ...

## Expert (single document types)
/archcore:adr, /archcore:prd, /archcore:spec, ...

Tip: you can also just describe what you need in natural language.
Claude will pick the right document type automatically.
```

### Plan Type Skill Absorption

The `/archcore:plan` Tier 3 type skill (which creates a single `plan` document) is absorbed into the `/archcore:plan` Tier 1 intent skill. The intent version routes to either:
- A single plan document (if user says "just a plan")
- The full product-track flow (default for most inputs)

The `skills/plan/` directory contains the intent skill, not the type skill. Users who need a single plan document can use `/archcore:plan` and answer the scope question, or use the MCP tools directly.

## Normative Behavior

- Tier 1 commands MUST NOT require knowledge of Archcore internals to use.
- Tier 1 commands MUST route to the correct types/tracks without user type selection.
- Tier 2 commands MUST assume the user knows which flow they want.
- Tier 3 commands MUST work as quick-create shortcuts (check duplicates → ask 1-2 questions → create → suggest relations).
- All creation commands MUST call `list_documents` before `create_document` to prevent duplicates.
- All creation commands MUST suggest `add_relation` calls after document creation.
- The help command MUST present Tier 1 commands first, with Tiers 2-3 as secondary sections.

## Constraints

- No sub-namespaces. All commands are `archcore:<name>`.
- Intent commands ask at most one scope-confirmation question before starting execution.
- Type quick-create commands ask at most 2-3 content questions.

## Invariants

- Every intent skill has `disable-model-invocation: true`.
- Every track skill has `disable-model-invocation: true`.
- Every track skill description starts with "Advanced —".
- Every non-high-frequency type skill description starts with "Expert —".
- Every creation command checks for duplicates first and suggests relations after.
- The help command accurately reflects the current tier structure.

## Error Handling

- If MCP server is unavailable, inform user with install/init instructions.
- If `create_document` fails due to duplicate filename, suggest an alternative slug.
- If intent routing is ambiguous, ask one scope question. If still ambiguous, default to `/archcore:capture` behavior.

## Conformance

A user-invoked skill conforms to this specification if:

1. It resides at `skills/<name>/SKILL.md`
2. Its description carries the appropriate tier prefix (or none for Tier 1 / high-frequency Tier 3)
3. It uses MCP tools exclusively for document operations
4. It checks for duplicates before creation
5. It suggests relations after creation
6. Its argument handling matches the tier pattern (intent description for Tier 1, topic for Tiers 2-3)