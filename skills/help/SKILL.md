---
name: help
description: "Show available Archcore commands and how to use them. Use when onboarding, exploring what skills are available, or when you're not sure which command to run."
---

# /archcore:help

Guide to what you can do with the Archcore plugin.

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
| `/archcore:graph [filter]` | Render the document relation graph (Mermaid) |

**Tip:** You can also just describe what you need in natural language. Claude will pick the right command automatically.

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

Create a specific document type directly. These commands are user-only shortcuts (the model routes through Quick Start commands for natural-language requests).

**Knowledge:** `/archcore:adr`, `/archcore:rfc`, `/archcore:rule`, `/archcore:guide`, `/archcore:doc`, `/archcore:spec`

**Vision:** `/archcore:prd`, `/archcore:idea`

**Experience:** `/archcore:task-type`, `/archcore:cpat`

## Hidden — Niche Discovery & ISO 29148 Types

Seven document types are not shown in `/` autocomplete to reduce cognitive load: `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs`. The model still knows about them and will reach them when you invoke the right track:

- **Market / business / user requirements (`mrd` / `brd` / `urd`):** use `/archcore:sources-track [topic]`
- **ISO 29148 cascade (`brs` → `strs` → `syrs` → `srs`):** use `/archcore:iso-track [topic]`
- **Direct creation for any type:** call the `mcp__archcore__create_document` tool with `type=<slug>`.

## Setup

If Archcore commands fail with MCP tool errors, the CLI needs to be installed:

1. **Install CLI:** `curl -fsSL https://archcore.ai/install.sh | bash`
2. **Initialize project:** `archcore init`
3. **Restart** the session

The plugin provides skills, agents, and hooks — but document operations (create, update, delete) require the Archcore CLI, which runs the MCP server.

---

## Result

The guide above, presented as-is. No additional commentary needed.
