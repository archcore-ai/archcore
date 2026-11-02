---
title: "Plugin Component Registry"
status: accepted
tags:
  - "component:plugin"
  - "plugin"
  - "reference"
---

## Overview

The Archcore plugin ships to four hosts: Claude Code, Cursor, Codex CLI, and GitHub Copilot CLI. The tree itself is the component list. This document keeps only what the tree cannot say: why two hosts get no plugin-shipped MCP server, and how the shared pieces are wired.

Per `four-command-palette.adr`, the visible `/` palette is four commands (`init`, `plan`, `document`, `review`) over a non-palette gated track layer (`track-layer.spec`). The skill at `skills/<name>/SKILL.md` is the single behavioral source of truth on all four hosts. Codex discovers slash commands only from `commands/*.md` wrappers, and Copilot loads the same wrappers behind its skills. No workflow logic lives in `commands/`.

The plugin root is `plugins/archcore/`. The three marketplace catalogs and `docs/cursor.mcp.example.json` live at the repo root and point at `./plugins/archcore`. A catalog `source.path` of `./` is never scanned by Codex marketplace discovery (`subdirectory-plugin-layout.adr`, issue #2). Copilot CLI has no marketplace and installs by subdirectory spec.

## Content

### Why Claude Code's `commands: []`

Claude Code still scans `commands/` by default and would list each wrapper beside its skill as a second `/archcore:<name>` row. `.claude-plugin/plugin.json` therefore declares `"commands": []`. Copilot does not inherit this value, because `.plugin/plugin.json` declares its own `commands`. Without that pointer the whole `/archcore:*` surface is missing on Copilot, because the field has no default path there.

### Why agents ship in three formats

Copilot loads plugin agents only from the `*.agent.md` extension. That extension still matches the `*.md` glob Claude Code and Cursor use. A sibling copy would hand both hosts two files declaring the same `name:`. The Copilot copies therefore live in `copilot-agents/`. The Codex TOML variants never had this problem, because their extension is foreign to every md-globbing host. Agent tool lists carry all three MCP namings (`mcp__archcore__*`, `mcp__plugin_archcore_archcore__*`, Copilot's flat `archcore-<tool>`). A deny-list that misses one naming fails open. `agent-system.spec` owns the turn budgets.

### Hook launchers

The hook policy runs inside the archcore CLI (`cli-owns-layers-4-5.adr`, `hooks-validation-system.spec`). The plugin registers one launcher per event. The launchers fail open: an absent CLI, a CLI older than 0.7.0, or a plugin-cache cwd yields a silent exit 0, so a deny can originate only in the CLI's own verdict. A prior `PostToolUse` entry with matcher `Write|Edit` was removed. PreToolUse already blocks every Write and Edit to `.archcore/` markdown, so the entry only forked a shell on every write in the repository.

Every PostToolUse matcher lists each MCP tool under both namings. A project-level `.claude.mcp.json` entry yields `mcp__archcore__*`, a plugin-bundled server yields `mcp__plugin_archcore_archcore__*`, and a Claude Code matcher without regex metacharacters is an exact match (`host-wiring-parity.adr`).

The Codex host id `ARCHCORE_HOST=codex` is pinned in the command, because a Codex `SessionStart` payload carries no `turn_id`. Before v0.7.3 every Codex session start was processed as Claude Code (`codex-adapter-conformance.adr`). Copilot commands probe three plugin-root variables in turn, because GitHub documents only `${PLUGIN_ROOT}` and every non-zero exit denies on that host (`copilot-adapter-design.adr`).

`bin/detect-host` has deliberately no Copilot branch. Every documented `COPILOT_*` variable is read from the user rather than set by the host. A Copilot session therefore resolves to `__UNKNOWN__` and reaches wiring through the init skill's ask path.

### MCP server

The plugin ships MCP registration for Claude Code only, through `.claude.mcp.json` at the plugin root. It registers one server named `archcore` with `command: "archcore"` and `args: ["mcp"]`, resolved through PATH. Codex points at the plugin-root `.codex.mcp.json` from `.codex-plugin/plugin.json`. The plugin bundles no CLI and no launcher (`remove-bundled-launcher-global-cli.idea`), so the CLI lifecycle is decoupled from plugin releases.

Cursor and GitHub Copilot CLI are the two exceptions, for the same reason. Each launches a plugin's MCP child outside the user's project.

Cursor spawns plugin MCPs from the plugin install directory, and its stdio schema has no `cwd` field. The plugin therefore ships no MCP file at the plugin root. Cursor users get `.cursor/mcp.json` written by `/archcore:init` or `archcore init --agent cursor`, carrying `--project ${workspaceFolder}`. Copying `docs/cursor.mcp.example.json` by hand remains the fallback before wiring has run (`cursor-mcp-architecture.adr`).

Copilot launches a plugin's MCP child with the working directory set to the plugin install root and passes it no project path at all, not even the `COPILOT_PROJECT_DIR` its hooks receive (github/copilot-cli#4234). Documents would be written into `~/.copilot/installed-plugins/` while every tool reported success. `.plugin/plugin.json` therefore declares no `mcpServers`, and Copilot users get a workspace-root `.mcp.json` from `archcore init --agent copilot` on CLI v0.6.4 or later. The plugin's own MCP file is named `.claude.mcp.json`, not `.mcp.json`, so Copilot's auto-discovery never picks it up (`copilot-mcp-architecture.adr`).

### Plugin-install-dir guard

`bin/session-start` exits silently when `$PWD` sits inside a plugin install cache or beside a plugin manifest. The check runs before the CLI availability check, so a misrouted working directory never sees the install nudge. The guard is shared with the launchers in `bin/lib/plugin-cache-guard.sh` (`cursor-mcp-architecture.adr`, extended by `host-wiring-parity.adr`).

### OpenCode package

`plugins/opencode/` holds `@archcore/opencode-plugin`, the npm package OpenCode loads. `opencode-adapter.spec` is its contract. The bridge spawns the `bin/` launchers and decides nothing.
