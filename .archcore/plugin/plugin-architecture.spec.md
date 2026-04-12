---
title: "Plugin Architecture — 4-Layer Intent-Based Command Hierarchy"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Purpose

Define how the Archcore Claude Plugin's components — intent skills, domain flow skills, typed artifact skills, workflow skills, agents, hooks, and MCP server — compose into a unified 4-layer system. This specification describes the layer hierarchy, invocation model, data flow, component interactions, and the architectural invariants that hold everything together.

Individual component contracts are defined in dedicated specs (skills-system.spec, commands-system.spec, agent-system.spec, hooks-validation-system.spec, actualize-system.spec). This document is the overarching architecture that explains _how they work together_.

## Scope

The entire Archcore Claude Plugin runtime: from a user message or model decision through skill activation, MCP tool calls, hook enforcement, and validation feedback. Covers the interaction between all component types and all four architectural layers.

## Authority

This specification is the architectural reference for cross-component behavior. For component-specific contracts, defer to the dedicated specs. In case of conflict, the dedicated spec wins for its own component; this spec wins for cross-component interactions.

## Subject

### System Overview

The plugin is a Claude Code plugin that makes Archcore effortless. It organizes all functionality into a 4-layer hierarchy where each layer has a distinct role, audience, and invocation model.

```
┌─────────────────────────────────────────────────────────────────┐
│                        User / Claude Model                      │
│                                                                 │
│  "plan this feature"   "record decision"   "/archcore:capture"  │
└──────┬──────────────────────┬──────────────────────┬────────────┘
       │                      │                      │
       ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│  LAYER 1 — Intent API (PRIMARY, 8 skills)                       │
│                                                                 │
│  /archcore:capture  /archcore:plan  /archcore:decide            │
│  /archcore:standard /archcore:review /archcore:status            │
│  /archcore:actualize /archcore:help                              │
│                                                                 │
│  → Routes user intent to correct types/tracks                   │
│  → Explicit routing tables, minimal elicitation                 │
│  → User-only (disable-model-invocation: true)                   │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 2 — Domain Flows (ADVANCED, 6 track skills)              │
│                                                                 │
│  product-track  sources-track  iso-track                        │
│  architecture-track  standard-track  feature-track              │
│                                                                 │
│  → Multi-document orchestration with relation chains            │
│  → User-only, description prefixed "Advanced —"                 │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 3 — Typed Artifacts (EXPERT, 18 type skills)             │
│                                                                 │
│  adr rfc rule guide doc spec prd idea plan                      │
│  mrd brd urd brs strs syrs srs task-type cpat                  │
│                                                                 │
│  → Domain knowledge per document type                           │
│  → Model-invoked (auto) + user-invoked, "Expert —" prefix      │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 4 — MCP Primitives (INFRASTRUCTURE, 8 tools)             │
│                                                                 │
│  create_document  update_document  remove_document              │
│  list_documents   get_document                                  │
│  add_relation  remove_relation  list_relations                  │
│                                                                 │
│  → Atomic CRUD + relations over .archcore/                      │
│  → Used by all layers above, not directly user-facing           │
├─────────────────────────────────────────────────────────────────┤
│  HOOKS LAYER (CROSS-CUTTING, event-driven)                      │
│                                                                 │
│  SessionStart → load context + staleness check                  │
│  PreToolUse → block direct writes                               │
│  PostToolUse → validate after mutations + cascade detection     │
└─────────────────────────────────────────────────────────────────┘
```

### Layer Roles and Audiences

| Layer | Role | Audience | Count | Invocation |
|---|---|---|---|---|
| 1 — Intent API | Translate user intent into document types/tracks | All users | 8 skills | User-only |
| 2 — Domain Flows | Orchestrate multi-document flows with relations | Advanced users | 6 skills | User-only |
| 3 — Typed Artifacts | Domain knowledge per document type | Expert users / Claude model | 18 skills | Model-invoked + user |
| 4 — MCP Primitives | Atomic CRUD + relations | Skills, agents, Claude | 8 tools | Tool calls |
| Hooks | Enforce invariants, load context, detect staleness | Automatic | 5 entries | Event-driven |
| Agents | Complex multi-document tasks, audits | Delegated | 2 agents | Model or user |

