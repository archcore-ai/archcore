---
name: srs
argument-hint: "[topic]"
description: Defines detailed software requirements including functional behavior, interfaces, and quality attributes following ISO 29148. Final level of the BRS→StRS→SyRS→SRS cascade.
---

# SRS — Software Requirements Specification (ISO 29148)

## When to use

- Translating system requirements (SyRS) into software-specific specs
- Defining detailed functional and non-functional software requirements

**Not SRS:**
- System-level → **syrs** (one level up)
- Informal product requirements → **prd**
- Technical contract for existing component → **spec**

## Quick create

1. `mcp__archcore__list_documents(types=["srs", "syrs"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What software is being specified? What SyRS does this implement?"
3. Compose content covering all SRS sections — using user's answers and upstream documents for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` — typically `implements` SyRS.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `implements` | SyRS | Formalizes system requirements |
| Outgoing | `implements` by | spec, plan | Realizes SRS requirements |

**Flows:** SyRS → **SRS** → spec/plan
