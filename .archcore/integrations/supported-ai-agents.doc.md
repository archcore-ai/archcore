---
title: "Supported AI Agents Registry"
status: accepted
tags:
  - "integrations"
---

## Summary

Archcore integrates with 8 AI coding agents. Each agent has its own combination of hooks support (lifecycle event interception), MCP support (document tool access), and an instruction-nudge file (discovery hint).

The canonical, tool-agnostic host roster lives in the `archcore` global source (`architecture/supported-ai-hosts`). This document is the CLI's per-host integration reference: config paths, hook formats, detection, and the add-an-agent recipe.

## Agent registry

| Agent | ID | Hooks | MCP | Detection Marker | Link |
|-------|----|-------|-----|------------------|------|
| Claude Code | `claude-code` | Yes | Yes | `.claude/` dir | [docs.anthropic.com](https://docs.anthropic.com/en/docs/claude-code) |
| Cursor | `cursor` | Yes | Yes | `.cursor/` dir | [cursor.com](https://www.cursor.com/) |
| Gemini CLI | `gemini-cli` | Yes | Yes | `.gemini/` dir | [github.com/google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli) |
| OpenCode | `opencode` | No | Yes | `opencode.json` file or `.opencode/` dir | [opencode.ai](https://opencode.ai/) |
| Codex CLI | `codex-cli` | No | Yes | `.codex/` dir | [github.com/openai/codex](https://github.com/openai/codex) |
| Roo Code | `roo-code` | No | Yes | `.roo/` dir | [roocode.com](https://roocode.com/) |
| Cline | `cline` | No | Manual | `.clinerules/` dir | [cline.bot](https://cline.bot/) |
| GitHub Copilot | `copilot` | Yes | Yes | `.github/copilot-instructions.md` file | [github.com/features/copilot](https://github.com/features/copilot) |

## Integration levels

### Full integration (hooks and MCP)

Agents: Claude Code, Cursor, Gemini CLI, GitHub Copilot.

These agents support the session-start lifecycle hook and MCP tool access. `archcore init` auto-detects them, installs the hook, and writes the MCP config.

### MCP only

Agents: OpenCode, Codex CLI, Roo Code.

These agents support MCP for document tool access but have no hooks mechanism compatible with Archcore. `archcore init` and `archcore mcp install` auto-detect them and write the MCP config.

### Manual

Agent: Cline.

Cline stores its MCP config in the VS Code `globalStorage`, not in a project-level file. The user adds the Archcore MCP server through Cline's MCP settings interface. Archcore prints a hint when it detects Cline.

## Instruction nudge files

Beyond hooks and MCP, `archcore init` (opt-in) and `archcore instructions install` write a short, always-on "use Archcore" hint into each agent's instruction file, so that a CLI-only user without the Archcore plugin discovers and invokes the MCP tools. Under Tool Search, MCP tools are deferred and only names load at startup, so this nudge is the discovery trigger. The related ADR on the per-agent usage nudge records the decision.

| Agent | Instruction file(s) | Write mode |
|-------|---------------------|------------|
| Claude Code | `CLAUDE.md` **and** `AGENTS.md` | fenced upsert (both) |
| Gemini CLI | `GEMINI.md` | fenced upsert |
| Cursor, OpenCode, Codex CLI, Roo Code, Cline, GitHub Copilot | `AGENTS.md` | fenced upsert |

- Claude Code gets `CLAUDE.md` and `AGENTS.md`. Per Anthropic's documentation (code.claude.com/docs/en/memory), Claude Code reads `CLAUDE.md` natively and does not auto-read `AGENTS.md`, so `CLAUDE.md` delivers the nudge. The `AGENTS.md` block is written as well, so the repository carries the standard that the plugin and the six other hosts converge on. `CLAUDE.md` holds the full nudge body directly, with no `@import`, so the two files stay uncoupled. Claude Code loads only `CLAUDE.md`, so the second file adds no duplicated context cost.
- Fenced upsert: for every target (`CLAUDE.md`, `AGENTS.md`, `GEMINI.md`), Archcore touches only the span between `<!-- archcore:start -->` and `<!-- archcore:end -->`. User content outside the markers stays unmodified, and a second write is idempotent, so Claude Code and a co-installed `AGENTS.md` agent both writing the `AGENTS.md` block collapse into one block.
- Dedup: the six `AGENTS.md`-only agents share one file, written once. Claude Code dedupes on its own `CLAUDE.md` path, which is distinct from `AGENTS.md`, so it always runs its own write — which also refreshes the `AGENTS.md` block — regardless of the agent order.
- Reverse: `archcore instructions remove [--agent <id>]` strips the fenced block and keeps user content. Removing Claude Code alone strips only its `CLAUDE.md` block and deletes the legacy `.claude/rules/archcore.md`, leaving the shared `AGENTS.md` block to the `AGENTS.md` agents' own remove. Both run in the remove-all path.
- Deprecated: earlier CLI versions wrote an owned `.claude/rules/archcore.md`. Re-wiring now removes that file.
- Source: `@internal/agents/instructions.go`, `@cmd/instructions.go`.

## Per-agent details

`SessionStart` is the only active lifecycle event. `Stop` and `UserPromptSubmit` / `beforeSubmitPrompt` / `BeforeAgent` were removed; the related ADR records the rationale.

### Claude Code

- Config paths: `.claude/settings.json` (hooks), `.mcp.json` (MCP)
- Hook events: `SessionStart`
- Hook commands: `archcore hooks claude-code session-start`
- MCP format: standard `mcpServers` JSON (`{"command": "archcore", "args": ["mcp"]}`)
- Instruction file: `CLAUDE.md` (fenced upsert, read natively by Claude Code) and `AGENTS.md` (fenced upsert, for the plugin and other hosts)
- Source: `@internal/agents/claude_code.go`, `@cmd/hooks_claude_code.go`

### Cursor

- Config paths: `.cursor/hooks.json` (hooks), `.cursor/mcp.json` (MCP)
- Hook events: `sessionStart`
- Hook commands: `archcore hooks cursor session-start`
- MCP format: standard `mcpServers` JSON
- Instruction file: `AGENTS.md` (fenced upsert)
- Source: `@internal/agents/cursor.go`, `@cmd/hooks_cursor.go`

### Gemini CLI

- Config paths: `.gemini/settings.json` (hooks and MCP in one shared file)
- Hook events: `SessionStart`
- Hook commands: `archcore hooks gemini-cli session-start`
- MCP format: standard `mcpServers` JSON inside the same `settings.json`
- Instruction file: `GEMINI.md` (fenced upsert; Gemini CLI's default `contextFileName`)
- Source: `@internal/agents/gemini_cli.go`, `@cmd/hooks_gemini_cli.go`

### GitHub Copilot

- Config paths: `.github/hooks/archcore.json` (hooks), `.vscode/mcp.json` (MCP)
- Hook events: `sessionStart`
- Hook commands: `archcore hooks copilot session-start`
- Hook format: uses the `bash` field instead of `command` (`{"type": "command", "bash": "..."}`)
- MCP format: VS Code-style `servers` JSON with `"type": "stdio"` (`{"servers": {"archcore": {"type": "stdio", "command": "archcore", "args": ["mcp"]}}}`)
- Detection: the `.github/copilot-instructions.md` file
- Instruction file: `AGENTS.md` (fenced upsert, read natively alongside `.github/copilot-instructions.md`)
- Source: `@internal/agents/copilot.go`, `@cmd/hooks_copilot.go`

### OpenCode

- Config path: `opencode.json` (MCP)
- MCP format: `{"mcp": {"archcore": {"type": "local", "command": ["archcore", "mcp"]}}}`
- Note: OpenCode uses a different MCP JSON structure, with `type` and with `command` as an array
- Instruction file: `AGENTS.md` (fenced upsert)
- Source: `@internal/agents/opencode.go`

### Codex CLI

- Config path: `.codex/config.toml` (MCP)
- MCP format: the TOML block `[mcp_servers.archcore]` with `command` and `args`
- Note: the only agent that uses a TOML config format
- Instruction file: `AGENTS.md` (fenced upsert)
- Source: `@internal/agents/codex_cli.go`

### Roo Code

- Config path: `.roo/mcp.json` (MCP)
- MCP format: standard `mcpServers` JSON
- Note: Roo Code supports `onSave` hooks only, which do not serve lifecycle events
- Instruction file: `AGENTS.md` (fenced upsert)
- Source: `@internal/agents/roo_code.go`

### Cline

- Config path: VS Code `globalStorage`, not project-level
- MCP format: manual installation through the Cline MCP settings interface
- Hint shown: "MCP config is stored in VS Code globalStorage — add manually via Cline MCP settings"
- Instruction file: `AGENTS.md` (fenced upsert)
- Source: `@internal/agents/cline.go`

## Adding a new agent

1. Add a new `AgentID` constant in `@internal/agents/agents.go`.
2. Add `internal/agents/<name>.go` implementing the `Agent` struct with `DetectFn`, `MCPConfigPath`, `WriteMCPConfig`, and, when the agent supports hooks, `WriteHooksConfig`.
3. Set `InstructionsPath`, `WriteInstructions`, and `RemoveInstructions` with the helpers in `@internal/agents/instructions.go`: `agentsMDInstructions*` for `AGENTS.md`, `geminiInstructions*` for `GEMINI.md`, or `claudeInstructions*` for the `CLAUDE.md` and `AGENTS.md` dual write. `TestAllAgents_RequiredFields` fails while any of the three is nil.
4. Add the agent constructor to the `all` slice in `@internal/agents/agents.go`.
5. Add `internal/agents/<name>_test.go`.
6. IF the agent supports hooks, THEN add `cmd/hooks_<name>.go` with the event handlers and the install logic, register the subcommand in `newHooksCmd()` in `@cmd/hooks.go`, and add the case to `installHooksForAgent()` in the same file.
7. Add the agent to the registry table and to the instruction-nudge table in this document.
8. Update the agent-hooks integration guide and the building-the-CLI guide.
