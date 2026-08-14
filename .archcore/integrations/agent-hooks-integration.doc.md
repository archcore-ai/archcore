---
title: "Integrating Archcore with AI Coding Agents"
status: accepted
tags:
  - "integrations"
---

## Overview

Archcore integrates with AI coding agents via three mechanisms:

- **Hooks** — Lifecycle event interception. Three events are active: `SessionStart` injects the
  project recap, `PreToolUse` blocks direct writes to `.archcore/` documents and injects the documents
  that constrain the file being edited, and `PostToolUse` reports validation, cascade, and precision
  findings after a document mutation. Wired for Claude Code, Cursor, Gemini CLI, Codex CLI, and GitHub
  Copilot. OpenCode runs the same three events, but through the Archcore OpenCode plugin rather than a
  written config. The `Stop` and `UserPromptSubmit` families stay unsupported — see the ADR on
  removing them.
- **MCP** — Model Context Protocol server providing document management tools (`init_project`,
  `list_documents`, `get_document`, `search_documents`, `create_document`, `update_document`,
  `remove_document`, `add_relation`, `remove_relation`, `list_relations`). Supported by all agents
  except Cline (manual setup).
- **Instruction nudge** — A short, always-on "use Archcore" hint written into each agent's instruction
  file (`AGENTS.md`, `GEMINI.md`, or `CLAUDE.md`) so agents discover the MCP tools without the Archcore
  plugin. See [Usage Nudge](#usage-nudge-instruction-files) below.

See [Supported AI Agents Registry](supported-ai-agents.doc.md) for the full agent list and
capabilities, and [CLI Hooks Reference](cli-hooks-reference.doc.md) for the event matrix, the guards
each event runs, and the per-host protocol dialects.

## Quick Start

Two paths are supported. Pick whichever matches how the user is onboarding.

### Path A — Shell-first (`archcore init`)

```bash
archcore init
```

`archcore init` automatically:

1. Creates the `.archcore/` directory structure
2. Detects installed agents by checking for marker directories (`.claude/`, `.cursor/`, `.gemini/`,
   `.codex/`, etc.)
3. If no agents are detected, prompts the user to pick one (or skips in non-interactive mode)
4. Installs hooks for agents that support them
5. Reports, per host, whether the written hook config can actually take effect
6. Installs MCP config for all detected agents
7. Offers (opt-in) to write a usage-nudge instruction file per agent — see
   [Usage Nudge](#usage-nudge-instruction-files)

Source: `@cmd/init.go` (`installAgents` loop, then `maybeInstallInstructions`).

### Path B — Agent-first (MCP `init_project`)

If the MCP config is already installed globally in the agent, the user can skip the shell step
entirely:

1. Start the agent in a repo without `.archcore/`. The MCP server boots normally and prints a hint that
   only `init_project` is useful until initialized.
2. The agent calls the `init_project` MCP tool (optionally with `language` / `sync_mode`). It creates
   `.archcore/settings.json` in-process.
3. From that point on, the rest of the MCP tools (`create_document`, etc.) work normally.

`init_project` is idempotent — calling it on an already-initialized project preserves existing
settings. See [MCP Server Starts Without .archcore/](../mcp/mcp-server-starts-without-archcore-dir.adr.md)
for the architectural decision.

## Manual Installation

Every command below reads the project root from `--project`, then `ARCHCORE_PROJECT_ROOT`, then the
working directory. Pass `--project` whenever the shell is not already in the repository — the
resolution also refuses a root inside a host's plugin install cache, which some hosts spawn agent
processes into.

### Hooks Only

```bash
archcore hooks install                 # auto-detect and install for all found agents
archcore hooks install --agent cursor  # install for a specific agent
```

Note: `archcore hooks install` also triggers MCP installation automatically.

Expected result: for each wired host, either silence (the wiring works as installed) or a note stating
that the file was written, why the hooks are inert, and the action that enables them. A host whose
hooks load as plugin code, such as OpenCode, is reported as such instead of as hookless.

### MCP Only

```bash
archcore mcp install                    # auto-detect and install for all found agents
archcore mcp install --agent codex-cli  # install for a specific agent
```

### Usage Nudge (instruction files)

```bash
archcore instructions install                 # auto-detect; write the hint for all found agents
archcore instructions install --agent cursor  # write for a specific agent
archcore instructions remove                  # strip the hint from every known target
```

The hint points agents at `.archcore/` through the MCP tools so they discover and use it even without
the Archcore plugin. `archcore init` offers this as an opt-in step (interactive only; non-interactive
runs skip it). Targets: `CLAUDE.md` **and** `AGENTS.md` (Claude Code — both, see below), `GEMINI.md`
(Gemini CLI), `AGENTS.md` (all others). Shared files use a `<!-- archcore:start -->` /
`<!-- archcore:end -->` fenced block — only that span is touched, so user content is preserved and
re-running is idempotent. See [Supported AI Agents Registry](supported-ai-agents.doc.md) and the
[instruction-nudge ADR](instruction-nudge-on-init.adr.md).

## Auto-Detection

Archcore detects agents by checking for marker directories or files in the project root:

| Agent          | Marker                                         |
| -------------- | ---------------------------------------------- |
| Claude Code    | `.claude/` directory                           |
| Cursor         | `.cursor/` directory                           |
| Gemini CLI     | `.gemini/` directory                           |
| Codex CLI      | `.codex/` directory                            |
| GitHub Copilot | `.github/copilot-instructions.md` file         |
| OpenCode       | `opencode.json` file or `.opencode/` directory |
| Roo Code       | `.roo/` directory                              |
| Cline          | `.clinerules/` directory                       |

If no markers are found, archcore prompts the user interactively or skips when running
non-interactively.

Source: `@internal/agents/agents.go` (`Detect` function), individual agent `DetectFn` in
`internal/agents/*.go`.

## What Gets Installed

### Hooks (5 agents, 3 events each)

One archcore-owned entry per (host, event) pair. The process dispatches by tool name internally.

| Agent          | Config File                   | Session event  | Pre-write event | Post-write event    |
| -------------- | ----------------------------- | -------------- | --------------- | ------------------- |
| Claude Code    | `.claude/settings.json`       | `SessionStart` | `PreToolUse`    | `PostToolUse`       |
| Cursor         | `.cursor/hooks.json`          | `sessionStart` | `preToolUse`    | `afterMCPExecution` |
| Gemini CLI     | `.gemini/settings.json`       | `SessionStart` | `BeforeTool`    | `AfterTool`         |
| Codex CLI      | `.codex/hooks.json`           | `SessionStart` | `PreToolUse`    | `PostToolUse`       |
| GitHub Copilot | `.github/hooks/archcore.json` | `sessionStart` | `preToolUse`    | `postToolUse`       |

Matchers and timeouts differ per host — see [CLI Hooks Reference](cli-hooks-reference.doc.md).

OpenCode is never wired, and this is permanent rather than pending: the host loads hooks as plugin
code, so there is no declarative file to write. It gets the same three events from the Archcore
OpenCode plugin, which registers the host's own events and calls
`archcore hooks opencode <event>`. Installing the CLI alone gives an OpenCode user the MCP document
tools but no hooks; installing the plugin adds the hooks.

### MCP (7 agents, Cline is manual)

| Agent          | Config File             | Format                                             |
| -------------- | ----------------------- | -------------------------------------------------- |
| Claude Code    | `.mcp.json`             | Standard `mcpServers` JSON                         |
| Cursor         | `.cursor/mcp.json`      | Standard `mcpServers` JSON                         |
| Gemini CLI     | `.gemini/settings.json` | Standard `mcpServers` JSON (shared with hooks)     |
| GitHub Copilot | `.mcp.json`             | Standard `mcpServers` JSON (the same file Claude Code gets) |
| OpenCode       | `opencode.json`         | Custom `mcp` section with `type` + `command` array |
| Codex CLI      | `.codex/config.toml`    | TOML `[mcp_servers.archcore]` block                |
| Roo Code       | `.roo/mcp.json`         | Standard `mcpServers` JSON                         |

Copilot shares Claude Code's file deliberately: both hosts key on `mcpServers` and accept a bare
`{command, args}` stdio entry, and the write merges idempotently, so one file serves both. Not
`.vscode/mcp.json` — Copilot CLI dropped that source in v1.0.37 (github/copilot-cli#3019) and it
belongs to VS Code alone.

### Instruction Nudge (8 agents, opt-in)

| Agent                                                        | Instruction File(s)             | Write Mode           |
| ------------------------------------------------------------ | ------------------------------- | -------------------- |
| Claude Code                                                  | `CLAUDE.md` **and** `AGENTS.md` | fenced upsert (both) |
| Gemini CLI                                                   | `GEMINI.md`                     | fenced upsert        |
| Cursor, OpenCode, Codex CLI, Roo Code, Cline, GitHub Copilot | `AGENTS.md`                     | fenced upsert        |

Written by the opt-in step in `archcore init` or by `archcore instructions install`. The six
`AGENTS.md`-only agents share one file (written once). Claude Code gets **both** `CLAUDE.md` (the file
it reads natively — this is what delivers the nudge) and the shared `AGENTS.md` block (for the plugin
and the other hosts; Claude Code does not auto-read AGENTS.md per Anthropic's docs). The `AGENTS.md`
upsert is idempotent, so a co-installed `AGENTS.md` agent writing the same block collapses to one.
Earlier CLIs wrote an owned `.claude/rules/archcore.md`; it is migrated away on (re)wiring.

## Per-Agent Config Examples

Non-normative examples. Matchers are abbreviated; the full values live in
`@internal/wiring/hooks_agents.go`.

### Claude Code — `.claude/settings.json`

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [{ "type": "command", "command": "archcore hooks claude-code session-start" }] }
    ],
    "PreToolUse": [
      { "matcher": "Write|Edit",
        "hooks": [{ "type": "command", "command": "archcore hooks claude-code pre-tool-use" }] }
    ],
    "PostToolUse": [
      { "matcher": "mcp__archcore__(create_document|update_document|…)",
        "hooks": [{ "type": "command", "command": "archcore hooks claude-code post-tool-use" }] }
    ]
  }
}
```

### Cursor — `.cursor/hooks.json`

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [{ "command": "archcore hooks cursor session-start", "type": "command" }],
    "preToolUse": [{ "command": "archcore hooks cursor pre-tool-use", "type": "command", "matcher": "Write" }],
    "afterMCPExecution": [{ "command": "archcore hooks cursor post-tool-use", "type": "command" }]
  }
}
```

