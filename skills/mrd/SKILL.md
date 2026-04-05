---
name: mrd
argument-hint: "[topic]"
description: Documents market analysis including landscape, competitive positioning, and TAM/SAM/SOM sizing. Activates for market research, competitive analysis, or opportunity assessment.
---

# MRD — Market Requirements Document

## When to use

- Analyzing a market before defining a product
- Documenting competitive landscape and market sizing

**Not MRD:**
- Business justification → **brd**
- User needs → **urd**
- Product requirements → **prd**

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
