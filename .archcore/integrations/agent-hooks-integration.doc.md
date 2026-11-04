---
title: "Integrating Archcore with AI Coding Agents"
status: accepted
tags:
  - "component:cli"
  - "integrations"
---

## Overview

Archcore integrates with AI coding agents via three mechanisms:

- **Hooks** — Lifecycle event interception. `SessionStart` injects the project recap, `PreToolUse` blocks direct writes to `.archcore/` documents and injects the documents that constrain the file being edited, and `PostToolUse` reports validation, cascade, and precision findings after a document mutation. Wired for Claude Code, Cursor, Gemini CLI, Codex CLI, and GitHub Copilot. OpenCode runs the same three events through the Archcore OpenCode plugin rather than a written config. The `Stop` and `UserPromptSubmit` families stay unsupported (@.archcore/integrations/disable-stop-and-prompt-hooks.adr.md).
- **MCP** — the document tools (`init_project`, `list_documents`, `get_document`, `search_documents`, `create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`, `list_relations`). Supported by all agents except Cline (manual setup).
- **Instruction nudge** — a short, always-on "use Archcore" block in each agent's instruction file (`AGENTS.md`, `GEMINI.md`, or `CLAUDE.md`) so agents discover the MCP tools without the plugin (@.archcore/integrations/instruction-nudge-on-init.adr.md).

The agent list and capabilities are in @.archcore/integrations/supported-ai-agents.doc.md. The hook contract is in @.archcore/integrations/hook-runtime.spec.md (guards, degradation) and @.archcore/integrations/hook-wire-protocol.spec.md (per-host protocol). Matchers, timeouts, and config shapes live in `@cli/internal/wiring/hooks_agents.go`.

## Quick Start

### Path A — Shell-first (`archcore init`)

`archcore init` creates `.archcore/`, detects installed agents by marker directories, prompts for one when none is found (interactive only), installs hooks and MCP config for the detected agents, reports per host whether the written hook config can take effect, and offers the opt-in usage-nudge file (`installAgents` and `maybeInstallInstructions` in @cli/cmd/init.go).

### Path B — Agent-first (MCP `init_project`)

If the MCP config is already installed globally, the agent starts in a repo without `.archcore/`; the server boots and only `init_project` is useful. The agent calls `init_project` (optionally with `language` / `sync_mode`), and the other tools then work. The call is idempotent (@.archcore/mcp/mcp-server-starts-without-archcore-dir.adr.md).

## Manual Installation

Every command reads the project root from `--project`, then `ARCHCORE_PROJECT_ROOT`, then the working directory. Pass `--project` whenever the shell is not in the repository; resolution also refuses a root inside a host's plugin install cache.

```bash
archcore hooks install [--agent cursor]       # also installs MCP
archcore mcp install [--agent codex-cli]
archcore instructions install [--agent cursor] # remove: archcore instructions remove
```

After `hooks install`, each wired host gets silence (the wiring works) or a note that says why the hooks are inert and what enables them.

The nudge targets are `CLAUDE.md` and `AGENTS.md` (Claude Code, both), `GEMINI.md` (Gemini CLI), and `AGENTS.md` (all others). Shared files use an `<!-- archcore:start -->` / `<!-- archcore:end -->` fenced block, so only that span is touched and re-running is idempotent. Claude Code gets both files because it reads `CLAUDE.md` natively and does not auto-read `AGENTS.md`.

## Auto-Detection

Archcore detects agents by marker in the project root (`Detect` in @cli/internal/agents/agents.go): `.claude/` (Claude Code), `.cursor/`, `.gemini/`, `.codex/`, `.github/copilot-instructions.md` (Copilot), `opencode.json` or `.opencode/`, `.roo/`, `.clinerules/`. Without a marker, init prompts or skips in non-interactive mode.

## What Gets Installed

- **Hooks**: one archcore-owned entry per (host, event) pair for five agents. OpenCode is never wired, and this is permanent: the host loads hooks as plugin code, so there is no declarative file. The OpenCode plugin registers the host events and calls `archcore hooks opencode <event>`. Installing the CLI alone gives an OpenCode user the MCP tools but no hooks; checking OpenCode in `archcore init`, or `archcore plugin install --agent opencode`, installs the plugin (@.archcore/integrations/plugin-delivery.spec.md).
- **MCP**: seven agents, Cline manual. Most use a standard `mcpServers` entry `{command: "archcore", args: ["mcp"]}`. OpenCode uses `opencode.json` with `{"mcp": {"archcore": {"type": "local", "command": ["archcore", "mcp"]}}}`; its optional keys are `enabled`, `cwd`, `timeout`, and `environment` (not `env`), because the host's schema rejects unknown keys. Codex CLI uses a TOML `[mcp_servers.archcore]` block. Copilot shares Claude Code's `.mcp.json` on purpose: both key on `mcpServers`, and the write merges idempotently. It does not use `.vscode/mcp.json`, which Copilot CLI dropped in v1.0.37.
- **Cline**: stores MCP config in VS Code `globalStorage`, so the user adds the server by hand with command `archcore` and args `["mcp"]`.

## Pointing the MCP Server at a Specific Project Root

Some editor integrations launch the binary from a directory that is not the workspace root. Two overrides exist: `archcore mcp --project /abs/path` and `ARCHCORE_PROJECT_ROOT`. Precedence is `--project`, then the variable, then the working directory, for every command that reads `.archcore/`. The path must be an existing directory; it need not hold `.archcore/` yet.

## Invalid Config Recovery

When archcore reads a config file with invalid JSON, it writes a `.bak` backup before proceeding with a fresh config (@.archcore/integrations/backup-invalid-configs.adr.md). To recover, restore the `.bak` file and re-run the install.

## Troubleshooting

- **`.archcore/` not initialized.** The MCP server starts anyway and exposes `init_project`. Hooks and the install commands need an initialized project: run `archcore init` or ask the agent to call `init_project`.
- **Wrong directory served, or a command reports on the wrong project.** The host launched the process from a directory that is not your workspace, sometimes inside its plugin install cache. Pass `--project` or set `ARCHCORE_PROJECT_ROOT`.
- **Agent not detected.** Check the marker directory, or target the agent with `--agent`.
- **Hooks not firing.** Re-run `archcore hooks install` and read its notes. Verify the config holds archcore entries and `archcore` is on `PATH`. On Codex CLI, enable the hooks feature (`codex --enable hooks`, or `hooks = true` under `[features]` in `~/.codex/config.toml`; `codex_hooks = true` before Codex 0.129.0), trust the project's `.codex/` layer, and note that Codex has no hook support on Windows.
- **No hooks on OpenCode.** Expected without the plugin; run `archcore plugin install --agent opencode`. A CLI older than the one that added the `archcore hooks opencode <event>` leaves makes the plugin degrade to no hooks rather than fail.
- **Context appears twice.** An older Archcore plugin still ships its own hooks, so both fire. Update the plugin; until then the cost is extra output, not a wrong verdict (@.archcore/integrations/plugin-cli-compatibility.rule.md).
- **No context before an edit on Copilot.** Expected: Copilot's `preToolUse` carries only a permission decision, so the write guard runs there and the code-alignment injection does not.
- **MCP tools not available.** Verify the MCP config for your agent and restart the agent. For Cline, add the server through the MCP settings UI.
