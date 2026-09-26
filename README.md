# Archcore - Spec-driven development and git-native context engineering for AI coding agents

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/archcore-ai/archcore)](https://github.com/archcore-ai/archcore/releases)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)](https://github.com/archcore-ai/archcore/releases)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Docs](https://img.shields.io/badge/docs-docs.archcore.ai-2563EB)](https://docs.archcore.ai)

> **Contributors:** source code and tests live on the [`dev`](https://github.com/archcore-ai/archcore/tree/dev) branch. `main` holds only the published plugin. Start with [CONTRIBUTING.md](https://github.com/archcore-ai/archcore/blob/dev/CONTRIBUTING.md).

**Stop re-explaining your repo to every AI coding agent.**

Archcore keeps your project's decisions, specs, and rules in the repo. Your coding agent reads them before it writes, so it builds by this repo's rules instead of the ones it happens to know.

## What you get

- **Code that fits this repo on the first try.** The decision that already chose Redis, and the rule for error shapes in `src/api/`, reach the agent before it edits the file.
- **Nothing to re-explain in a new session.** Each session opens with what is decided and what is in progress. Switch to another agent and it reads the same folder.
- **A broken decision caught before merge.** Review reads your branch against the spec and the decision record, and returns `code-wrong` on the file that ignored them.

## Get started

**Install.** One binary, nothing to run in the background.

```bash
curl -fsSL https://archcore.ai/install.sh | bash    # macOS, Linux, WSL
```

```powershell
irm https://archcore.ai/install.ps1 | iex            # Windows, PowerShell 5.1+
```

**Connect your agents.** In your project folder:

```bash
archcore init
```

This creates `.archcore/`, connects the agents it finds, and installs the Archcore plugin on the hosts you pick: Claude Code, Cursor, Codex CLI, GitHub Copilot CLI.

Already have a `CLAUDE.md`, `AGENTS.md`, rule files, or an ADR folder? Say `/archcore:init import` in your agent and they become typed documents. Keep the originals for host-specific guidance.

## Your first feature

Open your agent and say what you want. Plain sentences work in every connected agent; on Claude Code, Cursor, Codex CLI, and Copilot the slash command is the shortcut. Each step uses what the previous one saved, which is why the review at the end knows what the plan and the decision said.

The example below adds rate limiting to a public API.

| Shortcut             | You say                                               | What your agent leaves behind                                                                                                                                                                |
| -------------------- | ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/archcore:init`     | "Set up Archcore in this repo."                       | A proposal of documents for the architecture, the rules, and the key modules. You approve before anything is saved.                                                                          |
| `/archcore:plan`     | "Plan rate limiting for the public API."              | `api/rate-limiting.spec.md`, how rate limiting must behave, and `api/rate-limiting.plan.md`, the work broken into tasks. Both written from the rules the project already has.                |
| `/archcore:document` | "Record the decision to use a token bucket in Redis." | `api/token-bucket-in-redis.adr.md`: the choice and its reasoning, found by every later task that touches the API.                                                                            |
| `/archcore:review`   | "Review my branch before merge."                      | A verdict per finding, read against the spec and the decision: `code-wrong` on the handler that kept an in-memory counter, `spec-wrong` on the document the code outgrew, `ok` for the rest. |

Between these steps you code as usual. The spec and the rule for `src/api/` reach the agent before it edits a file there, with no command from you.

The decision from step 3, as it lands in `.archcore/api/token-bucket-in-redis.adr.md`:

```markdown
---
title: Rate limiting uses a token bucket in Redis
status: accepted
---

## Context

The public API needs per-client limits before the partner launch. Redis is already the shared store for sessions (src/session/store.go), and the API runs on three replicas, so a per-process counter never sees the whole client.

## Decision

Token bucket per API key, stored in Redis, refilled every 10 seconds.

## Alternatives Considered

1. In-memory counters per process: rejected because each of the three replicas would grant the full quota.
2. Rate limiting at the load balancer: deferred because it keys by IP, not by API key.

## Consequences

- Every handler in src/api/ reads the bucket from Redis. No in-memory counters.
- Adds one Redis round-trip per request. [expected] Under 2 ms inside the VPC.
```

Tomorrow, in a new session or in a different agent, the recap says what is decided and what is in progress. The next feature starts from there.

## Project knowledge becomes files

Specs define intent, and a spec is one part of the context. Decisions, rules, plans, and guides live beside it in `.archcore/`, as plain Markdown, versioned with the code they describe.

```text
.archcore/
├── architecture.doc.md
├── conventions.rule.md
├── api/
│   ├── rate-limiting.spec.md
│   ├── rate-limiting.plan.md
│   ├── token-bucket-in-redis.adr.md
│   └── error-shapes.rule.md
├── auth/
│   ├── session-model.adr.md
│   └── oauth-migration.rfc.md
├── billing/
│   ├── usage-based-pricing.prd.md
│   └── stripe-webhooks.spec.md
└── testing.guide.md
```

- **Typed by filename.** `adr`, `spec`, `rule`, `plan`, `guide`, `prd`: 23 types, each with a status that moves from `draft` to `accepted` when you approve it.
- **Linked, not piled.** Documents point at each other through seven kinds of relation, including `implements`, `depends_on`, and `supersedes`, so a decision carries the spec it serves and the one it replaced.
- **Reviewed like code.** A change to context is a diff in a pull request. It travels with every clone, and a company-wide `.archcore/` can supply defaults that a project overrides.
- **Loaded on demand.** The Archcore CLI serves the folder to your agent over MCP: a compact index at session start, full documents only when a task needs them. Your context window stays yours.

This repository's own [`.archcore/`](https://github.com/archcore-ai/archcore/tree/dev/.archcore) is a working example. Archcore is built with Archcore.

## Works with your agent

Claude Code, Cursor, Codex CLI, GitHub Copilot, Gemini CLI, OpenCode, Roo Code, and Cline read the same folder. Slash commands, skills, and guardrails run inside the first four; the rest reach the same documents over MCP. Where the host supports hooks, context arrives before the edit with no command from you.

`archcore init` opens a host picker with the agents it detects pre-checked and wires the ones you confirm. Per-host details, team rollouts, and uninstall: [Connect your agent](https://docs.archcore.ai/guides/connect-your-agent/).

<details>
<summary>Install the plugin without <code>archcore init</code></summary>

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

Cursor: open **Plugins**, paste `https://github.com/archcore-ai/archcore`, and add the plugin. Without `archcore init`, copy [`docs/cursor.mcp.example.json`](https://github.com/archcore-ai/archcore/blob/main/docs/cursor.mcp.example.json) into `~/.cursor/mcp.json` once.

</details>

## How it compares

| If you rely on…                                              | The gap                                                                     | What Archcore does instead                                                                     |
| ------------------------------------------------------------ | --------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| Instruction files (`CLAUDE.md`, `AGENTS.md`, `.cursorrules`) | One growing wall of text: no types, no links, no lifecycle, copied per tool | Typed documents, a relation graph, a draft → accepted lifecycle, one setup for every agent     |
| Memory tools (claude-mem, Mem0)                              | Remember what you did: volatile, opaque, vendor-bound                       | Store how the system is built and what was decided, versioned in Git and owned by you          |
| Methodology kits (BMAD, Spec Kit, Agent OS, Superpowers)     | Prescribe a process, often as a one-shot handoff                            | Keep the artifacts alive as a context graph that evolves with the code; run a kit on top of it |
| RAG or a bigger context window                               | Retrieves what the code says, not what was decided and why                  | Keeps decisions and rationale explicit and selective: the agent loads what applies             |

Not for: chat memory, a prompt library, or a one-shot spec-to-code generator.

## Questions

**Does my code leave my machine?** Archcore stores project documents locally in `.archcore/`. Your coding agent may send document excerpts to its model provider, as it does with any file. Install and update analytics carry version and platform information, not your project content. Details and opt-out: [privacy](https://archcore.ai/privacy).

**I already have a `CLAUDE.md` or `.cursor/rules`. Do I start over?** No. `/archcore:init import` turns the useful parts into typed documents, and the files stay for host-specific guidance.

**Do I need both the plugin and the CLI?** You install one thing. The CLI is the context infrastructure; `archcore init` adds the plugin, the command surface and guardrails, on the hosts you pick. On any other MCP-aware agent the CLI is all there is.

## Documentation

- [Install](https://docs.archcore.ai/start/install/) · [Quick start](https://docs.archcore.ai/start/quick-start/) · [Connect your agent](https://docs.archcore.ai/guides/connect-your-agent/) · [Commands](https://docs.archcore.ai/guides/commands/) · [CLI reference](https://docs.archcore.ai/cli/commands/) · [Document format](https://docs.archcore.ai/reference/document-format/)
- [archcore.ai](https://archcore.ai) · [How to use](https://archcore.ai/how-to-use/) · [Privacy](https://archcore.ai/privacy)

## Contributing

One repository holds both components: the CLI under [`cli/`](https://github.com/archcore-ai/archcore/tree/dev/cli) and the plugin under [`plugin/`](https://github.com/archcore-ai/archcore/tree/dev/plugin), developed on `dev` and released together from one tag. Setup, tests, and the release process: [CONTRIBUTING.md](https://github.com/archcore-ai/archcore/blob/dev/CONTRIBUTING.md). Bugs and ideas: [issues](https://github.com/archcore-ai/archcore/issues).

## License

[Apache-2.0](LICENSE)
