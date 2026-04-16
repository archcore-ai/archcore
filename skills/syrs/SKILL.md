---
name: syrs
argument-hint: "[topic]"
description: "Expert — Formalizes system requirements per ISO 29148 by translating StRS into system-level specifications covering boundaries, interfaces, and operational modes. Activates after completing StRS in an ISO 29148 cascade. Not for software-specific requirements — use /archcore:srs."
---

# SyRS — System Requirements Specification (ISO 29148)

## When to use

- Translating stakeholder requirements (StRS) into system-level specs
- Defining system boundaries, interfaces, and operational modes

**Not SyRS:**
- Stakeholder needs → **strs** (one level up)
- Software-specific → **srs** (one level down)
- Informal product requirements → **prd**

## Prerequisite

Requires Archcore MCP tools. If `mcp__archcore__*` tools are not available in this session, **do not proceed** — tell the user:

**Archcore CLI is not installed.** To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

## Quick create

1. `mcp__archcore__list_documents(types=["syrs", "strs"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What system is being specified? What StRS does this implement?"
3. Compose content covering all SyRS sections — using user's answers and upstream documents for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` — typically `implements` StRS.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `implements` | StRS | Formalizes stakeholder requirements |
| Outgoing | `implements` by | SRS | Next level in cascade |

**Flows:** StRS → **SyRS** → SRS
