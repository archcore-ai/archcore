---
title: "User-Invoked Skills — Tiered Command Surface Specification"
status: accepted
tags:
  - "commands"
  - "plugin"
---

## Purpose

Define the contract for user-invoked skills: their tier classification, discoverability, naming, argument handling, and behavior when users invoke them via slash commands. Reflects the Inverted Invocation Policy (`inverted-invocation-policy.adr.md`) for the intent/track/utility classes; the type-skill portion of that policy is superseded by `remove-document-type-skills.adr.md` — there are no per-type skills. The `status` and `graph` intents were merged into `review` and removed respectively per `merge-review-status-remove-graph.adr.md`.

Note: Claude Code has merged commands into skills. Codex still discovers slash commands from root-level `commands/*.md`, so those files are thin host-adapter wrappers that delegate to `skills/<name>/SKILL.md`. The skill file remains the behavioral source of truth.

## Scope

This specification covers the user-invoked surface of the plugin: how users discover, invoke, and interact with the skills or Codex command wrappers that appear in the `/` menu. It does not cover MCP tools.

## Authority

This specification is the authoritative reference for user-invoked skill behavior and discoverability in the plugin. The Skills System Specification defines internal skill structure; this spec defines the external-facing contract. The Inverted Invocation Policy ADR governs per-class invocation flags.

## Subject

### Tiered Command Surface

User-invocable skills are organized into two tiers plus one utility class. All visible commands are on disk; no hidden surface.

```
┌──────────────────────────────────────────────────────┐
│  TIER 1 — PRIMARY (9 skills, auto-invocable)         │
│  What most users see and use daily                   │
│                                                      │
│  /archcore:bootstrap  "initialize archcore"          │
│  /archcore:capture    "document this"                │
│  /archcore:plan       "plan this feature"            │
│  /archcore:decide     "record this decision"         │
│  /archcore:standard   "make this a standard"         │
│  /archcore:review     "show dashboard / audit docs"  │
│  /archcore:actualize  "are any docs stale?"          │
│  /archcore:help       "what can I do?"               │
│  /archcore:context    "what rules apply here?"       │
├──────────────────────────────────────────────────────┤
│  TIER 2 — ADVANCED (6 skills, auto-invocable)        │
│  For users who know which multi-doc flow they need   │
│                                                      │
│  /archcore:product-track       idea → prd → plan     │
│  /archcore:sources-track       mrd → brd → urd       │
│  /archcore:iso-track           brs → strs → syrs → srs │
│  /archcore:architecture-track  adr → spec → plan     │
│  /archcore:standard-track      adr → (opt cpat) → rule → guide │
│  /archcore:feature-track       prd → spec → plan → tt │
├──────────────────────────────────────────────────────┤
│  UTILITY (1 skill, user-only via /)                  │
│                                                      │
│  /archcore:verify — plugin integrity checks          │
└──────────────────────────────────────────────────────┘
```

Total visible in `/` menu: **16 commands**. Total skills on disk: 16. No hidden surface.

### Tier 1 — Primary Commands

Primary commands are intent-based. The user describes what they want to do, and the command routes to the correct document types, flows, or analysis. These skills are auto-invocable: the model picks them up from user phrasing without an explicit `/` invocation. Creation-oriented intents inline per-type elicitation so they are self-contained.

| Command               | Description (in skill picker)                                         | Argument                  | Behavior                                                |
| --------------------- | --------------------------------------------------------------------- | ------------------------- | ------------------------------------------------------- |
| `/archcore:bootstrap` | First-time onboarding — seed stack rule + run guide + imports         | —                         | Three-step flow, each step accept/edit/skip             |
| `/archcore:capture`   | Document a module/component/system                                    | `[topic]`                 | Routes to adr/spec/doc/guide based on context           |
| `/archcore:plan`      | Plan a feature or initiative end-to-end                               | `[topic]`                 | Routes to product-track or single plan                  |
| `/archcore:decide`    | Record a decision (ADR) or draft a proposal (RFC)                     | `[topic]`                 | Creates adr or rfc; offers rule+guide after ADR         |
| `/archcore:standard`  | Establish a team standard                                             | `[topic]`                 | Routes to standard-track (adr → optional cpat → rule → guide) |
| `/archcore:review`    | Documentation health (dashboard or `--deep` audit)                    | `[--deep] [category or tag]` | Default: compact dashboard. With `--deep` or filter: gaps, staleness, recommendations |
| `/archcore:actualize` | Detect stale documentation and suggest updates                        | `[scope: tag, cat, 'all']`| Code drift + cascade + temporal analysis                |
| `/archcore:help`      | Guide to Archcore commands and capabilities                           | —                         | Layer navigation, onboarding                            |
| `/archcore:context`   | Surface rules/decisions for a code area or pickup                     | `[path or topic]`         | search_documents-backed grouped markdown                |

