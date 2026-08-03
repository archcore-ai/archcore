---
title: "CLI Hooks Reference"
status: accepted
tags:
  - "integrations"
---

## Overview

Archcore hooks intercept AI agent lifecycle events and inject documentation context at session start. SessionStart is the only active hook event. It gives the agent a list of existing documents and the available MCP tools.

The related documents on the supported agent registry, on config-file backup behavior, and on the removal of the Stop and UserPromptSubmit hooks carry the neighboring detail.

## Commands

### `archcore hooks install`

Installs hooks for every detected agent, and triggers `archcore mcp install` for the MCP config.

```
archcore hooks install                 # auto-detect agents
archcore hooks install --agent cursor  # specific agent only
```

### `archcore hooks <agent> session-start`

Handles the SessionStart hook event for one agent. The agent invokes these commands; a user does not run them directly. Each reads JSON from stdin and writes JSON to stdout.

```
archcore hooks claude-code session-start
archcore hooks cursor session-start
archcore hooks gemini-cli session-start
archcore hooks copilot session-start
```

## Hook input (stdin JSON)

Every hook command reads one JSON object from stdin.

| Field             | Type   | Description                   |
| ----------------- | ------ | ----------------------------- |
| `session_id`      | string | Unique session identifier     |
| `cwd`             | string | Current working directory     |
| `hook_event_name` | string | Name of the hook event        |
| `source`          | string | How the session was initiated |

Source: the `hookInput` struct in `@cmd/hooks_claude_code.go`.

## Hook output (stdout JSON)

### SessionStart response

```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "<injected text>"
  }
}
```

Source: the `hookOutput` struct in `@cmd/hooks_claude_code.go`.

## Session context (SessionStart)

`buildSessionContext()` in `@cmd/hooks_common.go` builds the injected text for every agent. It contains:

1. Header — identifies Archcore and the available MCP tools (`list_documents`, `get_document`, `create_document`, `update_document`, `add_relation`, `remove_relation`, `list_relations`).
2. Existing documents — grouped by category (`knowledge`, `vision`, `experience`) with filenames and titles.
3. Tags — the top tag frequencies, up to 30, when any document carries tags.
4. Document relations — a summary count and the relation-management tools.
5. MCP referral — points to the MCP server instructions for document types and workflow rules.

## Per-agent event mapping

Each agent maps to a single hook event.

| Agent          | Hook Event   | Config File                   | Command                                    |
| -------------- | ------------ | ----------------------------- | ------------------------------------------ |
| Claude Code    | SessionStart | `.claude/settings.json`       | `archcore hooks claude-code session-start` |
| Cursor         | sessionStart | `.cursor/hooks.json`          | `archcore hooks cursor session-start`      |
| Gemini CLI     | SessionStart | `.gemini/settings.json`       | `archcore hooks gemini-cli session-start`  |
| GitHub Copilot | sessionStart | `.github/hooks/archcore.json` | `archcore hooks copilot session-start`     |

## Removed hooks (historical)

The Stop and UserPromptSubmit hooks were removed because keyword matching produced too many false positives. The related ADR records the full rationale.

Previously supported events:

- Stop — scanned assistant messages for keywords such as "decided to" or "root cause", then blocked the agent to suggest creating a document.
- UserPromptSubmit, BeforeSubmitPrompt, and BeforeAgent — scanned user prompts for keywords such as "should we use" or "debug", then injected task-specific instructions.
