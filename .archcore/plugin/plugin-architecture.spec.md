---
title: "Plugin Architecture — 4-Layer Intent-Based Command Hierarchy"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Purpose

Define how the Archcore Claude Plugin's components — intent skills, domain flow skills, utility skills, agents, hooks, and MCP server — compose into a unified system. This specification describes the layer hierarchy, invocation model, data flow, component interactions, and the architectural invariants that hold everything together.

Individual component contracts are defined in dedicated specs (skills-system.spec, commands-system.spec, agent-system.spec, hooks-validation-system.spec, actualize-system.spec). This document is the overarching architecture that explains _how they work together_.

Invocation policy (which layers auto-invoke vs user-only) is defined in `inverted-invocation-policy.adr.md`. The intent/track/utility portion of that ADR remains authoritative. The type-skill portion is superseded by `remove-document-type-skills.adr.md`; type skills no longer exist.

## Scope

The entire Archcore Claude Plugin runtime: from a user message or model decision through skill activation, MCP tool calls, hook enforcement, and validation feedback. Covers the interaction between all component types and all architectural layers.

## Authority

This specification is the architectural reference for cross-component behavior. For component-specific contracts, defer to the dedicated specs. In case of conflict, the dedicated spec wins for its own component; this spec wins for cross-component interactions.

## Subject

### System Overview

The plugin is a Claude Code plugin that makes Archcore effortless. It organizes functionality into an intent-based hierarchy where each layer has a distinct role, audience, and invocation model.

```
┌─────────────────────────────────────────────────────────────────┐
│                        User / Claude Model                      │
│                                                                 │
│  "plan this feature"   "record decision"   "/archcore:capture"  │
└──────┬──────────────────────┬──────────────────────┬────────────┘
       │                      │                      │
       ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│  LAYER 1 — Intent API (PRIMARY, 11 skills)                      │
│                                                                 │
│  /archcore:bootstrap /archcore:capture  /archcore:plan          │
│  /archcore:decide   /archcore:standard  /archcore:review        │
│  /archcore:status   /archcore:actualize /archcore:graph         │
│  /archcore:help     /archcore:context                           │
│                                                                 │
│  → Routes user intent to correct types/tracks                   │
│  → Explicit routing tables, minimal elicitation                 │
│  → Inline per-type creation recipes                             │
│  → Auto-invocable (model + user) per inverted policy            │
├─────────────────────────────────────────────────────────────────┤
│  LAYER 2 — Domain Flows (ADVANCED, 6 track skills)              │
│                                                                 │
│  product-track  sources-track  iso-track                        │
│  architecture-track  standard-track  feature-track              │
│                                                                 │
│  → Multi-document orchestration with relation chains            │
│  → Inline per-type elicitation at each step                     │
│  → Auto-invocable, description prefixed "Advanced —"            │
├─────────────────────────────────────────────────────────────────┤
│  MCP PRIMITIVES (INFRASTRUCTURE, 8 tools)                       │
│                                                                 │
│  create_document  update_document  remove_document              │
│  list_documents   get_document     search_documents             │
│  add_relation  remove_relation  list_relations  init_project    │
│                                                                 │
│  → Atomic CRUD + relations over .archcore/                      │
│  → Accepts any document type by type parameter                  │
│  → Used by all layers above, also directly callable             │
├─────────────────────────────────────────────────────────────────┤
│  UTILITY (1 skill, user-only)                                   │
│                                                                 │
│  verify — plugin integrity checks (tests, lint, config audit)   │
├─────────────────────────────────────────────────────────────────┤
│  HOOKS LAYER (CROSS-CUTTING, event-driven)                      │
│                                                                 │
│  SessionStart → load context + staleness check                  │
│  PreToolUse → block direct writes                               │
│  PostToolUse → validate after MCP mutations + cascade detection │
└─────────────────────────────────────────────────────────────────┘
```

### Layer Roles and Audiences

| Layer | Role | Audience | Count | Invocation |
|---|---|---|---|---|
| 1 — Intent API | Translate user intent into document types/tracks; inline creation recipes | All users | 11 skills | Auto (model + user) |
| 2 — Domain Flows | Orchestrate multi-document flows with relations; inline per-type elicitation | Advanced users | 6 skills | Auto (model + user) |
| MCP Primitives | Atomic CRUD + relations; accepts any type | Skills, agents, Claude | 8 tools | Tool calls |
| Utility | Plugin integrity checks | Plugin developers | 1 skill | User-only |
| Hooks | Enforce invariants, load context, detect staleness | Automatic | 4 entries | Event-driven |
| Agents | Complex multi-document tasks, audits | Delegated | 2 agents | Model or user |

### Component Inventory