### Tier 2 — Advanced Commands

Advanced commands are track-based. The user explicitly chooses a multi-document flow. Also auto-invocable so the model can orchestrate large cascades from natural language. Each track step inlines per-type elicitation.

| Command                        | Description (in skill picker)                                                        |
| ------------------------------ | ------------------------------------------------------------------------------------ |
| `/archcore:product-track`      | Advanced — Lightweight product requirements flow: idea → PRD → plan                  |
| `/archcore:sources-track`      | Advanced — Discovery flow: MRD → BRD → URD                                           |
| `/archcore:iso-track`          | Advanced — Formal ISO 29148 cascade: BRS → StRS → SyRS → SRS                         |
| `/archcore:architecture-track` | Advanced — ADR → spec → plan                                                         |
| `/archcore:standard-track`     | Advanced — ADR → (optional CPAT) → rule → guide                                      |
| `/archcore:feature-track`      | Advanced — PRD → spec → plan → task-type                                             |

### Utility

| Command            | Description                                              |
| ------------------ | -------------------------------------------------------- |
| `/archcore:verify` | Run plugin integrity checks — tests, lint, config audits |

User-only (`disable-model-invocation: true`) — maintenance skill for plugin developers, not end users.

### Document-type access without per-type commands

Every Archcore document type is reachable via:

1. An intent skill that inlines its creation (e.g., ADR via `/archcore:decide`, spec via `/archcore:capture`).
2. A track skill that creates it as part of a flow (e.g., brs via `/archcore:iso-track`, task-type via `/archcore:feature-track`).
3. A direct MCP call — `mcp__archcore__create_document(type=<any>)`.

See `skills-system.spec.md` → "Document-type coverage without type skills" for the full mapping of each type to the intents/tracks that reach it.

## Contract Surface

### Naming Conventions

- All commands use the `archcore:` plugin prefix
- Tier 1 commands use **action verbs or clear nouns**: bootstrap, capture, plan, decide, standard, review, actualize, help, context
- Tier 2 commands use **`<domain>-track`** pattern: product-track, iso-track, etc.
- No sub-namespaces (no `archcore:track:iso` or similar) — Claude Code uses a single colon as plugin separator

### Argument Handling

All visible commands accept an optional `[topic]` argument:

- **With argument**: `/archcore:plan auth-redesign` — the topic is passed as `$ARGUMENTS`, skill uses it to scope the work and check for duplicates
- **Without argument**: `/archcore:plan` — skill asks an initial question to establish topic/scope

Intent skills (Tier 1) treat the argument as a **description of intent** or **scope filter**, not a document slug. Track skills (Tier 2) treat it as a **topic identifier**.

The `/archcore:actualize` command treats the argument as a **scope filter** — a tag, category, or type name to narrow the analysis. The `/archcore:review` command accepts an optional `--deep` flag plus an optional filter (tag, category, or type) for the deep mode. The `/archcore:context` command treats it as a **path or topic** for its grouped-markdown search.

The `/archcore:bootstrap` command takes no argument; its flow is deterministic (three sequential steps).

### Discoverability

Claude Code shows all user-invocable skills in a flat list. The tiered hierarchy is communicated through:

1. **Description prefixes** — "Advanced —" for Tier 2. Tier 1 and Utility use clean descriptions.
2. **`/archcore:help`** — dedicated skill that explains the tier structure and guides users to the right command.
3. **SessionStart empty-state nudge** — on fresh repos, the session-start hook points users at `/archcore:bootstrap` so onboarding is self-routing.
4. **Natural conversation** — when a user describes an intent ("record the decision to use X", "show docs status", "set up archcore", "draft an RFC"), Claude auto-invokes the matching Tier 1 skill thanks to the Inverted Invocation Policy. The user does not need to know which command exists.

The `/archcore:help` output structure:

