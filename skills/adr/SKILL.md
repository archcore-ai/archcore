---
name: adr
argument-hint: "[topic]"
description: Records architectural decisions with context, alternatives, and consequences. Activates for finalized technical decisions, technology choices, or trade-off discussions.
---

# ADR — Architecture Decision Record

## When to use

- A technical decision has been finalized
- Recording why a specific approach was chosen over alternatives

**Not ADR:**

- Still discussing → **rfc**
- Mandatory standard → **rule**
- How-to instructions → **guide**

## Prerequisite

Requires Archcore MCP tools. If `mcp__archcore__*` tools are not available in this session, **do not proceed** — tell the user:

**Archcore CLI is not installed.** To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

## Quick create

1. `mcp__archcore__list_documents(types=["adr"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What was the decision? What alternatives were considered?"
3. Compose content covering Context, Decision, Alternatives Considered, Consequences — using user's answers for depth. Pass as `content` parameter to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type         | Target     | When                      |
| --------- | ------------ | ---------- | ------------------------- |
| Incoming  | `implements` | plan, spec | Implements this decision  |
| Outgoing  | `related`    | rule       | Codifies this decision    |
| Outgoing  | `related`    | guide      | Explains how to follow it |

**Flows:** RFC → **ADR** → rule → guide