### Gemini CLI — `.gemini/settings.json`

Timeouts are in milliseconds on this host.

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [{ "type": "command", "command": "archcore hooks gemini-cli session-start", "timeout": 3000 }] }
    ],
    "BeforeTool": [
      { "matcher": "write_file",
        "hooks": [{ "type": "command", "command": "archcore hooks gemini-cli pre-tool-use", "timeout": 1000 }] }
    ]
  },
  "mcpServers": { "archcore": { "command": "archcore", "args": ["mcp"] } }
}
```

### Codex CLI — `.codex/hooks.json`

```json
{
  "hooks": {
    "PreToolUse": [
      { "matcher": "Write|Edit|apply_patch",
        "hooks": [{ "type": "command", "command": "archcore hooks codex-cli pre-tool-use" }] }
    ]
  }
}
```

### GitHub Copilot — `.github/hooks/archcore.json`

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [{ "type": "command", "bash": "archcore hooks copilot session-start", "timeoutSec": 3 }],
    "preToolUse": [{ "type": "command", "bash": "archcore hooks copilot pre-tool-use",
                     "matcher": "create|edit|str_replace_editor|apply_patch", "timeoutSec": 1 }]
  }
}
```

### MCP config examples

```json
{ "mcpServers": { "archcore": { "command": "archcore", "args": ["mcp"] } } }
```