### Component Inventory

| Component | Count | Layer | Files |
|---|---|---|---|
| Intent skills | 8 | 1 | `skills/{capture,plan,decide,standard,review,status,actualize,help}/SKILL.md` |
| Track skills | 6 | 2 | `skills/{product,sources,iso,architecture,standard,feature}-track/SKILL.md` |
| Type skills | 18 | 3 | `skills/{adr,rfc,rule,...}/SKILL.md` |
| Agents | 2 | cross-layer | `agents/{archcore-assistant,archcore-auditor}.md` |
| Hooks | 5 entries | cross-layer | `hooks/hooks.json` |
| Bin scripts | 5 | cross-layer | `bin/{session-start,check-archcore-write,validate-archcore,check-staleness,check-cascade}` |
| MCP server | 1 | 4 | Provided by archcore CLI |

## Contract Surface

### Invocation Model

The plugin has five invocation paths. Every path converges on the MCP tool layer.

#### Path 1: Intent Skill (primary user entry)

```
User types /archcore:plan auth-redesign →
  Intent skill activates →
    Routing table classifies intent →
      Scope question (if ambiguous) →
        Sequential document creation (question → create → relate per doc) →
          MCP tool calls → Hooks validate
```

Trigger: User explicitly invokes a Layer 1 command. Intent skills use `disable-model-invocation: true` — they never auto-activate.

Example: User types `/archcore:plan auth-redesign` → routing table picks product-track flow → asks "Full feature plan (idea + PRD + plan) or just a plan document?" → creates 3 documents with `implements` relations.

#### Path 2: Model-Invoked Type Skill (automatic)

```
User says something → Claude matches context to Layer 3 skill description →
  Type skill activates → Claude follows skill guidance →
    MCP tool calls → Hooks validate
```

Trigger: Claude's model recognizes conversation context matching a type skill's `description`. Only Layer 3 skills (18 type skills) are model-invoked.

Example: User says "let's record the decision to use PostgreSQL" → Claude activates `skills/adr/SKILL.md` → follows Quick Create flow → calls `create_document(type="adr")`.

#### Path 3: Direct Track/Type Skill (expert user)

```
User types /archcore:iso-track auth system →
  Track skill activates →
    Sequential document creation with questions →
      MCP tool calls → Hooks validate
```

Trigger: User explicitly invokes a Layer 2 or Layer 3 skill. Advanced/expert users who know which flow or type they need.

#### Path 4: Agent Delegation (complex tasks)

```
User request or Claude judgment → Agent spawned →
  Agent uses MCP tools directly → Hooks validate
```

Trigger: Claude decides the task is complex enough for a subagent, or user explicitly requests agent help.

#### Path 5: Actualize (freshness detection)

```
Session starts → SessionStart hook → check-staleness → drift warning injected
Document updated → PostToolUse hook → check-cascade → cascade warning injected
User invokes /archcore:actualize → deep analysis → report + interactive fixes
```

Trigger: Automatic (Layers 1-2 of Actualize) or user-invoked (Layer 3 of Actualize). The actualize path is the only path that reads and analyzes documents without creating new ones.

### Data Flow: Intent Skill Execution

Intent skills follow this pattern:

```
1. Verify  → list_documents()                    // MCP available?
2. Route   → Routing table + $ARGUMENTS           // classify intent
3. Scope   → AskUserQuestion (if ambiguous)        // one question max
4. Check   → list_documents(types=[...])           // prevent duplicates, detect pickup point
5. Create  → For each document in the flow:
   a. Ask  → AskUserQuestion                      // content questions (1-2 per doc)
   b. Make → create_document(type, content, ...)   // MCP tool
      ├──→ [PostToolUse] archcore validate         // integrity check
   c. Link → add_relation(source, target, type)    // connect to chain
      └──→ [PostToolUse] archcore validate         // integrity check
6. Cross   → Suggest relations to existing docs    // outside the flow
7. Report  → Summary of created docs + relations   // what was done
```

