---
name: urd
argument-hint: "[topic]"
description: "Expert — Documents user requirements, personas, and journeys. Activates when defining who the users are and what they need — e.g., 'user research', 'define our personas', 'document user journeys', 'usability requirements'."
---

# URD — User Requirements Document

## When to use

- Documenting who the users are and what they need
- Mapping user journeys and defining usability requirements

**Not URD:**
- Market analysis → **mrd**
- Business justification → **brd**
- Product scope → **prd**

## Prerequisite

Requires Archcore MCP tools. If `mcp__archcore__*` tools are not available in this session, **do not proceed** — tell the user:

**Archcore CLI is not installed.** To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

## Quick create

1. `mcp__archcore__list_documents(types=["urd", "mrd", "brd"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "Who are the users? What are their key needs?"
3. Compose content covering User Personas, User Journeys, User Requirements, Usability Requirements, Acceptance Criteria — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `implements` by | StRS | Formal stakeholder requirements |
| Peer | `related` | MRD, BRD | Peer source documents |
| Outgoing | `related` | PRD | User context informs product |

**Flows:** MRD+BRD+**URD** → PRD; **URD** → StRS
