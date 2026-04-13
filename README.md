# Archcore Plugin

Git-native context for AI coding agents.

Archcore keeps your architecture decisions, requirements, specs, and patterns in `.archcore/` — right next to the code they describe. This plugin makes your AI coding tool a first-class citizen of that system: it can read, create, update, and connect documents without you having to explain the structure every time.

## Supported Hosts

| Host | Status | Install |
|------|--------|---------|
| **Claude Code** | Production | Plugin marketplace |
| **Cursor** | Implemented | Plugin marketplace |
| GitHub Copilot | Planned | — |
| Codex CLI | Planned | — |

The plugin uses open standards (Agent Skills, MCP) — skills, agents, and MCP tools are shared across all hosts. Only hooks and manifests are host-specific.

## What is Archcore?

[Archcore](https://archcore.ai) stores project knowledge as Markdown files with typed frontmatter in a `.archcore/` directory inside your repo. Each file has a type (ADR, PRD, spec, rule, plan, etc.), a status, tags, and semantic relations to other documents. Everything is version-controlled alongside your code.

## What this plugin adds

- **MCP Server** — exposes Archcore's document and relation tools directly to Claude
- **33 Skills** — teach your AI agent about every document type and orchestrate multi-step workflows
- **2 Agents** — handle complex documentation tasks and background audits
- **5 Hooks** — load context on session start, block unsafe writes, validate changes, detect cascade staleness

The result: your AI agent understands your `.archcore/` structure out of the box. Ask it to draft an ADR, trace requirements to implementation plans, or audit your documentation health — it knows how.

## Quick Start

**Prerequisites:** [Archcore CLI](https://archcore.ai)

```bash
curl -fsSL https://archcore.ai/install.sh | bash
```

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

### Initialize a project

```bash
archcore init
```

## How it works

1. **Session starts** — the session hook loads your project's document index and relations into context
2. **You ask for something** — "create a PRD for the auth redesign", "what ADRs relate to payments?", "audit the docs"
3. **Skills activate** — the agent matches your request to the right skill, which provides document-type knowledge, required sections, and relation guidance
4. **MCP tools execute** — all reads and writes go through `archcore mcp`, ensuring validation, template generation, and sync manifest updates
5. **Hooks guard quality** — direct `.archcore/` writes are blocked (MCP-only), and every change is validated automatically

## Skills

### Document Types (18)

Each skill teaches Claude about one document type: when to use it, required sections, best practices, common mistakes, and how to relate it to other documents.

| Type        | Category   | What it captures                                     |
| ----------- | ---------- | ---------------------------------------------------- |
| `adr`       | knowledge  | Architecture decisions with context and consequences |
| `rfc`       | knowledge  | Proposals open for review and feedback               |
| `rule`      | knowledge  | Mandatory team standards with rationale              |
| `guide`     | knowledge  | Step-by-step how-to instructions                     |
| `doc`       | knowledge  | Reference material — registries, glossaries, lookups |
| `spec`      | knowledge  | Technical contracts for systems and components       |
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

### Intent Commands (8)

The primary user interface — describe what you want, the skill routes to the right types and flows.

| Skill     | Command                | What it does                                                                    |
| --------- | ---------------------- | ------------------------------------------------------------------------------- |
| Capture   | `/archcore:capture`    | Document a module, component, or API — routes to adr, spec, doc, or guide       |
| Plan      | `/archcore:plan`       | Plan a feature end-to-end — routes to product-track or single plan              |
| Decide    | `/archcore:decide`     | Record a finalized decision — creates ADR, offers rule + guide follow-up        |
| Standard  | `/archcore:standard`   | Establish a team standard — drives standard-track (adr → rule → guide)          |
| Review    | `/archcore:review`     | Documentation health review — gaps, staleness, orphaned docs, missing relations |
| Status    | `/archcore:status`     | Dashboard — counts by category, type, status, relation coverage, tags           |
| Actualize | `/archcore:actualize`  | Detect stale docs via code drift, cascade analysis, and temporal staleness       |
| Help      | `/archcore:help`       | Navigate the system — discover skills, types, and workflows                     |

## Agents

**archcore-assistant** — Universal read/write agent for complex multi-document tasks. Creates and updates documents, manages relations, handles requirement cascades. Uses all 8 MCP tools.

**archcore-auditor** — Read-only background agent for documentation health. Detects coverage gaps, orphaned documents, stale statuses, broken relation chains, and naming inconsistencies. Safe by design — no write tools.

## Guardrails

The plugin enforces the **MCP-only principle**: all `.archcore/` operations must go through Archcore's MCP tools, never through direct file writes. This ensures every change is validated, templated, and synced.

- **Session start** — loads document index and relations into context, detects code-document drift
- **Write blocking** — intercepts and blocks direct Write/Edit calls targeting `.archcore/`
- **Validation** — runs `archcore validate` after every document mutation
- **Cascade detection** — warns when an updated document has dependents that may need review

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
