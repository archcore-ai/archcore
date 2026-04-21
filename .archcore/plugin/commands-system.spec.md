---
title: "User-Invoked Skills — Tiered Command Surface Specification"
status: draft
tags:
  - "commands"
  - "plugin"
---

## Purpose

Define the contract for user-invoked skills: their tier classification, discoverability, naming, argument handling, and behavior when users invoke them via slash commands. Reflects the Inverted Invocation Policy (`inverted-invocation-policy.adr.md`).

Note: Claude Code has merged commands into skills. The `commands/` directory is legacy. All user-invoked workflows use `skills/<name>/SKILL.md`.

## Scope

This specification covers the user-invoked surface of the plugin: how users discover, invoke, and interact with the skills that appear in the `/` menu. It also covers the hidden niche-type surface that users reach via track orchestration. It does not cover MCP tools.

## Authority

This specification is the authoritative reference for user-invoked skill behavior and discoverability in the plugin. The Skills System Specification defines internal skill structure; this spec defines the external-facing contract. The Inverted Invocation Policy ADR governs per-class invocation flags.

## Subject

### Tiered Command Surface

User-invocable skills are organized into tiers with decreasing prominence; a fifth class is hidden from the `/` menu entirely.

```
┌──────────────────────────────────────────────────────┐
│  TIER 1 — PRIMARY (9 skills, auto-invocable)         │
│  What most users see and use daily                   │
│                                                      │
│  /archcore:capture    "document this"                │
│  /archcore:plan       "plan this feature"            │
│  /archcore:decide     "record this decision"         │
│  /archcore:standard   "make this a standard"         │
│  /archcore:review     "check docs health"            │
│  /archcore:status     "show dashboard"               │
│  /archcore:actualize  "are any docs stale?"          │
│  /archcore:graph      "show the relation graph"      │
│  /archcore:help       "what can I do?"               │
├──────────────────────────────────────────────────────┤
│  TIER 2 — ADVANCED (6 skills, auto-invocable)        │
│  For users who know which multi-doc flow they need   │
│                                                      │
│  /archcore:product-track       idea → prd → plan     │
│  /archcore:sources-track       mrd → brd → urd       │
│  /archcore:iso-track           brs → strs → syrs → srs │
│  /archcore:architecture-track  adr → spec → plan     │
│  /archcore:standard-track      adr → rule → guide    │
│  /archcore:feature-track       prd → spec → plan → tt │
├──────────────────────────────────────────────────────┤
│  TIER 3 — EXPERT (10 skills, user-only via /)        │
│  For power users who know the exact document type    │
│                                                      │
│  /archcore:adr    /archcore:prd    /archcore:rfc     │
│  /archcore:rule   /archcore:guide  /archcore:doc     │
│  /archcore:spec   /archcore:idea   /archcore:task-type │
│  /archcore:cpat                                      │
├──────────────────────────────────────────────────────┤
│  UTILITY (1 skill, user-only via /)                  │
│                                                      │
│  /archcore:verify — plugin integrity checks          │
├──────────────────────────────────────────────────────┤
│  HIDDEN (7 skills, model-only, not in /)             │
│  Niche discovery + ISO 29148 types                   │
│                                                      │
│  mrd, brd, urd, brs, strs, syrs, srs                 │
│  Reached by the model through sources-track          │
│  or iso-track orchestration.                         │
└──────────────────────────────────────────────────────┘
```

Total visible in `/` menu: **26 commands**. Total skills on disk: 33 (26 visible + 7 hidden).

### Tier 1 — Primary Commands

Primary commands are intent-based. The user describes what they want to do, and the command routes to the correct document types, flows, or analysis. These skills are auto-invocable: the model picks them up from user phrasing without an explicit `/` invocation.

| Command               | Description (in skill picker)                                  | Argument                  | Behavior                                          |
| --------------------- | -------------------------------------------------------------- | ------------------------- | ------------------------------------------------- |
| `/archcore:capture`   | Document a module/component/system                              | `[topic]`                 | Routes to adr/spec/doc/guide based on context    |
| `/archcore:plan`      | Plan a feature or initiative end-to-end                         | `[topic]`                 | Routes to product-track or single plan            |
| `/archcore:decide`    | Record an architectural or technical decision                   | `[topic]`                 | Creates adr, offers rule+guide follow-up          |
| `/archcore:standard`  | Establish a team standard                                       | `[topic]`                 | Routes to standard-track (adr→rule→guide)        |
| `/archcore:review`    | Audit documentation for gaps, staleness, and issues             | `[category or tag]`       | Produces actionable findings                      |
| `/archcore:status`    | Show documentation dashboard                                    | —                         | Compact counts and coverage                       |
| `/archcore:actualize` | Detect stale documentation and suggest updates                  | `[scope: tag, cat, 'all']`| Code drift + cascade + temporal analysis          |
| `/archcore:graph`     | Render the document relation graph as a Mermaid flowchart       | `[filter]`                | Grouped subgraphs, styled edges, orphan list      |
| `/archcore:help`      | Guide to Archcore commands and capabilities                     | —                         | Layer navigation, onboarding, hidden-skill access |

