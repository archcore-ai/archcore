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

- **Purpose**: Translate user intent and orchestrate multi-document flows.
- **Location**: `skills/<skill-name>/SKILL.md`
- **Behavior**: Every skill is auto-invocable — Claude picks the right one from user phrasing. Skills inline per-type elicitation (questions + sections + MCP calls + relation suggestions). Per-flow logic lives as references loaded on demand under `skills/<name>/references/` or `skills/audit/lib/`.
- **Count**: **7 skills** — `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help` (per `skill-surface-collapse.adr.md`). No per-document-type skills, no track skills, no utility skills.

### Commands (user-invoked slash commands)

- **Purpose**: Accelerate common workflows with explicit user intent.
- **Note**: Claude Code and Cursor surface user-invoked workflows directly from skills. Codex CLI does not — it discovers slash commands from root-level `commands/*.md` wrappers, host-adapter shims that delegate to the matching `skills/<name>/SKILL.md`. Wrappers carry only `description:` frontmatter and a delegate instruction; they MUST NOT duplicate workflow logic. The skill remains the single behavioral source of truth across all three hosts.
- **User-facing palette (7 commands)**:
  - `/archcore:init` — First-time onboarding
  - `/archcore:capture` — Document a module/component
  - `/archcore:decide` — Record a decision (ADR/RFC); optional standard cascade
  - `/archcore:plan` — Plan a feature; pick a flow (single plan / product / sources / iso / feature)
  - `/archcore:audit` — Dashboard (default), `--deep` audit, or `--drift` detection
  - `/archcore:context` — Surface rules/decisions for a code area
  - `/archcore:help` — Command guide

### Agents (subagents)

- **Purpose**: Handle complex multi-document tasks requiring domain expertise.
- **Location**: `agents/archcore-assistant.md`, `agents/archcore-auditor.md`.
- **Behavior**: `archcore-assistant` covers all scenarios — requirements engineering, decision recording, documentation review, relation management. Restricted to MCP tools + read-only file access. `archcore-auditor` is the read-only auditor.

### Hooks (event-driven validation)

- **Purpose**: Enforce quality and the MCP-only principle.
- **Location**: `hooks/hooks.json`, `hooks/cursor.hooks.json`, `hooks/codex.hooks.json`.
- **Events**:
  - SessionStart — load project context, run staleness check
  - PreToolUse (Write|Edit) — block direct `.archcore/` writes; inject code-aligned context for source-file edits
  - PostToolUse — validate after MCP mutations; detect cascade after `update_document`; emit precision warnings

### MCP Server

- **Purpose**: Provide document CRUD and relation management tools.
- **Provider**: Archcore CLI (`archcore mcp`).
- **Registration**: shipped as `.mcp.json` (Claude Code) and `.codex.mcp.json` (Codex CLI). For Cursor, users copy `docs/cursor.mcp.example.json` into `~/.cursor/mcp.json` or `.cursor/mcp.json` (the plugin deliberately does not ship a Cursor plugin-MCP — see `cursor-mcp-architecture.adr.md`).

## Alternatives Considered

### Everything as commands

All functionality exposed as slash commands. Rejected because:

- Misses context-aware invocation — Claude wouldn't automatically know about document types.
- Users must remember and invoke every command manually.
- No model-invoked guidance.

### Everything as agents

Multiple specialized agents for each concern. Rejected because:

- Overhead for simple tasks (creating a single document doesn't need an agent).
- Agent switching adds latency and cognitive load.
- Skills handle the "teach Claude about types" use case more efficiently.

### Skills only, no commands

Rely entirely on model-invoked skills. Rejected because:

- Users sometimes want explicit control (e.g., "show the documentation dashboard now").
- Explicit modes (`--deep`, `--drift`) are easier as arguments to a slash command than as inferred intent.

### Ship MCP config inside the plugin (Cursor case)

Bundle a Cursor plugin-MCP for zero-config MCP after install. Rejected because:

- Cursor 2.5+ spawns plugin-MCPs from the plugin install dir, leaking bundled state instead of the user's workspace.
- Cursor's MCP stdio schema has no `cwd` field, so there is no way to redirect the server to the workspace from a plugin-MCP config.
- The workaround (copying `docs/cursor.mcp.example.json` into `~/.cursor/mcp.json` with `--project ${workspaceFolder}`) is documented in `plugin-development.guide.md` and `cursor-mcp-architecture.adr.md`.

## Consequences

### Positive

- Clear separation: skills route intent, agents orchestrate, hooks guard.
- Each component type used for its natural invocation model.
- Single 7-skill surface is easy to learn and easy to teach.
- All three hosts see the same 7 commands.

### Negative

- 7 SKILL.md files plus per-flow references and lib files to maintain.
- Codex CLI requires 7 thin slash-command wrappers in `commands/` to surface skills in the `/` menu (mechanical parity, no logic duplication).
- Must ensure consistency between skills, references, and agent system prompt.
- Cursor users must register MCP separately (one-time copy); `bin/session-start` mitigates with actionable guidance when the server is unreachable.
