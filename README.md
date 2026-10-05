# Archcore - Spec-driven development and git-native context engineering for AI coding agents

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/archcore-ai/archcore)](https://github.com/archcore-ai/archcore/releases)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)](https://github.com/archcore-ai/archcore/releases)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Docs](https://img.shields.io/badge/docs-docs.archcore.ai-2563EB)](https://docs.archcore.ai)

**The agent stops guessing and starts following the system.**

A coding agent can write the code. It does not know your project: what the feature must do, where the code belongs, which decisions and rules already apply. So it guesses, and you explain the same things again in the next session.

Archcore keeps specs, architecture, decisions, rules, and plans in Git, and makes the right project context available to AI coding agents as they work. It helps your coding agent make changes that fit your repo's architecture, rules, and past decisions.

- `/archcore:plan` before you build. The agent implements from a spec, examples, and tasks, sized to the change.
- `/archcore:document` as you go. One sentence from you becomes a finished, linked document.
- `/archcore:review` before merge. Archcore compares the branch with the documents and names the side that is wrong.

![Archcore: from idea to reviewed code](docs/promo.gif)

## Install

On macOS, Linux, or WSL:

```bash
curl -fsSL https://archcore.ai/install.sh | bash
```

On Windows (PowerShell 5.1+):

```powershell
irm https://archcore.ai/install.ps1 | iex
```

Then, in your project:

```bash
archcore init
```

The installer adds the CLI and plugins for Claude Code, Codex CLI, and GitHub Copilot CLI when it finds them. `archcore init` connects the agents you choose to this project. It also supports Gemini CLI, OpenCode, Roo Code, and Cline. Cursor needs [one extra setup step](https://docs.archcore.ai/guides/connect-your-agent/).

Already have a `CLAUDE.md`, `AGENTS.md`, or rule files? Say `/archcore:init import` in your agent to turn them into project documents.

The documents stay in your repo, in `.archcore/`. Details: [privacy](https://archcore.ai/privacy).

## Plan: the agent builds from documents, not from a chat message

```text
/archcore:plan [your feature]
```

`/archcore:plan` reads the repo and the existing documents first. Then it asks you only what it cannot find there. Your answers become documents in `.archcore/`: a spec with requirements, and a plan with tasks mapped to files. A user-facing change also gets examples in Given/When/Then form. Larger work can add a PRD, research, or a formal requirements chain.

Archcore weighs the change, computes the route, and reports its size from S to XL. You never choose a template or a size.

| The change                    | What plan prepares                                             |
| ----------------------------- | -------------------------------------------------------------- |
| A small fix                   | No documents                                                   |
| A settled choice              | A decision record                                              |
| A change to existing behavior | A check of the covering spec: update the spec, or fix the code |
| One new capability            | A spec and a plan                                              |
| Several capabilities          | A PRD, one spec per capability, and a plan                     |

Risk raises the size. A security requirement adds a formal requirements chain. A data migration adds a migration runbook.

The agent then implements from the spec, the examples, and the tasks. The open questions are settled before the code, not after it.

## Document: you say it once, Archcore writes the document

```text
/archcore:document decision why we chose [X]
/archcore:document code [module]
```

`/archcore:document` records what is true now: a decision, a team standard, how a module works, or a how-to. You do not choose a format. Archcore selects the document type, reads the code and the existing documents, and checks that the document does not exist yet. It asks you only what it cannot find. It links the new document to the related ones. A decision can also produce the rule and the guide that follow from it.

With no subject, `/archcore:document` reads the changes on your branch and asks one question about what to record.

The document outlives the session. It is in Git with the code, it has a status (`draft`, then `accepted`), and review checks it against the code.

## Review: the code and the documents agree before merge

```text
/archcore:review
```

**When documents cover the change.** `/archcore:review` compares the branch with them in both directions. Each finding names the code and the document that disagree, and carries one verdict: `code-wrong` when the code breaks a document that still stands, `spec-wrong` when the document is out of date. You fix the right side. When a plan covers the branch, review checks its tasks and closes the plan when the work is done.

**When no document covers the change.** Review still checks the branch against the decisions and rules the project has. If the branch repeats a pattern that no document records, review offers to record it. To record work that shipped without a plan, run `/archcore:document`.

Use the three commands together on one change, or use one alone. Archcore does not need a plan for every change.

The slash commands run in Claude Code, Cursor, Codex CLI, and GitHub Copilot. In every connected agent, a plain sentence works too: “Plan [your feature]”, “Record why we chose [X]”, “Review my branch”.

## Project knowledge becomes files

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

A spec is one part of context, not the whole context: decisions, rules, plans, and guides live beside it. A change to a document is a diff in a pull request, like a change to code. This repository's own [`.archcore/`](https://github.com/archcore-ai/archcore/tree/dev/.archcore) is a working example.

## Go deeper

[How Archcore works](https://archcore.ai/how-to-use/) · [Quick start](https://docs.archcore.ai/start/quick-start/) · [Commands](https://docs.archcore.ai/guides/commands/) · [Contributing](https://github.com/archcore-ai/archcore/blob/dev/CONTRIBUTING.md) (source is on `dev`) · [Apache 2.0 license](LICENSE)
