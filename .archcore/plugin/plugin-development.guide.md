---
title: "Plugin Development Guide"
status: accepted
tags:
  - "development"
  - "plugin"
---

## Prerequisites

- Claude Code, Cursor, or Codex CLI installed with plugin support
- Git for version control
- bats-core for tests (`brew install bats-core` on macOS)
- jq for JSON validation (`brew install jq`)
- ShellCheck (optional, `brew install shellcheck`)
- **Archcore CLI** installed globally via the official installer at https://docs.archcore.ai/cli/install/ — `curl -fsSL https://archcore.ai/install.sh | bash` (macOS/Linux/WSL) or `irm https://archcore.ai/install.ps1 | iex` (Windows PowerShell 5.1+). Verify with `archcore --version`.

That's it. The plugin does not bundle a launcher — it assumes users have the Archcore CLI installed globally on PATH. MCP is registered automatically for Claude Code via plugin-root `.mcp.json`, and for Codex CLI via `.codex-plugin/plugin.json` pointing at plugin-root `.codex.mcp.json`. Both `.mcp.json` and `.codex.mcp.json` simply name `archcore` as the command — host runtimes resolve it via PATH.

For Cursor development, you register MCP externally by copying `docs/cursor.mcp.example.json` into `~/.cursor/mcp.json` (user-scoped) or `.cursor/mcp.json` (project-scoped). The plugin deliberately does **not** ship a Cursor plugin-MCP — see `cursor-mcp-architecture.adr.md` for the three-layer rationale (Cursor 2.5+ spawns plugin-MCPs from the plugin install dir rather than the workspace, and its MCP stdio schema has no `cwd` field). The template passes `--project ${workspaceFolder}` in `args` so the server always resolves the workspace, regardless of how Cursor invokes it.

For Codex development, `codex plugin marketplace add /path/to/plugin` registers the marketplace. The current CLI loads enabled plugins from its installed plugin cache; run `make test-codex-smoke` for the local installed-cache smoke that verifies skill discovery and plugin-managed MCP.

Initialize a project for testing with `mcp__archcore__init_project` (via a Claude Code or Cursor session) rather than an out-of-band CLI command; the plugin routes initialization through MCP.

## Steps

### 1. Clone the plugin repository

```bash
git clone https://github.com/archcore-ai/plugin.git
cd plugin
git checkout dev               # development happens on dev, main is synthesized
git submodule update --init    # pulls bats-support and bats-assert
```

The plugin uses a `dev → main` split: all PRs land on `dev`. The `main` branch is synthesized by `.github/workflows/release.yml` from a tagged commit on `dev`, with dev-only artifacts stripped (`.archcore/`, `reference-materials/`, `test/`, `Makefile`, `.github/`, etc.). See `docs/release.md` for the full blocklist and release procedure.

### 2. Run the host with the plugin loaded locally

```bash
claude --plugin-dir .    # Claude Code
cursor --plugin-dir .    # Cursor
```

This loads the plugin from the current directory without requiring marketplace installation. Changes to plugin files are picked up after running `/reload-plugins` inside the session.

### 3. Modify an existing skill

The plugin ships **7 skills** (per `skill-surface-collapse.adr.md`): `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`. Each lives at `skills/<name>/SKILL.md`. Adding an eighth top-level skill requires a new ADR — prefer adding flow logic under `skills/plan/references/` or `skills/decide/references/` instead.

Edit `skills/<name>/SKILL.md`. Required frontmatter fields: `name` (must match directory name), `description`. Optional: `argument-hint`. No skill carries `disable-model-invocation` — all 7 are auto-invocable.

Reload and test: `/reload-plugins`, then try `/archcore:<name>`.

#### 3a. Add a Codex slash command wrapper (required for user-facing skills)

Claude Code and Cursor surface skills directly in the `/` menu. Codex CLI does not — it discovers slash commands from root-level `commands/<name>.md` files. The plugin ships 7 wrappers, one per skill. If you ever add a new top-level skill (requires a new ADR), add the matching wrapper:

```markdown
---
description: <one-line description, ideally matching the skill's first sentence>
---

# /archcore:my-skill

## Arguments

The user invoked this command with: $ARGUMENTS

## Instructions

Use the Archcore skill at `skills/my-skill/SKILL.md`.
```

Wrappers carry no workflow logic — behavior lives in the skill, the single source of truth. `test/structure/codex-plugin.bats` enforces parity: every wrapper must exist, carry `description:`, and reference its matching `skills/<name>/SKILL.md`.

### 4. Add or modify hooks

Edit `hooks/hooks.json` (Claude Code), `hooks/cursor.hooks.json` (Cursor), or `hooks/codex.hooks.json` (Codex CLI) to add event handlers.

