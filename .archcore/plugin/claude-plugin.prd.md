---
title: "Archcore Claude Plugin"
status: accepted
tags:
  - "plugin"
  - "vision"
---

## Vision

Make Archcore effortless in Claude Code, Cursor, Codex CLI, and GitHub Copilot CLI. The plugin turns a passive MCP-and-hook integration into a guided experience: intent skills route user phrasing into the right document type or flow, a universal agent assists complex documentation tasks, and hooks enforce quality by blocking a direct file write **and auto-injecting relevant context before a source edit**.

Every interaction with the `.archcore/` knowledge base flows through MCP tools, so validation, templates, relations, and the sync manifest stay consistent. Every source-code edit carries the applicable rules, ADRs, specs, and patterns into the agent's context without the user asking for them.

## Problem Statement

The original plugin at v0.0.1 was a thin wrapper that registered the MCP server and a SessionStart hook, which left five gaps.

- **No guidance.** The model does not know when or how to use each document type, so users must instruct the agent about Archcore conventions by hand.
- **No guardrails.** Nothing prevents the agent from writing a `.archcore/` file directly through Write or Edit, bypassing validation, templates, and the sync manifest.
- **No workflows.** A common task such as creating an ADR or auditing documentation health requires manual multi-step instructions every time.
- **No domain expertise.** Complex work — requirements engineering, ISO 29148 cascades, multi-document planning — has no specialized assistance.
- **Passive context rather than applied context.** The SessionStart index tells the agent that documents exist but does not force their content into the decision loop before the agent edits source code.

**Target users.** Anyone using Claude Code, Cursor, Codex CLI, or GitHub Copilot CLI with Archcore: individual developers, team leads, architects, and product managers.

## Goals and Success Metrics

**Goals.**

1. **Type-aware assistance.** The model applies the right document type, template, and content contract from context.
2. **Workflow acceleration.** A common documentation task reduces to a single command.
3. **Quality enforcement.** A direct `.archcore/` file write is blocked at the hook level and redirected to an MCP tool.
4. **Expert assistance.** A universal agent handles complex multi-document tasks.
5. **Applied repo alignment.** Applicable rules, ADRs, specs, and patterns reach the agent's context both on demand and automatically on a source-file edit.

**Success metrics.**

- All 18 document types are reachable through the four commands or directly through MCP, with no per-type skill required.
- The commands cover the common workflows: onboarding, documentation, decisions with the standard cascade, any forward-looking flow, health and drift, the on-demand pull, and help.
- The PreToolUse guard intercepts every direct Write or Edit attempt on a `.archcore/` file.
- Users never explain Archcore conventions to the model by hand.
- Every source-file edit outside `.archcore/` triggers automatic top-3 context injection whenever a document references that path.

## Requirements

### Functional

**FR-1 — the four-command palette.** The plugin ships four auto-invocable commands per `four-command-palette.adr`: `init`, `plan`, `document`, `review`. Each maps to a distinct user intent and auto-invokes from user phrasing. Per-type elicitation lives inline inside the matching intent; per-flow logic for the product, sources, ISO, and feature cascades lives under `skills/plan/references/`; continuation logic for the decision and standard cascade lives at `skills/decide/references/continuations.md`; and drift detection lives at `skills/audit/lib/drift-detection.md`.

**FR-2 — slash commands.** The user-invoked surface is `/archcore:{init,plan,document,review}` per `command-surface-v2.spec`. `capture` and `decide` are absorbed by `document`; `audit` becomes `review`; `context` is removed, with CLI hooks and command grounding absorbing the pull moment; `help` is removed, with command descriptions and CLI help absorbing it. For any type, `mcp__archcore__create_document(type=<any>)` remains a direct path that bypasses skill mediation.

**FR-3 — the universal agent.** One subagent, `archcore-assistant`, covers every documentation scenario: full knowledge of all 18 document types and their templates, requirements-engineering expertise across the product flow, sources flow, and ISO 29148 cascade, relation-pattern knowledge across the four relation types, and a tool set restricted to the archcore MCP tools plus Read, Grep, and Glob with no Write or Edit on `.archcore/`. It is invokable manually or automatically. Alongside it, `archcore-auditor` runs read-only documentation health checks in the background with a restricted tool set.

