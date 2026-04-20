# Archcore Plugin

Git-native context for AI coding agents.

> Git versions your code.
> CI/CD ships it.
> **Archcore makes your AI understand it.**

## Quick Start

**Prerequisites:** [Archcore CLI](https://archcore.ai) — required.

```bash
# 1. Install the CLI (macOS / Linux)
curl -fsSL https://archcore.ai/install.sh | bash

# 2. Initialize a project
archcore init
```

On **Windows**, run in PowerShell: `irm https://archcore.ai/install.ps1 | iex`. For WSL, `go install`, and other options, see the [full install guide](https://docs.archcore.ai/cli/install/).

### Claude Code

```bash
claude plugin marketplace add archcore-ai/archcore-plugin
claude plugin install archcore@archcore-plugins
```

Or from within Claude Code:

```bash
/plugin marketplace add archcore-ai/archcore-plugin
/plugin install archcore@archcore-plugins
```

### Cursor

Install from the Cursor plugin marketplace, or locally:

```bash
cursor --plugin-dir ./archcore-plugin
```

### Local development (any host)

```bash
claude --plugin-dir ./archcore-plugin    # Claude Code
cursor --plugin-dir ./archcore-plugin    # Cursor
```

---

## What it does

Archcore Plugin turns your AI coding agent into a first-class citizen of your project's knowledge base. Four things change the moment you install it:

- **Architecture** — the agent places code where your system expects it
- **Rules** — it follows your team's standards instead of improvising
- **Decisions** — prior ADRs aren't re-litigated
- **Workflows** — multi-step tasks (PRD → plan, ADR → rule → guide, ISO 29148 cascades) execute end-to-end

## Without vs. with Archcore

Without Archcore, ask for "a new service" and the agent:

- guesses the folder structure
- ignores conventions it has never been told about
- produces code that drifts from decisions buried in docs
- rediscovers patterns you've already written down

With Archcore, the same ask and the agent:

- places files where your architecture says they belong
- follows rules defined in `.archcore/`
- respects ADRs and existing specs
- reuses patterns (`cpat`) you've already captured

## Mental model

Two pieces work together. Keep the analogy in your head:

- **Archcore CLI — the compiler.** Reads `.archcore/`, builds the context graph, exposes it over MCP.
- **Archcore Plugin — the runtime.** Applies that context inside your AI agent — skills, guardrails, workflows.

Result: your agent ships code aligned with your system by default, not by accident.

## Supported Hosts

| Host            | Status      | Install            |
| --------------- | ----------- | ------------------ |
| **Claude Code** | Production  | Plugin marketplace |
| **Cursor**      | Implemented | Plugin marketplace |
| GitHub Copilot  | Planned     | —                  |
| Codex CLI       | Planned     | —                  |

The plugin uses open standards (Agent Skills, MCP) — skills, agents, and MCP tools are shared across hosts. Only hooks and manifests are host-specific.

## What ships in the box

- **32 Skills** — 18 document types, 8 intent commands, 6 multi-step tracks
- **2 Agents** — a universal assistant and a read-only auditor
- **Hooks** — session-start context loading, MCP-only write enforcement, post-mutation validation, cascade staleness detection

The plugin does **not** ship its own MCP server — it uses the one provided by the [Archcore CLI](https://archcore.ai) (`archcore mcp`). This avoids duplicate-server conflicts in repos that already register `archcore` in `.mcp.json` or via `claude mcp add`.

## How it works

1. **Session starts** — the session hook loads your project's document index and relations into context
2. **You ask for something** — "create a PRD for the auth redesign", "what ADRs relate to payments?", "audit the docs"
3. **Skills activate** — the agent matches your request to the right skill, which provides document-type knowledge, required sections, and relation guidance
4. **MCP tools execute** — all reads and writes go through `archcore mcp`, ensuring validation, template generation, and sync manifest updates
5. **Hooks guard quality** — direct `.archcore/` writes are blocked (MCP-only), and every change is validated automatically

## Skills

### Intent commands (8)

The primary user interface — describe what you want, the skill routes to the right types and flows.

| Skill     | Command               | What it does                                                                    |
| --------- | --------------------- | ------------------------------------------------------------------------------- |
| Capture   | `/archcore:capture`   | Document a module, component, or API — routes to adr, spec, doc, or guide       |
| Plan      | `/archcore:plan`      | Plan a feature end-to-end — routes to product-track or single plan              |
| Decide    | `/archcore:decide`    | Record a finalized decision — creates ADR, offers rule + guide follow-up        |
| Standard  | `/archcore:standard`  | Establish a team standard — drives standard-track (adr → rule → guide)          |
| Review    | `/archcore:review`    | Documentation health review — gaps, staleness, orphaned docs, missing relations |
| Status    | `/archcore:status`    | Dashboard — counts by category, type, status, relation coverage, tags           |
| Actualize | `/archcore:actualize` | Detect stale docs via code drift, cascade analysis, and temporal staleness      |
| Help      | `/archcore:help`      | Navigate the system — discover skills, types, and workflows                     |

### Document types (18)

Each skill teaches the agent about one document type: when to use it, required sections, best practices, common mistakes, and how to relate it to other documents.

| Type        | Category   | What it captures                                     |
| ----------- | ---------- | ---------------------------------------------------- |
| `prd`       | vision     | Product requirements — goals, scope, success metrics |
| `idea`      | vision     | Low-commitment concepts and explorations             |
| `plan`      | vision     | Phased implementation with tasks and dependencies    |
| `mrd`       | vision     | Market landscape, TAM/SAM/SOM, competition           |
| `brd`       | vision     | Business objectives, stakeholders, ROI               |
| `urd`       | vision     | User personas, journeys, usability requirements      |
| `brs`       | vision     | Formal business requirements spec (ISO 29148)        |
| `strs`      | vision     | Formal stakeholder requirements spec (ISO 29148)     |
| `syrs`      | vision     | System boundary and interface spec (ISO 29148)       |
| `srs`       | vision     | Software functional/non-functional spec (ISO 29148)  |
| `adr`       | knowledge  | Architecture decisions with context and consequences |
| `rfc`       | knowledge  | Proposals open for review and feedback               |
| `rule`      | knowledge  | Mandatory team standards with rationale              |
| `guide`     | knowledge  | Step-by-step how-to instructions                     |
| `doc`       | knowledge  | Reference material — registries, glossaries, lookups |
| `spec`      | knowledge  | Technical contracts for systems and components       |
| `task-type` | experience | Recurring task patterns with proven workflows        |
| `cpat`      | experience | Before/after code pattern changes with scope         |

Invoke any type directly: `/archcore:adr`, `/archcore:prd`, `/archcore:spec`, etc.

### Tracks (6)

Tracks orchestrate multi-document flows. Each step builds on the previous one, with proper relations created automatically.

| Track                | Flow                                         | Use when                                               |
| -------------------- | -------------------------------------------- | ------------------------------------------------------ |
| `product-track`      | idea &rarr; prd &rarr; plan                  | Lightweight product flow — simple and fast             |
| `sources-track`      | mrd &rarr; brd &rarr; urd                    | Discovery-focused — market, business, then user inputs |
| `iso-track`          | brs &rarr; strs &rarr; syrs &rarr; srs       | Formal ISO 29148 cascade with traceability             |
| `architecture-track` | adr &rarr; spec &rarr; plan                  | Design decisions flowing into implementation           |
| `standard-track`     | adr &rarr; rule &rarr; guide                 | Decision &rarr; codified standard &rarr; instructions  |
| `feature-track`      | prd &rarr; spec &rarr; plan &rarr; task-type | Full feature lifecycle                                 |

Invoke: `/archcore:product-track`, `/archcore:architecture-track`, etc.

## Agents

**archcore-assistant** — Universal read/write agent for complex multi-document tasks. Creates and updates documents, manages relations, handles requirement cascades. Uses all 8 MCP tools.

**archcore-auditor** — Read-only background agent for documentation health. Detects coverage gaps, orphaned documents, stale statuses, broken relation chains, and naming inconsistencies. Safe by design — no write tools.

## Guardrails

The plugin enforces the **MCP-only principle**: all `.archcore/` operations must go through Archcore's MCP tools, never through direct file writes. This ensures every change is validated, templated, and synced.

- **Session start** — loads document index and relations into context, detects code-document drift
- **Write blocking** — intercepts and blocks direct Write/Edit calls targeting `.archcore/`
- **Validation** — runs `archcore validate` after every document mutation
- **Cascade detection** — warns when an updated document has dependents that may need review

## Philosophy

- **Context is a first-class artifact** — typed, validated, relation-aware Markdown in Git. Not a hidden prompt, not tribal knowledge.
- **Opinionated workflows over raw tool access** — skills route intent to the right document type and the right multi-step flow.
- **Minimal effort at the boundary** — the agent already knows the structure, so you describe intent, not schema.

## Roadmap

- Deeper IDE integrations (VS Code, JetBrains)
- Additional hosts (GitHub Copilot, Codex CLI)
- Multi-agent coordination for long cascades
- Richer staleness and drift analytics

## Uninstallation

Claude Code:

```bash
/plugin uninstall archcore@archcore-plugins
```

Cursor: remove from plugin settings.

## License

[Apache-2.0](LICENSE)

## Contributing

Issues and ideas: [GitHub Issues](https://github.com/archcore-ai/archcore-plugin/issues)
