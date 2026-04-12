---
title: "Skills System Specification — Intent, Track, Type, and Workflow Skills"
status: draft
tags:
  - "plugin"
  - "skills"
---

## Purpose

Define the contract for how skills are structured, discovered, and used within the Archcore Claude Plugin. Skills are organized into a 4-group hierarchy: intent skills (Layer 1), track skills (Layer 2), type skills (Layer 3), and utility skills. Each group has distinct structure, invocation behavior, and audience.

## Scope

This specification covers all skill files in the `skills/` directory: 7 intent skills, 6 track skills, and 18 document-type skills. It defines their naming convention, content structure, invocation triggers, relationship to MCP tools, and tier classification. It does not cover agents (subagents).

## Authority

This specification is the authoritative reference for all skill files in the plugin. The Skill File Structure Standard (rule) derives from this specification. The Plugin Architecture spec defines how skills interact with other components.

## Subject

The skills system consists of directories under `skills/`, each containing a `SKILL.md` file. Skills fall into four groups organized into a hierarchy.

### Intent Skills (7) — Layer 1: Primary User Entry

Intent skills are the main user-facing entry points. They translate user intent into the correct document types and tracks. They use explicit routing tables to classify input and create documents via MCP tools.

| Directory | Skill | User intent | Routes to |
|---|---|---|---|
| `skills/capture/` | capture | Document a module/component | adr, spec, doc, guide by context |
| `skills/plan/` | plan | Plan a feature/initiative | product-track flow or single plan |
| `skills/decide/` | decide | Record a decision | adr, offer rule+guide follow-up |
| `skills/standard/` | standard | Establish a team standard | standard-track flow (adr→rule→guide) |
| `skills/review/` | review | Check documentation health | analysis + recommendations |
| `skills/status/` | status | Show dashboard | counts, relations, issues |
| `skills/help/` | help | Navigate the system | layer guide, onboarding |

Intent skills are always user-only (`disable-model-invocation: true`). They never auto-activate from ambient context because false-positive activation of orchestration flows is disruptive.

### Track Skills (6) — Layer 2: Advanced Domain Flows

Each orchestrates a complete multi-document flow, creating documents in sequence with proper relations. For advanced users who know which flow they need.

| Directory | Track | Flow |
|---|---|---|
| `skills/product-track/` | Product Track | idea → prd → plan |
| `skills/sources-track/` | Sources Track | mrd → brd → urd |
| `skills/iso-track/` | ISO 29148 Track | brs → strs → syrs → srs |
| `skills/architecture-track/` | Architecture Track | adr → spec → plan |
| `skills/standard-track/` | Standard Track | adr → rule → guide |
| `skills/feature-track/` | Feature Track | prd → spec → plan → task-type |

Track skills are user-only (`disable-model-invocation: true`). Their descriptions are prefixed with "Advanced —" for tier signaling in the flat skill picker.

Track skills do NOT duplicate type skill content. They define the flow — sequence of steps, relation chain, and scope detection.

### Type Skills (18) — Layer 3: Expert Typed Artifacts

Each teaches Claude about one Archcore document type. They serve two roles: domain knowledge for model invocation (Claude auto-activates) and quick-create for expert users who know exactly which type they need.

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
| `skills/plan/` | Implementation Plan | vision |
| `skills/mrd/` | Market Requirements | vision |
| `skills/brd/` | Business Requirements | vision |
| `skills/urd/` | User Requirements | vision |
| `skills/brs/` | Business Requirements Specification | vision |
| `skills/strs/` | Stakeholder Requirements Specification | vision |
| `skills/syrs/` | System Requirements Specification | vision |
| `skills/srs/` | Software Requirements Specification | vision |
| `skills/task-type/` | Recurring Task Pattern | experience |
| `skills/cpat/` | Code Pattern Change | experience |

Type skills are model-invoked (Claude activates automatically) AND user-invokable via `/archcore:<type> <topic>`. Their descriptions are prefixed with "Expert —" for tier signaling, except high-frequency types (adr, prd, rule, guide, plan) which keep clean descriptions for better model invocation matching.

### Absorbed Skills

The previous workflow category (create, review, status) is absorbed into Layer 1 intent skills:
- `create` wizard → absorbed by `/archcore:capture` (intent routing replaces type selection)
- `review` → becomes Layer 1 `/archcore:review`
- `status` → becomes Layer 1 `/archcore:status`

The previous `plan` type skill is absorbed by the `/archcore:plan` intent skill, which routes to either a single plan document or the full product-track flow.

## Contract Surface

### File Location

Each skill resides at `skills/<name>/SKILL.md` where `<name>` is:
- The intent name (for intent skills): `capture`, `plan`, `decide`, `standard`, `review`, `status`, `help`
- The track name (for track skills): `product-track`, `sources-track`, etc.
- The Archcore type identifier (for type skills): `adr`, `prd`, `spec`, etc.

### SKILL.md Frontmatter

**Intent skills (Layer 1):**
```yaml
---
name: <intent-name>
argument-hint: "[topic or description]"
description: <What this intent does — user-facing, no prefix>
disable-model-invocation: true
---
```

**Track skills (Layer 2):**
```yaml
---
name: <track-name>
argument-hint: "[topic]"
description: "Advanced — <what this track orchestrates>"
disable-model-invocation: true
---
```