### Data Flow: Direct Write Interception

When Claude attempts to Write/Edit a `.archcore/*.md` file directly:

```
1. Claude calls Write/Edit with .archcore/ path
2. [PreToolUse] bin/check-archcore-write
   ├──→ Extracts file_path from stdin JSON
   ├──→ Matches .archcore/**/*.md pattern
   ├──→ Exit 2 + stderr message → BLOCKED
   └──→ Claude receives feedback: "Use MCP tools instead"
3. Claude retries via create_document or update_document
```

### Data Flow: Staleness Detection

The actualize system operates at three depths:

```
SessionStart:
  bin/session-start → bin/check-staleness
  ├──→ git log: last .archcore/ commit
  ├──→ git diff: code files changed since
  ├──→ grep: match against document content
  └──→ Output: "[Archcore Staleness] N files changed..."

PostToolUse (update_document):
  bin/check-cascade
  ├──→ Parse updated document path
  ├──→ Query relation graph (implements/depends_on/extends targets)
  └──→ Output: "[Archcore Cascade] Updated X. Check Y, Z..."

/archcore:actualize (on demand):
  ├──→ list_documents + list_relations + git log
  ├──→ Code→Doc drift analysis
  ├──→ Doc→Doc cascade analysis
  ├──→ Temporal staleness analysis
  └──→ Report + interactive fixes via update_document
```

### Skill Taxonomy

Skills are organized into four groups across three layers.

#### Intent Skills (8) — Layer 1

- **Frontmatter**: `name`, `description`, `argument-hint`, `disable-model-invocation: true`
- **User-only**: explicit `/archcore:<name> <topic>` invocation
- **Content structure**: When to Use, Routing Table, Execution (steps), Result
- **Key feature**: Explicit routing table that maps user input to document types/tracks
- **Self-contained**: includes inline creation recipes per document type (question + sections + create + relate)

| Skill | User intent | Routes to |
|---|---|---|
| `capture` | Document a module/component | adr, spec, doc, guide by context |
| `plan` | Plan a feature/initiative | product-track or single plan |
| `decide` | Record a decision | adr, offer rule+guide |
| `standard` | Establish a team standard | standard-track (adr→rule→guide) |
| `review` | Check documentation health | analysis + recommendations |
| `status` | Show dashboard | counts, relations, issues |
| `actualize` | Detect stale docs, suggest updates | code drift, cascade, temporal analysis |
| `help` | Navigate the system | layer guide, onboarding |

#### Track Skills (6) — Layer 2

- **Frontmatter**: `name`, `description` prefixed "Advanced —", `argument-hint`, `disable-model-invocation: true`
- **User-only**: explicit `/archcore:<track-name> <topic>` invocation
- **Content structure**: sequential steps (Check → Scope → Create doc 1 → Create doc 2 → ... → Cross-relate)
- **Defines**: document sequence, relation chain, scope detection (resume from existing docs)

| Track | Flow | Relation Chain |
|---|---|---|
| product-track | idea → prd → plan | each `implements` previous |
| sources-track | mrd → brd → urd | `related` peers |
| iso-track | brs → strs → syrs → srs | each `implements` previous |
| architecture-track | adr → spec → plan | spec `implements` adr, plan `implements` spec |
| standard-track | adr → rule → guide | rule `implements` adr, guide `related` rule |
| feature-track | prd → spec → plan → task-type | spec `implements` prd, plan `implements` spec, task-type `related` plan |

#### Type Skills (18) — Layer 3

- **Frontmatter**: `name`, `description` prefixed "Expert —", `argument-hint`
- **Dual invocation**: model auto-activates based on context; user invokes for quick create
- **Content structure**: When to Use, Quick Create, Relations — concise guidance around the template
- **Does NOT contain**: full template content, multi-document flows

| Category | Types |
|---|---|
| Knowledge | adr, rfc, rule, guide, doc, spec |
| Vision | prd, idea, plan, mrd, brd, urd, brs, strs, syrs, srs |
| Experience | task-type, cpat |

