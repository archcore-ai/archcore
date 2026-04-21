---
name: brs
argument-hint: "[topic]"
description: "Expert — Formalizes business requirements into a traceable specification per ISO 29148. Activates when converting informal business requirements into structured, auditable specs, or starting an ISO 29148 cascade. Use /archcore:brd for informal business cases."
---

# BRS — Business Requirements Specification (ISO 29148)

## When to use

- Formalizing business requirements from MRD/BRD into traceable specs
- Starting the ISO 29148 requirements cascade

**Not BRS:**
- Informal business case → **brd**
- Product requirements → **prd**
- Stakeholder requirements → **strs** (next level)

## Quick create

1. `mcp__archcore__list_documents(types=["brs", "brd", "mrd"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What business goals does this formalize? What source documents exist?"
3. Compose content covering all BRS sections — using user's answers and source documents for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` — typically `implements` BRD/MRD.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `implements` | MRD or BRD | Formalizes informal requirements |
| Outgoing | `implements` by | StRS | Next level in cascade |

**Flows:** MRD/BRD → **BRS** → StRS → SyRS → SRS