**Type skills (Layer 3):**
```yaml
---
name: <type-name>
argument-hint: "[topic]"
description: "Expert — <when Claude should activate this skill>"
---
```

Note: High-frequency type skills (adr, prd, rule, guide, plan) omit the "Expert —" prefix to preserve model invocation quality.

### Intent Skill Content Structure (Layer 1)

Every intent skill file MUST contain these sections in order:

1. **Title and one-liner** — What this intent does, in user terms.

2. **When to Use** — Natural language signals that lead to this intent. Contrast with adjacent intents (e.g., "Not /archcore:decide — that's for single decisions. /archcore:capture is for documenting a component comprehensively.").

3. **Routing Table** — Explicit decision tree mapping user input to document types or tracks. Each branch terminates in a named type list or named track. Maximum one clarifying question when input is ambiguous between two paths.

4. **Execution** — Step-by-step flow:
   - Step 0: Verify MCP availability
   - Step 1: `list_documents` to detect existing docs, prevent duplicates, detect pickup point
   - Step 2: Scope confirmation (one `AskUserQuestion` if `$ARGUMENTS` is ambiguous)
   - Steps 3–N: Sequential document creation (one `AskUserQuestion` per document for content, then `create_document` + `add_relation`)
   - Final step: Suggest relations to existing documents outside the flow

5. **Result** — Summary of what was created, the relation chain, and recommended next actions.

### Track Skill Content Structure (Layer 2)

Every track skill file MUST contain:

1. **Title and summary** — Track name, flow diagram, when to use.
2. **Step 0: Verify MCP** — Check MCP availability.
3. **Step 1: Check existing** — `list_documents` to detect existing documents and prevent duplicates.
4. **Step 2: Determine scope** — Logic for picking up where existing documents left off.
5. **Steps 3–N: One step per document** — Focused questions, content sections to compose, `create_document` call, `add_relation` calls.
6. **Final step: Relate to existing** — Suggest links to documents outside the track.
7. **Result** — Summary of what was created and the relation chain.

### Type Skill Content Structure (Layer 3)

Every type skill file MUST contain these sections in order:

1. **When to Use** — Specific scenarios and context signals. Contrast with similar types.
2. **Quick Create** — Elicitation questions + MCP `create_document` call + `add_relation` suggestion.
3. **Relations** — Typical relation types and targets for this document type.

Type skills are deliberately concise. They provide disambiguation, elicitation, and relation guidance — not full template content (which lives in the MCP server).

## Normative Behavior

- Intent skills MUST use `disable-model-invocation: true`.
- Intent skills MUST contain explicit routing tables with bounded decision branches terminating in named types or tracks.
- Intent skills MUST default to minimum viable path. Expansion requires a binary scope question.
- Intent skills MUST be self-contained with inline creation recipes (question + sections + create + relate per document type).
- Track skills MUST use `disable-model-invocation: true`.
- Track skills MUST create documents sequentially, asking focused questions before each creation step.
- Type skills are model-invoked: Claude activates them automatically based on `description` matching.
- All skills MUST use `create_document` MCP tool for document creation. MUST NOT instruct direct file writes.
- Skills MUST reference MCP tools by exact name.
- Skills provide guidance around the template, not the template itself.
- When multiple type skills could apply, Claude should prefer the most specific type.
- Track skill descriptions MUST be prefixed with "Advanced —".
- Type skill descriptions MUST be prefixed with "Expert —" (except adr, prd, rule, guide, plan).

## Constraints

- Intent skill files must not exceed 300 lines.
- Track skill files must not exceed 200 lines.
- Type skill files must not exceed 100 lines.
- Skill files must not include code blocks longer than 20 lines.
- Skills must not reference internal CLI implementation details — only the MCP tool interface.
- Skills must not embed full document templates.
- Track skills must not duplicate content from type skills.
- Intent skills must not duplicate content from track skills (they inline creation recipes, not track flow definitions).

## Invariants

- There are exactly 7 intent skills (Layer 1).
- There are exactly 6 track skills (Layer 2).
- There is exactly one type skill per Archcore document type (18 total, Layer 3).
- Every intent skill has a routing table section.
- Every track skill follows the sequential step structure.
- Every type skill has When to Use, Quick Create, and Relations sections.
- Every skill references `create_document` in its workflow.
- No skill instructs direct Write/Edit to `.archcore/` files.

## Error Handling

- If MCP tools are unavailable, skills inform the user with install/init instructions.
- If `create_document` fails (duplicate filename), skills suggest alternative filename.
- If intent routing is ambiguous after one scope question, default to the most general path.
- If a track skill detects existing documents mid-flow, skip already-created documents and resume.

## Conformance

A skill file conforms to this specification if:

1. It resides at the correct path (`skills/<name>/SKILL.md`)
2. It has valid frontmatter with `name` and `description` fields
3. Intent skills contain all 5 required sections (title, when-to-use, routing-table, execution, result)
4. Track skills contain the sequential step structure
5. Type skills contain When to Use, Quick Create, and Relations sections
6. It references `create_document` MCP tool (not Write/Edit) in its workflow
7. It stays within its line limit (300/200/100)
8. It does not embed full template content
9. Description fields carry appropriate tier prefixes