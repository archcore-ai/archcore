---
title: "Integrating Archcore with AI Coding Agents"
status: accepted
tags:
  - "integrations"
---

## Overview

Archcore integrates with AI coding agents via three mechanisms:

- **Hooks** — Lifecycle event interception (session start) to inject context. Supported by Claude Code, Cursor, Gemini CLI, and GitHub Copilot. Only the `SessionStart` event is active — see [Disable Stop and Prompt Hooks ADR](disable-stop-and-prompt-hooks.adr.md).
- **MCP** — Model Context Protocol server providing document management tools (`init_project`, `list_documents`, `get_document`, `create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`, `list_relations`). Supported by all agents except Cline (manual setup).
- **Instruction nudge** — A short, always-on "use Archcore" hint written into each agent's instruction file (`AGENTS.md`, `GEMINI.md`, or `.claude/rules/archcore.md`) so agents discover the MCP tools without the Archcore plugin. See [Usage Nudge](#usage-nudge-instruction-files) below.

See [Supported AI Agents Registry](supported-ai-agents.doc.md) for the full agent list and capabilities.

## Quick Start

Two paths are supported. Pick whichever matches how the user is onboarding.

### Path A — Shell-first (`archcore init`)

```bash
archcore init
```

`archcore init` automatically:

1. Creates the `.archcore/` directory structure
2. Detects installed agents by checking for marker directories (`.claude/`, `.cursor/`, `.gemini/`, etc.)
3. If no agents are detected, prompts the user to pick one (or skips in non-interactive mode)
4. Installs hooks for agents that support them (Claude Code, Cursor, Gemini CLI, GitHub Copilot)
5. Installs MCP config for all detected agents
6. Offers (opt-in) to write a usage-nudge instruction file per agent — see [Usage Nudge](#usage-nudge-instruction-files)

Source: `cmd/init.go` (`installHooksForAgent` + `installMCPForAgent` loop, then `maybeInstallInstructions`).

### Path B — Agent-first (MCP `init_project`)

If the MCP config is already installed globally in the agent, the user can skip the shell step entirely:

1. Start the agent in a repo without `.archcore/`. The MCP server boots normally and prints a hint that only `init_project` is useful until initialized.
2. The agent calls the `init_project` MCP tool (optionally with `language` / `sync_mode`). It creates `.archcore/settings.json` in-process.
3. From that point on, the rest of the MCP tools (`create_document`, etc.) work normally.

`init_project` is idempotent — calling it on an already-initialized project preserves existing settings. See [MCP Server Starts Without .archcore/](../mcp/mcp-server-starts-without-archcore-dir.adr.md) for the architectural decision.

## Manual Installation

### Hooks Only

```bash
archcore hooks install              # auto-detect and install for all found agents
archcore hooks install --agent cursor  # install for a specific agent
```

Note: `archcore hooks install` also triggers MCP installation automatically.

### MCP Only

```bash
archcore mcp install                # auto-detect and install for all found agents
archcore mcp install --agent codex-cli  # install for a specific agent
```

### Usage Nudge (instruction files)

```bash
archcore instructions install              # auto-detect; write the hint for all found agents
archcore instructions install --agent cursor  # write for a specific agent
archcore instructions remove               # strip the hint from every known target
```

The hint points agents at `.archcore/` through the MCP tools so they discover and use it even without the Archcore plugin. `archcore init` offers this as an opt-in step (interactive only; non-interactive runs skip it). Targets: `.claude/rules/archcore.md` (Claude Code, owned file), `GEMINI.md` (Gemini CLI), `AGENTS.md` (all others). Shared files use a `<!-- archcore:start -->` / `<!-- archcore:end -->` fenced block — only that span is touched, so user content is preserved and re-running is idempotent. See [Supported AI Agents Registry](supported-ai-agents.doc.md) and the [instruction-nudge ADR](instruction-nudge-on-init.adr.md).

## Auto-Detection

Archcore detects agents by checking for marker directories or files in the project root:

| Agent          | Marker                                         |
| -------------- | ---------------------------------------------- |
| Claude Code    | `.claude/` directory                           |
| Cursor         | `.cursor/` directory                           |
| Gemini CLI     | `.gemini/` directory                           |
| GitHub Copilot | `.github/copilot-instructions.md` file         |
| OpenCode       | `opencode.json` file or `.opencode/` directory |
| Codex CLI      | `.codex/` directory                            |
| Roo Code       | `.roo/` directory                              |
| Cline          | `.clinerules/` directory                       |

If no markers are found, archcore prompts the user interactively or skips when running non-interactively.

Source: `internal/agents/agents.go` (`Detect` function), individual agent `DetectFn` in `internal/agents/*.go`.

## What Gets Installed

### Hooks (4 agents, SessionStart only)

| Agent          | Config File                   | Event          |
| -------------- | ----------------------------- | -------------- |
| Claude Code    | `.claude/settings.json`       | `SessionStart` |
| Cursor         | `.cursor/hooks.json`          | `sessionStart` |
| Gemini CLI     | `.gemini/settings.json`       | `SessionStart` |
| GitHub Copilot | `.github/hooks/archcore.json` | `sessionStart` |

### MCP (7 agents, Cline is manual)

| Agent          | Config File             | Format                                             |
| -------------- | ----------------------- | -------------------------------------------------- |
| Claude Code    | `.mcp.json`             | Standard `mcpServers` JSON                         |
| Cursor         | `.cursor/mcp.json`      | Standard `mcpServers` JSON                         |
| Gemini CLI     | `.gemini/settings.json` | Standard `mcpServers` JSON (shared with hooks)     |
| GitHub Copilot | `.vscode/mcp.json`      | VS Code-style `servers` JSON with `"type": "stdio"` |
| OpenCode       | `opencode.json`         | Custom `mcp` section with `type` + `command` array |
| Codex CLI      | `.codex/config.toml`    | TOML `[mcp_servers.archcore]` block                |
| Roo Code       | `.roo/mcp.json`         | Standard `mcpServers` JSON                         |

### Instruction Nudge (8 agents → 3 files, opt-in)

| Agent                                                        | Instruction File            | Write Mode         |
| ------------------------------------------------------------ | --------------------------- | ------------------ |
| Claude Code                                                  | `.claude/rules/archcore.md` | owned (whole file) |
| Gemini CLI                                                   | `GEMINI.md`                 | fenced upsert      |
| Cursor, OpenCode, Codex CLI, Roo Code, Cline, GitHub Copilot | `AGENTS.md`                 | fenced upsert      |

Written by the opt-in step in `archcore init` or by `archcore instructions install`. The six `AGENTS.md` agents share one file (written once).

## Per-Agent Config Examples

### Claude Code — `.claude/settings.json`

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "", "hooks": [{ "type": "command", "command": "archcore hooks claude-code session-start" }] }
    ]
  }
}
```

### Claude Code — `.mcp.json`

```json
{
  "mcpServers": {
    "archcore": { "command": "archcore", "args": ["mcp"] }
  }
}
```

### Cursor — `.cursor/hooks.json`

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [{ "command": "archcore hooks cursor session-start", "type": "command" }]
  }
}
```