Hook scripts go in `bin/` and must:

- Start with `#!/bin/sh`
- Be executable (`chmod +x`)
- Source `bin/lib/normalize-stdin.sh` if they read hook stdin
- Add `# shellcheck source=lib/normalize-stdin.sh` before the source line
- Invoke the CLI directly as `archcore` (resolved via PATH); the plugin no longer ships a launcher wrapper
- If the script reads `.archcore/` or emits user-visible context, guard against being launched from a plugin install directory by exiting silently when cwd contains a sibling `.cursor-plugin/`, `.claude-plugin/`, or `.codex-plugin/` manifest (see `bin/session-start` for the canonical pattern, and `cursor-mcp-architecture.adr.md` for the rationale)

Each host's hook config uses its host's canonical plugin-root env var:

- `${CLAUDE_PLUGIN_ROOT}` — Claude Code's native injection (`hooks/hooks.json`).
- `${CURSOR_PLUGIN_ROOT}` — Cursor's native injection (`hooks/cursor.hooks.json`).
- `${PLUGIN_ROOT}` — Codex CLI's canonical, host-neutral env var (`hooks/codex.hooks.json`). Codex's hooks engine (`codex-rs/hooks/src/engine/discovery.rs`) injects `PLUGIN_ROOT` as the canonical name; `CLAUDE_PLUGIN_ROOT` is also injected but only as a backward-compat alias for porting old Claude plugins — do NOT use it in a Codex-native hook config. `CODEX_PLUGIN_ROOT` does not exist in Codex.

Plugin-shipped Codex hooks require `codex features enable plugin_hooks` to actually fire (the `plugin_hooks` feature is `under development, false` by default in Codex 0.130.0). See `codex-path-resolution.adr.md` for the full mechanism.

### 5. Modify agents

Edit `agents/archcore-assistant.md` or `agents/archcore-auditor.md`:

- Frontmatter: `name`, `description`, `model`, `maxTurns`, `tools`
- The auditor must remain read-only (only list_documents, get_document, list_relations MCP tools)

For Codex CLI, also update the matching TOML variant (`agents/archcore-assistant.toml`, `agents/archcore-auditor.toml`) — TOML and MD must keep identical `developer_instructions` content; structural drift is detected by `test/structure/agents.bats`.

### 6. Run tests

After any change, verify everything works:

```bash
make verify    # full check: JSON + permissions + shellcheck + tests
```

Or run individual checks:

```bash
make test           # all bats tests
make test-unit      # unit tests (bin script logic)
make test-structure # structure tests (configs, frontmatter)
make lint           # shellcheck
make check-json     # JSON validity
make check-perms    # executable permissions
```

`make verify` is the canonical way to run plugin integrity checks; there is no `/archcore:verify` skill (removed by `skill-surface-collapse.adr.md`).

See `plugin-testing.guide.md` for detailed testing instructions.

### 7. Test all components manually

