---
name: mrd
argument-hint: "[topic]"
description: "Expert — Documents market analysis, competitive positioning, and market sizing (TAM/SAM/SOM). Activates when analyzing a market before defining a product — e.g., 'market research', 'competitive analysis', 'market opportunity document'."
---

# MRD — Market Requirements Document

## When to use

- Analyzing a market before defining a product
- Documenting competitive landscape and market sizing

**Not MRD:**
- Business justification → **brd**
- User needs → **urd**
- Product requirements → **prd**

## Prerequisite

Requires Archcore MCP tools. If `mcp__archcore__*` tools are not available in this session, **do not proceed** — tell the user:

**Archcore CLI is not installed.** To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

## Quick create

1. `mcp__archcore__list_documents(types=["mrd", "brd", "urd"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What market are you analyzing? What's the key opportunity?"
3. Compose content covering Market Landscape, TAM/SAM/SOM, Competitive Analysis, Market Needs, Opportunity and Timing — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Outgoing | `implements` by | BRS | Formal requirements from market |
| Peer | `related` | BRD, URD | Peer source documents |
| Outgoing | `related` | PRD | Market context informs product |

**Flows:** **MRD**+BRD+URD → PRD; **MRD** → BRS → StRS → SyRS → SRS