### Tier 2 — Advanced Commands

Advanced commands are track-based. The user explicitly chooses a multi-document flow. Also auto-invocable so the model can orchestrate large cascades from natural language.

| Command                        | Description (in skill picker)                                                    |
| ------------------------------ | -------------------------------------------------------------------------------- |
| `/archcore:product-track`      | Advanced — Lightweight product requirements flow: idea → PRD → plan              |
| `/archcore:sources-track`      | Advanced — Discovery flow: MRD → BRD → URD (orchestrates hidden niche types)     |
| `/archcore:iso-track`          | Advanced — Formal ISO 29148 cascade: BRS → StRS → SyRS → SRS (orchestrates hidden niche types) |
| `/archcore:architecture-track` | Advanced — ADR → spec → plan                                                     |
| `/archcore:standard-track`     | Advanced — ADR → rule → guide                                                    |
| `/archcore:feature-track`      | Advanced — PRD → spec → plan → task-type                                         |

### Tier 3 — Expert Commands

Expert commands create a single document of a specific type. User-only via `/` — the model does NOT auto-invoke these; it routes through Tier 1 instead. Descriptions are NOT in the model's initial context (`disable-model-invocation: true`).

| Command               | Description (in skill picker)                               |
| --------------------- | ----------------------------------------------------------- |
| `/archcore:adr`       | Record an architectural decision                            |
| `/archcore:prd`       | Create product requirements                                 |
| `/archcore:rfc`       | Expert — Propose a change for team review                   |
| `/archcore:rule`      | Expert — Define a mandatory team standard                   |
| `/archcore:guide`     | Expert — Write step-by-step instructions                    |
| `/archcore:doc`       | Expert — Create reference material                          |
| `/archcore:spec`      | Expert — Define a normative technical contract              |
| `/archcore:idea`      | Expert — Explore a product or technical concept             |
| `/archcore:task-type` | Expert — Document a recurring task pattern                  |
| `/archcore:cpat`      | Expert — Document a code pattern change                     |

High-frequency types (adr, prd) omit the "Expert —" prefix for cleaner display.

### Utility

| Command            | Description                                              |
| ------------------ | -------------------------------------------------------- |
| `/archcore:verify` | Run plugin integrity checks — tests, lint, config audits |

User-only (`disable-model-invocation: true`) — maintenance skill for plugin developers, not end users.

### Hidden — Niche Type Skills

Seven document types are not shown in the `/` autocomplete menu (`user-invocable: false`) to reduce cognitive load for typical users. The model can still invoke them — their descriptions stay in context — and `sources-track` / `iso-track` do so programmatically.

| Skill | Type                          | Reached via                 |
| ----- | ----------------------------- | --------------------------- |
| mrd   | Market Requirements           | `/archcore:sources-track`   |
| brd   | Business Requirements         | `/archcore:sources-track`   |
| urd   | User Requirements             | `/archcore:sources-track`   |
| brs   | Business Requirements Spec    | `/archcore:iso-track`       |
| strs  | Stakeholder Requirements Spec | `/archcore:iso-track`       |
| syrs  | System Requirements Spec      | `/archcore:iso-track`       |
| srs   | Software Requirements Spec    | `/archcore:iso-track`       |

Users who need direct access to these types can call `mcp__archcore__create_document` with the matching `type` parameter.

## Contract Surface

### Naming Conventions

- All commands use the `archcore:` plugin prefix
- Tier 1 commands use **action verbs or clear nouns**: capture, plan, decide, standard, review, status, actualize, graph, help
- Tier 2 commands use **`<domain>-track`** pattern: product-track, iso-track, etc.
- Tier 3 commands use **Archcore type identifiers**: adr, prd, spec, etc.
- No sub-namespaces (no `archcore:track:iso` or `archcore:type:strs`) — Claude Code uses a single colon as plugin separator

### Argument Handling

All visible commands accept an optional `[topic]` argument (graph accepts `[filter]` — a tag, type, category, or document slug for scoping the subgraph):

- **With argument**: `/archcore:plan auth-redesign` — the topic is passed as `$ARGUMENTS`, skill uses it to scope the work and check for duplicates
- **Without argument**: `/archcore:plan` — skill asks an initial question to establish topic/scope

Intent skills (Tier 1) treat the argument as a **description of intent** or **scope filter**, not a document slug. Track and type skills (Tiers 2-3) treat it as a **topic identifier**.

The `/archcore:actualize` command treats the argument as a **scope filter** — a tag, category, or type name to narrow the analysis. The `/archcore:graph` command does the same.

### Discoverability

Claude Code shows all user-invocable skills in a flat list. The tiered hierarchy is communicated through:

1. **Description prefixes** — "Advanced —" for Tier 2, "Expert —" for Tier 3 (except high-frequency types).
2. **`/archcore:help`** — dedicated skill that explains the tier structure, guides users to the right command, and documents how to reach hidden niche types.
3. **Natural conversation** — when a user describes an intent ("record the decision to use X", "show the graph"), Claude auto-invokes the matching Tier 1 skill thanks to the Inverted Invocation Policy. The user does not need to know which command exists.

