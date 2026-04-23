# Archcore Plugin

**Stop your AI agent from forgetting how your repo works.**

> Git versions your code.
> CI/CD ships it.
> **Archcore makes your AI understand it.**

Make your AI agent code with your project's architecture, rules, and decisions — loaded on session start, auto-injected before source edits, surfaced on demand for any code area, and carried across sessions and subagents.

_Not a spec workflow kit. Not chat memory. Archcore is a Git-backed repo context layer for coding agents._

## What changes the moment you install it

- **Architecture** — the agent places code where your system expects it, not where it guesses
- **Rules** — it follows your team's standards instead of improvising
- **Decisions** — prior ADRs aren't re-litigated, they're respected
- **Workflows** — multi-step tasks (PRD → plan, ADR → rule → guide) execute end-to-end

## Install

No prerequisites. The plugin bundles a launcher that downloads the Archcore CLI on first use (cached between sessions).

**Claude Code** — inside `claude`:

```bash
/plugin marketplace add archcore-ai/archcore-plugin
/plugin install archcore@archcore-plugins
```

**Cursor** — requires Cursor 2.5+. Archcore is not yet on the official [Cursor Marketplace](https://cursor.com/marketplace), so install directly from GitHub — open a new Agent chat and run:

```sh
/add-plugin archcore@https://github.com/archcore-ai/archcore-plugin
```

`/add-plugin` doesn't appear in autocomplete — type the full command.

<details>
<summary>Local development, offline, enterprise, team rollouts</summary>

**Claude Code** — load the plugin for the current session:

```bash
claude --plugin-dir /path/to/archcore-claude-plugin
```

**Cursor** — no `--plugin-dir` flag. Symlink the repo into Cursor's local plugins directory and reload the window:

```bash
ln -s /path/to/archcore-claude-plugin ~/.cursor/plugins/local/archcore
# then in Cursor: Cmd/Ctrl+Shift+P → "Developer: Reload Window"
```

Both manifests (`.claude-plugin/plugin.json` and `.cursor-plugin/plugin.json`) live at the repo root.

**Cursor team rollouts** — add the GitHub URL under Dashboard → Settings → Plugins → Team Marketplaces → Import.

**Offline / BYO CLI** — if you already have the Archcore CLI installed (via `curl -fsSL https://archcore.ai/install.sh | bash`, `go install`, etc.), the launcher respects it — a global install on `PATH` wins over the plugin-managed cache.

For fully offline environments: install the CLI manually and set `ARCHCORE_SKIP_DOWNLOAD=1` to disable the launcher's auto-download. Alternatively, set `ARCHCORE_BIN=/abs/path/to/archcore` to pin an explicit binary.

</details>

## Try these 3 prompts first

Install, open your project, and try these — each shows a different side of what your agent can now do:

_Empty repo? Run `/archcore:bootstrap` first to seed a stack rule, a run-the-app guide, and (optionally) imports from your existing CLAUDE.md / AGENTS.md / .cursorrules._

**1. "What rules and decisions apply to `src/` in this repo?"**
Agent runs `/archcore:context src/`, surfacing the rules, ADRs, specs, and patterns that reference that path — grouped by type and ranked by specificity — before you change anything. Works the same way for a file or topic, or with no argument to recap what you're in the middle of.

**2. "Create a rule: API handlers live in `src/api/handlers/`."**
Agent creates the rule, validates it, and from now on places new handlers there without being reminded.

**3. "Record an ADR that we picked PostgreSQL, then show me what future work it should affect."**
Agent creates the ADR, scans the relation graph, and lists related specs, plans, and rules that should be reviewed.

If any of these feels valuable, the rest of Archcore is more of the same, just structured.

## When to use Archcore

Archcore is for teams whose agents already write code but keep missing project-specific context. Install it when:

- Your `CLAUDE.md` / `.cursorrules` / `AGENTS.md` keeps growing and drifting
- You work with 2+ agents or 2+ host tools (Claude Code + Cursor + Copilot)
- The agent keeps re-deciding things your team already decided
- You want decisions, rules, and specs in Git — not in chat scrollback

Archcore is **not** the right fit if you just want chat memory, a prompt library, or a one-shot spec-to-code generator.

## Is Archcore like BMAD / Spec Kit / claude-mem / Memory Bank?

No — these solve different problems. Quick map:

| Tool                     | Category    | What it is                                                                   | How Archcore differs                                                                                                                         |
| ------------------------ | ----------- | ---------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| **BMAD**                 | Methodology | Agentic SDLC methodology — 12+ roles, 34+ workflows, installer               | Archcore stores _artifacts_; BMAD prescribes _process_. Durable knowledge in BMAD lives in generated skills, not relation-aware repo memory  |
| **Superpowers**          | Methodology | Skills framework + dev methodology (TDD, plan writing, subagent-driven dev)  | Shapes _agent behavior_ during coding; Archcore provides _canonical project knowledge_ any agent can read                                    |
| **Spec Kit**             | Methodology | Spec-driven workflow: `specify → plan → tasks → implement`, one-shot         | Spec Kit is a one-shot handoff; Archcore maintains a living graph that evolves with the codebase                                             |
| **Agent OS**             | Methodology | Codebase standards extraction + spec-driven development, alongside IDE tools | Closest positioning. Archcore adds typed documents, validated relations, and an optional ISO 29148 cascade for regulated teams               |
| **claude-mem**           | Memory      | Auto-captures session memory (SQLite + Chroma, MCP search, web viewer)       | claude-mem remembers _what you did_; Archcore stores _how the system is built and what was decided_                                          |
| **agentmemory**          | Memory      | Cross-agent memory server (hooks, BM25 + vector + graph, 4-tier consolidation) | Infrastructure for recall over observations; Archcore is repo-native canonical knowledge                                                     |
| **OpenMemory / Mem0**    | Memory      | Memory infrastructure — SDK, MCP, self-hosted or managed                     | General-purpose agent memory; Archcore is project truth for coding agents                                                                    |
| **claude-brain**         | Memory      | One-file local memory (`.claude/mind.mv2`), searchable, portable             | Solo session continuity; Archcore is a team-grade, relation-aware layer                                                                      |
| **Cline Memory Bank**    | Docs        | Fixed-schema markdown files (`projectbrief`, `activeContext`, `systemPatterns`…) | Same spirit, lower ceremony. Archcore adds typed relations, MCP validation, and multi-step cascades                                          |
| **codeplow / obsidian-kb** | Docs      | Per-project Obsidian vault with explicit handoff and file:line doc-audit     | Knowledge vault + auditing; Archcore is a typed context _compiler_ — less "notes", more "artifacts"                                          |

**Choose by what you need.** Pick a methodology tool (BMAD, Superpowers, Spec Kit, Agent OS) for an opinionated dev flow. Pick a memory tool (claude-mem, Mem0, agentmemory, claude-brain) for session continuity in general-purpose agents. Pick Archcore when you want typed, queryable _project truth_ — the decisions, rules, and architecture of _this_ repo — that your coding agent respects on every request.

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

## Supported hosts

| Host            | Status      | Install            |
| --------------- | ----------- | ------------------ |
| **Claude Code** | Production  | Plugin marketplace |
| **Cursor**      | Implemented | Plugin marketplace |
| GitHub Copilot  | Planned     | —                  |
| Codex CLI       | Planned     | —                  |

The plugin uses open standards (Agent Skills, MCP) — skills, agents, and MCP tools are shared across hosts. Only hooks and manifests are host-specific.

## What ships in the box

- **35 Skills** — 17 document types, 11 intent commands, 6 multi-step tracks, 1 utility
- **2 Agents** — a universal assistant and a read-only auditor
- **Hooks** — session-start context loading, MCP-only write enforcement, post-mutation validation, cascade staleness detection

The plugin ships a launcher that resolves the [Archcore CLI](https://archcore.ai) (`archcore mcp`) and registers the MCP server automatically via the plugin's bundled `.mcp.json`. If the CLI isn't on `PATH`, the launcher downloads it on first use and caches it under `$CLAUDE_PLUGIN_DATA/archcore/cli/` (survives plugin updates). An existing global `archcore` install on `PATH` always wins — no duplicate-server conflicts.

## How it works

1. **Session starts** — the session hook loads your project's document index and relations into context
2. **You ask for something** — "create a PRD for the auth redesign", "what ADRs relate to payments?", "audit the docs"
3. **Skills activate** — the agent matches your request to the right skill, which provides document-type knowledge, required sections, and relation guidance
4. **MCP tools execute** — all reads and writes go through `archcore mcp`, ensuring validation, template generation, and sync manifest updates
5. **Hooks guard quality** — direct `.archcore/` writes are blocked (MCP-only), and every change is validated automatically

### Mental model

Two pieces work together:

- **Archcore CLI — the compiler.** Reads `.archcore/`, builds the context graph, exposes it over MCP.
- **Archcore Plugin — the runtime.** Applies that context inside your AI agent — skills, guardrails, workflows.

## Skills

### Intent commands (11)

The primary user interface — describe what you want, the skill routes to the right types and flows.

| Skill     | Command               | What it does                                                                    |
| --------- | --------------------- | ------------------------------------------------------------------------------- |
| Bootstrap | `/archcore:bootstrap` | First-time onboarding — seeds a stack rule, run-the-app guide, optional imports |
| Capture   | `/archcore:capture`   | Document a module, component, or API — routes to adr, spec, doc, or guide       |
| Context   | `/archcore:context`   | Pull applicable rules, ADRs, specs, and patterns for a code area, file, or topic |
| Plan      | `/archcore:plan`      | Plan a feature end-to-end — routes to product-track or single plan              |
| Decide    | `/archcore:decide`    | Record a finalized decision — creates ADR, offers rule + guide follow-up        |
| Standard  | `/archcore:standard`  | Establish a team standard — drives standard-track (adr → rule → guide)          |
| Review    | `/archcore:review`    | Documentation health review — gaps, staleness, orphaned docs, missing relations |
| Status    | `/archcore:status`    | Dashboard — counts by category, type, status, relation coverage, tags           |
| Actualize | `/archcore:actualize` | Detect stale docs via code drift, cascade analysis, and temporal staleness      |
| Graph     | `/archcore:graph`     | Render the document relation graph as a Mermaid flowchart                       |
| Help      | `/archcore:help`      | Navigate the system — discover skills, types, and workflows                     |

### Document types (17)

Each skill teaches the agent about one document type: when to use it, required sections, best practices, common mistakes, and how to relate it to other documents. The `plan` type is served by the `/archcore:plan` intent skill rather than a standalone type skill.

| Type        | Category   | What it captures                                     |
| ----------- | ---------- | ---------------------------------------------------- |
| `prd`       | vision     | Product requirements — goals, scope, success metrics |
| `idea`      | vision     | Low-commitment concepts and explorations             |
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

Invoke mainstream types directly: `/archcore:adr`, `/archcore:prd`, `/archcore:spec`, etc. The 7 niche types (`mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs`) are hidden from `/` autocomplete to reduce cognitive load — reach them through `/archcore:sources-track` or `/archcore:iso-track`, or call `mcp__archcore__create_document` directly.

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
