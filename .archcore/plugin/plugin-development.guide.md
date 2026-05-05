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

That's it for developing against the plugin. The Archcore CLI is **not** a prerequisite — the plugin bundles a launcher (`bin/archcore{,.cmd,.ps1}`) that resolves the CLI on first use (from `$ARCHCORE_BIN`, `PATH`, a plugin-managed cache, or a checksum-verified download). MCP is registered automatically for Claude Code via plugin-root `.mcp.json`, and for Codex CLI via `.codex-plugin/plugin.json` pointing at plugin-root `.codex.mcp.json`.

For Cursor development, you still register MCP externally (via Cursor's MCP settings or a project `mcp.json`). Point Cursor's MCP config at `${CURSOR_PLUGIN_ROOT}/bin/archcore` with args `["mcp"]`, or at a globally-installed `archcore`.

For Codex development, `codex plugin marketplace add /path/to/plugin` registers the marketplace. The current CLI loads enabled plugins from its installed plugin cache; run `make test-codex-smoke` for the local installed-cache smoke that verifies skill discovery and plugin-managed MCP.

If you want to run the MCP server against a pre-existing global install or a locally-built CLI, set `ARCHCORE_BIN=/abs/path/to/archcore` — the launcher will use that binary and skip the cache/download path.

Initialize a project for testing with `mcp__archcore__init_project` (via a Claude Code or Cursor session) rather than an out-of-band CLI command; the plugin routes initialization through MCP.

## Steps

### 1. Clone the plugin repository

```bash
git clone https://github.com/archcore-ai/plugin.git
cd plugin
git submodule update --init   # pulls bats-support and bats-assert
```

### 2. Run the host with the plugin loaded locally

```bash
claude --plugin-dir .    # Claude Code
cursor --plugin-dir .    # Cursor
```

This loads the plugin from the current directory without requiring marketplace installation. Changes to plugin files are picked up after running `/reload-plugins` inside the session.

### 3. Add a new skill

Create a directory for the skill under `skills/`:

```bash
mkdir -p skills/my-skill
```

Create `skills/my-skill/SKILL.md` with YAML frontmatter:

```yaml
---
name: my-skill
argument-hint: "[topic]"
description: What this skill does and when to activate it.
---
```

Required frontmatter fields: `name` (must match directory name), `description`. Optional: `argument-hint`, `disable-model-invocation: true` (for user-only skills).

Reload and test: `/reload-plugins`, then try `/archcore:my-skill`.

#### 3a. Add a Codex slash command wrapper (required for user-facing skills)

Claude Code and Cursor surface skills directly in the `/` menu. Codex CLI does not — it discovers slash commands from root-level `commands/<name>.md` files. For every user-facing skill (intent, track, or utility), add a thin wrapper at `commands/my-skill.md`:

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

Wrappers carry no workflow logic — behavior lives in the skill, the single source of truth. `test/structure/codex-plugin.bats` enforces parity: every wrapper must exist, carry `description:`, and reference its matching `skills/<name>/SKILL.md`. Skills with `disable-model-invocation: true` (user-only utilities like `verify`) still get wrappers because they are user-invocable in the `/` menu.

### 4. Add or modify hooks

Edit `hooks/hooks.json` (Claude Code), `hooks/cursor.hooks.json` (Cursor), or `hooks/codex.hooks.json` (Codex CLI) to add event handlers.

Hook scripts go in `bin/` and must:

- Start with `#!/bin/sh`
- Be executable (`chmod +x`)
- Source `bin/lib/normalize-stdin.sh` if they read hook stdin
- Add `# shellcheck source=lib/normalize-stdin.sh` before the source line
- Invoke the CLI through `"$SCRIPT_DIR/archcore"` (the launcher) rather than a bare `archcore`, so the resolution order (ARCHCORE_BIN → PATH → cache → download) applies

Use `${CLAUDE_PLUGIN_ROOT}` (Claude Code), `${CURSOR_PLUGIN_ROOT}` (Cursor), or plugin-relative `./bin/...` (Codex CLI — does not expose a documented plugin-root variable) in hook configs.

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
make test-unit      # unit tests (bin script logic, incl. launcher.bats)
make test-structure # structure tests (configs, frontmatter)
make lint           # shellcheck
make check-json     # JSON validity
make check-perms    # executable permissions
```

Run `/archcore:verify` inside a Claude Code session for AI-assisted verification including live MCP tool checks.

See `plugin-testing.guide.md` for detailed testing instructions.

### 7. Test all components manually

- Skills: discuss relevant topics and verify Claude activates the skill
- Commands: run each `/archcore:<name>` command (in all three hosts where applicable) and verify behavior — Codex pulls these from `commands/`, Claude Code and Cursor pull them from `skills/`
- Agent: invoke the agent on a multi-document task
- Hooks: trigger Write/Edit on `.archcore/` and verify PreToolUse blocks it
- Launcher: temporarily unset `ARCHCORE_BIN`, remove the cached binary, and confirm the next MCP call downloads and caches the CLI without prompting
- Verify: `/archcore:verify`

### 8. Bumping the bundled CLI version

When you want the plugin to pull in a new Archcore CLI release:

1. Edit `bin/CLI_VERSION` — replace with the new semver (e.g., `0.1.7`).
2. Run `make verify` — the structure tests confirm all launcher scripts still reference the file correctly.
3. Manually exercise the launcher: unset `ARCHCORE_BIN`, delete the cached `archcore-v<old>` binary, trigger an MCP call. Confirm the new binary downloads, verifies, caches, and runs.

No other changes required — the cache is version-keyed by filename so old binaries don't need explicit eviction.

## Verification

- `make verify` exits 0 with "All checks passed"
- `/reload-plugins` shows correct count of skills, agents, hooks
- `/help` lists all `/archcore:*` commands
- `/agents` lists `archcore-assistant` and `archcore-auditor`
- Writing to `.archcore/*.md` via Write/Edit is blocked with a redirect message
- A fresh install (no global `archcore`, no cache) resolves a CLI binary on first MCP call

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

### Tests failing

- Run `git submodule update --init` if bats helpers are missing
- On macOS, the test suite provides a `timeout` shim automatically
- See `plugin-testing.guide.md` for detailed troubleshooting

### MCP server not connecting (Claude Code / Codex CLI)

The plugin ships `.mcp.json` for Claude Code and `.codex.mcp.json` for Codex CLI. Diagnose in this order:

1. **Plugin loaded?** — `/plugin` (Claude Code) or `codex mcp list --json` (Codex CLI) should show `archcore`. If `.mcp.json`, `.codex.mcp.json`, or the Codex `mcpServers` pointer was modified or removed, the MCP server won't register; restore it from git.
2. **Launcher resolves?** — run `bin/archcore --version` from the plugin root. Expected: prints a version. Errors indicate:
   - Missing `bin/CLI_VERSION` → restore from git.
   - Network failure on first run → re-run with network, or set `ARCHCORE_BIN=/abs/path/to/archcore` to bypass download.
   - Checksum mismatch → corrupt download; delete the cache dir and retry.
3. **Duplicate suppression?** — if `/plugin` shows "Errors (1)" with an `archcore` MCP message, a user- or project-registered `archcore` has the same command. This is benign; the resolved binary is the same either way. To silence the warning, remove the redundant user/project registration.
4. **Using a custom CLI?** — if `ARCHCORE_BIN` is set but points at a non-existent or non-executable path, the launcher falls back to PATH/cache/download. Check the path and permissions.

### MCP server not connecting (Cursor)

Cursor does not auto-register the plugin's MCP. Configure it in Cursor's MCP settings or a project `mcp.json`:

```json
{
  "mcpServers": {
    "archcore": {
      "command": "/abs/path/to/plugin/bin/archcore",
      "args": ["mcp"]
    }
  }
}
```

Alternatively, install the CLI globally and point `command` at `archcore`. In both cases the launcher / resolved binary is the same.
