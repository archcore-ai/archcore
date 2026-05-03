---
title: "Skills System Specification — Intent, Track, Type, and Workflow Skills"
status: accepted
tags:
  - "plugin"
  - "skills"
---

## Purpose

Define the contract for how skills are structured, discovered, and used within the Archcore Claude Plugin. Skills are organized into a 3-group hierarchy: intent skills (Layer 1), track skills (Layer 2), and utility skills. Each group has distinct structure, invocation behavior, and audience. Per-type elicitation lives inline in intent and track skills — there are no per-document-type skills.

Per-class invocation flags follow the Inverted Invocation Policy ADR (`inverted-invocation-policy.adr.md`) for the intent/track/utility portion of the policy; the type-skill portion of that ADR is superseded by `remove-document-type-skills.adr.md`.

## Scope

This specification covers all skill files in the `skills/` directory: 9 intent skills, 6 track skills, and 1 utility skill — 16 in total. It defines naming convention, content structure, invocation triggers, relationship to MCP tools, and tier classification. It does not cover agents (subagents).

## Authority

This specification is the authoritative reference for all skill files in the plugin. The Skill File Structure Standard (rule) derives from this specification. The Plugin Architecture spec defines how skills interact with other components. The Inverted Invocation Policy ADR defines per-class invocation flag requirements for the remaining classes.

## Subject

The skills system consists of directories under `skills/`, each containing a `SKILL.md` file. Skills fall into three groups organized into a hierarchy.

### Intent Skills (9) — Layer 1: Primary User Entry

Intent skills are the main user-facing entry points. They translate user intent into the correct document types, tracks, or analysis modes. They use explicit routing tables to classify input and operate via MCP tools. Creation-oriented intents inline per-type elicitation (questions + sections + MCP calls + relation suggestions).

| Directory | Skill | User intent | Routes to |
|---|---|---|---|
| `skills/bootstrap/` | bootstrap | Seed an empty `.archcore/` on first install | stack rule + run guide + optional imports (opt-in) |
| `skills/capture/` | capture | Document a module/component | adr, spec, doc, guide by context |
| `skills/plan/` | plan | Plan a feature/initiative | product-track flow or single plan |
| `skills/decide/` | decide | Record a decision or draft a proposal | adr (finalized) or rfc (open); offers rule+guide after ADR |
| `skills/standard/` | standard | Establish a team standard | standard-track flow (adr → optional cpat → rule → guide) |
| `skills/review/` | review | Documentation health (dashboard or `--deep` audit) | counts/relations/issues (default), or full coverage-gap and recommendation report |
| `skills/actualize/` | actualize | Detect stale docs, suggest updates | code drift, cascade, temporal analysis |
| `skills/help/` | help | Navigate the system | layer guide, onboarding |
| `skills/context/` | context | Surface rules/decisions for a code area or pickup | search_documents-backed grouped markdown |

Intent skills are auto-invocable (no invocation-restricting flag) per the Inverted Invocation Policy. The model picks them up from user phrasing, which is the routing entry point for natural-language requests. Their descriptions enumerate explicit triggers and anti-triggers ("Activate when X. Do NOT activate for Y (use /archcore:other).") so routing is deterministic.

### Track Skills (6) — Layer 2: Advanced Domain Flows

Each orchestrates a complete multi-document flow, creating documents in sequence with proper relations. For users (or the model) who know which flow they need. Each track step inlines the per-type elicitation (questions + sections + create + relate).

| Directory | Track | Flow |
|---|---|---|
| `skills/product-track/` | Product Track | idea → prd → plan |
| `skills/sources-track/` | Sources Track | mrd → brd → urd |
| `skills/iso-track/` | ISO 29148 Track | brs → strs → syrs → srs |
| `skills/architecture-track/` | Architecture Track | adr → spec → plan |
| `skills/standard-track/` | Standard Track | adr → (optional cpat) → rule → guide |
| `skills/feature-track/` | Feature Track | prd → spec → plan → task-type |

Track skills are auto-invocable so the model can route rich natural-language descriptions of full cascades through them. Their descriptions are prefixed with "Advanced —".

### Utility Skills (1)

