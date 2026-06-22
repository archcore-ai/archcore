---
title: "Supported AI Agents Registry"
status: accepted
tags:
  - "integrations"
---

## Summary

Archcore integrates with 8 AI coding agents. Each agent has a unique combination of hooks support (lifecycle event interception), MCP support (document tool access), and an instruction-nudge file (discovery hint). The canonical, tool-agnostic host roster lives in the `archcore` global source (`architecture/supported-ai-hosts`); this document is the CLI's per-host integration reference (config paths, hook formats, detection, and the add-an-agent recipe).

## Agent Registry

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

## Integration Levels

### Full Integration (Hooks + MCP)

Agents: **Claude Code**, **Cursor**, **Gemini CLI**, **GitHub Copilot**

These agents support the session-start lifecycle hook and MCP tool access. Archcore auto-detects them, installs the hook, and writes MCP config during `archcore init`.

### MCP Only

Agents: **OpenCode**, **Codex CLI**, **Roo Code**

These agents support MCP for document tool access but do not have a hooks mechanism compatible with archcore. They are auto-detected and receive MCP config during `archcore init` or `archcore mcp install`.

### Manual

Agent: **Cline**

Cline stores MCP config in VS Code `globalStorage`, not in a project-level file. Users must add the archcore MCP server manually via Cline's MCP settings UI. Archcore prints a hint when Cline is detected.

## Instruction Nudge Files

Beyond hooks and MCP, `archcore init` (opt-in) and `archcore instructions install` write a short, always-on "use Archcore" hint into each agent's instruction file so that CLI-only users (no Archcore plugin) discover and invoke the MCP tools. Under Tool Search, MCP tools are deferred — only names load at startup — so this nudge is the discovery trigger. See [Write a Usage-Nudge Instruction File per Agent on init](instruction-nudge-on-init.adr.md).

| Agent | Instruction file | Write mode |
|-------|------------------|------------|
| Claude Code | `.claude/rules/archcore.md` | owned (whole file) |
| Gemini CLI | `GEMINI.md` | fenced upsert |
| Cursor, OpenCode, Codex CLI, Roo Code, Cline, GitHub Copilot | `AGENTS.md` | fenced upsert |

- **Owned files** — archcore owns the whole file and overwrites it freely. Claude Code does not read `AGENTS.md`; it reads `.claude/rules/*.md` (auto-loaded at CLAUDE.md priority), so it gets a dedicated file.
- **Fenced upsert** — for shared files, archcore only ever touches the span between `<!-- archcore:start -->` and `<!-- archcore:end -->`. User content outside the markers is never modified, and writing twice is idempotent.
- **Dedup** — the six `AGENTS.md` agents share one file, written once.
- **Reverse** — `archcore instructions remove [--agent <id>]` strips the fenced block (keeping user content) or deletes the owned file.
- **Source:** `internal/agents/instructions.go`, `cmd/instructions.go`.

## Per-Agent Details

Only the `SessionStart` lifecycle event is active. `Stop` and `UserPromptSubmit`/`beforeSubmitPrompt`/`BeforeAgent` were removed — see [Disable Stop and Prompt Hooks ADR](disable-stop-and-prompt-hooks.adr.md).

### Claude Code

- **Config paths:** `.claude/settings.json` (hooks), `.mcp.json` (MCP)
- **Hook events:** `SessionStart`
- **Hook commands:** `archcore hooks claude-code session-start`
- **MCP format:** Standard `mcpServers` JSON (`{"command": "archcore", "args": ["mcp"]}`)
- **Instruction file:** `.claude/rules/archcore.md` (owned)
- **Source:** `internal/agents/claude_code.go`, `cmd/hooks_claude_code.go`

### Cursor

- **Config paths:** `.cursor/hooks.json` (hooks), `.cursor/mcp.json` (MCP)
- **Hook events:** `sessionStart`
- **Hook commands:** `archcore hooks cursor session-start`
- **MCP format:** Standard `mcpServers` JSON
- **Instruction file:** `AGENTS.md` (fenced upsert)
- **Source:** `internal/agents/cursor.go`, `cmd/hooks_cursor.go`

