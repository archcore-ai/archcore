---
name: task-type
argument-hint: "[topic]"
description: "Expert — Documents a recurring task pattern with steps, variations, and known pitfalls. Activates when a task is performed repeatedly — e.g., 'document our release process', 'playbook for X', 'we always do this the same way'. For one-time procedures use /archcore:guide."
---

# Task-Type — Recurring Task Pattern

## When to use

- A task is performed repeatedly with a standard approach
- Documenting tribal knowledge before it's lost

**Not Task-Type:**
- One-time procedure → **guide**
- Mandatory standard → **rule**
- One-off implementation → **plan**

## Prerequisite

Requires Archcore MCP tools. If `mcp__archcore__*` tools are not available in this session, **do not proceed** — tell the user:

**Archcore CLI is not installed.** To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

## Quick create

1. `mcp__archcore__list_documents(types=["task-type", "rule", "guide"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What recurring task does this cover? What are the key steps?"
3. Compose content covering What, When to Use, Steps, Example, Things to Watch Out For — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `depends_on` | rule | Rule the pattern follows |
| Peer | `related` | guide | Related procedures |

**Flows:** rule → **Task-Type**