Claude Code and GitHub Copilot (both `.mcp.json`, one shared file), Cursor (`.cursor/mcp.json`),
Gemini CLI (inside `.gemini/settings.json`), and Roo Code (`.roo/mcp.json`) all use that shape. The
two that differ:

```json
{ "mcp": { "archcore": { "type": "local", "command": ["archcore", "mcp"] } } }
```

OpenCode, `opencode.json`. Optional per-server keys are `enabled`, `cwd`, `timeout`, and
`environment` — spelled that way, not `env`. The host's schema rejects an unknown key rather than
ignoring it, so the wrong spelling fails the config instead of silently dropping the value.

```toml
[mcp_servers.archcore]
command = "archcore"
args = ["mcp"]
```

Codex CLI, `.codex/config.toml`.

### Cline — Manual Setup

Cline stores MCP config in VS Code `globalStorage`, not in project files. To add archcore:

1. Open Cline MCP settings in VS Code
2. Add an MCP server with command `archcore` and args `["mcp"]`

## Pointing the MCP Server at a Specific Project Root

By default `archcore mcp` serves documents from the current working directory. Some editor
integrations launch the binary from a directory that isn't the workspace root (e.g. a desktop app's
install dir, or a Cline globalStorage profile). In those cases the server may see an empty or wrong
project.

