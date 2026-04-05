---
title: "Plugin Development Guide"
status: draft
tags:
  - "development"
  - "plugin"
---

## Prerequisites

- Archcore CLI installed and in PATH (`archcore --version`)
- Claude Code installed with plugin support (`claude --version`)
- Git for version control
- A project with `.archcore/` initialized (`archcore init`)

## Steps

### 1. Clone the plugin repository

```bash
git clone https://github.com/archcore-ai/archcore-claude-plugin.git
cd archcore-claude-plugin
```

### 2. Run Claude Code with the plugin loaded locally

```bash
claude --plugin-dir .
```

This loads the plugin from the current directory without requiring marketplace installation. Changes to plugin files are picked up after running `/reload-plugins` inside the session.

### 3. Add a new skill

Create a directory for the document type under `skills/`:

```bash
mkdir -p skills/adr
```

Create `skills/adr/SKILL.md` following the Skill File Structure Standard:

- Frontmatter with `name: archcore-adr` and `description` (trigger conditions)
- 7 required sections: Overview, When to Use, Required Sections, Best Practices, Common Mistakes, Relation Guidance, Example Workflow
- Example Workflow must use `create_document` MCP tool, never Write/Edit

Reload and test: `/reload-plugins`, then discuss a topic that should trigger the skill.

### 4. Add a new command

Create a Markdown file under `commands/`:

```bash
touch commands/create.md
```

Add frontmatter with `description` and prompt instructions that use MCP tools. Commands are invoked as `/archcore:<filename>` (e.g., `/archcore:create`).

Reload and test: `/reload-plugins`, then run the command.

### 5. Modify the agent

Edit `agents/archcore-assistant.md`:

- Frontmatter: `name`, `description`, `tools` (MCP tools + Read/Grep/Glob)
- System prompt: cover all 18 document types, requirements engineering patterns, relation guidance

Reload and test: `/reload-plugins`, then check `/agents` to see the agent listed.

### 6. Add or modify hooks

Edit `hooks/hooks.json` to add event handlers. Supported events: SessionStart, PreToolUse, PostToolUse, and others.

Hook scripts go in `bin/` and must be executable. Use `${CLAUDE_PLUGIN_ROOT}` to reference them in hooks.json.

### 7. Test all components

- Skills: discuss relevant topics and verify Claude activates the skill
- Commands: run each `/archcore:<name>` command and verify behavior
- Agent: invoke the agent on a multi-document task
- Hooks: trigger Write/Edit on `.archcore/` and verify PreToolUse blocks it
- Validate: `claude plugin validate .`

## Verification

- `/reload-plugins` shows correct count of skills, commands, agents, hooks
- `/help` lists all `/archcore:*` commands
- `/agents` lists `archcore-assistant`
- Writing to `.archcore/*.md` via Write/Edit is blocked with a redirect message
- `claude plugin validate .` passes with no errors

## Common Issues

### Plugin not loading

- Ensure `.claude-plugin/plugin.json` exists and has valid JSON
- Check that directories (skills/, commands/, agents/, hooks/) are at the plugin root, NOT inside `.claude-plugin/`
- Run `claude --debug` to see plugin loading details

### Skill not activating

- Check the `description` field in SKILL.md frontmatter — it determines when Claude activates the skill
- Ensure the description contains specific trigger words/phrases
- Run `/reload-plugins` after changes

### Hook not firing

- Ensure bin/ scripts are executable: `chmod +x bin/check-archcore-write`
- Check the shebang line: `#!/bin/sh`
- Verify the hook JSON structure matches the expected format
- Test scripts manually: `echo '{"tool_input":{"file_path":".archcore/test.adr.md"}}' | bin/check-archcore-write`

### MCP server not connecting

- Verify `archcore` is in PATH: `which archcore`
- Check `.mcp.json` contains the correct server config
- Run `archcore mcp` manually to check for errors