```
## Quick Start (most users start here)
/archcore:bootstrap  — seed an empty repo (stack rule, run guide, optional imports)
/archcore:capture    — document a module or component
/archcore:plan       — plan a feature end-to-end
/archcore:decide     — record a decision (ADR) or draft a proposal (RFC)
/archcore:standard   — establish a team standard
/archcore:review     — dashboard (default) or full audit (`--deep`)
/archcore:actualize  — detect stale docs, suggest updates
/archcore:context    — rules/decisions for a code area or pickup
/archcore:help       — this guide

## Advanced (multi-document flows)
/archcore:product-track       — idea → prd → plan
/archcore:sources-track       — mrd → brd → urd
/archcore:iso-track           — brs → strs → syrs → srs
/archcore:architecture-track  — adr → spec → plan
/archcore:standard-track      — adr → (optional cpat) → rule → guide
/archcore:feature-track       — prd → spec → plan → task-type

## Utility
/archcore:verify              — run plugin integrity checks

## Direct document creation
For any document type (including niche types like brs/strs/syrs/srs/mrd/brd/urd),
call mcp__archcore__create_document with the matching `type` parameter.

Tip: you can also just describe what you need in natural language.
Claude auto-invokes the right intent skill and routes from there.
```

### Plan Type Absorption

There is no per-type `plan` skill. The `/archcore:plan` Tier 1 intent skill is the entry point — it routes to either:
- A single plan document (if user says "just a plan")
- The full product-track flow (default for most inputs)

The `skills/plan/` directory contains the intent skill. Users who need a single plan document can use `/archcore:plan` and answer the scope question, or use MCP tools directly.

## Normative Behavior

- Tier 1 commands MUST NOT require knowledge of Archcore internals to use.
- Tier 1 commands MUST route to the correct types/tracks/analysis without user type selection.
- Tier 2 commands MUST assume the user knows which flow they want; they MAY auto-invoke from rich natural-language descriptions of a full cascade.
- Creation-oriented Tier 1 intents MUST be self-contained with inline creation recipes per document type they produce (check duplicates → ask questions → create → suggest relations).
- Tier 2 track skills MUST inline per-type elicitation for each step (they are the authoritative home for per-type creation recipes within the plugin).
- All creation commands MUST call `list_documents` before `create_document` to prevent duplicates.
- All creation commands MUST suggest `add_relation` calls after document creation.
- Analysis commands (review, actualize, context) MUST use MCP read tools for data gathering.
- The help command MUST present Tier 1 commands first, Tier 2 as a secondary section, Utility as a tertiary section, and direct-MCP access for any document type as a final note.
- The `/archcore:bootstrap` command MUST be idempotent: each step detects existing artifacts and asks before regenerating; re-runs on a partially-bootstrapped repo only offer the missing steps.
- The `/archcore:review` command MUST default to the short dashboard mode when invoked without arguments and MUST switch to the full audit mode when `--deep` or any non-flag filter is present.

## Constraints

- No sub-namespaces. All commands are `archcore:<name>`.
- Intent commands ask at most one scope-confirmation question before starting execution — except `/archcore:bootstrap`, which runs its three-step flow and asks per-step accept/edit/skip.
- Track skills ask at most 1–2 content questions per document step.

## Invariants

- Every intent skill is auto-invocable (no `disable-model-invocation`).
- Every track skill is auto-invocable (no `disable-model-invocation`).
- Every utility skill (verify) has `disable-model-invocation: true`.
- Every track skill description starts with "Advanced —".
- Every intent and track skill description enumerates trigger phrases and anti-triggers using the "Activate when X. Do NOT activate for Y (use /archcore:other)." format.
- Every creation command checks for duplicates first and suggests relations after.
- Every analysis command gathers data via MCP read tools before producing output.
- The help command accurately reflects the current tier structure and notes direct-MCP access for any document type.
- Every Archcore document type has at least one intent or track skill path that can create it.

## Error Handling

- If MCP server is unavailable, inform user with install/init instructions.
- If `create_document` fails due to duplicate filename, suggest an alternative slug.
- If intent routing is ambiguous, ask one scope question. If still ambiguous, default to `/archcore:capture` behavior.
- If git is unavailable for `/archcore:actualize`, skip code-drift analysis and perform cascade + temporal only.

## Conformance

A user-invoked skill or Codex command wrapper conforms to this specification if:

1. Its behavior resides at `skills/<name>/SKILL.md`; Codex may also expose a matching `commands/<name>.md` wrapper.
2. Its invocation flags match its class per the Inverted Invocation Policy ADR.
3. Its description carries the appropriate tier prefix (or none for Tier 1 / Utility).
4. It uses MCP tools exclusively for document operations.
5. Creation commands check for duplicates before creation and suggest relations after.
6. Analysis commands gather data via MCP read tools.
7. Its argument handling matches the tier pattern (intent description/filter for Tier 1, topic for Tier 2).
