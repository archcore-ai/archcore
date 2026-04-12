---
name: syrs
argument-hint: "[topic]"
description: "Expert — Formalizes system requirements per ISO 29148."
---

# SyRS — System Requirements Specification (ISO 29148)

## When to use

- Translating stakeholder requirements (StRS) into system-level specs
- Defining system boundaries, interfaces, and operational modes

**Not SyRS:**
- Stakeholder needs → **strs** (one level up)
- Software-specific → **srs** (one level down)
- Informal product requirements → **prd**

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
