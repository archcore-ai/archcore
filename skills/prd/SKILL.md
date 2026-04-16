---
name: prd
argument-hint: "[topic]"
description: Defines product requirements with vision, goals, and success metrics. Activates for product scoping, feature definitions, or when establishing what to build and why.
---

# PRD — Product Requirements Document

## When to use

- Defining requirements for a new product or major feature
- Establishing goals and success metrics before implementation

**Not PRD:**
- Market analysis → **mrd**
- Business justification → **brd**
- User needs → **urd**
- Formal software spec → **srs**
- Informal concept → **idea**

## Prerequisite

Requires Archcore MCP tools. If `mcp__archcore__*` tools are not available in this session, **do not proceed** — tell the user:

**Archcore CLI is not installed.** To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

## Quick create

1. `mcp__archcore__list_documents(types=["prd", "idea"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What problem are you solving? What are the success metrics?"
3. Compose content covering Vision, Problem Statement, Goals and Success Metrics, Requirements — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Incoming | `implements` | idea | PRD formalizes an idea |
| Incoming | `related` | MRD, BRD, URD | Sources inform PRD |
| Outgoing | `implements` by | plan, spec | Implements requirements |

**Flows:** idea → **PRD** → plan; MRD+BRD+URD → **PRD**