| Directory | Skill | Purpose |
|---|---|---|
| `skills/verify/` | verify | Plugin integrity checks (tests, lint, config audit, cross-reference validation) |

Utility skills carry `disable-model-invocation: true`. They are maintenance tools for plugin developers, not end users.

### Document-type coverage without type skills

Every Archcore document type is reachable through an intent or track skill. Direct creation via `mcp__archcore__create_document(type=<any>)` also remains available without any skill mediating.

| Type | Reached via |
|---|---|
| adr | `/archcore:decide`, `/archcore:capture`, `/archcore:standard`, `/archcore:architecture-track`, `/archcore:standard-track` |
| rfc | `/archcore:decide` (open-proposal branch) |
| rule | `/archcore:standard`, `/archcore:standard-track` |
| guide | `/archcore:capture`, `/archcore:standard`, `/archcore:standard-track` |
| doc | `/archcore:capture` |
| spec | `/archcore:capture`, `/archcore:architecture-track`, `/archcore:feature-track` |
| prd | `/archcore:plan`, `/archcore:product-track`, `/archcore:feature-track` |
| idea | `/archcore:plan`, `/archcore:product-track` |
| plan | `/archcore:plan`, `/archcore:product-track`, `/archcore:architecture-track`, `/archcore:feature-track` |
| task-type | `/archcore:feature-track` |
| cpat | `/archcore:standard-track` (optional step between ADR and rule) |
| mrd, brd, urd | `/archcore:sources-track` |
| brs, strs, syrs, srs | `/archcore:iso-track` |

## Contract Surface

### File Location

Each skill resides at `skills/<name>/SKILL.md` where `<name>` is:
- The intent name (for intent skills): `bootstrap`, `capture`, `plan`, `decide`, `standard`, `review`, `actualize`, `help`, `context`
- The track name (for track skills): `product-track`, `sources-track`, `iso-track`, `architecture-track`, `standard-track`, `feature-track`
- The utility name: `verify`

### SKILL.md Frontmatter

**Intent skills (Layer 1) — auto-invocable:**
```yaml
---
name: <intent-name>
argument-hint: "[topic or description]"
description: <What this intent does. Activate when X. Do NOT activate for Y (use /archcore:other).>
---
```

**Track skills (Layer 2) — auto-invocable:**
```yaml
---
name: <track-name>
argument-hint: "[topic]"
description: "Advanced — <what this track orchestrates>. Activate when X. Do NOT activate for Y."
---
```

**Utility skills:**
```yaml
---
name: <utility-name>
argument-hint: "[options]"
description: <What this utility does>
disable-model-invocation: true
---
```

### Intent Skill Content Structure (Layer 1)

Every intent skill file MUST contain these sections in order:

1. **Title and one-liner** — What this intent does, in user terms.

2. **When to Use** — Natural language signals that lead to this intent. Contrast with adjacent intents (e.g., "Not /archcore:decide — that's for single decisions. /archcore:capture is for documenting a component comprehensively.").

3. **Routing Table** — Explicit decision tree mapping user input to document types, tracks, or analysis modes. Each branch terminates in a named type list, named track, or analysis scope. Maximum one clarifying question when input is ambiguous between two paths. (Flow-style intents such as `bootstrap` may replace the routing table with a numbered step sequence, as long as each step has a deterministic set of branches.)

4. **Execution** — Step-by-step flow:
   - Step 1: Gather data (list_documents, list_relations, git log as needed)
   - Step 2: Scope confirmation (one `AskUserQuestion` if `$ARGUMENTS` is ambiguous)
   - Steps 3–N: Core execution (document creation, analysis, or reporting). Creation steps include per-type elicitation inline: question → compose sections → create_document → add_relation.
   - Final step: Summary and suggested next actions

5. **Result** — Summary of what was created or found, and recommended next actions.

Note: Creation-oriented intents (bootstrap, capture, plan, decide, standard) include inline creation recipes covering every document type they can produce. Analysis-oriented intents (review, actualize) include analysis logic. The `review` intent has two output modes (short dashboard and `--deep` full audit) inside one skill. The help intent includes the layer navigation guide.