The `/archcore:help` output structure:

```
## Quick Start (most users start here)
/archcore:capture    — document a module or component
/archcore:plan       — plan a feature end-to-end
/archcore:decide     — record a technical decision
/archcore:standard   — establish a team standard
/archcore:review     — check documentation health
/archcore:status     — show dashboard
/archcore:actualize  — detect stale docs, suggest updates
/archcore:graph      — render the relation graph
/archcore:help       — this guide

## Advanced (multi-document flows)
/archcore:product-track, /archcore:architecture-track, ...

## Expert (single document types)
/archcore:adr, /archcore:prd, /archcore:rfc, /archcore:rule,
/archcore:guide, /archcore:doc, /archcore:spec, /archcore:idea,
/archcore:task-type, /archcore:cpat

## Hidden niche types
MRD/BRD/URD: use /archcore:sources-track
BRS/StRS/SyRS/SRS: use /archcore:iso-track
Direct creation: use the mcp__archcore__create_document tool.

Tip: you can also just describe what you need in natural language.
Claude auto-invokes the right intent skill and routes from there.
```

### Plan Type Skill Absorption

The `/archcore:plan` Tier 3 type skill (which creates a single `plan` document) is absorbed into the `/archcore:plan` Tier 1 intent skill. The intent version routes to either:
- A single plan document (if user says "just a plan")
- The full product-track flow (default for most inputs)

The `skills/plan/` directory contains the intent skill, not a type skill. Users who need a single plan document can use `/archcore:plan` and answer the scope question, or use MCP tools directly.

## Normative Behavior

- Tier 1 commands MUST NOT require knowledge of Archcore internals to use.
- Tier 1 commands MUST route to the correct types/tracks/analysis without user type selection.
- Tier 2 commands MUST assume the user knows which flow they want; they MAY auto-invoke from rich natural-language descriptions of a full cascade.
- Tier 3 commands MUST work as quick-create shortcuts (check duplicates → ask 1–2 questions → create → suggest relations).
- Tier 3 commands MUST NOT be auto-invoked by the model; routing flows through Tier 1 instead.
- Hidden niche type skills MUST be reachable by the model via track orchestration; tracks that use them (`sources-track`, `iso-track`) MUST remain auto-invocable.
- All creation commands MUST call `list_documents` before `create_document` to prevent duplicates.
- All creation commands MUST suggest `add_relation` calls after document creation.
- Analysis commands (review, status, actualize, graph) MUST use MCP read tools for data gathering.
- The help command MUST present Tier 1 commands first, Tiers 2–3 as secondary sections, and the hidden-niche-type access path as a tertiary section.

## Constraints

- No sub-namespaces. All commands are `archcore:<name>`.
- Intent commands ask at most one scope-confirmation question before starting execution.
- Type quick-create commands ask at most 2–3 content questions.
- The hidden niche-type surface MUST be kept in sync with the `sources-track` and `iso-track` document lists; removing a niche type requires removing its track step.

## Invariants

- Every intent skill is auto-invocable (no `disable-model-invocation`, default `user-invocable`).
- Every track skill is auto-invocable (no `disable-model-invocation`, default `user-invocable`).
- Every mainstream type skill (adr, prd, rfc, rule, guide, doc, spec, idea, task-type, cpat) has `disable-model-invocation: true`.
- Every niche type skill (mrd, brd, urd, brs, strs, syrs, srs) has `user-invocable: false`.
- Every utility skill (verify) has `disable-model-invocation: true`.
- Every track skill description starts with "Advanced —".
- Every non-high-frequency mainstream type skill description starts with "Expert —".
- Every intent and track skill description enumerates trigger phrases and anti-triggers using the "Activate when X. Do NOT activate for Y (use /archcore:other)." format.
- Every creation command checks for duplicates first and suggests relations after.
- Every analysis command gathers data via MCP read tools before producing output.
- The help command accurately reflects the current tier structure and documents the hidden-niche-type access path.

## Error Handling

- If MCP server is unavailable, inform user with install/init instructions.
- If `create_document` fails due to duplicate filename, suggest an alternative slug.
- If intent routing is ambiguous, ask one scope question. If still ambiguous, default to `/archcore:capture` behavior.
- If git is unavailable for `/archcore:actualize`, skip code-drift analysis and perform cascade + temporal only.

## Conformance

A user-invoked skill conforms to this specification if:

1. It resides at `skills/<name>/SKILL.md`
2. Its invocation flags match its class per the Inverted Invocation Policy ADR
3. Its description carries the appropriate tier prefix (or none for Tier 1 / high-frequency Tier 3)
4. It uses MCP tools exclusively for document operations
5. Creation commands check for duplicates before creation and suggest relations after
6. Analysis commands gather data via MCP read tools
7. Its argument handling matches the tier pattern (intent description/filter for Tier 1, topic for Tiers 2–3)
