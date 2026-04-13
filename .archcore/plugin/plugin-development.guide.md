---
title: "Plugin Development Guide"
status: draft
tags:
  - "development"
  - "plugin"
---

## Prerequisites

- Archcore CLI installed and in PATH (`archcore --version`)
- Claude Code or Cursor installed with plugin support
- Git for version control
- A project with `.archcore/` initialized (`archcore init`)
- bats-core for tests (`brew install bats-core` on macOS)
- jq for JSON validation (`brew install jq`)
- ShellCheck (optional, `brew install shellcheck`)

## Steps

### 1. Clone the plugin repository

```bash
git clone https://github.com/archcore-ai/archcore-plugin.git
cd archcore-plugin
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

### 4. Add or modify hooks

Edit `hooks/hooks.json` (Claude Code) or `hooks/cursor.hooks.json` (Cursor) to add event handlers.

Hook scripts go in `bin/` and must:
- Start with `#!/bin/sh`
- Be executable (`chmod +x`)
- Source `bin/lib/normalize-stdin.sh` if they read hook stdin
- Add `# shellcheck source=lib/normalize-stdin.sh` before the source line

Use `${CLAUDE_PLUGIN_ROOT}` (Claude Code) or `${CURSOR_PLUGIN_ROOT}` (Cursor) in hook configs.

### 5. Modify agents

Edit `agents/archcore-assistant.md` or `agents/archcore-auditor.md`:

- Frontmatter: `name`, `description`, `model`, `maxTurns`, `tools`
- The auditor must remain read-only (only list_documents, get_document, list_relations MCP tools)

### 6. Run tests

After any change, verify everything works:

```bash
make verify    # full check: JSON + permissions + shellcheck + tests
```

Or run individual checks:

```bash
make test           # all 119 bats tests
make test-unit      # 69 unit tests (bin script logic)
make test-structure # 50 structure tests (configs, frontmatter)
make lint           # shellcheck
make check-json     # JSON validity
make check-perms    # executable permissions
```

Run `/archcore:verify` inside a Claude Code session for AI-assisted verification including live MCP tool checks.

See `plugin-testing.guide.md` for detailed testing instructions.

### 7. Test all components manually

- Skills: discuss relevant topics and verify Claude activates the skill
- Commands: run each `/archcore:<name>` command and verify behavior
- Agent: invoke the agent on a multi-document task
- Hooks: trigger Write/Edit on `.archcore/` and verify PreToolUse blocks it
- Verify: `/archcore:verify`

## Verification

- `make verify` exits 0 with "All checks passed"
- `/reload-plugins` shows correct count of skills, agents, hooks
- `/help` lists all `/archcore:*` commands
- `/agents` lists `archcore-assistant` and `archcore-auditor`
- Writing to `.archcore/*.md` via Write/Edit is blocked with a redirect message

## Common Issues

### Plugin not loading

- Ensure `.claude-plugin/plugin.json` (Claude Code) or `.cursor-plugin/plugin.json` (Cursor) exists and has valid JSON
- Check that directories (skills/, agents/, hooks/) are at the plugin root
- Run `claude --debug` to see plugin loading details

### Skill not activating

- Check the `description` field in SKILL.md frontmatter — it determines when Claude activates the skill
- Ensure `name` matches the directory name
- Run `/reload-plugins` after changes

### Hook not firing

- Ensure bin/ scripts are executable: `chmod +x bin/<name>`
- Check the shebang line: `#!/bin/sh`
- Verify the hook JSON structure matches the expected format
- Test scripts manually: `echo '{"tool_name":"Write","tool_input":{"file_path":".archcore/test.adr.md"}}' | bin/check-archcore-write`

### Tests failing

- Run `git submodule update --init` if bats helpers are missing
- On macOS, the test suite provides a `timeout` shim automatically
- See `plugin-testing.guide.md` for detailed troubleshooting

### MCP server not connecting

- Verify `archcore` is in PATH: `which archcore`
- Check `.mcp.json` (Claude Code) or `mcp.json` (Cursor) contains the correct server config
- Run `archcore mcp` manually to check for errors