- Skills: discuss relevant topics and verify Claude activates the skill
- Commands: run each `/archcore:<name>` command (in all three hosts where applicable) and verify behavior — Codex pulls these from `commands/`, Claude Code and Cursor pull them from `skills/`
- Agent: invoke the agent on a multi-document task
- Hooks: trigger Write/Edit on `.archcore/` and verify PreToolUse blocks it
- MCP availability: ensure `archcore` is on PATH and `archcore --version` works
- For Codex: from a directory **outside** the plugin source repo (e.g., `cd $(mktemp -d)`), call any `mcp__archcore__*` tool and verify the MCP starts.
- For Cursor: after copying `docs/cursor.mcp.example.json` into `.cursor/mcp.json`, open an empty project. `list_documents` should return empty (not the plugin's own dev docs). If it returns dev docs, the plugin-install-dir guards regressed — file an issue against this repo and `archcore-ai/cli`.
- Integrity check: `make verify`

## Verification

- `make verify` exits 0 with "All checks passed"
- `/reload-plugins` shows correct count of skills (7), agents (2), hooks (6 entries)
- `/help` lists all `/archcore:*` commands (7)
- `/agents` lists `archcore-assistant` and `archcore-auditor`
- Writing to `.archcore/*.md` via Write/Edit is blocked with a redirect message
- `archcore --version` works (CLI is on PATH or installed globally)
- For all hosts: MCP tools work from any project directory (the CLI resolves via PATH)

## Common Issues

### Plugin not loading

- Ensure `.claude-plugin/plugin.json` (Claude Code), `.cursor-plugin/plugin.json` (Cursor), or `.codex-plugin/plugin.json` (Codex CLI) exists and has valid JSON
- Check that directories (skills/, agents/, hooks/, commands/) are at the plugin root
- Run `claude --debug` to see plugin loading details

### Skill not activating

- Check the `description` field in SKILL.md frontmatter — it determines when Claude activates the skill
- Ensure `name` matches the directory name
- Run `/reload-plugins` after changes

### `/archcore:<name>` missing in Codex `/` menu

- Confirm `commands/<name>.md` exists and has `description:` frontmatter
- Confirm it references `skills/<name>/SKILL.md` (the bats parity test enforces this)
- Run `make test-structure` — `codex-plugin.bats` will flag missing or malformed wrappers
- Restart Codex after adding new wrappers (the marketplace cache is read once on session start)

### Hook not firing

- Ensure bin/ scripts are executable: `chmod +x bin/<name>`
- Check the shebang line: `#!/bin/sh`
- Verify the hook JSON structure matches the expected format
- Test scripts manually: `echo '{"tool_name":"Write","tool_input":{"file_path":".archcore/test.adr.md"}}' | bin/check-archcore-write`
- For Codex specifically: hooks require `codex features enable plugin_hooks` (the `plugin_hooks` feature is under development; absent the flag, Codex does not run plugin-shipped hooks)

### Tests failing

- Run `git submodule update --init` if bats helpers are missing
- On macOS, the test suite provides a `timeout` shim automatically
- Ensure `archcore` CLI is installed on PATH (`archcore --version`)
- See `plugin-testing.guide.md` for detailed troubleshooting

### MCP server not connecting (Claude Code / Codex CLI)

The plugin ships `.mcp.json` for Claude Code and `.codex.mcp.json` for Codex CLI. Diagnose in this order:

1. **Plugin loaded?** — `/plugin` (Claude Code) or `codex mcp list --json` (Codex CLI) should show `archcore`. If `.mcp.json`, `.codex.mcp.json`, or the Codex `mcpServers` pointer was modified or removed, the MCP server won't register; restore it from git.
2. **CLI available?** — run `archcore --version` from the terminal. Expected: prints a version.
   - Not found? → Install via the official installer: `curl -fsSL https://archcore.ai/install.sh | bash` (macOS/Linux/WSL) or `irm https://archcore.ai/install.ps1 | iex` (Windows). Full docs: https://docs.archcore.ai/cli/install/
   - Permission denied? → Check that the CLI binary is executable
3. **Session lifecycle** — Claude Code registers MCP servers at session start. If the CLI was missing at that moment, installing it mid-session will NOT reconnect the server. Restart the host (Claude Code / Codex CLI) after a fresh install.
4. **Duplicate suppression?** — if `/plugin` shows "Errors (1)" with an `archcore` MCP message, a user- or project-registered `archcore` has the same command. This is benign; the resolved binary is the same either way. To silence the warning, remove the redundant user/project registration.

### MCP server not connecting (Cursor)

Cursor uses a user-installed MCP, not a plugin-shipped one (deliberate — see `cursor-mcp-architecture.adr.md`). Copy `docs/cursor.mcp.example.json` into one of:

- `~/.cursor/mcp.json` — user-scoped, available in every workspace
- `.cursor/mcp.json` — project-scoped, only this project

The file ships with the right shape:

```json
{
  "mcpServers": {
    "archcore": {
      "type": "stdio",
      "command": "archcore",
      "args": ["mcp", "--project", "${workspaceFolder}"]
    }
  }
}
```

Key points:

- **`--project ${workspaceFolder}` is mandatory.** Cursor's MCP stdio schema has no `cwd` field; without `--project`, the server falls back to `os.Getwd()` which is unreliable for plugin-launched processes.
- **Do not add a `cwd` field.** Cursor silently ignores it; doing so is just confusing.
- **Do not copy the template to the plugin root.** A `cursor.mcp.json` at the plugin root would let Cursor's plugin-MCP auto-detection register the server with cwd = plugin install dir, leaking any bundled `.archcore/` (or other unintended state) instead of the user's workspace.

### "Plugin MCP Servers → archcore" appears in Cursor settings with stale documents

The plugin deliberately ships no Cursor plugin-MCP. If Cursor's "Plugin MCP Servers" section shows `archcore`, then either (a) an older plugin version with a plugin-root `cursor.mcp.json` is still cached, or (b) a regression introduced a plugin-root MCP file. Steps:

1. Uninstall the plugin from Cursor.
2. Remove `~/.cursor/plugins/cache/archcore-plugins/` (or the relevant cache subtree).
3. Reinstall the plugin from `main` (which is synthesized by the release workflow and has no plugin-root `cursor.mcp.json`).
4. Verify `test/structure/cursor-plugin.bats` passes — it asserts no legacy `cursor.mcp.json` at the plugin root.

If the symptom persists after a fresh `main` install, file an issue: the `cursor-mcp-architecture.adr.md` layered defense has a gap.