#### Workflow absorbed into Intent

The previous workflow skills (create, review, status) are absorbed into Layer 1 intent skills:
- `create` → absorbed by `/archcore:capture` (intent routing replaces type selection wizard)
- `review` → becomes Layer 1 intent skill `/archcore:review`
- `status` → becomes Layer 1 intent skill `/archcore:status`

### Agent Integration

Agents are an escalation path, not the primary interface. Most documentation tasks are handled by intent skills (Layer 1) or track skills (Layer 2).

#### When to use which layer

| Scenario | Layer | Component |
|---|---|---|
| "Plan this feature" | 1 | `/archcore:plan` (intent) |
| "Record this decision" | 1 | `/archcore:decide` (intent) |
| "Document this module" | 1 | `/archcore:capture` (intent) |
| "Check docs health" | 1 | `/archcore:review` (intent) |
| "Are any docs stale?" | 1 | `/archcore:actualize` (intent) |
| Run ISO requirements cascade | 2 | `/archcore:iso-track` (track) |
| Create a single ADR | 3 | Model-invoked `adr` skill or `/archcore:adr` |
| Restructure all auth docs with relations | agent | archcore-assistant |
| Audit documentation quality | agent | archcore-auditor |

#### Agent tool boundaries

Both agents are restricted: no Write, Edit, or Bash on `.archcore/` files. The assistant gets all 8 MCP tools + Read/Grep/Glob. The auditor gets only 3 read MCP tools + Read/Grep/Glob.

### Hook Enforcement Layer

Hooks form a cross-cutting layer that enforces architectural invariants and detects documentation staleness regardless of which layer initiated the operation.