**FR-4 — validation hooks.** The pre-mutation block guard denies a Write or Edit whose target matches `.archcore/**/*.md` and returns a message naming the MCP tool to use. The pre-mutation injection guard, for a target outside `.archcore/` and inside a configured source root, scans for documents referencing the path and injects the top 3 by specificity and then type priority, without ever blocking. The post-mutation validator runs `archcore doctor` after each document mutation and reports issues. The cascade hook lists, after an update, the documents that reference the updated one through `implements`, `depends_on`, or `extends`. The precision hook emits soft warnings for forbidden vague words, missing mandatory sections, frontmatter gaps, and stub-length bodies, and never blocks. SessionStart loads the project context — the document index, tags, and relation count — and runs the staleness check, rate-limited to once per 24 hours.

**FR-5 — the empty-state session nudge.** When a session starts in a repository that is missing `.archcore/` or functionally empty, meaning it holds no `.md` file of at least 200 bytes, the SessionStart hook emits a one-line advisory pointing at `/archcore:init`. The nudge is informational, never blocks, is disabled by `ARCHCORE_HIDE_EMPTY_NUDGE=1`, and disappears once any substantial document exists.

**FR-6 — the init skill.** `/archcore:init` seeds an empty `.archcore/` with a useful starting set: a short imperative stack rule of 6 lines or fewer, carrying no versions and at most 5 signals, derived from the project manifests; a short run-the-app guide derived from the README's install section or from the manifest scripts, with monorepo awareness; and an opt-in import of the existing agent-instruction files, defaulting to link mode, which writes a doc with a single-line pointer and duplicates no content, with an optional extract mode routing content into typed documents behind a cost warning that requires explicit confirmation for a large input.

The skill detects scale as small, medium, or large, and seeds the extras that scale calls for — an entry-point inventory, a top-level domain map, and hotspot capture candidates. Every step is skippable, a re-run is idempotent through the `imported` and `source:<slug>` tags, and every creation goes through an MCP tool. It auto-invokes on phrasing such as "init archcore" or "set up archcore". `magic-first-day-init.adr` extends this requirement with the tiered extractive-facts and confirmed-synthesis model.

### Non-functional

- **NFR-1 — MCP-only operations.** Every `.archcore/` document operation MUST go through an MCP tool.
- **NFR-2 — idempotent hooks.** A hook MUST be safe to run repeatedly with no side effect.
- **NFR-3 — performance.** A blocking hook MUST complete within 1 second, and a non-blocking validation hook within 3 seconds.
- **NFR-4 — graceful degradation.** IF the `archcore` CLI is not installed, THEN the plugin MUST inform the user and supply the installation instructions. The injection hook MUST NOT block an edit on any internal error.
- **NFR-5 — no template duplication.** A skill MUST reference the template system rather than embed template content that could drift from the CLI.
- **NFR-6 — multi-host parity.** Every hook, skill, and command MUST behave identically across the supported hosts, with each documented host gap declared by name.

## Delivered capabilities

- **SessionStart index** — loads documents, tags, and the relation count at session start.
- **SessionStart empty-state nudge** — on a missing or functionally empty `.archcore/`, emits a one-line pointer at `/archcore:init`, suppressible by environment variable.
- **PreToolUse guardrails** — the block guard denies a direct `.archcore/` markdown write, and the injection guard supplies the applicable rules, ADRs, specs, and cpats for a source-file edit.
- **PostToolUse validation** — the validator, the cascade detector, and the precision check all run on MCP mutations.
- **Four commands** — routing natural-language intent into the right document type or workflow, with `review` covering the dashboard, the deep audit, and drift detection (absorbing former `audit`), `plan` covering a single plan and all cascades through on-demand references, and `document` covering module capture plus ADR/RFC creation and the standard cascade (absorbing former `capture` and `decide`).
- **Two agents** — `archcore-assistant` for complex multi-document tasks and `archcore-auditor` as the read-only reviewer.

`development-roadmap.plan` records what remains.
