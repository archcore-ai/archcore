# Archcore

**Git-native project context for AI coding agents.**

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/archcore-ai/archcore)](https://github.com/archcore-ai/archcore/releases)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)](https://github.com/archcore-ai/archcore/releases)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Docs](https://img.shields.io/badge/docs-docs.archcore.ai-2563EB)](https://docs.archcore.ai)

Archcore keeps your specs, architecture decisions, rules, and plans in `.archcore/`, versioned with the code, and hands the relevant part to your coding agent while it works: at session start, on every file edit, and through MCP tools the agent can query. Claude Code, Cursor, Codex CLI, GitHub Copilot CLI, Gemini CLI, OpenCode, Roo Code, and Cline read the same context.

Without Archcore, an agent re-learns the repository every session and re-litigates settled decisions. With Archcore, it loads what applies to the files it touches, puts code where the architecture says, and records new decisions as documents you review in pull requests.

## Get started

1. Install the CLI.

   ```bash
   curl -fsSL https://archcore.ai/install.sh | bash    # macOS, Linux, WSL
   ```

   ```powershell
   irm https://archcore.ai/install.ps1 | iex            # Windows, PowerShell 5.1+
   ```

2. Initialize the project.

   ```bash
   cd your-project
   archcore init
   ```

   `archcore init` creates `.archcore/`, wires hooks and MCP for the agents it detects, and installs the Archcore plugin for the hosts you select: Claude Code, Cursor, Codex CLI, and GitHub Copilot CLI.

3. Open your agent and say what you want.

   > "We're using PostgreSQL for primary storage. Record this decision."

   The agent writes an ADR into `.archcore/`. Every later session, in any of your agents, sees it.

Already have a `CLAUDE.md`, `AGENTS.md`, rule files, or an ADR folder? Run `/archcore:init import` inside the agent to convert them into typed documents.

## Everyday use

Most of the time you type nothing Archcore-specific. Hooks inject the rules and specs that apply when the agent edits a file, the session opens with a recap of what is decided and in progress, and the agent reads or writes documents through MCP when a question or a decision comes up.

Four commands cover explicit work:

| Command | What it does |
| --- | --- |
| `/archcore:init` | Detects the repository's scale, wires host configs, measures the context you already wrote, and seeds a first-day pack: a stack rule, a run guide, an architecture overview, specs for hotspot modules. `init import` converts existing instruction files and ADR folders. |
| `/archcore:plan` | Turns a feature, refactor, or initiative into a scoped package. The route is computed from what the work changes: a small fix exits with no documents, one capability gets a spec and a plan, a large initiative gets a PRD with one spec per capability. |
| `/archcore:document` | Records the present state: `decision` writes an ADR or RFC, `code` describes a module or API that has no doc yet, `research` files a report or one external material as evidence. |
| `/archcore:review` | Checks the branch against recorded rules and decisions before merge. `drift` finds stale docs, `deep` audits the whole corpus, `closeout` closes a finished feature. |

Say what you want in plain language. The first word after a command is a mode (`plan research`, `document decision`, `review closeout`) and the rest is the subject. A fully specified request runs question-free, a vague one stays within five questions, and an interrupted flow resumes in a later session.

## What lives in `.archcore/`

```text
.archcore/
├── settings.json
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

- **Typed Markdown.** A document's type is its filename suffix, `slug.type.md`: 23 types across knowledge (`adr`, `rfc`, `rule`, `spec`, `guide`, `doc`, `evidence`, `scenario`), vision (`prd`, `plan`, `idea`, `research`, `rnd`, `journey`, and the requirements tracks `mrd` / `brd` / `urd` and `brs` / `strs` / `syrs` / `srs`), and experience (`task-type`, `cpat`). Each carries a title, a status (`draft`, `accepted`, `rejected`), and optional tags in YAML frontmatter.
- **Relations.** Documents link through seven directed relations: `related`, `implements`, `extends`, `depends_on`, `supports`, `contradicts`, `supersedes`.
- **Free-form layout.** Organize by domain, feature, or team. Global sources let a company-wide `.archcore/` supply read-only defaults that a project's own documents override.
- **Git owns it.** Context changes arrive as pull requests, get reviewed like code, and travel with every clone. This repository's own [`.archcore/`](https://github.com/archcore-ai/archcore/tree/dev/.archcore) is a working example.

The agent reaches it through 11 MCP tools served by `archcore mcp`, a local stdio server: `list_documents`, `search_documents`, `get_document`, `create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`, `list_relations`, `init_project`, and `install_host_config`.

## Works with your agent

| Agent | Plugin: commands, skills, guardrails | Hooks | MCP |
| --- | --- | --- | --- |
| Claude Code | yes | yes | yes |
| Cursor 2.5+ | yes | yes | yes, user-level config |
| Codex CLI 0.117+ | yes | via the plugin | yes |
| GitHub Copilot CLI | yes | yes | yes, project-wired |
| Gemini CLI | — | yes | yes |
| OpenCode | — | — | yes |
| Roo Code | — | — | yes |
| Cline | — | — | manual |

`archcore init` wires the detected agents. On the four hosts with a plugin system the plugin adds the commands, the skills, two agents, and the hook launchers:

```bash
# Claude Code
/plugin marketplace add archcore-ai/archcore
/plugin install archcore@archcore-plugins

# Codex CLI, then /plugins → Archcore → Install plugin
codex plugin marketplace add archcore-ai/archcore

# GitHub Copilot CLI: a plugin cannot ship an MCP server here, so wire the project as well
copilot plugin install archcore-ai/archcore:plugins/archcore
archcore init --agent copilot --project "$PWD"
```

Cursor: open **Plugins**, paste `https://github.com/archcore-ai/archcore`, and add the plugin. If you skipped `archcore init`, copy [`docs/cursor.mcp.example.json`](https://github.com/archcore-ai/archcore/blob/main/docs/cursor.mcp.example.json) into `~/.cursor/mcp.json` once. Per-host details, team rollouts, and uninstall: [docs.archcore.ai/guides/connect-your-agent](https://docs.archcore.ai/guides/connect-your-agent/).

## How it compares

| If you rely on… | The gap | What Archcore does instead |
| --- | --- | --- |
| Instruction files (`CLAUDE.md`, `AGENTS.md`, `.cursorrules`) | One growing wall of text: no types, no links, no lifecycle, copied per tool | Typed documents, a relation graph, a draft → accepted lifecycle, one setup for every agent |
| Memory tools (claude-mem, Mem0) | Remember what you did: volatile, opaque, vendor-bound | Store how the system is built and what was decided, versioned in Git and owned by you |
| Methodology kits (BMAD, Spec Kit, Agent OS, Superpowers) | Prescribe a process, often as a one-shot handoff | Keep the artifacts alive as a context graph that evolves with the code; run a kit on top of it |
| RAG or a bigger context window | Retrieves what the code says, not what was decided and why | Keeps decisions and rationale explicit and selective: the agent loads what applies |

Not for: chat memory, a prompt library, or a one-shot spec-to-code generator.

## Documentation

- [Install](https://docs.archcore.ai/start/install/) · [Connect your agent](https://docs.archcore.ai/guides/connect-your-agent/) · [Commands](https://docs.archcore.ai/guides/commands/) · [CLI reference](https://docs.archcore.ai/cli/commands/) · [Document format](https://docs.archcore.ai/reference/document-format/)
- [archcore.ai](https://archcore.ai) · [Privacy](https://archcore.ai/privacy)

## Contributing

One repository holds both components: the CLI under [`cli/`](https://github.com/archcore-ai/archcore/tree/dev/cli) and the plugin under [`plugin/`](https://github.com/archcore-ai/archcore/tree/dev/plugin), developed on the `dev` branch and released together from one tag. `main` is the generated plugin distribution. Setup, tests, and the release process: [CONTRIBUTING.md](https://github.com/archcore-ai/archcore/blob/dev/CONTRIBUTING.md). Bugs and ideas: [issues](https://github.com/archcore-ai/archcore/issues).

## License

[Apache-2.0](LICENSE)
