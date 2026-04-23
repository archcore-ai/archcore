---
title: "Skills System Specification — Intent, Track, Type, and Workflow Skills"
status: accepted
tags:
  - "plugin"
  - "skills"
---

## Purpose

Define the contract for how skills are structured, discovered, and used within the Archcore Claude Plugin. Skills are organized into a 4-group hierarchy: intent skills (Layer 1), track skills (Layer 2), type skills (Layer 3), and utility skills. Each group has distinct structure, invocation behavior, and audience.

Per-class invocation flags follow the Inverted Invocation Policy ADR (`inverted-invocation-policy.adr.md`). This spec defines structural contracts; the ADR governs which classes auto-invoke vs which are user-only.

## Scope

This specification covers all skill files in the `skills/` directory: 11 intent skills, 6 track skills, 17 document-type skills (10 mainstream + 7 niche), and 1 utility skill — 35 in total. It defines naming convention, content structure, invocation triggers, relationship to MCP tools, and tier classification. It does not cover agents (subagents).

## Authority

This specification is the authoritative reference for all skill files in the plugin. The Skill File Structure Standard (rule) derives from this specification. The Plugin Architecture spec defines how skills interact with other components. The Inverted Invocation Policy ADR defines per-class invocation flag requirements.

## Subject

The skills system consists of directories under `skills/`, each containing a `SKILL.md` file. Skills fall into four groups organized into a hierarchy.

### Intent Skills (11) — Layer 1: Primary User Entry

Intent skills are the main user-facing entry points. They translate user intent into the correct document types, tracks, or analysis modes. They use explicit routing tables to classify input and operate via MCP tools.

| Directory | Skill | User intent | Routes to |
|---|---|---|---|
| `skills/bootstrap/` | bootstrap | Seed an empty `.archcore/` on first install | stack rule + run guide + optional imports (opt-in) |
| `skills/capture/` | capture | Document a module/component | adr, spec, doc, guide by context |
| `skills/plan/` | plan | Plan a feature/initiative | product-track flow or single plan |
| `skills/decide/` | decide | Record a decision | adr, offer rule+guide follow-up |
| `skills/standard/` | standard | Establish a team standard | standard-track flow (adr→rule→guide) |
| `skills/review/` | review | Check documentation health | analysis + recommendations |
| `skills/status/` | status | Show dashboard | counts, relations, issues |
| `skills/actualize/` | actualize | Detect stale docs, suggest updates | code drift, cascade, temporal analysis |
| `skills/graph/` | graph | Render the relation graph | Mermaid flowchart, orphan list |
| `skills/help/` | help | Navigate the system | layer guide, onboarding |
| `skills/context/` | context | Surface rules/decisions for a code area or pickup | search_documents-backed grouped markdown |

Intent skills are auto-invocable (no invocation-restricting flag) per the Inverted Invocation Policy. The model picks them up from user phrasing, which is the routing entry point for natural-language requests. Their descriptions enumerate explicit triggers and anti-triggers ("Activate when X. Do NOT activate for Y (use /archcore:other).") so routing is deterministic.

### Track Skills (6) — Layer 2: Advanced Domain Flows

Each orchestrates a complete multi-document flow, creating documents in sequence with proper relations. For users (or the model) who know which flow they need.

| Directory | Track | Flow |
|---|---|---|
| `skills/product-track/` | Product Track | idea → prd → plan |
| `skills/sources-track/` | Sources Track | mrd → brd → urd |
| `skills/iso-track/` | ISO 29148 Track | brs → strs → syrs → srs |
| `skills/architecture-track/` | Architecture Track | adr → spec → plan |
| `skills/standard-track/` | Standard Track | adr → rule → guide |
| `skills/feature-track/` | Feature Track | prd → spec → plan → task-type |

Track skills are auto-invocable so the model can route rich natural-language descriptions of full cascades through them. Tracks that orchestrate niche types (`sources-track`, `iso-track`) rely on the niche skills remaining model-visible (see niche type policy below). Their descriptions are prefixed with "Advanced —".

Track skills do NOT duplicate type skill content. They define the flow — sequence of steps, relation chain, and scope detection.

### Type Skills (17) — Layer 3: Typed Artifacts

Each teaches Claude about one Archcore document type. Two policy classes:

#### Mainstream Type Skills (10) — User-only via `/`

