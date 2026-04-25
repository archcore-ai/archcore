# Archcore CLI

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/archcore-ai/cli)](https://github.com/archcore-ai/cli/releases)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)](https://github.com/archcore-ai/cli/releases)

**Git-native context for AI coding agents**

> Git ships your code. CI/CD ships your delivery. Archcore ships your understanding.

Archcore is a context layer for your repository. Architecture decisions, rules, plans, and incidents live as structured documents that every AI coding agent — Claude Code, Cursor, Copilot, Gemini CLI, Codex — can read, reuse, and follow.

## Quick Start

```bash
# Install (macOS / Linux)
curl -fsSL https://archcore.ai/install.sh | bash

# Initialize in your project
cd your-project
archcore init

# Verify setup
archcore status
```

> On **Windows**? Use PowerShell: `irm https://archcore.ai/install.ps1 | iex`. For WSL, `go install`, and other options, see the [full install guide](https://docs.archcore.ai/cli/install/).

Now open your AI agent and say: _"Record an ADR: we're using PostgreSQL for primary storage."_ The agent will create a structured decision document in `.archcore/` that every future session can read.

## Table of Contents

- [Why Archcore](#why-archcore)
- [Installation](#installation)
- [How It Works](#how-it-works)
- [What Lives in `.archcore/`](#what-lives-in-archcore)
- [Why Not Just Instruction Files?](#why-not-just-instruction-files)
- [Try It](#try-it)
- [Document Types](#document-types)
- [Document Relations](#document-relations)
- [Commands](#commands)
- [AI Agent Integration](#ai-agent-integration)
- [Configuration](#configuration)
- [Development](#development)
- [Links & License](#links--license)

## Why Archcore

AI coding agents don't understand your system. They:

- ignore your architecture
- break your conventions
- duplicate logic that already exists
- make inconsistent decisions across sessions

The real context lives in your head, in scattered docs, and in implicit patterns that never reach the model. Archcore makes that context explicit, structured, and versioned alongside your code:

- **Shared repo context** — architecture, rules, plans, and postmortems next to the code they describe
- **Works across agents** — Claude Code, Cursor, Gemini CLI, GitHub Copilot, Codex CLI, OpenCode, Roo Code, and Cline
- **Structured document types** — ADRs, RFCs, rules, guides, plans, PRDs, incidents, and more
- **Git-native** — local-first, version-controlled, reviewable with code
- **MCP-powered** — agents read, create, update, and link documents in real time
- **Built for teams** — context survives sessions, teammates, and tool changes

> **AI should follow your system, not guess it.**

## Installation

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

### Go Install

```bash
go install github.com/archcore-ai/cli@latest
```

### From Source

```bash
git clone https://github.com/archcore-ai/cli.git
cd cli
go build -o archcore .
```

**Supported platforms:** macOS, Linux, Windows — amd64 and arm64.

For all install methods, environment variables (`ARCHCORE_VERSION`, `ARCHCORE_INSTALL_DIR`, `GITHUB_TOKEN`), and PATH troubleshooting, see the [full install guide on docs.archcore.ai](https://docs.archcore.ai/cli/install/).

## How It Works

1. **Initialize your repo**  
   `archcore init` creates `.archcore/` and installs integrations for supported agents.

2. **Capture durable context**  
   Store architecture decisions, rules, plans, product docs, and incident learnings as structured Markdown files.

3. **Let agents reuse it**  
   Hooks and MCP let your coding agents read existing context and create or update documents during real work.

4. **Keep it in Git**  
   Review context changes like code, evolve them over time, and keep them portable across tools.

### Mental model

Archcore CLI is a **context compiler** for your repository. It turns scattered documents into structured, machine-readable context. MCP and hooks are the **runtime** — the surface agents use to consume that context during real work.

```text
implicit repo knowledge  →  structured context  →  AI-readable system
```

## What Lives in `.archcore/`

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

## Why Not Just Instruction Files?

`CLAUDE.md`, `AGENTS.md`, and repository instructions are useful starting points, but they break down when your team needs:

- more than one flat memory file
- structured document types — ADRs, rules, plans, incidents
- reusable context across multiple AI tools
- versioned project knowledge that grows with the codebase
- relations between documents (a plan that _implements_ a PRD, an RFC that _extends_ an ADR)
- incident learnings and recurring workflows that agents can pick up later

Instruction files tell the agent _what you want_. Archcore tells the agent _how your system works_ — so the agent can follow your system instead of guessing it.

## Try It

After `archcore init`, open your AI agent and start talking. The agent already knows your existing documents and has tools to create new ones.

> “We decided to use PostgreSQL instead of MongoDB for our primary database. Record this decision.”

Creates `infrastructure/use-postgres.adr.md` with context, decision, alternatives considered, and consequences.

> “We have a team convention: always wrap errors with context using fmt.Errorf and %w. Make this a rule.”

Creates `backend/error-wrapping.rule.md` with imperative guidance, rationale, and good/bad code examples.

> “Last week we had a connection pool exhaustion incident because idle connections weren't being recycled. Document this so we don't repeat it.”

Creates `incidents/connection-pool-exhaustion.cpat.md` with root-cause analysis and prevention steps.

> “I need a PRD for the user notifications feature — push, email digests, and in-app alerts.”

Creates `notifications/user-notifications.prd.md` with goals, user stories, requirements, and success metrics.

> “Create an implementation plan for the notifications PRD and link them together.”

Creates `notifications/notifications-implementation.plan.md`, then links it to the PRD with an `implements` relation.

## Document Types

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
---

## Context

...
```

Valid statuses: `draft`, `accepted`, and `rejected`.

## Document Relations

Documents can be linked with directed relations to other documents:

- **related** — general association
- **implements** — source implements what target specifies
- **extends** — source builds upon target
- **depends_on** — source requires target to proceed

Relations are stored in `.sync-state.json` and managed automatically by the AI agent through MCP tools.

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
# Update to the latest version
archcore update
```

The command checks GitHub Releases for a newer version, downloads it, verifies the SHA-256 checksum, and atomically replaces the current binary.

### Examples

```bash
# Install integrations for a specific agent
archcore hooks install --agent cursor
archcore mcp install --agent gemini-cli
```

## AI Agent Integration

Archcore integrates with AI coding agents in two ways:

- **Hooks** inject context at session start, so the agent is aware of your `.archcore/` documents from the first message.
- **MCP** (Model Context Protocol) gives the agent tools to list, read, create, update, and link documents in real time.

### Supported Agents

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

### Install Integrations

```bash
# Auto-detect agents in your project and install everything
archcore hooks install

# Or target a specific agent
archcore mcp install --agent opencode
```

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

### Build & Test

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

### Project Structure

```text
├── cmd/              # Cobra commands (init, doctor, config, status, hooks, mcp, ...)
├── internal/
│   ├── agents/       # Supported AI agents with hooks/MCP capabilities
│   ├── api/          # HTTP client for archcore server
│   ├── config/       # Settings management and directory init
│   ├── display/      # Terminal output formatting (lipgloss)
│   ├── update/       # Self-update logic (version check, download, verify, replace)
│   ├── mcp/          # MCP stdio server implementation
│   └── sync/         # Sync logic
├── templates/        # Document type templates
├── install.sh        # Install script
└── .goreleaser.yaml  # Release configuration
```

## Links & License

- **Documentation:** [docs.archcore.ai](https://docs.archcore.ai)
- **Website:** [archcore.ai](https://archcore.ai)
- **Issues:** [github.com/archcore-ai/cli/issues](https://github.com/archcore-ai/cli/issues)
- **License:** [Apache 2.0](LICENSE)
