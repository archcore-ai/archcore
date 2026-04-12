---
name: help
description: Guide to Archcore commands and capabilities.
disable-model-invocation: true
---

# /archcore:help

Guide to what you can do with Archcore in Claude Code.

## When to use

- "What can I do with Archcore?"
- "Help"
- "What commands are available?"

## Routing table

No routing needed. Single behavior: present the command guide.

## Execution

Present the following guide:

---

## Quick Start

Most users start here. Describe what you need — the system picks the right document types automatically.

| Command | What it does |
|---|---|
| `/archcore:capture [topic]` | Document a module, component, or topic |
| `/archcore:plan [feature]` | Plan a feature end-to-end (idea → PRD → plan) |
| `/archcore:decide [topic]` | Record a technical decision |
| `/archcore:standard [topic]` | Establish a team standard (decision → rule → guide) |
| `/archcore:review` | Check documentation health and find gaps |
| `/archcore:status` | Quick dashboard of document counts and stats |
| `/archcore:actualize` | Detect stale docs and suggest updates |

**Tip:** You can also just describe what you need in natural language. Claude will pick the right document type automatically.

## Advanced — Multi-Document Flows

For users who know which documentation flow they need:

| Command | Flow |
|---|---|
| `/archcore:product-track [topic]` | idea → PRD → plan |
| `/archcore:sources-track [topic]` | MRD → BRD → URD |
| `/archcore:iso-track [topic]` | BRS → StRS → SyRS → SRS |
| `/archcore:architecture-track [topic]` | ADR → spec → plan |
| `/archcore:standard-track [topic]` | ADR → rule → guide |
| `/archcore:feature-track [topic]` | PRD → spec → plan → task-type |

## Expert — Single Document Types

Create a specific document type directly:

**Knowledge:** `/archcore:adr`, `/archcore:rfc`, `/archcore:rule`, `/archcore:guide`, `/archcore:doc`, `/archcore:spec`

**Vision:** `/archcore:prd`, `/archcore:idea`, `/archcore:mrd`, `/archcore:brd`, `/archcore:urd`, `/archcore:brs`, `/archcore:strs`, `/archcore:syrs`, `/archcore:srs`

**Experience:** `/archcore:task-type`, `/archcore:cpat`

---

## Result

The guide above, presented as-is. No additional commentary needed.