| Directory | Document Type | Category |
|---|---|---|
| `skills/adr/` | Architecture Decision Record | knowledge |
| `skills/rfc/` | Request for Comments | knowledge |
| `skills/rule/` | Team Standard | knowledge |
| `skills/guide/` | How-To Instructions | knowledge |
| `skills/doc/` | Reference Material | knowledge |
| `skills/spec/` | Technical Specification | knowledge |
| `skills/prd/` | Product Requirements | vision |
| `skills/idea/` | Product/Technical Concept | vision |
| `skills/task-type/` | Recurring Task Pattern | experience |
| `skills/cpat/` | Code Pattern Change | experience |

Mainstream type skills carry `disable-model-invocation: true` per the Inverted Invocation Policy. The model does NOT auto-invoke them — natural-language routing flows through Layer 1 intent skills. Users who know exactly which type they need can use `/archcore:<type> <topic>` directly. Descriptions are prefixed with "Expert —" except for high-frequency types (adr, prd, rule, guide, idea) which keep clean descriptions for skill-picker readability.

#### Niche Type Skills (7) — Model-only via track orchestration

| Directory | Document Type | Reached via |
|---|---|---|
| `skills/mrd/` | Market Requirements | `/archcore:sources-track` |
| `skills/brd/` | Business Requirements | `/archcore:sources-track` |
| `skills/urd/` | User Requirements | `/archcore:sources-track` |
| `skills/brs/` | Business Requirements Specification | `/archcore:iso-track` |
| `skills/strs/` | Stakeholder Requirements Specification | `/archcore:iso-track` |
| `skills/syrs/` | System Requirements Specification | `/archcore:iso-track` |
| `skills/srs/` | Software Requirements Specification | `/archcore:iso-track` |

Niche type skills carry `user-invocable: false`. Hidden from `/` autocomplete to reduce cognitive load for typical users; their descriptions remain in model context so `sources-track` and `iso-track` can orchestrate them programmatically.

Note: The `plan` document type does NOT have a corresponding type skill — it is absorbed by the `/archcore:plan` intent skill, which routes to either a single plan document or the product-track flow.

### Utility Skills (1)

| Directory | Skill | Purpose |
|---|---|---|
| `skills/verify/` | verify | Plugin integrity checks (tests, lint, config audit, cross-reference validation) |

Utility skills carry `disable-model-invocation: true`. They are maintenance tools for plugin developers, not end users.

### Absorbed Skills

The previous workflow category (create, review, status) is absorbed into Layer 1 intent skills:
- `create` wizard → absorbed by `/archcore:capture` (intent routing replaces type selection)
- `review` → becomes Layer 1 `/archcore:review`
- `status` → becomes Layer 1 `/archcore:status`

The previous `plan` type skill is absorbed by the `/archcore:plan` intent skill, which routes to either a single plan document or the full product-track flow.

## Contract Surface

### File Location

Each skill resides at `skills/<name>/SKILL.md` where `<name>` is:
- The intent name (for intent skills): `bootstrap`, `capture`, `plan`, `decide`, `standard`, `review`, `status`, `actualize`, `graph`, `help`, `context`
- The track name (for track skills): `product-track`, `sources-track`, etc.
- The Archcore type identifier (for type skills): `adr`, `prd`, `spec`, etc.
- The utility name (for utility skills): `verify`

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

**Mainstream type skills (Layer 3) — user-only:**
```yaml
---
name: <type-name>
argument-hint: "[topic]"
description: "Expert — <when to use this type>"
disable-model-invocation: true
---
```

High-frequency type skills (adr, prd, rule, guide, idea) omit the "Expert —" prefix.

