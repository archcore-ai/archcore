---
name: plan
argument-hint: "[topic]"
description: Creates implementation plans with phased tasks, acceptance criteria, and dependencies. Activates for roadmaps, task breakdowns, or phased rollout planning.
---

# Plan — Implementation Plan

## When to use

- Breaking a PRD or feature into implementation tasks
- Creating a phased rollout or migration plan

**Not Plan:**
- Product requirements → **prd**
- Informal concept → **idea**
- Recurring task pattern → **task-type**

## Quick create

1. `mcp__archcore__list_documents(types=["plan", "prd", "adr"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What is the goal? What are the key phases?"
3. Compose content covering Goal, Tasks (phased), Acceptance Criteria, Dependencies — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `implements` | PRD | Executes requirements |
| Outgoing | `depends_on` | ADR | Relies on a decision |
| Peer | `related` | guides | Help execute tasks |

**Flows:** PRD → **Plan** → implementation
