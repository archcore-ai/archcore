---
name: strs
argument-hint: "[topic]"
description: "Expert — Formalizes stakeholder requirements per ISO 29148 by decomposing BRS into stakeholder-specific specs. Activates when multiple stakeholder groups have different requirement sets, or progressing through an ISO 29148 cascade after BRS. Use /archcore:urd for informal user needs."
user-invocable: false
---

# StRS — Stakeholder Requirements Specification (ISO 29148)

## When to use

- Decomposing BRS into stakeholder-specific requirements
- Multiple stakeholder groups with different needs

**Not StRS:**
- Informal user needs → **urd**
- Business-level → **brs** (one level up)
- System-level → **syrs** (one level down)

## Quick create

1. `mcp__archcore__list_documents(types=["strs", "brs", "urd"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What stakeholder classes exist? What BRS does this implement?"
3. Compose content covering all StRS sections — using user's answers and upstream documents for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` — typically `implements` BRS.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `implements` | BRS | Formalizes business requirements |
| Incoming | `implements` | URD | Formalizes user needs |
| Outgoing | `implements` by | SyRS | Next level in cascade |

**Flows:** BRS → **StRS** → SyRS → SRS