**Niche type skills (Layer 3) — model-only via tracks:**
```yaml
---
name: <type-name>
argument-hint: "[topic]"
description: "Expert — <when to use this type, typically via track orchestration>"
user-invocable: false
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
   - Steps 3–N: Core execution (document creation, analysis, or reporting)
   - Final step: Summary and suggested next actions

5. **Result** — Summary of what was created or found, and recommended next actions.

Note: Creation-oriented intents (bootstrap, capture, plan, decide, standard) include inline creation recipes. Analysis-oriented intents (review, status, actualize, graph) include analysis logic. The help intent includes the layer navigation guide.

### Track Skill Content Structure (Layer 2)

Every track skill file MUST contain:

1. **Title and summary** — Track name, flow diagram, when to use.
2. **Step 1: Check existing** — `list_documents` to detect existing documents and prevent duplicates.
3. **Step 2: Determine scope** — Logic for picking up where existing documents left off.
4. **Steps 3–N: One step per document** — Focused questions, content sections to compose, `create_document` call, `add_relation` calls.
5. **Final step: Relate to existing** — Suggest links to documents outside the track.
6. **Result** — Summary of what was created and the relation chain.

### Type Skill Content Structure (Layer 3)

Every type skill file MUST contain these sections in order:

1. **When to Use** — Specific scenarios and context signals. Contrast with similar types.
2. **Quick Create** — Elicitation questions + MCP `create_document` call + `add_relation` suggestion.
3. **Relations** — Typical relation types and targets for this document type.

Type skills are deliberately concise. They provide disambiguation, elicitation, and relation guidance — not full template content (which lives in the MCP server).

## Normative Behavior

- Intent skills MUST NOT carry `disable-model-invocation` — they are the auto-invocation entry point.
- Intent skills MUST contain explicit routing tables with bounded decision branches (flow-style intents may substitute a numbered step sequence with deterministic branches per step).
- Intent and track skill descriptions MUST enumerate triggers and anti-triggers using the "Activate when X. Do NOT activate for Y." format.
- Intent skills MUST default to minimum viable path. Expansion requires a binary scope question.
- Creation-oriented intent skills MUST be self-contained with inline creation recipes (question + sections + create + relate per document type).
- Analysis-oriented intent skills (review, status, actualize, graph) MUST use MCP read tools (list_documents, get_document, list_relations) and may use git/Grep/Glob for cross-referencing.
- Track skills MUST NOT carry `disable-model-invocation` — they are auto-invocable to support model-initiated cascades.
- Track skills MUST create documents sequentially, asking focused questions before each creation step.
- Mainstream type skills MUST carry `disable-model-invocation: true`. Routing flows through intent skills.
- Niche type skills MUST carry `user-invocable: false`. The model still sees them so tracks can orchestrate them.
- Utility skills MUST carry `disable-model-invocation: true`.
- All skills MUST use MCP tools for document operations. MUST NOT instruct direct file writes.
- Skills MUST reference MCP tools by exact name.
- Skills provide guidance around the template, not the template itself.
- Track skill descriptions MUST be prefixed with "Advanced —".
- Mainstream type skill descriptions MUST be prefixed with "Expert —" (except high-frequency types: adr, prd, rule, guide, idea).
- Niche type skill descriptions MUST be prefixed with "Expert —".

## Constraints

- Intent skill files must not exceed 300 lines.
- Track skill files must not exceed 200 lines.
- Type skill files must not exceed 100 lines.
- Skill files must not include code blocks longer than 20 lines.
- Skills must not reference internal CLI implementation details — only the MCP tool interface.
- Skills must not embed full document templates.
- Track skills must not duplicate content from type skills.
- Intent skills must not duplicate content from track skills (creation intents inline creation recipes, not track flow definitions).

## Invariants

- There are exactly 11 intent skills (Layer 1).
- There are exactly 6 track skills (Layer 2).
- There are exactly 17 type skills, one per Archcore document type except `plan` (which is served by the `/archcore:plan` intent skill): 10 mainstream + 7 niche.
- There is exactly 1 utility skill (`verify`).
- Total skills on disk: 35. Visible in `/` menu: 28 (35 − 7 niche hidden).
- Every intent skill has a routing table section or a numbered step sequence with deterministic branches.
- Every track skill follows the sequential step structure.
- Every type skill has When to Use, Quick Create, and Relations sections.
- Every creation skill references `create_document` in its workflow.
- Every analysis skill references `list_documents` and `list_relations` in its workflow.
- No skill instructs direct Write/Edit to `.archcore/` files.
- Auto-invocable surface: intent (11) + track (6) = 17. User-only surface: mainstream type (10) + utility (1) = 11. Hidden surface: niche type (7).

## Error Handling

- If MCP tools are unavailable, skills inform the user with install/init instructions.
- If `create_document` fails (duplicate filename), skills suggest alternative filename.
- If intent routing is ambiguous after one scope question, default to the most general path.
- If a track skill detects existing documents mid-flow, skip already-created documents and resume.
- If git is unavailable for actualize, skip code-drift analysis and perform cascade + temporal only.

## Conformance

A skill file conforms to this specification if:

1. It resides at the correct path (`skills/<name>/SKILL.md`)
2. It has valid frontmatter with `name` and `description` fields
3. Its invocation flag matches its class per the Inverted Invocation Policy ADR
4. Intent skills contain all 5 required sections (title, when-to-use, routing-table-or-step-sequence, execution, result)
5. Track skills contain the sequential step structure
6. Type skills contain When to Use, Quick Create, and Relations sections
7. It references appropriate MCP tools in its workflow
8. It stays within its line limit (300/200/100)
9. It does not embed full template content
10. Description fields carry appropriate tier prefixes
