---
name: rfc
argument-hint: "[topic]"
description: "Expert — Proposes a significant technical change for team review before a decision is finalized. Activates when proposing a design, exploring a change that needs buy-in, or when the user says 'draft an RFC', 'design proposal', or 'let's get feedback before deciding'."
---

# RFC — Request for Comments

## When to use

- Proposing a significant technical change that needs buy-in
- Exploring a design that affects multiple systems

**Not RFC:**
- Decision already made → **adr**
- Informal concept → **idea**
- Defining a standard → **rule**

## Prerequisite

Requires Archcore MCP tools. If `mcp__archcore__*` tools are not available in this session, **do not proceed** — tell the user:

**Archcore CLI is not installed.** To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

## Quick create

1. `mcp__archcore__list_documents(types=["rfc", "adr"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What change are you proposing? What problem does it solve?"
3. Compose content covering Summary, Motivation, Detailed Design, Drawbacks, Alternatives — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `extends` | existing ADR | Revising a past decision |
| Outgoing | `related` | ADR | ADR recorded after approval |
| Incoming | `related` | idea | Idea that inspired this |

**Flows:** idea → **RFC** → ADR
