---
name: cpat
argument-hint: "[topic]"
description: "Expert — Documents a code pattern change with before/after examples and migration scope. Activates when a coding pattern has changed — e.g., 'we switched from X to Y', 'document this refactor pattern', 'before/after code change'. For deciding to change, use /archcore:adr."
disable-model-invocation: true
---

# CPAT — Code Pattern Change

## When to use

- The team has decided to change a coding pattern
- Recording a before/after code shift with scope

**Not CPAT:**
- Deciding whether to change → **adr**
- Defining the new standard → **rule**
- Step-by-step migration → **guide**

## Quick create

1. `mcp__archcore__list_documents(types=["cpat", "adr", "rule"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What pattern changed? Show the before and after."
3. Compose content covering What Changed, Why, Before, After, Scope — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `implements` | ADR | Implements decision to change |
| Outgoing | `extends` | rule | Elaborates with before/after |
| Peer | `related` | rule | New standard being adopted |

**Flows:** ADR → **CPAT** + rule
