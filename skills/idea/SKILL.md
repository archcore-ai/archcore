---
name: idea
argument-hint: "[topic]"
description: 'Expert — Captures product or technical concepts worth exploring before commitment. Activates for brainstorming, "what if" discussions, or early-stage concept exploration.'
---

# Idea — Concept Exploration

## When to use

- Capturing a concept during brainstorming
- Recording a "what if" that deserves later exploration

**Not Idea:**

- Ready to commit → **prd**
- Technical proposal needing review → **rfc**
- Decision already made → **adr**

## Prerequisite

Requires Archcore MCP tools. If `mcp__archcore__*` tools are not available in this session, **do not proceed** — tell the user:

**Archcore CLI is not installed.** To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

## Quick create

1. `mcp__archcore__list_documents(types=["idea"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What's the core concept? Who would benefit?"
3. Compose content covering Idea, Value, Possible Implementation, Risks and Constraints — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type            | Target      | When                         |
| --------- | --------------- | ----------- | ---------------------------- |
| Outgoing  | `implements` by | PRD         | Formalized into requirements |
| Outgoing  | `implements` by | RFC         | Developed into a proposal    |
| Peer      | `related`       | other ideas | Related concepts             |

**Flows:** **Idea** → PRD → plan; **Idea** → RFC → ADR