| Component | Count | Layer | Files |
|---|---|---|---|
| Intent skills | 11 | 1 | `skills/{bootstrap,capture,plan,decide,standard,review,status,actualize,graph,help,context}/SKILL.md` |
| Track skills | 6 | 2 | `skills/{product,sources,iso,architecture,standard,feature}-track/SKILL.md` |
| Utility skills | 1 | cross-layer | `skills/verify/SKILL.md` |
| Agents | 2 | cross-layer | `agents/{archcore-assistant,archcore-auditor}.md` |
| Hooks | 4 entries | cross-layer | `hooks/hooks.json` (Claude Code), `hooks/cursor.hooks.json` (Cursor) |
| Bin scripts | 5 | cross-layer | `bin/{session-start,check-archcore-write,validate-archcore,check-staleness,check-cascade}` |
| MCP server | 1 | MCP | Provided by archcore CLI |

Total skills on disk: 18 (11 + 6 + 1). All 18 are visible in the `/` menu — no hidden or flagged-out skills.

## Contract Surface

### Invocation Model

The plugin has four invocation paths. Every path converges on the MCP tool layer.

#### Path 1: Intent Skill (primary user entry)

```
User types /archcore:plan auth-redesign →
  Intent skill activates →
    Routing table classifies intent →
      Scope question (if ambiguous) →
        Sequential document creation (question → create → relate per doc) →
          MCP tool calls → Hooks validate
```

Trigger: User explicitly invokes a Layer 1 command, or the model auto-invokes from natural language. Intent skills are auto-invocable under the Inverted Invocation Policy — their descriptions enumerate triggers and anti-triggers so routing is deterministic.

Example: User types `/archcore:plan auth-redesign` → routing table picks product-track flow → asks "Full feature plan (idea + PRD + plan) or just a plan document?" → creates 3 documents with `implements` relations.

#### Path 2: Track Skill (model or user)

```
User says "run ISO requirements cascade" OR types /archcore:iso-track →
  Track skill activates →
    Sequential document creation with questions →
      MCP tool calls → Hooks validate
```

Trigger: Auto-invocation from rich natural-language descriptions of a full cascade, or explicit `/archcore:<track>` invocation. Each track step inlines per-type elicitation (questions + sections + create + relate) — there is no per-type skill mediating.

#### Path 3: Agent Delegation (complex tasks)

```
User request or Claude judgment → Agent spawned →
  Agent uses MCP tools directly → Hooks validate
```

Trigger: Claude decides the task is complex enough for a subagent, or user explicitly requests agent help.

#### Path 4: Direct MCP (any document type)

```
Model or user invokes mcp__archcore__create_document(type=<any>, ...) →
  MCP tool call → Hooks validate
```

Trigger: The model decides to create a document type that matches user intent (e.g., user says "draft an RFC" which routes through `/archcore:decide` open-proposal branch, or the model calls MCP directly when no skill routing fits). MCP accepts every Archcore document type; no skill is required for creation.

#### Path 5: Actualize (freshness detection)

```
Session starts → SessionStart hook → check-staleness → drift warning injected
Document updated → PostToolUse hook → check-cascade → cascade warning injected
User invokes /archcore:actualize → deep analysis → report + interactive fixes
```

Trigger: Automatic (hook layers of Actualize) or user-invoked (the intent skill). The actualize path is the only path that reads and analyzes documents without creating new ones.

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

Note: PreToolUse blocks the write BEFORE it happens, so PostToolUse never fires for blocked `.archcore/*.md` writes. There is no PostToolUse `Write|Edit` validate-archcore entry — it would be dead weight forking a shell on every write anywhere in the repo. Validation runs only on the MCP path.

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

Skills are organized into two layers plus one utility class.

#### Intent Skills (11) — Layer 1

- **Frontmatter**: `name`, `description`, `argument-hint`. No `disable-model-invocation` flag — auto-invocable per Inverted Invocation Policy.
- **Auto-invocable**: model routes user phrasing to the matching intent skill.
- **Content structure**: When to Use, Routing Table, Execution (steps), Result
- **Key feature**: Explicit routing table that maps user input to document types/tracks
- **Self-contained**: includes inline creation recipes per document type (question + sections + create + relate)

| Skill | User intent | Routes to |
|---|---|---|
| `bootstrap` | Seed an empty `.archcore/` | stack rule + run guide + optional imports |
| `capture` | Document a module/component | adr, spec, doc, guide by context |
| `plan` | Plan a feature/initiative | product-track or single plan |
| `decide` | Record a decision or draft a proposal | adr (finalized) or rfc (open); offer rule+guide after ADR |
| `standard` | Establish a team standard | standard-track (adr → optional cpat → rule → guide) |
| `review` | Check documentation health | analysis + recommendations |
| `status` | Show dashboard | counts, relations, issues |
| `actualize` | Detect stale docs, suggest updates | code drift, cascade, temporal analysis |
| `graph` | Render the relation graph | Mermaid flowchart, orphan list |
| `help` | Navigate the system | layer guide, onboarding |
| `context` | Surface rules/decisions for a code area or pickup | search_documents-backed grouped markdown |

