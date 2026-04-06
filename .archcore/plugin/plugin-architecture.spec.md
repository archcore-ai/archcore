---
title: "Plugin Architecture — Skills, Tracks, Workflows"
status: draft
tags:
  - "architecture"
  - "plugin"
  - "skills"
---

## Purpose

Define how the Archcore Claude Plugin's components — skills, tracks, workflows, agents, hooks, and MCP server — compose into a unified system. This specification describes the invocation model, data flow, component interactions, and the architectural invariants that hold everything together.

Individual component contracts are defined in dedicated specs (skills-system.spec, commands-system.spec, agent-system.spec, hooks-validation-system.spec). This document is the overarching architecture that explains _how they work together_.

## Scope

The entire Archcore Claude Plugin runtime: from a user message or model decision through skill activation, MCP tool calls, hook enforcement, and validation feedback. Covers the interaction between all component types, not the internal details of each.

## Authority

This specification is the architectural reference for cross-component behavior. For component-specific contracts, defer to the dedicated specs. In case of conflict, the dedicated spec wins for its own component; this spec wins for cross-component interactions.

## Subject

### System Overview

The plugin is a Claude Code plugin that makes Archcore effortless. It teaches Claude about 18 document types, provides multi-document workflows, enforces MCP-only operations, and assists with complex documentation tasks.

```
┌─────────────────────────────────────────────────────────────┐
│                     User / Claude Model                     │
│                                                             │
│  "record this decision"          "/archcore:adr topic"      │
│  (model-invoked)                 (user-invoked)             │
└──────────┬──────────────────────────────┬───────────────────┘
           │                              │
           ▼                              ▼
┌─────────────────────┐    ┌─────────────────────────────────┐
│  Document-Type Skill │    │  Track / Workflow / Type Skill  │
│  (18 skills)         │    │  (6 tracks + 3 workflows)      │
│                      │    │                                 │
│  Knowledge about     │    │  Multi-step orchestration       │
│  one document type   │    │  with sequential questions      │
└──────────┬───────────┘    └──────────────┬──────────────────┘
           │                               │
           │    ┌──────────────────────┐   │
           │    │  Agents (optional)   │   │
           │    │  assistant / auditor │   │
           │    └──────────┬───────────┘   │
           │               │               │
           ▼               ▼               ▼
┌─────────────────────────────────────────────────────────────┐
│                    MCP Tool Layer                            │
│                                                             │
│  create_document  update_document  remove_document          │
│  add_relation     remove_relation  list_relations           │
│  list_documents   get_document                              │
└──────────┬──────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Hooks Layer                               │
│                                                             │
│  PreToolUse ──→ Block direct Write/Edit to .archcore/*.md   │
│  PostToolUse ──→ archcore validate after mutations          │
│  SessionStart ──→ Load project context                      │
└─────────────────────────────────────────────────────────────┘
```

### Component Types and Roles

| Component            | Count     | Role                                                          | Invocation                                               |
| -------------------- | --------- | ------------------------------------------------------------- | -------------------------------------------------------- |
| Document-type skills | 18        | Teach Claude about one document type each                     | Model-invoked (auto) + user-invoked (`/archcore:<type>`) |
| Track skills         | 6         | Orchestrate multi-document flows with relations               | User-only (`/archcore:<track>`)                          |
| Workflow skills      | 3         | Utility workflows (create wizard, review, status)             | User-only (`/archcore:<name>`)                           |
| Agents               | 2         | Complex multi-document tasks (assistant) and audits (auditor) | Model-invoked or user-delegated                          |
| Hooks                | 4 entries | Enforce MCP-only principle and validate mutations             | Event-driven (automatic)                                 |
| MCP server           | 1         | Document CRUD, relations, validation                          | Tool calls from skills/agents/model                      |
| Bin scripts          | 3         | Hook handlers (session-start, check-write, validate)          | Called by hooks                                          |

## Contract Surface

### Invocation Model

The plugin has three invocation paths. Every path converges on the MCP tool layer.

#### Path 1: Model-Invoked Skill (automatic)

```
User says something → Claude matches context to skill description →
  Skill activates → Claude follows skill guidance →
    MCP tool calls → Hooks validate
```

Trigger: Claude's model recognizes that the conversation matches a skill's `description` field. Only document-type skills (18) are model-invoked.

Example: User says "let's record the decision to use PostgreSQL" → Claude activates `skills/adr/SKILL.md` → follows the Quick Create flow → calls `create_document(type="adr")`.

#### Path 2: User-Invoked Skill (explicit)

```
User types /archcore:<name> → Skill activates →
  Claude follows skill instructions →
    MCP tool calls → Hooks validate
```

Trigger: User explicitly invokes via slash command. All 27 skills are user-invokable. Track and workflow skills (9) are user-only (`disable-model-invocation: true`).

