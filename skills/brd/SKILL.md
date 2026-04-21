---
name: brd
argument-hint: "[topic]"
description: "Expert — Documents business requirements, objectives, and ROI to justify a project. Activates when making a business case or documenting ROI expectations — e.g., 'business case for X', 'justify this project', 'business requirements document'."
user-invocable: false
---

# BRD — Business Requirements Document

## When to use

- Justifying a project from a business perspective
- Documenting business objectives and ROI expectations

**Not BRD:**

- Market analysis → **mrd**
- User needs → **urd**
- Product scope → **prd**

## Quick create

1. `mcp__archcore__list_documents(types=["brd", "mrd", "urd"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What are the business objectives? What's the expected ROI?"
3. Compose content covering Business Objectives, Stakeholders, Business Rules, Success Metrics and ROI, Dependencies — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type            | Target   | When                             |
| --------- | --------------- | -------- | -------------------------------- |
| Outgoing  | `implements` by | BRS      | Formal business requirements     |
| Peer      | `related`       | MRD, URD | Peer source documents            |
| Outgoing  | `related`       | PRD      | Business context informs product |

**Flows:** MRD+**BRD**+URD → PRD; **BRD** → BRS
