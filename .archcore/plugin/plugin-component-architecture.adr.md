---
title: "Plugin Component Architecture"
status: accepted
tags:
  - "architecture"
  - "plugin"
---

## Context

Claude Code plugins support multiple component types: skills (model-invoked), commands (user-invoked slash commands), agents (subagents with restricted tools), hooks (event handlers), bin (executables), and settings. We need to decide how to map Archcore's needs to these capabilities, creating a clear separation of concerns.

The plugin must cover: document type guidance, workflow acceleration, complex documentation assistance, and quality enforcement.

## Decision

Each plugin capability maps to a specific Claude Code component type based on its invocation model and complexity:

### Skills (model-invoked, context-aware)

- **Purpose**: Teach Claude about each of the 18 document types
- **Location**: `skills/<type-name>/SKILL.md`
- **Behavior**: Claude activates skills automatically when the conversation context matches a document type. Skills provide type overview, required sections, best practices, common mistakes, relation guidance, and example MCP workflows.
- **Count**: 18 skills (one per document type)

### Commands (user-invoked slash commands)

- **Purpose**: Accelerate common workflows with explicit user intent
- **Location**: `commands/<name>.md`
- **Commands**:
  - `/archcore:create` — Interactive creation wizard
  - `/archcore:review` — Documentation health review
  - `/archcore:status` — Dashboard of documents, relations, tags
  - Type shortcuts — `/archcore:adr`, `/archcore:prd`, `/archcore:rule`, etc.

### Agent (universal subagent)

- **Purpose**: Handle complex multi-document tasks requiring domain expertise
- **Location**: `agents/archcore-assistant.md`
- **Behavior**: One universal agent covering all scenarios — requirements engineering, decision recording, documentation review, relation management. Restricted to MCP tools + read-only file access.

### Hooks (event-driven validation)

- **Purpose**: Enforce quality and the MCP-only principle
- **Location**: `hooks/hooks.json`
- **Events**:
  - SessionStart — load project context (existing)
  - PreToolUse (Write|Edit) — block direct `.archcore/` writes, redirect to MCP
  - PostToolUse ��� validate `.archcore/` files after changes

### MCP Server (existing)

- **Purpose**: Provide document CRUD and relation management tools
- **Location**: `.mcp.json`
- **Behavior**: Delegates to `archcore mcp` CLI command. No changes needed.

## Alternatives Considered

### Everything as commands

All functionality exposed as slash commands. Rejected because:

- Misses context-aware invocation — Claude wouldn't automatically know about document types
- Users must remember and invoke every command manually
- No model-invoked guidance

### Everything as agents

Multiple specialized agents for each concern. Rejected because:

- Overhead for simple tasks (creating a single document doesn't need an agent)
- Agent switching adds latency and cognitive load
- Skills handle the "teach Claude about types" use case more efficiently

### Skills only, no commands

Rely entirely on model-invoked skills. Rejected because:

- Users sometimes want explicit control (e.g., "run a documentation review now")
- Status dashboards are better as explicit commands than implicit suggestions

## Consequences

### Positive

- Clear separation: skills teach, commands act, agent orchestrates, hooks guard
- Each component type used for its natural invocation model
- Skills provide passive knowledge; commands provide active workflows
- Single agent simplifies maintenance while covering all scenarios

### Negative

- 18 skill files to maintain (one per type)
- Multiple command files to maintain
- Must ensure consistency between skills, commands, and agent system prompt