### Cursor — `.cursor/mcp.json`

```json
{
  "mcpServers": {
    "archcore": { "command": "archcore", "args": ["mcp"] }
  }
}
```

### Gemini CLI — `.gemini/settings.json`

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "", "hooks": [{ "type": "command", "command": "archcore hooks gemini-cli session-start" }] }
    ]
  },
  "mcpServers": {
    "archcore": { "command": "archcore", "args": ["mcp"] }
  }
}
```

### GitHub Copilot — `.github/hooks/archcore.json`

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [{ "type": "command", "bash": "archcore hooks copilot session-start" }]
  }
}
```

### GitHub Copilot — `.vscode/mcp.json`

```json
{
  "servers": {
    "archcore": { "type": "stdio", "command": "archcore", "args": ["mcp"] }
  }
}
```

### OpenCode — `opencode.json`

```json
{
  "mcp": {
    "archcore": { "type": "local", "command": ["archcore", "mcp"] }
  }
}
```

### Codex CLI — `.codex/config.toml`

```toml
[mcp_servers.archcore]
command = "archcore"
args = ["mcp"]
```

### Roo Code — `.roo/mcp.json`

```json
{
  "mcpServers": {
    "archcore": { "command": "archcore", "args": ["mcp"] }
  }
}
```

### Cline — Manual Setup

