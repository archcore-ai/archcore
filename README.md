# Archcore CLI

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/archcore-ai/cli)](https://github.com/archcore-ai/cli/releases)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)](https://github.com/archcore-ai/cli/releases)

**Your AI agent stops guessing and starts following your architecture.**

> Git ships your code. CI/CD ships your delivery. Archcore ships your understanding.

Archcore stores your decisions, rules, and conventions in Git — so your AI agent follows them automatically. Works across Claude Code, Cursor, Copilot, Gemini CLI, Codex, OpenCode, Roo Code, and Cline.

Archcore ships as a CLI **and** a local stdio MCP server — any MCP-compatible coding agent can read and write your repo context through standard tools, while the Claude Code / Cursor plugin adds a higher-level workflow layer.

> **Using Claude Code or Cursor?** Pair the CLI with the [Archcore Plugin](https://github.com/archcore-ai/archcore-plugin) — same engine, plus skills, intent commands, and guardrails out of the box. Sticking with the CLI is great too — it works across every other agent.

## In 60 seconds

```bash
curl -fsSL https://archcore.ai/install.sh | bash
cd your-project && archcore init
```

Then open your AI agent and say:

> _"We're using PostgreSQL for primary storage. Record this decision."_

Done. There's now a structured ADR in `.archcore/` that every future session — in any agent — can read.

> On **Windows**? Use PowerShell: `irm https://archcore.ai/install.ps1 | iex`. For WSL, `go install`, and other options, see [Install methods](#install-methods) or the [full install guide](https://docs.archcore.ai/cli/install/).

## Ask your AI things like

Once your repo has a few documents, your agent can use them. Try:

> _"Before I touch the auth module, what ADRs and rules apply here?"_

The agent loads the relevant decisions and rules tied to that area before it edits a single line.

> _"Add a new API handler and follow this repo's conventions."_

The agent surfaces the matching rule (e.g. "handlers live in `src/api/handlers/`") and places code where your architecture says it belongs.

> _"What's our error-handling rule?"_

The agent reads `error-wrapping.rule.md` straight from `.archcore/` instead of guessing from a few examples in the codebase.

## Try these first

These prompts capture _new_ context — decisions, rules, plans, incidents. Each creates a structured document the agent (or any teammate) can later reuse.

_New repo? `archcore init` creates `.archcore/`. The MCP server also works in an empty repo and exposes an `init_project` tool, so the agent can bootstrap for you._

> _"We decided to use PostgreSQL instead of MongoDB for our primary database. Record this decision."_

Creates `infrastructure/use-postgres.adr.md` with context, decision, alternatives considered, and consequences.

> _"We have a team convention: always wrap errors with context using fmt.Errorf and %w. Make this a rule."_

Creates `backend/error-wrapping.rule.md` with imperative guidance, rationale, and good/bad code examples.

> _"Last week we had a connection pool exhaustion incident because idle connections weren't being recycled. Document this so we don't repeat it."_

Creates `incidents/connection-pool-exhaustion.cpat.md` with root-cause analysis and prevention steps.

> _"I need a PRD for the user notifications feature — push, email digests, and in-app alerts."_

Creates `notifications/user-notifications.prd.md` with goals, user stories, requirements, and success metrics.

> _"Create an implementation plan for the notifications PRD and link them together."_

Creates `notifications/notifications-implementation.plan.md`, then links it to the PRD with an `implements` relation.

If any of these resonates, the rest of Archcore is more of the same — just structured.

## What changes after install

Without Archcore, the agent:

- ignores your architecture
- breaks your conventions
- duplicates logic that already exists
- re-litigates decisions your team already made
- needs the same conventions repeated in every chat
- loses project truth the moment the session ends

With Archcore, the same asks produce code that:

- lands where your architecture says it belongs
- respects ADRs, specs, and rules already in Git
- follows team conventions loaded automatically on session start
- reflects new decisions as future guardrails, not markdown graveyards

> **AI should follow your system, not guess it.**

## Use Archcore when

- Your agent writes code, but not in the way this repo expects
- Your `CLAUDE.md` / `.cursorrules` / `AGENTS.md` keeps growing and drifting
- You work with 2+ agents or 2+ host tools (Claude Code + Cursor + Copilot)
- You want decisions, rules, and specs in Git — not in chat scrollback

**Not for** — chat memory, a prompt library, or a one-shot spec-to-code generator. Archcore is a repo truth layer for coding agents, not a methodology kit.

## Why not just instruction files?

`CLAUDE.md`, `AGENTS.md`, and repository instructions are useful starting points, but they break down when your team needs:

- more than one flat memory file
- structured document types — ADRs, rules, plans, incidents
- reusable context across multiple AI tools
- versioned project knowledge that grows with the codebase
- relations between documents (a plan that _implements_ a PRD, an RFC that _extends_ an ADR)
- incident learnings and recurring workflows that agents can pick up later

Instruction files tell the agent _what you want_. Archcore tells the agent _how your system works_ — so the agent can follow your system instead of guessing it.

## Supported agents

Archcore CLI is itself a local stdio MCP server — that is the shared integration surface for every MCP-compatible agent in the table below. Hooks add proactive session-start context where the agent supports them.

| Agent          | Hooks | MCP    |
| -------------- | ----- | ------ |
| Claude Code    | yes   | yes    |
| Cursor         | yes   | yes    |
| Gemini CLI     | yes   | yes    |
| GitHub Copilot | yes   | yes    |
| OpenCode       | —     | yes    |
| Codex CLI      | —     | yes    |
| Roo Code       | —     | yes    |
| Cline          | —     | manual |

---

## How it works

1. **Initialize your repo**
   `archcore init` creates `.archcore/` and installs integrations for supported agents.

2. **Capture durable context**
   Store architecture decisions, rules, plans, product docs, and incident learnings as structured Markdown files.

3. **Let agents reuse it**
   Hooks and MCP let your coding agents read existing context and create or update documents during real work.

4. **Keep it in Git**
   Review context changes like code, evolve them over time, and keep them portable across tools.

### Mental model

Archcore CLI is the **context compiler** — it turns scattered documents into structured, machine-readable context. MCP and hooks are the **runtime** — the surface agents use to consume that context during real work. The [Archcore Plugin](https://github.com/archcore-ai/archcore-plugin) for Claude Code and Cursor is a higher-level runtime built on top.

```text
implicit repo knowledge  →  structured context  →  AI-readable system
```

## What lives in `.archcore/`

```text
.archcore/
├── settings.json
├── .sync-state.json
├── auth/
│   ├── jwt-strategy.adr.md
│   └── auth-redesign.prd.md
├── backend/
│   └── error-wrapping.rule.md
├── incidents/
│   └── connection-pool-exhaustion.cpat.md
└── notifications/
    └── notifications-implementation.plan.md
```

The structure is **free-form** — organize documents by domain, feature, team, or whatever fits your repo. Categories are virtual and inferred from the document type in the filename (`slug.type.md`).

Use `.archcore/` for:

- architecture decisions
- coding rules and conventions
- implementation plans
- product requirements
- incidents and postmortems
- reusable workflow knowledge

See the Archcore CLI repository itself for a working example: [`.archcore/` in this repo](https://github.com/archcore-ai/cli/tree/main/.archcore)

## What ships in the box

- **18 document types** across vision, knowledge, and experience
- **4 relation types** — `related`, `implements`, `extends`, `depends_on`
- **10 MCP tools** — `list_documents`, `get_document`, `create_document`, `update_document`, `remove_document`, `search_documents`, `init_project`, plus relation management (`add_relation`, `remove_relation`, `list_relations`)
- **5 multi-document prompts** — track cascades invokable as slash commands from MCP-compatible agents
- **Hook integrations** for 4 agents (Claude Code, Cursor, Gemini CLI, GitHub Copilot) and **MCP integrations** for 8

## Document types

Archcore organizes context into 3 layers of knowledge: Vision, Knowledge, and Experience.

### Vision

| Type   | Full Name                     | Description                                                               |
| ------ | ----------------------------- | ------------------------------------------------------------------------- |
| `prd`  | Product Requirements Document | Goals, user stories, acceptance criteria, and success metrics             |
| `idea` | Idea                          | Lightweight capture of a product or technical idea for future exploration |
| `plan` | Plan                          | Phased task list with acceptance criteria and dependencies                |

Archcore also supports two additional requirements tracks for teams that need structured discovery or formal decomposition:

**Sources track** (MRD → BRD → URD) — captures _where_ requirements come from:

| Type  | Full Name                      | Description                                                              |
| ----- | ------------------------------ | ------------------------------------------------------------------------ |
| `mrd` | Market Requirements Document   | Market landscape, TAM/SAM/SOM, competitive analysis, and market needs    |
| `brd` | Business Requirements Document | Business objectives, stakeholders, ROI, and business rules               |
| `urd` | User Requirements Document     | User personas, journeys, usability requirements, and acceptance criteria |

**ISO/IEC/IEEE 29148:2018 track** (BRS → StRS → SyRS → SRS) — captures _how_ requirements decompose:

| Type   | Full Name                              | Description                                                            |
| ------ | -------------------------------------- | ---------------------------------------------------------------------- |
| `brs`  | Business Requirements Specification    | Mission, goals, objectives, and business operational concept           |
| `strs` | Stakeholder Requirements Specification | Stakeholder needs, operational concept, and user requirements          |
| `syrs` | System Requirements Specification      | System functions, interfaces, performance, and design constraints      |
| `srs`  | Software Requirements Specification    | Software functions, external interfaces, and detailed behavioral specs |

Use PRD for most projects. Add the sources track when you need structured requirement discovery. Add ISO 29148 when you need formal traceability for regulated or complex multi-team systems. Mix freely — some features can use a PRD while others use the full cascade.

### Knowledge

| Type    | Full Name                    | Description                                                                          |
| ------- | ---------------------------- | ------------------------------------------------------------------------------------ |
| `adr`   | Architecture Decision Record | Captures a finalized technical decision with context, alternatives, and consequences |
| `rfc`   | Request for Comments         | Proposes a significant change open for team review and feedback                      |
| `rule`  | Rule                         | Coding or process standard with imperative guidance and examples                     |
| `guide` | Guide                        | Step-by-step instructions for completing a specific task                             |
| `doc`   | Document                     | Reference documentation, registries, and descriptive material                        |
| `spec`  | Specification                | Canonical normative contract for a system, component, interface, or protocol         |

### Experience

| Type        | Full Name           | Description                                                    |
| ----------- | ------------------- | -------------------------------------------------------------- |
| `task-type` | Task Type           | Reusable checklist and workflow for a recurring task           |
| `cpat`      | Code Change Pattern | Root-cause analysis of a bug or incident with prevention steps |

Each document is a Markdown file with YAML frontmatter:

```markdown
---
title: "Use PostgreSQL for Primary Storage"
status: draft
tags: [database, infrastructure]
---

## Context

...
```

Valid statuses: `draft`, `accepted`, and `rejected`. Tags are optional and free-form — use them to mark cross-cutting topics (`security`, `golang`, `frontend`).

## Document relations

Documents can be linked with directed relations to other documents:

- **related** — general association
- **implements** — source implements what target specifies
- **extends** — source builds upon target
- **depends_on** — source requires target to proceed

Relations are stored in `.sync-state.json` and managed automatically by the AI agent through MCP tools.

## AI agent integration

Archcore integrates with AI coding agents in three ways:

- **Hooks** inject context at session start, so the agent is aware of your `.archcore/` documents from the first message.
- **MCP tools** give the agent capabilities to list, search, read, create, update, and link documents in real time. The MCP server also works in an empty repo and exposes an `init_project` tool, so agents can bootstrap `.archcore/` themselves.
- **MCP prompts** are ready-made multi-document workflows you trigger from your agent as slash commands.

### Prompts

Prompts orchestrate full document cascades in one call — the agent creates and links every document in the track for you. Most MCP-compatible agents surface them as slash commands (e.g. `/architecture_track`); the exact prefix depends on the client.

| Prompt               | What it does                                          |
| -------------------- | ----------------------------------------------------- |
| `product_track`      | idea → PRD → plan (lightweight feature flow)          |
| `architecture_track` | ADR → spec → plan (technical design + implementation) |
| `standard_track`     | ADR → rule → guide (codify a team standard)           |
| `sources_track`      | MRD → BRD → URD (market / business / user discovery)  |
| `iso_track`          | BRS → StRS → SyRS → SRS (formal ISO 29148 cascade)    |

**Example.** In your agent, run `/product_track feature="user notifications"`. The agent drafts an idea, derives a PRD, builds an implementation plan, and links them automatically.

### Local MCP server

Archcore does not require a hosted service. The CLI runs a local stdio MCP server:

```bash
archcore mcp
```

By default `archcore mcp` serves documents from the current directory. Pass `--project /path/to/repo` (or set `ARCHCORE_PROJECT_ROOT`) to point it elsewhere — useful when the server is launched from a directory that isn't your workspace (for example, by an editor integration).

Wire it into Claude Code:

```bash
claude mcp add --transport stdio archcore -- archcore mcp
```

Or install automatically for a supported agent:

```bash
archcore mcp install --agent cursor
```

### Install integrations

```bash
# Auto-detect agents in your project and install everything
archcore hooks install

# Or target a specific agent
archcore mcp install --agent opencode
archcore hooks install --agent cursor
```

## Commands

| Command                  | Description                                     |
| ------------------------ | ----------------------------------------------- |
| `archcore init`          | Initialize `.archcore/` directory interactively |
| `archcore doctor`        | Check your archcore setup and fix issues        |
| `archcore status`        | Check .archcore/ structure and document health  |
| `archcore config`        | View or modify settings                         |
| `archcore hooks install` | Install hooks for detected AI agents            |
| `archcore update`        | Update Archcore to the latest version           |
| `archcore mcp`           | Run the MCP stdio server                        |
| `archcore mcp install`   | Install MCP config for detected agents          |

### Update

```bash
archcore update
```

The command checks GitHub Releases for a newer version, downloads it, verifies the SHA-256 checksum, and atomically replaces the current binary.

## Install methods

### macOS / Linux

```bash
curl -fsSL https://archcore.ai/install.sh | bash
```

### Windows

```powershell
irm https://archcore.ai/install.ps1 | iex
```

Installs `archcore.exe` under `%LOCALAPPDATA%\Programs\archcore` and adds it to your user `PATH`. Open a new PowerShell window after install so the `PATH` change is picked up.

### Windows (WSL)

Install [WSL](https://learn.microsoft.com/en-us/windows/wsl/install), then run inside it:

```bash
curl -fsSL https://archcore.ai/install.sh | bash
```

### Go install

```bash
go install github.com/archcore-ai/cli@latest
```

### From source

```bash
git clone https://github.com/archcore-ai/cli.git
cd cli
go build -o archcore .
```

**Supported platforms:** macOS, Linux, Windows — amd64 and arm64.

For environment variables (`ARCHCORE_VERSION`, `ARCHCORE_INSTALL_DIR`, `GITHUB_TOKEN`) and PATH troubleshooting, see the [full install guide on docs.archcore.ai](https://docs.archcore.ai/cli/install/).

## Configuration

Settings are stored in `.archcore/settings.json` and created during `archcore init`.

| Field      | Description                                                                      | Values                                  |
| ---------- | -------------------------------------------------------------------------------- | --------------------------------------- |
| `sync`     | Sync mode. Cloud and on-prem are coming soon.                                    | `none` (local only), `cloud`, `on-prem` |
| `language` | Document language. Helps the agent generate documentation in the right language. | String, defaults to `en`                |

```bash
archcore config                              # show all settings
archcore config get <key>                    # get a specific value
archcore config set <key> <value>            # set a value
```

## Development

### Prerequisites

- Go 1.24+

### Build & test

```bash
# Build
go build -o archcore .

# Run all tests
go test ./...

# Run a specific package
go test ./cmd/

# Run a single test
go test ./cmd/ -run TestConfigCmd
```

### Project structure

```text
├── cmd/              # Cobra commands (init, doctor, config, status, hooks, mcp, ...)
├── internal/
│   ├── agents/       # Supported AI agents with hooks/MCP capabilities
│   ├── api/          # HTTP client for archcore server
│   ├── config/       # Settings management and directory init
│   ├── display/      # Terminal output formatting (lipgloss)
│   ├── update/       # Self-update logic (version check, download, verify, replace)
│   ├── mcp/          # MCP stdio server, tools, and prompts
│   └── sync/         # Sync logic
├── templates/        # Document type templates
├── install.sh        # Install script
└── .goreleaser.yaml  # Release configuration
```

## Is Archcore like BMAD / Spec Kit / Memory Bank?

No — these solve different problems. Quick map:

| Tool                  | Category    | What it is                                                                       | How Archcore differs                                                                                  |
| --------------------- | ----------- | -------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| **BMAD**              | Methodology | Agentic SDLC methodology — 12+ roles, 34+ workflows                              | Archcore stores _artifacts_; BMAD prescribes _process_                                                |
| **Spec Kit**          | Methodology | Spec-driven workflow: `specify → plan → tasks → implement`, one-shot             | Spec Kit is a one-shot handoff; Archcore maintains a living graph that evolves with the codebase     |
| **Agent OS**          | Methodology | Codebase standards extraction + spec-driven development                          | Closest positioning. Archcore adds typed documents, validated relations, and an optional ISO cascade  |
| **claude-mem / Mem0** | Memory      | Auto-captures session memory, cross-agent recall                                 | Memory tools remember _what you did_; Archcore stores _how the system is built and what was decided_ |
| **Cline Memory Bank** | Docs        | Fixed-schema markdown files (`projectbrief`, `activeContext`, `systemPatterns`…) | Same spirit, lower ceremony. Archcore adds typed relations, MCP validation, and multi-step cascades  |
| **CLAUDE.md / .cursorrules** | Instructions | Single flat file the agent reads at session start                          | Archcore replaces a growing instruction file with typed, related, queryable documents                 |

Pick a methodology tool for an opinionated dev flow. Pick a memory tool for session continuity. Pick Archcore when you want typed, queryable _project truth_ — the decisions, rules, and architecture of _this_ repo — that your coding agent respects on every request.

## Links & license

- **Documentation:** [docs.archcore.ai](https://docs.archcore.ai)
- **Website:** [archcore.ai](https://archcore.ai)
- **Plugin (Claude Code, Cursor):** [github.com/archcore-ai/archcore-plugin](https://github.com/archcore-ai/archcore-plugin)
- **Issues:** [github.com/archcore-ai/cli/issues](https://github.com/archcore-ai/cli/issues)
- **License:** [Apache 2.0](LICENSE)
