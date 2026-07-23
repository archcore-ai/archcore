---
title: "Plugin Architecture — Seven-Skill Intent Surface"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Purpose

Define how the Archcore Plugin's components — intent skills, agents, hooks, and MCP server — compose into a unified multi-host system (Claude Code, Cursor, Codex CLI). This specification describes the runtime model, invocation pathways, data flow, component interactions, and the architectural invariants that hold everything together.

Individual component contracts are defined in dedicated specs (`skills-system.spec`, `commands-system.spec`, `agent-system.spec`, `hooks-validation-system.spec`). This document is the overarching architecture that explains *how they work together*.

The current skill surface is governed by `skill-surface-collapse.adr.md`, which supersedes the prior Layer 2 (track) tier from `intent-based-skill-architecture.adr.md`, the standalone `actualize`/`review` split from `merge-review-status-remove-graph.adr.md`, the mainstream/niche type-skill stratification from `inverted-invocation-policy.adr.md`, and the standalone `standard` and `verify` intents from `commands-system.spec.md`.

## Scope

The entire Archcore Plugin runtime: from a user message or model decision through skill activation, MCP tool calls, hook enforcement, and validation feedback. Covers the interaction between all component types.

## Authority

This specification is the architectural reference for cross-component behavior. For component-specific contracts, defer to the dedicated specs. In case of conflict, the dedicated spec wins for its own component; this spec wins for cross-component interactions.

## Subject

### System Overview

The plugin makes Archcore effortless. It exposes a flat surface of **7 auto-invocable intent skills** — every user-facing entry point is a skill in this set. Per-flow logic and per-mode logic live as reference files loaded on demand by the matching skill.

```
┌─────────────────────────────────────────────────────────────────┐
│                        User / Claude Model                      │
│                                                                 │
│  "plan this feature"   "record decision"   "/archcore:capture"  │
└──────┬──────────────────────┬──────────────────────┬────────────┘
       │                      │                      │
       ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│  INTENT SKILLS (7, auto-invocable)                              │
│                                                                 │
│  /archcore:init     /archcore:capture   /archcore:decide        │
│  /archcore:plan     /archcore:audit     /archcore:context       │
│  /archcore:help                                                 │
│                                                                 │
│  → Routes user intent to types / flows / analysis modes         │
│  → Inline per-type creation recipes                             │
│  → Per-flow references under skills/<name>/references/          │
│  → Per-mode lib under skills/audit/lib/                         │
│  → Auto-invocable in Claude Code, Cursor, Codex CLI             │
├─────────────────────────────────────────────────────────────────┤
│  MCP PRIMITIVES (INFRASTRUCTURE)                                │
│                                                                 │
│  create_document  update_document  remove_document              │
│  list_documents   get_document     search_documents             │
│  add_relation  remove_relation  list_relations  init_project    │
│                                                                 │
│  → Atomic CRUD + relations over .archcore/                      │
│  → Accepts any document type by `type` parameter                │
│  → Used by all skills, also directly callable                   │
├─────────────────────────────────────────────────────────────────┤
│  HOOKS LAYER (CROSS-CUTTING, event-driven)                      │
│                                                                 │
│  SessionStart → load context + staleness check                  │
│  PreToolUse → block direct writes + inject code-aligned context │
│  PostToolUse → validate after MCP mutations + cascade detection │
│              + precision check                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Skill Roles

| Skill | Role | Audience | Invocation |
|---|---|---|---|
| `init` | Seed `.archcore/` on first install | All users | Auto + user |
| `capture` | Document a module/component | All users | Auto + user |
| `decide` | Record decisions; optional standard cascade | All users | Auto + user |
| `plan` | Plan a feature or initiative; pick a flow | All users | Auto + user |
| `audit` | Documentation health and drift | All users | Auto + user |
| `context` | Surface rules/decisions for a code area | All users | Auto + user |
| `help` | Command catalogue, onboarding | All users | Auto + user |

Other component types:

| Component | Count | Layer | Files |
|---|---|---|---|
| Per-flow references | 4 | inside `plan` | `skills/plan/references/{product,sources,iso,feature}-flow.md` |
| Per-mode lib | 1 | inside `audit` | `skills/audit/lib/drift-detection.md` |
| Continuation references | 1 | inside `decide` | `skills/decide/references/continuations.md` |
| Shared assets | 2 | shared | `skills/_shared/{precision-rules,adr-contract}.md` |
| Agents | 2 | cross-cutting | `agents/archcore-{assistant,auditor}.{md,toml}` |
| Hooks | 6 entries | cross-cutting | `hooks/{hooks,cursor.hooks,codex.hooks}.json` |
| Bin scripts | 7 | cross-cutting | `bin/{session-start,check-archcore-write,check-code-alignment,validate-archcore,check-staleness,check-cascade,check-precision}` |
| MCP server | 1 | infra | Provided by archcore CLI |
| Codex command wrappers | 7 | host adapter | `commands/<name>.md` |

Total skills on disk: **7**. All are visible in the `/` menu of every supported host.

## Contract Surface

### Invocation Model

The plugin has four invocation paths. Every path converges on the MCP tool layer.

#### Path 1: Skill Invocation (primary user entry)

```
User types /archcore:plan auth-redesign --feature →
  Skill activates →
    Routing table picks the flow →
      Loads skills/plan/references/feature-flow.md on demand →
        Sequential document creation (question → create → relate per doc) →
          MCP tool calls → Hooks validate