Example: User types `/archcore:architecture-track auth system` → Claude follows `skills/architecture-track/SKILL.md` → creates ADR, spec, plan in sequence with relations.

#### Path 3: Agent Delegation (complex tasks)

```
User request or Claude judgment → Agent spawned →
  Agent uses MCP tools directly → Hooks validate
```

Trigger: Claude decides the task is complex enough to warrant a subagent, or user explicitly requests agent help. Agents have their own tool allowlists.

Example: User asks "restructure all auth documentation with proper relations" → Claude spawns `archcore-assistant` → agent reads existing docs, plans changes, creates/updates documents via MCP.

### Data Flow: Document Creation

Every document creation, regardless of invocation path, follows this flow:

```
1. Check ──→ list_documents(types=[...])     // prevent duplicates
2. Ask   ──→ AskUserQuestion                 // gather context (1-3 questions)
3. Create ──→ create_document(type, ...)     // MCP tool
   │
   ├──→ [PreToolUse] no-op (MCP, not Write)
   ├──→ [MCP server] validates, applies template, writes file, updates manifest
   └──→ [PostToolUse] archcore validate      // integrity check
4. Relate ──→ add_relation(source, target, type)  // link to existing docs
   │
   └──→ [PostToolUse] archcore validate      // integrity check
```

### Data Flow: Direct Write Interception

When Claude (or a skill/agent) attempts to Write/Edit a `.archcore/*.md` file directly:

```
1. Claude calls Write/Edit with .archcore/ path
2. [PreToolUse] bin/check-archcore-write
   ├──→ Extracts file_path from stdin JSON
   ├──→ Matches .archcore/**/*.md pattern
   ├──→ Exit 2 + stderr message → BLOCKED
   └──→ Claude receives feedback: "Use MCP tools instead"
3. Claude retries via create_document or update_document
```

### Skill Taxonomy

Skills are organized into three groups with distinct behavior:

#### Document-Type Skills (18)

- **Frontmatter**: `name`, `description`, `argument-hint` — no `disable-model-invocation`
- **Dual invocation**: model auto-activates based on context; user invokes for quick create
- **Content structure**: When to Use, Quick Create, Relations — concise guidance around the template
- **Does NOT contain**: full template content, implementation details, multi-document flows

#### Track Skills (6)

- **Frontmatter**: adds `disable-model-invocation: true`
- **User-only**: explicit `/archcore:<track-name> <topic>` invocation
- **Content structure**: sequential steps (Check → Scope → Create doc 1 → Create doc 2 → ... → Cross-relate)
- **Defines**: document sequence, relation chain, scope detection (resume from existing docs)
- **Does NOT contain**: document-type guidance (delegates to type skills conceptually)

| Track              | Flow                          | Relation Chain                                                          |
| ------------------ | ----------------------------- | ----------------------------------------------------------------------- |
| product-track      | idea → prd → plan             | each `implements` previous                                              |
| sources-track      | mrd → brd → urd               | `related` peers                                                         |
| iso-track          | brs → strs → syrs → srs       | each `implements` previous                                              |
| architecture-track | adr → spec → plan             | spec `implements` adr, plan `implements` spec                           |
| standard-track     | adr → rule → guide            | rule `implements` adr, guide `related` rule                             |
| feature-track      | prd → spec → plan → task-type | spec `implements` prd, plan `implements` spec, task-type `related` plan |

#### Workflow Skills (3)

- **Frontmatter**: adds `disable-model-invocation: true`
- **User-only**: explicit `/archcore:<name>` invocation
- **Purpose**: utility operations, not document-type-specific

| Workflow | Purpose                     | Key Behavior                                       |
| -------- | --------------------------- | -------------------------------------------------- |
| create   | Interactive creation wizard | Infers type from context, asks focused questions   |
| review   | Documentation health review | Produces actionable findings, not verbose analysis |
| status   | Documentation dashboard     | Compact, scannable output of counts and coverage   |

### Agent Integration

Agents are an escalation path, not the primary interface. Most documentation tasks are handled by skills directly.

#### When Claude uses agents vs. skills

| Scenario                                     | Component                                   |
| -------------------------------------------- | ------------------------------------------- |
| Create one document                          | Document-type skill (model or user invoked) |
| Create 2-4 related documents in a known flow | Track skill (user invoked)                  |
| Create documents with complex dependencies   | archcore-assistant agent                    |
| Restructure existing documentation           | archcore-assistant agent                    |
| Audit documentation health                   | archcore-auditor agent                      |
| Quick status check                           | status workflow skill                       |

#### Agent tool boundaries

Both agents are restricted: no Write, Edit, or Bash on `.archcore/` files. The assistant gets all 8 MCP tools + Read/Grep/Glob. The auditor gets only 3 read MCP tools + Read/Grep/Glob. This is enforced by the agent's `tools` allowlist in the definition file, not by hooks.