| Hook | Event | Purpose |
|---|---|---|
| session-start | SessionStart | Load .archcore/ context, check CLI, detect code-doc drift |
| check-archcore-write | PreToolUse (Write\|Edit) | Block direct .archcore/*.md writes |
| validate-archcore #1 | PostToolUse (Write\|Edit) | Defense-in-depth validation |
| validate-archcore #2 | PostToolUse (MCP mutations) | Primary validation after MCP mutations |
| check-cascade | PostToolUse (update_document) | Cascade staleness detection via relation graph |

### Cross-Layer Interaction Patterns

#### Pattern 1: Intent → Track → MCP → Hook (full flow)

An intent skill routes to a track flow. The track creates documents sequentially via MCP. Hooks validate after each mutation.

Example: `/archcore:plan` → routes to product-track → creates idea, prd, plan with `implements` relations.

#### Pattern 2: Intent → Single Type → MCP → Hook (simple flow)

An intent skill determines only one document is needed. Creates directly via MCP.

Example: `/archcore:decide` with a simple decision → creates single adr document.

#### Pattern 3: Model → Type Skill → MCP → Hook (automatic)

Claude auto-activates a type skill from conversation context. Creates via MCP.

Example: User discusses a decision → Claude activates `adr` skill → creates ADR.

#### Pattern 4: Agent → MCP → Hook (complex flow)

An agent makes multiple MCP calls autonomously. Hooks validate each one.

#### Pattern 5: Write → Hook → Block → MCP (correction flow)

When any component attempts a direct write, PreToolUse blocks it. Claude retries via MCP.

#### Pattern 6: Update → Hook → Cascade Warning (freshness flow)

When a document is updated via MCP, the cascade hook detects potentially stale dependents and warns.

Example: User updates a PRD → Hook 5 fires → finds plan that `implements` this PRD → injects "plan may need review" warning.

## Normative Behavior

- All document operations MUST flow through MCP tools. This is the **MCP-only principle** — the single most important architectural invariant.
- Layer 1 intent skills MUST use `disable-model-invocation: true`.
- Layer 1 intent skills MUST contain explicit routing tables with bounded decision branches.
- Layer 1 intent skills MUST default to minimum viable path, offering expansion via one scope question.
- Layer 1 intent skills that create documents MUST be self-contained with inline creation recipes per document type.
- Layer 2 track skills MUST create documents sequentially, asking questions before each step.
- Layer 2 track skills MUST add relations between created documents as defined in their relation chain.
- Layer 2 track skill descriptions MUST be prefixed with "Advanced —".
- Layer 3 type skill descriptions MUST be prefixed with "Expert —" (except for high-frequency types: adr, prd, rule, guide, plan).
- Layer 3 type skills remain model-invoked for automatic context-based activation.
- Skills MUST NOT instruct direct file writes to `.archcore/`. They reference MCP tools by exact name.
- Agents MUST use MCP tools exclusively for `.archcore/` operations.
- Hooks MUST fire for every relevant tool call, regardless of which layer initiated it.
- The PreToolUse hook MUST block `.archcore/**/*.md` writes with exit code 2.
- PostToolUse validation hooks MUST run `archcore validate` after every document mutation.
- PostToolUse cascade hook MUST run after `update_document` to detect relation-graph staleness.
- SessionStart MUST include staleness check after context loading.

## Constraints

- Maximum 8 intent skills (Layer 1). New intent skills require updating this spec.
- Maximum 6 track skills (Layer 2). New tracks require an ADR.
- Maximum 18 type skills (Layer 3). Matches Archcore document types.
- Maximum 2 agents. New agents require an ADR.
- Hooks must complete within their timeout (PreToolUse: 1s, PostToolUse: 3s).
- Intent skills must not exceed 300 lines.
- Track skills must not exceed 200 lines.
- Type skills must not exceed 100 lines.

## Invariants

- Every user-facing entry point maps to exactly one of the four layers.
- Every document mutation passes through the MCP tool layer (Layer 4).
- Every MCP mutation triggers PostToolUse validation.
- Every `update_document` triggers cascade detection in addition to validation.
- Every direct `.archcore/*.md` write attempt is blocked by PreToolUse.
- Every session starts with project context loaded and staleness check run (or a warning if CLI is missing).
- Intent skills and track skills never duplicate type skill content.
- Intent skills contain routing tables; track skills contain flow definitions; type skills contain type knowledge.
- Agents never have Write/Edit/Bash access to `.archcore/` files.
- Layer 1 commands are the only "primary" surface; Layer 2-3 are discoverable but secondary.
- Staleness detection never modifies documents autonomously — only the `/archcore:actualize` skill modifies, and only with user confirmation.

## Error Handling

- **MCP server unavailable**: All layers inform the user with install/init instructions. Hooks degrade gracefully.
- **Duplicate document**: `create_document` fails. Skills suggest alternative filename.
- **Intent routing ambiguous**: Intent skill asks one scope-confirmation question. If still ambiguous, falls back to the `capture` intent (most general).
- **Track interrupted mid-flow**: Track skills detect existing documents via `list_documents` and resume from the next step.
- **Hook timeout**: PostToolUse fails open. PreToolUse fail-closed behavior handled by Claude Code.
- **Agent exceeds turn limit**: Agent returns partial results. User can re-invoke or continue manually.
- **Git unavailable for staleness**: SessionStart skips staleness check. `/archcore:actualize` skips code-drift analysis but performs cascade and temporal.

## Conformance

The plugin architecture conforms to this specification if:

1. All document operations flow through MCP tools
2. Layer 1 has exactly 8 intent skills, all with `disable-model-invocation: true`
3. Layer 2 has exactly 6 track skills, all with `disable-model-invocation: true`
4. Layer 3 has exactly 18 type skills matching Archcore document types
5. PreToolUse hook blocks 100% of direct `.archcore/**/*.md` writes
6. PostToolUse validation fires after every MCP document mutation
7. PostToolUse cascade detection fires after every `update_document`
8. SessionStart includes staleness check after context loading
9. No layer duplicates another's responsibility
10. Description fields carry tier prefixes for Layer 2-3 skills
11. Intent skills contain routing tables and inline creation recipes (for creation-oriented intents)
12. The invocation model matches: user-only for Layers 1-2, model+user for Layer 3, tool calls for Layer 4