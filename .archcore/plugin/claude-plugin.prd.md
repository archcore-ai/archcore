---
title: "Archcore Claude Plugin"
status: accepted
tags:
  - "plugin"
  - "vision"
---

## Vision

Make Archcore effortless in Claude Code. The plugin transforms the passive MCP+hook integration into a rich, guided experience where skills teach document types, commands accelerate workflows, a universal agent assists complex documentation tasks, and hooks enforce quality by blocking direct file writes **and auto-inject relevant context before source edits**.

Every interaction with the `.archcore/` knowledge base flows through MCP tools — ensuring validation, templates, relations, and sync manifest are always consistent. Every source-code edit benefits from the applicable rules, ADRs, specs, and patterns being surfaced automatically, without the user having to ask.

## Problem Statement

The current Archcore Claude Plugin (v0.0.1) is a thin wrapper: it registers the MCP server and a SessionStart hook. This leaves significant gaps:

- **No guidance**: Claude doesn't know when or how to use each of the 18 document types. Users must manually instruct the agent about Archcore conventions.
- **No guardrails**: Nothing prevents the agent from writing `.archcore/` files directly via Write/Edit, bypassing validation, templates, and the sync manifest.
- **No workflows**: Common tasks (create an ADR, review documentation health) require manual multi-step instructions every time.
- **No domain expertise**: Complex documentation tasks (requirements engineering, ISO 29148 cascades, multi-document planning) lack specialized assistance.
- **Passive context, not applied context**: The SessionStart index tells the agent documents exist but does not force their content into the decision loop before the agent edits source code. Repo-alignment is a passive nudge, not an active guardrail.

### Target Users

Anyone using Claude Code with Archcore — individual developers, team leads, architects, product managers. The plugin is tool-agnostic within Claude Code: it enhances the Archcore experience regardless of project type or team size.

## Goals and Success Metrics

### Goals

1. **Type-aware assistance**: Claude automatically applies the right document type, template, and best practices based on context
2. **Workflow acceleration**: Common documentation tasks reduced to single slash commands
3. **Quality enforcement**: Direct `.archcore/` file writes blocked at the hook level, redirected to MCP tools
4. **Expert assistance**: Universal agent handles complex multi-document tasks (requirements engineering, documentation review, relation management)
5. **Applied repo alignment**: Applicable rules, ADRs, specs, and patterns reach the agent's context window both on demand (`/archcore:context`) and automatically on source-file edits (PreToolUse injection hook)

### Success Metrics

- All 18 document types reachable through intent skills, track skills, or direct MCP — no per-type skill required
- Slash commands cover the most common workflows (review with dashboard or `--deep` audit, capture, plan, decide, standard, actualize, context, bootstrap)
- PreToolUse hook intercepts 100% of direct Write/Edit attempts on `.archcore/` files
- Users never need to manually explain Archcore conventions to Claude
- Every source-file edit outside `.archcore/` triggers automatic top-3 context injection when any document references that path

## Requirements

### Functional Requirements

#### FR-1: Skills (intent + track + utility)

The plugin ships 16 skills total: 9 intent skills (Layer 1, primary user surface), 6 track skills (Layer 2, multi-document flows), and 1 utility skill (`verify`, plugin-developer maintenance). There are no per-document-type skills — per-type elicitation lives inline inside intent and track skills. Skills are auto-invoked by the model based on user phrasing; the utility skill is user-only. Each intent/track skill covers: when to use, routing/flow, MCP tool calls, and relation guidance.

#### FR-2: Slash Commands

User-invoked commands for common workflows:

- `/archcore:review` — Default short dashboard (counts, status breakdown, relations, orphans). With `--deep` or any filter argument, runs a full health audit with coverage gaps, staleness, and prioritized recommendations.
- `/archcore:capture` — Document a module, component, or system (routes to adr/spec/doc/guide).
- `/archcore:plan` — Plan a feature end-to-end (idea → PRD → plan or single plan).
- `/archcore:decide` — Record a decision (ADR) or draft a proposal (RFC).
- `/archcore:standard` — Establish a team standard (ADR → optional CPAT → rule → guide).
- `/archcore:actualize` — Detect stale documentation via code drift, cascade, and temporal analysis.
- `/archcore:context` — On-demand pull of applicable rules/ADRs/specs/cpats for a code area, topic, or current-focus pickup.
- `/archcore:bootstrap` — First-time onboarding (stack rule + run guide + optional imports).
- `/archcore:help` — Guide to commands and capabilities.

For any document type not directly covered by an intent (including niche ISO and discovery types), use the matching track skill (`/archcore:iso-track`, `/archcore:sources-track`, `/archcore:architecture-track`, `/archcore:standard-track`, `/archcore:product-track`, `/archcore:feature-track`) or call `mcp__archcore__create_document(type=<any>)` directly.

#### FR-3: Universal Agent (archcore-assistant)

One subagent that covers all documentation scenarios:

- Full knowledge of all 18 document types and their templates
- Requirements engineering expertise (product track, sources track, ISO 29148 cascade)
- Relation pattern knowledge (implements, extends, depends_on, related)
- Tool restrictions: archcore MCP tools + Read + Grep + Glob (no Write/Edit on `.archcore/`)
- Invokable manually or automatically by Claude when complex documentation tasks arise