### Gemini CLI

- **Config paths:** `.gemini/settings.json` (hooks and MCP, shared file)
- **Hook events:** `SessionStart`
- **Hook commands:** `archcore hooks gemini-cli session-start`
- **MCP format:** Standard `mcpServers` JSON (inside same `settings.json`)
- **Instruction file:** `GEMINI.md` (fenced upsert — Gemini CLI's default `contextFileName`)
- **Source:** `internal/agents/gemini_cli.go`, `cmd/hooks_gemini_cli.go`

### GitHub Copilot

- **Config paths:** `.github/hooks/archcore.json` (hooks), `.vscode/mcp.json` (MCP)
- **Hook events:** `sessionStart`
- **Hook commands:** `archcore hooks copilot session-start`
- **Hook format:** Uses `bash` field instead of `command` (`{"type": "command", "bash": "..."}`)
- **MCP format:** VS Code-style `servers` JSON with `"type": "stdio"` (`{"servers": {"archcore": {"type": "stdio", "command": "archcore", "args": ["mcp"]}}}`)
- **Detection:** `.github/copilot-instructions.md` file
- **Instruction file:** `AGENTS.md` (fenced upsert — read natively alongside `.github/copilot-instructions.md`)
- **Source:** `internal/agents/copilot.go`, `cmd/hooks_copilot.go`

### OpenCode

- **Config path:** `opencode.json` (MCP)
- **MCP format:** `{"mcp": {"archcore": {"type": "local", "command": ["archcore", "mcp"]}}}`
- **Note:** OpenCode uses a different MCP JSON structure with `type` and `command` as array
- **Instruction file:** `AGENTS.md` (fenced upsert)
- **Source:** `internal/agents/opencode.go`

### Codex CLI

- **Config path:** `.codex/config.toml` (MCP)
- **MCP format:** TOML block `[mcp_servers.archcore]` with `command` and `args`
- **Note:** Only agent using TOML config format
- **Instruction file:** `AGENTS.md` (fenced upsert)
- **Source:** `internal/agents/codex_cli.go`

### Roo Code

- **Config path:** `.roo/mcp.json` (MCP)
- **MCP format:** Standard `mcpServers` JSON
- **Note:** Roo Code only supports `onSave` hooks, not useful for lifecycle events
- **Instruction file:** `AGENTS.md` (fenced upsert)
- **Source:** `internal/agents/roo_code.go`

### Cline

- **Config path:** VS Code `globalStorage` (not project-level)
- **MCP format:** Manual installation via Cline MCP settings UI
- **Hint shown:** "MCP config is stored in VS Code globalStorage — add manually via Cline MCP settings"
- **Instruction file:** `AGENTS.md` (fenced upsert)
- **Source:** `internal/agents/cline.go`

## Adding a New Agent

1. **Define the ID** — Add a new `AgentID` constant in `internal/agents/agents.go`
2. **Create agent file** — Add `internal/agents/<name>.go` implementing the `Agent` struct with `DetectFn`, `MCPConfigPath`, `WriteMCPConfig`, and optionally `WriteHooksConfig`
3. **Wire the instruction nudge** — Set `InstructionsPath`, `WriteInstructions`, and `RemoveInstructions`, pointing at the right shared target via the helpers in `internal/agents/instructions.go` (`agentsMDInstructions*` for `AGENTS.md`, or a dedicated owned/fenced target). `TestAllAgents_RequiredFields` fails if any are nil.
4. **Register** — Add the agent constructor to the `all` slice in `internal/agents/agents.go`
5. **Add tests** — Create `internal/agents/<name>_test.go`
6. **If hooks supported** — Create `cmd/hooks_<name>.go` with event handlers and install logic; register the subcommand in `cmd/hooks.go:newHooksCmd()`; add the case to `installHooksForAgent()` in `cmd/hooks.go`
7. **Update this document** — Add the agent to the registry table and the Instruction Nudge Files table above
8. **Cross-reference** — Update `agent-hooks-integration` guide and `building-the-cli` guide
