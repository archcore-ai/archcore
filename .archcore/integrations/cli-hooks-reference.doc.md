---
title: "CLI Hooks Reference"
status: accepted
tags:
  - "integrations"
---

## Overview

Archcore hooks intercept AI agent lifecycle events to inject documentation context at session start. The only active hook event is **SessionStart**, which provides agents with a list of existing documents and available MCP tools.

See also: [Supported AI Agents Registry](supported-ai-agents.doc.md) for agent-specific details, [Backup Invalid Configs](backup-invalid-configs.adr.md) for config file recovery behavior, [Disable Stop and Prompt Hooks ADR](disable-stop-and-prompt-hooks.adr.md) for why Stop/UserPromptSubmit hooks were removed.

## Commands

### `archcore hooks install`

Installs hooks for all detected agents. Also triggers `archcore mcp install` for MCP config.

```
archcore hooks install              # auto-detect agents
archcore hooks install --agent cursor  # specific agent only
```

### `archcore hooks <agent> session-start`

Handles the SessionStart hook event for an agent. These commands are invoked by the agent, not by users directly. They read JSON from stdin and write JSON to stdout.

```
archcore hooks claude-code session-start
archcore hooks cursor session-start
archcore hooks gemini-cli session-start
archcore hooks copilot session-start
```

## Hook Input (stdin JSON)

All hook commands read a JSON object from stdin.

| Field             | Type   | Description                   |
| ----------------- | ------ | ----------------------------- |
| `session_id`      | string | Unique session identifier     |
| `cwd`             | string | Current working directory     |
| `hook_event_name` | string | Name of the hook event        |
| `source`          | string | How the session was initiated |

Source: `cmd/hooks_claude_code.go` (`hookInput` struct)

## Hook Output (stdout JSON)

### SessionStart Response

```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "<injected text>"
  }
}
```

Source: `cmd/hooks_claude_code.go` (`hookOutput` struct)

## Session Context (SessionStart)

Built by `buildSessionContext()` in `cmd/hooks_common.go`. Injected at session start for all agents. Contains:

1. **Header** — Identifies archcore and available MCP tools (list_documents, get_document, create_document, update_document, add_relation, remove_relation, list_relations)
2. **Existing documents** — Grouped by category (`knowledge`, `vision`, `experience`) with filenames and titles
3. **Tags** — Top tag frequencies (up to 30) when tags are present in any document
4. **Document relations** — Summary count and available relation management tools
5. **MCP referral** — Points to MCP server instructions for document types and workflow rules

## Per-Agent Event Mapping

Each agent maps to a single hook event:

| Agent          | Hook Event   | Config File                   | Command                                    |
| -------------- | ------------ | ----------------------------- | ------------------------------------------ |
| Claude Code    | SessionStart | `.claude/settings.json`       | `archcore hooks claude-code session-start` |
| Cursor         | sessionStart | `.cursor/hooks.json`          | `archcore hooks cursor session-start`      |
| Gemini CLI     | SessionStart | `.gemini/settings.json`       | `archcore hooks gemini-cli session-start`  |
| GitHub Copilot | sessionStart | `.github/hooks/archcore.json` | `archcore hooks copilot session-start`     |

## Removed Hooks (Historical)

Stop and UserPromptSubmit hooks were removed due to excessive false positives from keyword matching. See [disable-stop-and-prompt-hooks.adr.md](disable-stop-and-prompt-hooks.adr.md) for full rationale.

Previously supported events:

- **Stop** — Scanned assistant messages for keywords (e.g., "decided to", "root cause") and blocked the agent to suggest creating documents.
- **UserPromptSubmit / BeforeSubmitPrompt / BeforeAgent** — Scanned user prompts for keywords (e.g., "should we use", "debug") and injected task-specific instructions.
