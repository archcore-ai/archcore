---
title: "Plugin Component Architecture"
status: accepted
tags:
  - "architecture"
  - "plugin"
---

## Context

A host plugin system supports several component types — skills that the model invokes, commands the user invokes, agents with restricted tools, hooks that handle events, executables, and settings — and Archcore had to map its needs onto them with a clear separation of concerns. The plugin must cover four capabilities: document type guidance, workflow acceleration, complex documentation assistance, and quality enforcement.

## Decision

Map each plugin capability to the component type whose invocation model matches it: skills route intent, command wrappers surface skills on hosts that need them, agents orchestrate complex work, hooks guard quality, and the MCP server provides document operations.

**Skills — model-invoked and context-aware.** They translate user intent and orchestrate multi-document flows, living at `skills/<skill-name>/SKILL.md`. Every skill is auto-invocable, so the model picks the right one from user phrasing, and each inlines its per-type elicitation — questions, sections, MCP calls, relation suggestions — with per-flow logic loaded on demand from `skills/<name>/references/` or `skills/audit/lib/`. There are four: `init`, `plan`, `document`, `review`, per `four-command-palette.adr`, with a non-palette gated track layer per `track-layer.spec`. There are no per-document-type skills, no track skills, and no utility skills.

**Commands — user-invoked slash commands.** Claude Code, Cursor, and Copilot CLI surface user-invoked workflows directly from skills. Codex CLI does not, and discovers slash commands from `commands/*.md` wrappers — host-adapter shims carrying only `description:` frontmatter and a delegate instruction. A wrapper MUST NOT duplicate workflow logic; the skill stays the single behavioral source of truth on every host. Copilot loads the same wrappers behind its skills. The user-facing palette is the four commands `/archcore:init`, `/archcore:plan`, `/archcore:document`, and `/archcore:review`.

**Agents — subagents.** They handle complex multi-document tasks needing domain expertise, at `agents/archcore-assistant.md` and `agents/archcore-auditor.md`. The assistant covers requirements engineering, decision recording, documentation review, and relation management, restricted to MCP tools plus read-only file access; the auditor is its read-only counterpart.

**Hooks — event-driven validation.** They enforce quality and the MCP-only principle, configured in `hooks/hooks.json`, `hooks/cursor.hooks.json`, `hooks/codex.hooks.json`, and `hooks/copilot.hooks.json`. SessionStart loads project context and runs the staleness check; PreToolUse blocks a direct `.archcore/` write and injects code-aligned context for a source-file edit; PostToolUse validates after an MCP mutation, detects cascade after `update_document`, and emits precision warnings.

**MCP server.** It provides document CRUD and relation management, supplied by the Archcore CLI through `archcore mcp`. Registration is plugin-shipped for Claude Code through the manifest key pointing at `.claude.mcp.json`, and for Codex CLI through `.codex-plugin/plugin.json` pointing at `.codex.mcp.json`. Cursor and Copilot get no plugin-shipped MCP: `cursor-mcp-architecture.adr` and `copilot-mcp-architecture.adr` record why, and their users register the server per project or per user.

## Alternatives Considered

1. **Expose everything as commands** — rejected because it misses context-aware invocation, so the model would not know about document types on its own, users would have to remember and invoke every command manually, and no model-invoked guidance would exist.
2. **Expose everything as agents, one specialized agent per concern** — rejected because it adds overhead for simple tasks, since creating a single document needs no agent, because agent switching adds latency and cognitive load, and because skills handle the "teach the model about types" case more directly.
3. **Ship skills only, with no command wrappers** — rejected because users sometimes want explicit control, and because an explicit mode such as `--deep` or `--drift` reads more clearly as a slash-command argument than as inferred intent. Codex additionally has no other way to surface a skill in its `/` menu.
4. **Ship an MCP config inside the plugin for Cursor** — rejected because Cursor 2.5 and later spawn a plugin MCP from the plugin install directory, leaking bundled state instead of the user's workspace, and because Cursor's MCP stdio schema has no `cwd` field, so nothing can redirect the server to the workspace from a plugin-MCP config. The documented path is the user- or project-level config carrying `--project ${workspaceFolder}`.

## Consequences

- Responsibilities separate cleanly: skills route intent, agents orchestrate, hooks guard, and MCP performs operations.
- Each component type is used for the invocation model it was designed for.
- A single four-command surface is quick to learn and to teach, and all four hosts see the same four commands.
- Tradeoff: 4 `SKILL.md` files (plus non-palette track files) plus the per-flow references and lib files require maintenance, and consistency must hold between the skills, those references, and the agent system prompts.
- Tradeoff: Codex CLI requires 4 thin wrappers in `commands/` to surface the skills, which is mechanical parity rather than logic duplication, and Copilot needs the same wrappers plus an explicit manifest pointer.
- Tradeoff: Cursor and Copilot users must register MCP separately. `bin/session-start` mitigates this with actionable guidance when the server is unreachable, and `host-wiring-parity.adr` reduces it to a confirmed step inside `/archcore:init`.

## Superseded when

- Every supported host surfaces skills directly in its `/` menu, which would remove the `commands/` wrapper layer entirely.
- Every supported host launches a plugin-shipped MCP server in the user's project, which would let the plugin ship one registration for all four.