### Hook Enforcement Layer

Hooks form the bottom layer that enforces architectural invariants regardless of which component initiated the operation.

| Hook                 | Event                       | Fires When                              | Purpose                                         |
| -------------------- | --------------------------- | --------------------------------------- | ----------------------------------------------- |
| session-start        | SessionStart                | Session begins/resumes                  | Load .archcore/ context, check CLI availability |
| check-archcore-write | PreToolUse (Write\|Edit)    | Before any file write                   | Block direct .archcore/\*.md writes             |
| validate-archcore #1 | PostToolUse (Write\|Edit)   | After any file write                    | Defense-in-depth validation                     |
| validate-archcore #2 | PostToolUse (MCP mutations) | After create/update/remove/relation ops | Primary validation after MCP mutations          |

Hook enforcement is independent of the invocation path. Whether a skill, agent, or the model itself initiates an operation, hooks apply uniformly.

### Cross-Component Interaction Patterns

#### Pattern 1: Skill → MCP → Hook (standard flow)

The most common pattern. A skill guides Claude to call MCP tools. Hooks validate after each mutation.

#### Pattern 2: Track Skill → Multiple (Skill → MCP → Hook)

A track skill orchestrates N iterations of Pattern 1, with `add_relation` calls between documents. Each step asks questions before creating.

#### Pattern 3: Agent → MCP → Hook (complex flow)

An agent makes multiple MCP calls autonomously. Hooks validate each one. The agent may read existing documents between mutations to inform decisions.

#### Pattern 4: Model → Write → Hook → Block → MCP (correction flow)

When Claude attempts a direct write, the PreToolUse hook blocks it and provides feedback. Claude then retries via the correct MCP tool. This is the self-correcting enforcement loop.

## Normative Behavior

- All document operations MUST flow through MCP tools. This is the **MCP-only principle** — the single most important architectural invariant.
- Skills MUST NOT instruct direct file writes to `.archcore/`. They reference MCP tools by exact name.
- Track skills MUST create documents sequentially, asking questions before each step. They MUST NOT batch-create all documents at once.
- Track skills MUST add relations between created documents as defined in the relation chain.
- Track skills MUST check for existing documents and resume from where the chain left off.
- Agents MUST use MCP tools exclusively for `.archcore/` operations. Their tool allowlists enforce this.
- Hooks MUST fire for every relevant tool call, regardless of which component initiated it.
- The PreToolUse hook MUST block `.archcore/**/*.md` writes with exit code 2 and redirect to MCP tools.
- PostToolUse hooks MUST run `archcore validate` after every document mutation.
- When multiple skills could apply, Claude SHOULD prefer the most specific one (e.g., `adr` skill over `create` wizard).

## Constraints

- Maximum 27 skills total (18 type + 6 track + 3 workflow). New skills require updating this spec.
- Maximum 2 agents. New agents require an ADR.
- Hooks must complete within their timeout (PreToolUse: 1s, PostToolUse: 3s).
- Skills must not exceed 500 lines (type skills) or 200 lines (workflow skills).
- No component may reference internal CLI implementation details — only the MCP tool interface.

## Invariants

- Every document mutation passes through the MCP tool layer.
- Every MCP mutation triggers PostToolUse validation.
- Every direct `.archcore/*.md` write attempt is blocked by PreToolUse.
- Every session starts with project context loaded (or a warning if CLI is missing).
- Every track skill produces a chain of related documents, not isolated documents.
- Document-type skills and track skills never duplicate content — type skills own type guidance, track skills own flow definition.
- Agents never have Write/Edit/Bash access to `.archcore/` files.

## Error Handling

- **MCP server unavailable**: Skills and agents inform the user with install/init instructions. Hooks degrade gracefully (session-start warns, validate-archcore skips).
- **Duplicate document**: `create_document` fails. Skills guide Claude to suggest an alternative filename.
- **Track interrupted mid-flow**: Track skills detect existing documents via `list_documents` and resume from the next step.
- **Hook timeout**: Claude Code enforces timeouts. If a hook times out, the operation proceeds (fail-open for PostToolUse) or is blocked (fail-closed for PreToolUse timeout is handled by Claude Code).
- **Agent exceeds turn limit**: Agent returns partial results. User can re-invoke or continue manually.

## Conformance

The plugin architecture conforms to this specification if:

1. All document operations in skills, agents, and model behavior flow through MCP tools
2. The PreToolUse hook blocks 100% of direct `.archcore/**/*.md` writes
3. PostToolUse validation fires after every MCP document mutation
4. Track skills create documents sequentially with proper relations
5. Agents are restricted to their declared tool allowlists
6. No component duplicates another's responsibility (type guidance in type skills, flow definition in track skills, enforcement in hooks)
7. The invocation model matches: model-invoked for type skills, user-only for tracks and workflows, delegated for agents