Cline stores MCP config in VS Code `globalStorage`, not in project files. To add archcore:

1. Open Cline MCP settings in VS Code
2. Add an MCP server with command `archcore` and args `["mcp"]`

## Pointing the MCP Server at a Specific Project Root

By default `archcore mcp` serves documents from the current working directory. Some editor integrations launch the binary from a directory that isn't the workspace root (e.g. a desktop app's install dir, or a Cline globalStorage profile). In those cases the server may see an empty/wrong project.

Two overrides are available:

- **Flag**: `archcore mcp --project /absolute/path/to/repo`
- **Environment**: `ARCHCORE_PROJECT_ROOT=/absolute/path/to/repo archcore mcp`

Precedence: `--project` > `ARCHCORE_PROJECT_ROOT` > current working directory.

The path must point at an existing directory (it does not need to contain `.archcore/` yet — the server still starts and exposes `init_project`).

Example for an agent that needs an absolute workspace path:

```json
{
  "mcpServers": {
    "archcore": { "command": "archcore", "args": ["mcp", "--project", "/Users/me/code/my-repo"] }
  }
}
```

## Invalid Config Recovery

When archcore reads a config file that contains invalid JSON, it creates a `.bak` backup before proceeding with a fresh config. This prevents data loss while keeping the installation non-blocking.

Example: if `.cursor/hooks.json` is corrupted, archcore writes `.cursor/hooks.json.bak` and starts with an empty hooks config.

See [Backup Invalid Configs](backup-invalid-configs.adr.md) for the full decision record.

## Troubleshooting

### `.archcore/` not initialized

- The **MCP server** (`archcore mcp`) starts fine without `.archcore/` — it exposes `init_project` so the agent can bootstrap the directory in-session. If the agent sees an empty `list_documents` result and wants to create documents, it should call `init_project` first.
- **Hooks** and the `archcore hooks/mcp install` commands still require an initialized project. Run `archcore init` first, or ask the agent to call `init_project`.

### MCP server is serving the wrong directory

Symptoms: `list_documents` returns an empty array even though the workspace clearly has `.archcore/` documents, or `init_project` would create the directory in an unexpected location.

Cause: the agent launched `archcore mcp` from a working directory that isn't your workspace.

Fix: pass `--project` explicitly in the agent's MCP config, or set `ARCHCORE_PROJECT_ROOT` in the agent's environment. See **Pointing the MCP Server at a Specific Project Root** above.

### Agent not detected

Check that the agent's marker directory exists in your project root. You can also target a specific agent with `--agent`:

```bash
archcore hooks install --agent gemini-cli
archcore mcp install --agent opencode
```

### Hooks not firing

1. Verify the config file exists and contains archcore entries (see examples above)
2. Ensure `archcore` is on your `PATH`
3. Check the agent's logs for hook execution errors

### MCP tools not available

1. Verify MCP config file is correct for your agent
2. Restart the agent/IDE after installing
3. For Cline, ensure you added the server via the MCP settings UI

### Corrupted config after install

Check for a `.bak` file next to the config. Restore it and retry:

```bash
cp .cursor/hooks.json.bak .cursor/hooks.json
archcore hooks install --agent cursor
```
