---
name: guide
argument-hint: "[topic]"
description: Provides step-by-step instructions for completing a specific task with prerequisites and verification. Activates for how-to procedures, setup instructions, or runbooks.
---

# Guide — How-To Instructions

## When to use

- Documenting a procedure someone needs to follow
- Writing runbooks or onboarding instructions

**Not Guide:**
- Mandatory standard → **rule**
- Reference lookup → **doc**
- Recording a decision → **adr**

## Quick create

1. `mcp__archcore__list_documents(types=["guide", "rule"])` — check duplicates
2. Ask: "What task does this guide help complete? What prerequisites?"
3. Compose content covering Prerequisites, Steps (numbered), Verification, Common Issues — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `depends_on` | rule | Rule this guide helps implement |
| Outgoing | `related` | ADR | Decision context behind procedure |

**Flows:** ADR → rule → **Guide**