#### Track Skills (6) — Layer 2

- **Frontmatter**: `name`, `description` prefixed "Advanced —", `argument-hint`. No `disable-model-invocation` — auto-invocable.
- **Auto-invocable**: so the model can route multi-document requests through them.
- **Content structure**: sequential steps (Check → Scope → Create doc 1 → Create doc 2 → ... → Cross-relate)
- **Defines**: document sequence, relation chain, scope detection (resume from existing docs), per-type elicitation per step

| Track | Flow | Relation Chain |
|---|---|---|
| product-track | idea → prd → plan | each `implements` previous |
| sources-track | mrd → brd → urd | `related` peers |
| iso-track | brs → strs → syrs → srs | each `implements` previous |
| architecture-track | adr → spec → plan | spec `implements` adr, plan `implements` spec |
| standard-track | adr → (optional cpat) → rule → guide | rule `implements` adr, guide `related` rule, cpat `implements` adr (if created) |
| feature-track | prd → spec → plan → task-type | spec `implements` prd, plan `implements` spec, task-type `related` plan |

#### Utility Skills (1)

- `verify` — `disable-model-invocation: true`. Plugin integrity checks for developers (tests, lint, config audit, cross-reference validation).

#### Document-type coverage

There are no per-document-type skills. Every Archcore document type is reachable through:
- An intent skill that inlines creation for that type, or
- A track skill that creates it as part of a flow, or
- A direct MCP `create_document(type=<any>)` call.

See `skills-system.spec.md` → "Document-type coverage without type skills" for the full mapping.

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
| "Record this decision" | 1 | `/archcore:decide` (intent, ADR path) |
| "Draft an RFC" | 1 | `/archcore:decide` (intent, RFC path) |
| "Document this module" | 1 | `/archcore:capture` (intent) |
| "Check docs health" | 1 | `/archcore:review` (intent) |
| "Are any docs stale?" | 1 | `/archcore:actualize` (intent) |
| "Show the relation graph" | 1 | `/archcore:graph` (intent) |
| "What rules apply to src/X/" | 1 | `/archcore:context` (intent) |
| Run ISO requirements cascade | 2 | `/archcore:iso-track` (track) |
| Build full standard with pattern change | 2 | `/archcore:standard-track` (track, cpat optional) |
| Create a single niche document (e.g., task-type alone) | MCP | direct `mcp__archcore__create_document` |
| Restructure all auth docs with relations | agent | archcore-assistant |
| Audit documentation quality | agent | archcore-auditor |

#### Agent tool boundaries

Both agents are restricted: no Write, Edit, or Bash on `.archcore/` files. The assistant gets all MCP tools + Read/Grep/Glob. The auditor gets only read MCP tools + Read/Grep/Glob.

### Hook Enforcement Layer

Hooks form a cross-cutting layer that enforces architectural invariants and detects documentation staleness regardless of which layer initiated the operation.

