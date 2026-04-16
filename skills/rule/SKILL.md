---
name: rule
argument-hint: "[topic]"
description: Defines mandatory team standards and required behaviors with rationale and examples. Activates for coding conventions, enforceable practices, or standards codified from decisions.
---

# Rule — Team Standard

## When to use

- Defining a mandatory coding standard or convention
- Codifying a decision (ADR) into an enforceable practice

**Not Rule:**
- Recording a decision → **adr**
- Step-by-step instructions → **guide**
- Reference material → **doc**

## Prerequisite

Requires Archcore MCP tools. If `mcp__archcore__*` tools are not available in this session, **do not proceed** — tell the user:

**Archcore CLI is not installed.** To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

## Quick create

1. `mcp__archcore__list_documents(types=["rule", "adr"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What must the team always/never do? What motivated this standard?"
3. Compose content covering Rule (imperative statements), Rationale, Examples (Good/Bad), Enforcement — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Incoming | `implements` | ADR | Decision that led to this rule |
| Outgoing | `related` | guide | How to follow this rule |
| Outgoing | `related` | spec | Formalizes this rule's scope |

**Flows:** ADR → **Rule** → guide