Two overrides are available:

- **Flag**: `archcore mcp --project /absolute/path/to/repo`
- **Environment**: `ARCHCORE_PROJECT_ROOT=/absolute/path/to/repo archcore mcp`

Precedence: `--project` > `ARCHCORE_PROJECT_ROOT` > current working directory. The same precedence
applies to every command that reads `.archcore/`, not only `mcp`.

The path must point at an existing directory (it does not need to contain `.archcore/` yet — the
server still starts and exposes `init_project`).

```json
{
  "mcpServers": {
    "archcore": { "command": "archcore", "args": ["mcp", "--project", "/Users/me/code/my-repo"] }
  }
}
```

## Invalid Config Recovery

When archcore reads a config file that contains invalid JSON, it creates a `.bak` backup before
proceeding with a fresh config. This prevents data loss while keeping the installation non-blocking.

Example: if `.cursor/hooks.json` is corrupted, archcore writes `.cursor/hooks.json.bak` and starts with
an empty hooks config.

See [Backup Invalid Configs](backup-invalid-configs.adr.md) for the full decision record.

## Troubleshooting

### `.archcore/` not initialized

- The **MCP server** (`archcore mcp`) starts fine without `.archcore/` — it exposes `init_project` so
  the agent can bootstrap the directory in-session. If the agent sees an empty `list_documents` result
  and wants to create documents, it should call `init_project` first.
- **Hooks** and the `archcore hooks/mcp install` commands still require an initialized project. Run
  `archcore init` first, or ask the agent to call `init_project`.

### MCP server is serving the wrong directory

Symptoms: `list_documents` returns an empty array even though the workspace clearly has `.archcore/`
documents, or `init_project` would create the directory in an unexpected location.

Cause: the agent launched `archcore mcp` from a working directory that isn't your workspace.

Fix: pass `--project` explicitly in the agent's MCP config, or set `ARCHCORE_PROJECT_ROOT` in the
agent's environment. See **Pointing the MCP Server at a Specific Project Root** above.

### A command reports on the wrong project

Symptoms: `archcore status` or `archcore config get` describes a project that is not the one you are
in, or `archcore instructions remove` strips a hint you did not mean to touch.

Cause: the shell, or the host that spawned the process, is not in the repository. Some hosts launch
agent processes with the working directory inside their own plugin install cache.

Fix: pass `--project /path/to/repo` or set `ARCHCORE_PROJECT_ROOT`. Archcore refuses a root that
resolves inside a known plugin cache rather than reading it as a project.

### Agent not detected

Check that the agent's marker directory exists in your project root. You can also target a specific
agent with `--agent`:

```bash
archcore hooks install --agent gemini-cli
archcore mcp install --agent opencode
```

### Hooks not firing

1. Re-run `archcore hooks install` and read the notes it prints. A host that cannot run the written
   config says so, with the reason and the fix.
2. Verify the config file exists and contains archcore entries (see examples above).
3. Ensure `archcore` is on your `PATH`.
4. On Codex CLI, enable the experimental hooks feature (`codex --enable hooks`, or `[features]` with
   `hooks = true` in `~/.codex/config.toml`; `codex_hooks = true` before Codex 0.129.0), trust the
   project's `.codex/` layer, and note that Codex does not support hooks on Windows.
5. Check the agent's logs for hook execution errors.

### No hooks on OpenCode

Expected without the plugin. `archcore init` writes OpenCode's MCP config but no hook config, because
the host has no declarative hook file. Install the Archcore OpenCode plugin to get the three events;
the CLI already ships the `archcore hooks opencode <event>` leaves it calls. A CLI older than the one
that added those leaves makes the plugin degrade to no hooks rather than fail.

### Context appears twice

Cause: an older Archcore plugin is installed and still ships its own hooks. Both its entries and the
project entries fire.

Fix: update the plugin. Until then the duplication costs extra output, not a wrong verdict. See the
[plugin compatibility rule](plugin-cli-compatibility.rule.md).

### No context before an edit on Copilot

Expected. Copilot's `preToolUse` event carries only a permission decision, so the write guard runs
there and the code-alignment injection does not.

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