### Track Skill Content Structure (Layer 2)

Every track skill file MUST contain:

1. **Title and summary** — Track name, flow diagram, when to use.
2. **Step 1: Check existing** — `list_documents` to detect existing documents and prevent duplicates.
3. **Step 2: Determine scope** — Logic for picking up where existing documents left off.
4. **Steps 3–N: One step per document** — Focused questions, content sections to compose, `create_document` call, `add_relation` calls. Optional steps (like CPAT in standard-track) include a gating question before proceeding.
5. **Final step: Relate to existing** — Suggest links to documents outside the track.
6. **Result** — Summary of what was created and the relation chain.

## Normative Behavior

- Intent skills MUST NOT carry `disable-model-invocation` — they are the auto-invocation entry point.
- Intent skills MUST contain explicit routing tables with bounded decision branches (flow-style intents may substitute a numbered step sequence with deterministic branches per step).
- Intent and track skill descriptions MUST enumerate triggers and anti-triggers using the "Activate when X. Do NOT activate for Y." format.
- Intent skills MUST default to minimum viable path. Expansion requires a binary scope question.
- Creation-oriented intent skills MUST be self-contained with inline creation recipes (question + sections + create + relate per document type produced).
- Analysis-oriented intent skills (review, actualize) MUST use MCP read tools (list_documents, get_document, list_relations) and may use git/Grep/Glob for cross-referencing.
- Track skills MUST NOT carry `disable-model-invocation` — they are auto-invocable to support model-initiated cascades.
- Track skills MUST create documents sequentially, asking focused questions before each creation step.
- Track skills MUST inline per-type elicitation for each step (they are the authoritative home for that content within the plugin).
- Utility skills MUST carry `disable-model-invocation: true`.
- All skills MUST use MCP tools for document operations. MUST NOT instruct direct file writes.
- Skills MUST reference MCP tools by exact name.
- Skills provide guidance around the template, not the template itself.
- Track skill descriptions MUST be prefixed with "Advanced —".

## Constraints

- Intent skill files must not exceed 300 lines.
- Track skill files must not exceed 200 lines.
- Skill files must not include code blocks longer than 20 lines.
- Skills must not reference internal CLI implementation details — only the MCP tool interface.
- Skills must not embed full document templates.
- Intent skills and track skills may duplicate elicitation for the same document type when each is the self-contained entry point for a different user intent; per-type content is not maintained in a separate home.

## Invariants

- There are exactly 9 intent skills (Layer 1).
- There are exactly 6 track skills (Layer 2).
- There is exactly 1 utility skill (`verify`).
- Total skills on disk: 16. All are visible in `/` menu (no hidden skills).
- Every intent skill has a routing table section or a numbered step sequence with deterministic branches.
- Every track skill follows the sequential step structure.
- Every creation skill references `create_document` in its workflow.
- Every analysis skill references `list_documents` and `list_relations` in its workflow.
- No skill instructs direct Write/Edit to `.archcore/` files.
- Auto-invocable surface: intent (9) + track (6) = 15. User-only surface: utility (1).
- Every Archcore document type has at least one intent or track skill path.

## Error Handling

- If MCP tools are unavailable, skills inform the user with install/init instructions.
- If `create_document` fails (duplicate filename), skills suggest alternative filename.
- If intent routing is ambiguous after one scope question, default to the most general path.
- If a track skill detects existing documents mid-flow, skip already-created documents and resume.
- If git is unavailable for actualize, skip code-drift analysis and perform cascade + temporal only.

## Conformance

A skill file conforms to this specification if:

1. It resides at the correct path (`skills/<name>/SKILL.md`).
2. It has valid frontmatter with `name` and `description` fields.
3. Its invocation flag matches its class per the Inverted Invocation Policy ADR (utility: `disable-model-invocation: true`; intent/track: no restricting flag).
4. Intent skills contain all 5 required sections (title, when-to-use, routing-table-or-step-sequence, execution, result).
5. Track skills contain the sequential step structure.
6. It references appropriate MCP tools in its workflow.
7. It stays within its line limit (300/200).
8. It does not embed full template content.
9. Track skill descriptions carry the "Advanced —" prefix.