```

Trigger: User explicitly invokes a `/archcore:<name>` command, or the model auto-invokes from natural language. Every skill is auto-invocable — their descriptions enumerate triggers and anti-triggers so routing is deterministic.

Example: User says "show me the docs dashboard" → model auto-invokes `audit` → loads default short mode → returns dashboard. User says "are any docs out of date?" → model auto-invokes `audit` → loads `skills/audit/lib/drift-detection.md` → runs drift protocol.

#### Path 2: Agent Delegation (complex tasks)

```
User request or Claude judgment → Agent spawned →
  Agent uses MCP tools directly → Hooks validate
```

Trigger: Claude decides the task is complex enough for a subagent, or user explicitly requests agent help.

#### Path 3: Direct MCP (any document type)

```
Model or user invokes mcp__archcore__create_document(type=<any>, ...) →
  MCP tool call → Hooks validate
```

Trigger: The model decides to create a document type that matches user intent, or the user calls MCP directly. MCP accepts every Archcore document type; no skill is required.

#### Path 4: Staleness Detection (freshness pathway)

```
Session starts → SessionStart hook → check-staleness → drift warning injected
Document updated → PostToolUse hook → check-cascade → cascade warning injected
User invokes /archcore:audit --drift → deep analysis → report + interactive fixes
```

Trigger: Automatic (hook-driven background warnings) or user-invoked (the `audit --drift` mode).

### Data Flow: Plan Skill Execution (multi-flow example)

The `plan` skill demonstrates the per-flow reference pattern:

```
1. Verify   → list_documents()                       // MCP available?
2. Route    → Pick flow from argument flag or phrasing
              (--product / --sources / --iso / --feature / single plan)
3. Load     → Read skills/plan/references/<flow>.md
4. Scope    → AskUserQuestion (if topic ambiguous)
5. Check    → list_documents(types=[...])             // detect pickup point
6. Create   → For each document in the flow:
   a. Ask   → AskUserQuestion                         // 1-2 content questions
   b. Make  → create_document(type, content, ...)     // MCP tool
      ├──→ [PostToolUse] archcore doctor              // integrity check
      ├──→ [PostToolUse] check-precision               // precision warnings
   c. Link  → add_relation(source, target, type)       // connect to chain
      └──→ [PostToolUse] archcore doctor
7. Cross    → Suggest relations to existing docs       // outside the flow
8. Report   → Summary of created docs + relations
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

The audit-drift mode operates at three depths:

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

/archcore:audit --drift (on demand):
  ├──→ list_documents + list_relations + git log
  ├──→ Code→Doc drift analysis
  ├──→ Doc→Doc cascade analysis
  ├──→ Temporal staleness analysis
  └──→ Report + interactive fixes via update_document
