<div align="center">

# Archcore

**Git-native project context for AI coding agents.<br>The agent stops guessing and starts following the system.**

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE) [![Release](https://img.shields.io/github/v/release/archcore-ai/archcore)](https://github.com/archcore-ai/archcore/releases) [![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)](https://github.com/archcore-ai/archcore/releases) [![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev) [![Docs](https://img.shields.io/badge/docs-docs.archcore.ai-2563EB)](https://docs.archcore.ai)

**[Website](https://archcore.ai) · [Docs](https://docs.archcore.ai) · [Quick start](https://docs.archcore.ai/start/quick-start/)**

[Install](#install) · [Three commands](#three-commands) · [What you get](#what-you-get) · [How it compares](#how-it-compares) · [Go deeper](#go-deeper)

<code>curl -fsSL https://archcore.ai/install.sh | bash</code>

<br><br>

<img src="https://raw.githubusercontent.com/archcore-ai/archcore/dev/docs/promo.gif" alt="Archcore: from idea to reviewed code" width="900">

</div>

<br>

A coding agent can write the code. It does not know your project: what the feature must do, where the code belongs, and which decisions and rules already apply. So it guesses, and you explain the same things again in the next session.

Archcore keeps specs, decisions, rules, and plans in Git, next to the code. Your agent reads them before it changes the code, so the change fits your architecture and past decisions.

Ask your agent in plain words:

- *"Plan rate limiting for the public API."*
- *"Record why we chose Redis for the token bucket."*
- *"Document how the billing module works."*
- *"Review my branch before merge."*

The agent picks the document type, asks only what it cannot find in the repo, and writes plain Markdown into `.archcore/`.

---

## Install

**1. Install the CLI and the agent plugins.**

On macOS, Linux, or WSL:

```bash
curl -fsSL https://archcore.ai/install.sh | bash
```

On Windows (PowerShell 5.1+):

```powershell
irm https://archcore.ai/install.ps1 | iex
```

The installer adds the CLI. It also adds the plugins for Claude Code, Codex CLI, and GitHub Copilot CLI when it finds them.

**2. Connect your project.**

```bash
archcore init
```

`archcore init` connects the agents you choose to this project.

**3. Start in your agent.**

```text
/archcore:init
```

`/archcore:init` reads the repo and proposes a first set of documents: the stack, how to run the project, and an architecture overview. You confirm once.

<details>
<summary><b>Already have</b> <code>CLAUDE.md</code>, <code>AGENTS.md</code>, or rule files?</summary>

Say `/archcore:init import` in your agent. Archcore turns those files into typed project documents.

</details>

<details>
<summary><b>Supported agents</b></summary>

| Agent | Setup |
| --- | --- |
| Claude Code, Codex CLI, GitHub Copilot CLI | The installer adds the plugin |
| OpenCode | `archcore init` adds the plugin to the project |
| Gemini CLI, Roo Code, Cline | `archcore init` connects them |
| Cursor | [One extra setup step](https://docs.archcore.ai/guides/connect-your-agent/) |

</details>

<details>
<summary><b>Privacy</b></summary>

The documents stay in your repo, in `.archcore/`. Details: [privacy](https://archcore.ai/privacy).

</details>

---

## Three commands

| When | Command | What you get |
| --- | --- | --- |
| Before you build | `/archcore:plan [your feature]` | A spec, examples, and tasks, sized to the change |
| As you go | `/archcore:document [subject]` | One sentence from you becomes a finished, linked document |
| Before merge | `/archcore:review` | The branch checked against the documents, with the side that is wrong |

Use the three commands together on one change, or use one alone. The slash commands run in Claude Code, Cursor, Codex CLI, and GitHub Copilot. In every connected agent, a plain sentence works too.

### Plan: the agent builds from documents, not from a chat message

```text
/archcore:plan [your feature]
```

`/archcore:plan` reads the repo and the existing documents first. Then it asks you only what it cannot find there. You never choose a template or a mode: plan creates only the documents that the change needs.

| The change | What plan prepares |
| --- | --- |
| A small fix | No documents |
| A settled choice | A decision record |
| A change to existing behavior | A check of the covering spec: update the spec, or fix the code |
| A sequenced migration | A plan |
| One new capability | A spec and a plan |
| Several capabilities | A PRD, one spec per capability, and a plan |

<details>
<summary>Risks, examples, and larger work</summary>

A user-facing change also gets examples in Given/When/Then form. A risk adds the check it needs: a security fix gets a security check. A formal requirements chain starts only when an obligation asks for traceability. A data migration that people run by hand adds a migration runbook. Larger work can add a PRD or research.

The agent then implements from the spec, the examples, and the tasks. The open questions are settled before the code, not after it.

</details>

### Document: you say it once, Archcore writes the document

```text
/archcore:document why we chose [X]
/archcore:document how [module] works
```

`/archcore:document` records what is true now: a decision, a team standard, how a module works, or a how-to. Archcore selects the document type, reads the code, and checks that the document does not exist yet. It links the new document to the related ones.

<details>
<summary>No subject, statuses, and follow-up documents</summary>

With no subject, `/archcore:document` reads the changes on your branch and asks one question about what to record.

A decision can also produce the rule and the guide that follow from it. Each document has a status (`draft`, then `accepted`), and review checks it against the code.

</details>

### Review: the code and the documents agree before merge

```text
/archcore:review
```

When documents cover the change, `/archcore:review` compares the branch with them in both directions. Each finding names the code and the document that disagree, and carries one verdict:

- `code-wrong`: the code breaks a document that still stands. Fix the code.
- `spec-wrong`: the document is out of date. Fix the document.

<details>
<summary>Plans, and changes no document covers</summary>

When a plan covers the branch, review checks its tasks and closes the plan when the work is done.

When no document covers the change, review still checks the branch against the decisions and rules the project has. If the branch repeats a pattern that no document records, review offers to record it. To record work that shipped without a plan, run `/archcore:document`.

</details>

---

## What you get

Each command leaves plain Markdown in `.archcore/`, versioned with the code it describes. The document type is in the filename.

```text
.archcore/
├── architecture/
│   └── architecture-overview.doc.md    ← /archcore:init
├── conventions/
│   └── project-stack.rule.md           ← /archcore:init
└── api/
    ├── rate-limiting.spec.md           ← /archcore:plan
    ├── rate-limiting.plan.md           ← /archcore:plan
    ├── token-bucket-in-redis.adr.md    ← /archcore:document
    └── error-shapes.rule.md            ← /archcore:document
```

A change to a document is a diff in a pull request, like a change to code. This repository's own [`.archcore/`](https://github.com/archcore-ai/archcore/tree/dev/.archcore) is a working example.

---

## How it compares

| | Chat history | One `CLAUDE.md` or `AGENTS.md` | Archcore |
| --- | --- | --- | --- |
| Where context lives | In one session | In one file in the repo | In typed documents in `.archcore/` |
| Next session | Starts from zero | Reads the whole file | Reads the documents for the files it changes |
| Who writes it | Nobody | You, by hand | The agent, from one sentence from you |
| Kinds of context | Mixed in the chat | Mixed in one file | Specs, decisions, rules, plans, and guides, each in its own type |
| Checked against code | No | No | `/archcore:review` before merge |

The trade-offs are real. The documents are files that your team reviews and keeps current, like code. A small fix gets no documents, so Archcore adds little to a project that has only small fixes. A single `CLAUDE.md` is still enough for a small repo with few rules.

---

## Go deeper

- [How Archcore works](https://archcore.ai/how-to-use/)
- [Quick start](https://docs.archcore.ai/start/quick-start/)
- [Commands](https://docs.archcore.ai/guides/commands/)
- [Contributing](https://github.com/archcore-ai/archcore/blob/dev/CONTRIBUTING.md) (the source is on `dev`)
- [Apache 2.0 license](LICENSE)