| Hook | Event (Claude Code) | Event (Cursor) | Purpose |
|---|---|---|---|
| session-start | SessionStart | sessionStart | Load .archcore/ context, check CLI, detect code-doc drift |
| check-archcore-write | PreToolUse (Write\|Edit) | preToolUse (Write) | Block direct .archcore/*.md writes |
| validate-archcore | PostToolUse (MCP mutations) | afterMCPExecution | Primary validation after MCP mutations |
| check-cascade | PostToolUse (update_document) | afterMCPExecution (filtered) | Cascade staleness detection via relation graph |

Claude Code config (`hooks/hooks.json`) has 4 entries: 1 SessionStart, 1 PreToolUse, 2 PostToolUse. Cursor config (`hooks/cursor.hooks.json`) has 3 events: sessionStart, preToolUse, afterMCPExecution (afterMCPExecution runs both validate-archcore and check-cascade in sequence).

### Cross-Layer Interaction Patterns

#### Pattern 1: Intent → Track → MCP → Hook (full flow)

An intent skill routes to a track flow. The track creates documents sequentially via MCP. Hooks validate after each mutation.

Example: `/archcore:plan` → routes to product-track → creates idea, prd, plan with `implements` relations.

#### Pattern 2: Intent → Inline Creation → MCP → Hook (simple flow)

An intent skill creates documents inline using its self-contained recipe. No track needed.

Example: `/archcore:decide` with a finalized decision → inline ADR recipe → creates single adr document.

#### Pattern 3: Model → Intent Skill → Type Creation → MCP → Hook (auto routing)

Claude auto-activates an intent skill from conversation context. The intent routes to the correct type (using its inline recipe) and creates via MCP.

Example: User discusses a decision → Claude activates `decide` intent → routes to adr creation via MCP.

#### Pattern 4: Agent → MCP → Hook (complex flow)

An agent makes multiple MCP calls autonomously. Hooks validate each one.

#### Pattern 5: Write → Hook → Block → MCP (correction flow)

When any component attempts a direct write, PreToolUse blocks it. Claude retries via MCP.

#### Pattern 6: Update → Hook → Cascade Warning (freshness flow)

When a document is updated via MCP, the cascade hook detects potentially stale dependents and warns.

Example: User updates a PRD → check-cascade fires → finds plan that `implements` this PRD → injects "plan may need review" warning.

## Normative Behavior

- All document operations MUST flow through MCP tools. This is the **MCP-only principle** — the single most important architectural invariant.
- Layer 1 intent skills MUST NOT carry `disable-model-invocation` flags. They are the primary auto-invocation entry point.
- Layer 1 intent skills MUST contain explicit routing tables with bounded decision branches.
- Layer 1 intent skills MUST default to minimum viable path, offering expansion via one scope question.
- Layer 1 intent skills that create documents MUST be self-contained with inline creation recipes per document type they produce.
- Layer 2 track skills MUST NOT carry `disable-model-invocation` flags — they are auto-invocable to support model-initiated multi-document cascades.
- Layer 2 track skills MUST create documents sequentially, asking questions before each step.
- Layer 2 track skills MUST add relations between created documents as defined in their relation chain.
- Layer 2 track skills MUST inline per-type elicitation for each step (they are the authoritative home for that content within the plugin).
- Layer 2 track skill descriptions MUST be prefixed with "Advanced —".
- Skills MUST NOT instruct direct file writes to `.archcore/`. They reference MCP tools by exact name.
- Agents MUST use MCP tools exclusively for `.archcore/` operations.
- Hooks MUST fire for every relevant tool call, regardless of which layer initiated it.
- The PreToolUse hook MUST block `.archcore/**/*.md` writes with exit code 2.
- PostToolUse validation hooks MUST run `archcore validate` after every MCP document mutation.
- PostToolUse cascade hook MUST run after `update_document` to detect relation-graph staleness.
- No PostToolUse hook MUST be registered for `Write|Edit` — PreToolUse already blocks `.archcore/*.md` writes before they succeed.
- SessionStart MUST include staleness check after context loading.

## Constraints

- Maximum 11 intent skills (Layer 1). New intent skills require updating this spec.
- Maximum 6 track skills (Layer 2). New tracks require an ADR.
- Maximum 2 agents. New agents require an ADR.
- Hooks must complete within their timeout (PreToolUse: 1s, PostToolUse: 3s).
- Intent skills must not exceed 300 lines.
- Track skills must not exceed 200 lines.

## Invariants

- Every user-facing entry point maps to Layer 1, Layer 2, or the utility class.
- Every document mutation passes through the MCP tool layer.
- Every MCP mutation triggers PostToolUse validation.
- Every `update_document` triggers cascade detection in addition to validation.
- Every direct `.archcore/*.md` write attempt is blocked by PreToolUse.
- Every session starts with project context loaded and staleness check run (or a warning if CLI is missing).
- Intent skills and track skills inline per-type elicitation; this duplication is intentional and accepted per `skills-system.spec.md` to keep each entry point self-contained.
- Agents never have Write/Edit/Bash access to `.archcore/` files.
- Auto-invocable surface: Layer 1 (intent) + Layer 2 (track). Utility is user-only.
- Staleness detection never modifies documents autonomously — only the `/archcore:actualize` skill modifies, and only with user confirmation.
- Every Archcore document type is reachable through at least one intent or track skill (or directly via MCP).

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

1. All document operations flow through MCP tools.
2. Layer 1 has exactly 11 intent skills, all auto-invocable (no `disable-model-invocation`).
3. Layer 2 has exactly 6 track skills, all auto-invocable.
4. Utility class has exactly 1 skill (`verify`) with `disable-model-invocation: true`.
5. PreToolUse hook blocks 100% of direct `.archcore/**/*.md` writes.
6. PostToolUse validation fires after every MCP document mutation.
7. PostToolUse cascade detection fires after every `update_document`.
8. No PostToolUse hook is registered for `Write|Edit`.
9. SessionStart includes staleness check after context loading.
10. No layer duplicates another's responsibility (inline per-type elicitation in intents/tracks is not duplication — it is each entry point's self-containment).
11. Description fields carry the "Advanced —" prefix for Layer 2 skills.
12. Intent skills contain routing tables and inline creation recipes (for creation-oriented intents).
13. The invocation model matches: auto for Layers 1–2, user-only for utility, tool calls for MCP primitives.
14. Every Archcore document type is reachable through at least one intent or track skill.