#### FR-4: Validation Hooks

- **PreToolUse (Write|Edit) — block** (`check-archcore-write`): If the target file matches `.archcore/**/*.md`, block the operation and return a message redirecting to the appropriate MCP tool.
- **PreToolUse (Write|Edit) — inject** (`check-code-alignment`): If the target file is outside `.archcore/` and inside a configured source root, scan `.archcore/` for documents referencing the path and inject top-3 (by specificity → type priority) as `additionalContext`. Non-blocking by design.
- **PostToolUse (MCP mutations) — validate** (`validate-archcore`): After `create_document` / `update_document` / `remove_document` / `add_relation` / `remove_relation`, run `archcore doctor` and report issues.
- **PostToolUse (`update_document`) — cascade** (`check-cascade`): After document updates, list documents that reference the updated one via `implements` / `depends_on` / `extends` so the agent can review them for drift.
- **SessionStart**: Loads project context at session start (document index + tags + relation count).

#### FR-5: Empty-State Session Nudge

When a session starts in a repo that is either missing `.archcore/` or functionally empty (no `.md` file ≥ 200 bytes), the SessionStart hook emits a one-line advisory pointing the user at `/archcore:bootstrap`. The nudge is purely informational — never blocks — and can be disabled via `ARCHCORE_HIDE_EMPTY_NUDGE=1`. Once any substantial document exists, the nudge disappears automatically (no persistent flag).

#### FR-6: Bootstrap Skill

A `/archcore:bootstrap` intent skill that seeds an empty `.archcore/` with a useful starting set in three independently-confirmable steps:

1. A short imperative **stack rule** (≤ 6 lines, no versions, ≤ 5 signals) derived from project manifests (`package.json`, `pyproject.toml`, `Cargo.toml`, etc.).
2. A short **run-the-app guide** derived from the README's install/setup section or from manifest `scripts:`, with monorepo awareness.
3. An **opt-in import** of existing agent-instruction files (`CLAUDE.md`, `AGENTS.md`, `.cursorrules`, `.cursor/rules/*.mdc`, `.github/copilot-instructions.md`, `.windsurfrules`, `.junie/guidelines.md`, `CONVENTIONS.md`). Default mode is **link** (doc with single-line pointer, zero content duplication); optional **extract** mode routes content into typed `rule` / `adr` / `doc` documents. A cost warning fires for large inputs (> 50 KB OR > 5 files OR > 8 estimated docs) and requires explicit `do` confirmation.

Each step is skippable, re-runs are idempotent (detected via `imported` + `source:<slug>` tags), and every creation goes through MCP tools. The skill is auto-invocable on phrases like "bootstrap archcore", "initialize archcore", "set up archcore", "first-time setup".

### Non-Functional Requirements

- **NFR-1: MCP-only operations** — All `.archcore/` document operations MUST go through MCP tools. No plugin component (skill, command, agent) should ever instruct direct file writes.
- **NFR-2: Idempotent hooks** — Hooks must be safe to run multiple times without side effects.
- **NFR-3: Performance** — Blocking hooks must complete within 1 second (`timeout: 1` in manifests). Non-blocking validation hooks within 3 seconds. Skills and commands must not add perceptible latency.
- **NFR-4: Graceful degradation** — If `archcore` CLI is not installed, the plugin should inform the user and provide installation instructions rather than failing silently. The push-mode context injection hook must never block an edit on any internal error — it is strictly additive.
- **NFR-5: No template duplication** — Skills reference the template system; they don't embed template content that could drift from the CLI.
- **NFR-6: Multi-host parity** — Every hook, skill, and command works identically on Claude Code and Cursor. Hosts with a weaker hook contract (e.g. Cursor's `preToolUse` `additional_context` handling) degrade gracefully without regression on other JTBDs.

## Delivered capabilities (v0.3.0)

- **SessionStart index** — loads documents, tags, relation count on session start (JTBD #2).
- **SessionStart empty-state nudge** — on missing or functionally-empty `.archcore/`, emits a one-line pointer at `/archcore:bootstrap`; suppressible via `ARCHCORE_HIDE_EMPTY_NUDGE=1`.
- **PreToolUse guardrails** — `check-archcore-write` blocks direct `.archcore/*.md` writes; `check-code-alignment` injects applicable rules/ADRs/specs/cpats for source-file edits (JTBD #1 push mode).
- **PostToolUse validation** — `validate-archcore` + `check-cascade` run on every MCP mutation (JTBD #1/#3 back-pressure).
- **Intent skills** — 9 Layer 1 skills (**bootstrap**, capture, context, plan, decide, standard, review, actualize, help) route natural-language intent into the right document type or workflow. The `review` intent covers both the dashboard role (former `status`) and the full audit; the standalone `graph` intent has been removed.
- **Tracks** — 6 multi-step workflows (product, sources, iso, architecture, standard, feature) for JTBD #4.
- **Universal agent** — `archcore-assistant` for complex multi-document tasks; `archcore-auditor` as a read-only reviewer.

See `development-roadmap.plan.md` for what remains.