```

### Skill Composition

#### Intent Skill Structure

- **Frontmatter**: `name`, `description`, `argument-hint`. No `disable-model-invocation` — every skill is auto-invocable.
- **Content structure**: When to Use, Routing Table (or step sequence), Execution, Result.
- **Per-flow / per-mode content**: lives under `skills/<name>/references/<flow>.md`, `skills/<name>/lib/<mode>.md`, or `skills/_shared/` and is loaded by the SKILL.md on demand.

#### Document-type coverage

There are no per-document-type skills. Every Archcore document type is reachable through:
- An intent skill that inlines creation for that type, or
- A direct MCP `create_document(type=<any>)` call.

See `skills-system.spec.md` → "Document-type coverage" for the full mapping of each type to the intent skill(s) that reach it.

### Agent Integration

Agents are an escalation path, not the primary interface. Most documentation tasks are handled by one of the 7 intent skills.

#### When to use which path

| Scenario | Component |
|---|---|
| "Plan this feature" | `/archcore:plan` (skill; picks flow) |
| "Record this decision" | `/archcore:decide` (skill, ADR path) |
| "Draft an RFC" | `/archcore:decide` (skill, RFC path) |
| "Establish a standard" | `/archcore:decide` (skill, ADR + rule + guide continuation) |
| "Document this module" | `/archcore:capture` (skill) |
| "Show docs dashboard / counts" | `/archcore:audit` (skill, default short mode) |
| "Audit docs health" | `/archcore:audit --deep` (skill) |
| "Are any docs stale?" | `/archcore:audit --drift` (skill) |
| "What rules apply to src/X/" | `/archcore:context` (skill) |
| Run ISO requirements cascade | `/archcore:plan --iso` (skill, iso-flow reference) |
| Build full standard with pattern change | `/archcore:decide` (skill; CPAT step in continuation) |
| Create a single niche document directly | direct `mcp__archcore__create_document` |
| Restructure all auth docs with relations | `archcore-assistant` agent |
| Audit documentation quality | `archcore-auditor` agent |
| Run plugin integrity checks | `make verify` from plugin root (no skill — see `skill-surface-collapse.adr.md`) |

#### Agent tool boundaries

Both agents are restricted: no Write, Edit, or Bash on `.archcore/` files. The assistant gets all MCP tools + Read/Grep/Glob. The auditor gets only read MCP tools + Read/Grep/Glob.

### Hook Enforcement Layer

Hooks form a cross-cutting layer that enforces architectural invariants and detects documentation staleness regardless of which path initiated the operation.

| Hook | Event (Claude Code) | Event (Cursor) | Event (Codex) | Purpose |
|---|---|---|---|---|
| session-start | SessionStart | sessionStart | SessionStart | Load `.archcore/` context, check CLI, detect code-doc drift |
| check-archcore-write | PreToolUse (`Write\|Edit`) | preToolUse (`Write`) | PreToolUse (`Write\|Edit\|apply_patch`) | Block direct `.archcore/*.md` writes |
| check-code-alignment | PreToolUse (`Write\|Edit`) | preToolUse (`Write`) | PreToolUse (`Write\|Edit\|apply_patch`) | Inject relevant `.archcore/` context for source-file edits |
| validate-archcore | PostToolUse (MCP mutations) | afterMCPExecution | PostToolUse (MCP mutations) | Primary validation after MCP mutations |
| check-cascade | PostToolUse (`update_document`) | afterMCPExecution (filtered) | PostToolUse (`update_document`) | Cascade staleness detection via relation graph |
| check-precision | PostToolUse (`create_document\|update_document`) | afterMCPExecution (filtered) | PostToolUse (`create_document\|update_document`) | Forbidden vagueness + section + stub-length warnings |

### Cross-Layer Interaction Patterns

#### Pattern 1: Skill → Flow Reference → MCP → Hook

The skill loads a per-flow reference and creates documents sequentially via MCP. Hooks validate after each mutation.

Example: `/archcore:plan --feature` → loads `skills/plan/references/feature-flow.md` → creates prd, spec, plan, task-type with `implements` relations.

#### Pattern 2: Skill → Inline Creation → MCP → Hook

The skill creates a single document inline using its routing-table-selected recipe.

Example: `/archcore:decide` with a finalized decision → inline ADR recipe → creates one adr.

#### Pattern 3: Model → Auto-Routed Skill → MCP → Hook

Claude auto-activates a skill from conversation context.

Example: User discusses a decision → Claude activates `decide` → routes to ADR creation via MCP.

#### Pattern 4: Agent → MCP → Hook (complex flow)

An agent makes multiple MCP calls autonomously. Hooks validate each one.

#### Pattern 5: Write → Hook → Block → MCP (correction flow)

When any component attempts a direct `.archcore/` write, PreToolUse blocks it. Claude retries via MCP.

#### Pattern 6: Update → Hook → Cascade Warning (freshness flow)

When a document is updated via MCP, the cascade hook detects potentially stale dependents and warns.

Example: User updates a PRD → check-cascade fires → finds plan that `implements` this PRD → injects "plan may need review" warning.

## Normative Behavior

- All document operations MUST flow through MCP tools. This is the **MCP-only principle** — the single most important architectural invariant.
- All 7 intent skills MUST be auto-invocable. No skill carries `disable-model-invocation`.
- Skills MUST contain explicit routing tables with bounded decision branches (flow-style skills like `init` may substitute a numbered step sequence).
- Skills MUST default to minimum viable path, offering expansion via one scope question.
- Skills that create documents MUST be self-contained with inline creation recipes (or load a per-flow reference). The plan skill MUST hold per-flow logic in `skills/plan/references/<flow>.md`.
- Skills MUST NOT instruct direct file writes to `.archcore/`. They reference MCP tools by exact name.
- Agents MUST use MCP tools exclusively for `.archcore/` operations.
- Hooks MUST fire for every relevant tool call, regardless of which path initiated it.
- The PreToolUse hook MUST block `.archcore/**/*.md` writes with exit code 2.
- PostToolUse validation hooks MUST run `archcore doctor` after every MCP document mutation.
- PostToolUse cascade hook MUST run after `update_document` to detect relation-graph staleness.
- PostToolUse precision hook MUST run after `create_document` and `update_document`.
- No PostToolUse hook MUST be registered for `Write|Edit` — PreToolUse already blocks `.archcore/*.md` writes before they succeed.
- SessionStart MUST include staleness check after context loading.

## Constraints

- Exactly 7 visible intent skills. Adding an eighth requires a new ADR.
- Maximum 2 agents. New agents require an ADR.
- Hooks must complete within their timeout (PreToolUse: 1s, PostToolUse: 3s).
- Skill files must not exceed 300 lines.
- Per-flow reference files must not exceed 200 lines.

## Invariants

- Every user-facing entry point maps to one of the 7 intent skills.
- Every document mutation passes through the MCP tool layer.
- Every MCP mutation triggers PostToolUse validation.
- Every `update_document` triggers cascade detection in addition to validation.
- Every `create_document` and `update_document` triggers precision check.
- Every direct `.archcore/*.md` write attempt is blocked by PreToolUse.
- Every session starts with project context loaded and staleness check run (or a warning if the CLI is missing).
- Skills inline per-type elicitation; this duplication is intentional and accepted per `skills-system.spec.md` to keep each entry point self-contained.
- Agents never have Write/Edit/Bash access to `.archcore/` files.
- No skill carries `disable-model-invocation`. Every skill is auto-invocable.
- Staleness detection never modifies documents autonomously — only `/archcore:audit --drift` modifies, and only with user confirmation.
- Every Archcore document type is reachable through at least one intent skill (or directly via MCP).

## Error Handling

- **MCP server unavailable**: All skills inform the user with install/init instructions. Hooks degrade gracefully.
- **Duplicate document**: `create_document` fails. Skills suggest alternative filename.
- **Intent routing ambiguous**: Skill asks one scope-confirmation question. If still ambiguous, falls back to the `capture` skill (most general).
- **Flow interrupted mid-cascade**: `plan` skill detects existing documents via `list_documents` and resumes from the next step.
- **Hook timeout**: PostToolUse fails open. PreToolUse fail-closed behavior handled by the host.
- **Agent exceeds turn limit**: Agent returns partial results. User can re-invoke or continue manually.
- **Git unavailable for staleness**: SessionStart skips staleness check. `/archcore:audit --drift` skips code-drift analysis but performs cascade and temporal.

## Conformance

The plugin architecture conforms to this specification if:

1. All document operations flow through MCP tools.
2. The skill surface is exactly 7 skills: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`. All are auto-invocable (no `disable-model-invocation`).
3. PreToolUse hook blocks 100% of direct `.archcore/**/*.md` writes.
4. PostToolUse validation fires after every MCP document mutation.
5. PostToolUse cascade detection fires after every `update_document`.
6. PostToolUse precision check fires after every `create_document` and `update_document`.
7. No PostToolUse hook is registered for `Write|Edit`.
8. SessionStart includes staleness check after context loading.
9. Per-flow logic for multi-document cascades lives under `skills/plan/references/<flow>.md`.
10. Drift-mode logic for `audit` lives under `skills/audit/lib/drift-detection.md`.
11. Continuation logic for `decide` (ADR → CPAT → rule → guide) lives under `skills/decide/references/continuations.md`.
12. Every Archcore document type is reachable through at least one intent skill.
