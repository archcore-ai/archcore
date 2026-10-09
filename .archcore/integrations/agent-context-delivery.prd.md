---
title: "Agent Context Delivery — The Agent Knows Where to Fetch Project Context Before Every Edit"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "hooks"
  - "integrations"
  - "mcp"
  - "multi-host"
---

## Vision

In any project with `.archcore/`, an agent that is about to change, test, or explain code knows the one call that returns the project context for that file, and it receives that context before the edit. This holds on every supported host, with or without the Archcore MCP tools, and at any corpus size.

## Problem Statement

Teams that record rules and decisions in `.archcore/` lose them on the way to the agent, and the developer pays for it in review. In MR !7667 of the Litres monorepo (302 documents) the agent missed `guard-clauses-first.rule.md` and invented a defect that `app-router-data-fetching.doc.md` already disproved. Both documents existed. Today the instruction that teaches the agent where to look sits in channels that two of three probed hosts cut or drop. The pre-edit hint lists 3 documents by directory match, and the agent opens 16% of them. Agents read `.archcore/` through the shell 463 times against 293 MCP calls. A rule that names no path never reaches an edit. Without the CLI, Archcore goes silent and the user is not told.

## Goals and Success Metrics

- Bench, before and after, on a project with no hand-written `.archcore/` section in its instruction files: the agent reads the rule that applies to the task before its first write in at least 80% of runs. Baseline: measured by the same bench before the first change.
- Same bench: the agent calls an Archcore MCP tool before its first write in at least 80% of runs. Pilot on the monorepo with the current block: 0 of 6 runs.
- Marker probe on Claude Code, Codex, and Copilot CLI: the context address reaches the model on all three hosts. Today the server-instruction route reaches 1 host whole.
- [assumption] Share of documents listed by the pre-edit hint that the agent opens afterwards: at least 40%. Baseline: 16% (72 of 462).

## Requirements

1. An agent learns the call that returns the context for its file from whichever Archcore channel reaches it first.
2. The instruction that names this call survives the truncation of every host Archcore supports.
3. The context an agent receives for one file stays within a fixed size, whatever the corpus holds.
4. A rule that names no path reaches the agent before an edit.
5. A document named in the pre-edit hint opens with the path the hint shows.
6. The session-start context tells the agent where to fetch context before it lists project news.
7. An agent working without the Archcore MCP tools tells the user once and still reads the project's rules as files.
8. The language notice and the global-source notice reach the agent on hosts that truncate MCP server instructions.

## Out of Scope

- Experience capture from merge-request review threads (section 4 of the field report that started this work).
- An `applies_to` frontmatter field for rules.
- A live probe of Cursor, and localization of the managed block into the project language.

## Clarifications

- 2026-10-09 — Entry point: a `search_documents` mode, not a dedicated tool (user choice; recommendation was a dedicated tool).
- 2026-10-09 — A rule that names no path counts as general (user choice, recommended).
- 2026-10-09 — A new managed block goes to the top of the instruction file; an existing block keeps its position (user choice, recommended).
- 2026-10-09 — Success metric: the bench threshold of 80% for rule reads and for MCP calls (user choice, recommended).
