---
title: "Archcore Claude Plugin"
status: accepted
tags:
  - "plugin"
  - "vision"
---

## Vision

Make Archcore effortless in Claude Code, Cursor, and Codex CLI. The plugin transforms the passive MCP+hook integration into a rich, guided experience where intent skills route user phrasing into the right document type or flow, a universal agent assists complex documentation tasks, and hooks enforce quality by blocking direct file writes **and auto-inject relevant context before source edits**.

Every interaction with the `.archcore/` knowledge base flows through MCP tools — ensuring validation, templates, relations, and sync manifest are always consistent. Every source-code edit benefits from the applicable rules, ADRs, specs, and patterns being surfaced automatically, without the user having to ask.

## Problem Statement

The original Archcore Claude Plugin (v0.0.1) was a thin wrapper: it registered the MCP server and a SessionStart hook. This left significant gaps:

- **No guidance**: Claude doesn't know when or how to use each document type. Users must manually instruct the agent about Archcore conventions.
- **No guardrails**: Nothing prevents the agent from writing `.archcore/` files directly via Write/Edit, bypassing validation, templates, and the sync manifest.
- **No workflows**: Common tasks (create an ADR, audit documentation health) require manual multi-step instructions every time.
- **No domain expertise**: Complex documentation tasks (requirements engineering, ISO 29148 cascades, multi-document planning) lack specialized assistance.
- **Passive context, not applied context**: The SessionStart index tells the agent documents exist but does not force their content into the decision loop before the agent edits source code.

### Target Users

Anyone using Claude Code, Cursor, or Codex CLI with Archcore — individual developers, team leads, architects, product managers.

## Goals and Success Metrics

### Goals

1. **Type-aware assistance**: Claude automatically applies the right document type, template, and best practices based on context.
2. **Workflow acceleration**: Common documentation tasks reduced to single slash commands.
3. **Quality enforcement**: Direct `.archcore/` file writes blocked at the hook level, redirected to MCP tools.
4. **Expert assistance**: Universal agent handles complex multi-document tasks.
5. **Applied repo alignment**: Applicable rules, ADRs, specs, and patterns reach the agent's context window both on demand (`/archcore:context`) and automatically on source-file edits (PreToolUse injection hook).

### Success Metrics

- All 18 document types reachable through 7 intent skills or direct MCP — no per-type skill required.
- Slash commands cover the most common workflows: init (onboarding), capture (documentation), decide (decisions + standard cascade), plan (any forward-looking flow), audit (health + drift), context (pull), help.
- PreToolUse hook intercepts 100% of direct Write/Edit attempts on `.archcore/` files.
- Users never need to manually explain Archcore conventions to Claude.
- Every source-file edit outside `.archcore/` triggers automatic top-3 context injection when any document references that path.

## Requirements

### Functional Requirements

#### FR-1: Intent Skills (Seven-Skill Surface)

The plugin ships **7 auto-invocable intent skills** (per `skill-surface-collapse.adr.md`): `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`. Each maps to a clearly distinct user intent and is auto-invoked by the model based on user phrasing.

Per-type elicitation lives inline inside the matching intent. Per-flow logic (product/sources/iso/feature cascades) lives under `skills/plan/references/`. Continuation logic for the decision/standard cascade (ADR → optional CPAT → optional rule → optional guide) lives under `skills/decide/references/continuations.md`. Drift-detection logic lives under `skills/audit/lib/drift-detection.md`.

#### FR-2: Slash Commands

User-invoked commands for common workflows:

- `/archcore:init` — First-time onboarding (stack rule + run guide + optional agent-file imports).
- `/archcore:capture` — Document a module, component, or system (routes to adr/spec/doc/guide).
- `/archcore:decide` — Record a decision (ADR) or draft a proposal (RFC); optional standard cascade (CPAT → rule → guide).
- `/archcore:plan` — Plan a feature or initiative; routes to single plan or one of four flows (product, sources, iso, feature).
- `/archcore:audit` — Documentation health: dashboard (default), `--deep` audit, or `--drift` detection.
- `/archcore:context` — On-demand pull of applicable rules/ADRs/specs/cpats for a code area, topic, or current-focus pickup.
- `/archcore:help` — Guide to commands and capabilities.

For any document type, `mcp__archcore__create_document(type=<any>)` remains a direct path that bypasses skill mediation.

#### FR-3: Universal Agent (archcore-assistant)

One subagent that covers all documentation scenarios:

- Full knowledge of all 18 document types and their templates.
- Requirements engineering expertise (product flow, sources flow, ISO 29148 cascade).
- Relation pattern knowledge (implements, extends, depends_on, related).
- Tool restrictions: archcore MCP tools + Read + Grep + Glob (no Write/Edit on `.archcore/`).
- Invokable manually or automatically by Claude when complex documentation tasks arise.

Plus `archcore-auditor` — a read-only auditor for documentation health checks (background, restricted tool set).

#### FR-4: Validation Hooks

- **PreToolUse (Write|Edit) — block** (`check-archcore-write`): If the target file matches `.archcore/**/*.md`, block the operation and return a message redirecting to the appropriate MCP tool.
- **PreToolUse (Write|Edit) — inject** (`check-code-alignment`): If the target file is outside `.archcore/` and inside a configured source root, scan `.archcore/` for documents referencing the path and inject top-3 (by specificity → type priority) as `additionalContext`. Non-blocking by design.
- **PostToolUse (MCP mutations) — validate** (`validate-archcore`): After `create_document` / `update_document` / `remove_document` / `add_relation` / `remove_relation`, run `archcore doctor` and report issues.
- **PostToolUse (`update_document`) — cascade** (`check-cascade`): After document updates, list documents that reference the updated one via `implements` / `depends_on` / `extends` so the agent can review them for drift.
- **PostToolUse (`create_document`, `update_document`) — precision** (`check-precision`): Emits soft warnings for forbidden vague words, missing mandatory sections, frontmatter gaps, and stub-length bodies. Never blocks.
- **SessionStart**: Loads project context at session start (document index + tags + relation count) and runs `check-staleness` (rate-limited to once per 24h).

#### FR-5: Empty-State Session Nudge

When a session starts in a repo that is either missing `.archcore/` or functionally empty (no `.md` file ≥ 200 bytes), the SessionStart hook emits a one-line advisory pointing the user at `/archcore:init`. The nudge is purely informational — never blocks — and can be disabled via `ARCHCORE_HIDE_EMPTY_NUDGE=1`. Once any substantial document exists, the nudge disappears automatically.

#### FR-6: Init Skill

A `/archcore:init` intent skill that seeds an empty `.archcore/` with a useful starting set:

1. A short imperative **stack rule** (≤ 6 lines, no versions, ≤ 5 signals) derived from project manifests (`package.json`, `pyproject.toml`, `Cargo.toml`, etc.).
2. A short **run-the-app guide** derived from the README's install/setup section or from manifest `scripts:`, with monorepo awareness.
3. An **opt-in import** of existing agent-instruction files (`CLAUDE.md`, `AGENTS.md`, `.cursorrules`, `.cursor/rules/*.mdc`, `.github/copilot-instructions.md`, `.windsurfrules`, `.junie/guidelines.md`, `CONVENTIONS.md`). Default mode is **link** (doc with single-line pointer, zero content duplication); optional **extract** mode routes content into typed `rule` / `adr` / `doc` documents. A cost warning fires for large inputs and requires explicit `do` confirmation.

The skill scale-detects (small / medium / large) and seeds scale-appropriate extras (entry-point inventory, top-level domain map, hotspot capture candidates). Each step is skippable, re-runs are idempotent (detected via `imported` + `source:<slug>` tags), and every creation goes through MCP tools. Auto-invocable on phrases like "init archcore", "initialize archcore", "set up archcore", "first-time setup".

### Non-Functional Requirements

- **NFR-1: MCP-only operations** — All `.archcore/` document operations MUST go through MCP tools.
- **NFR-2: Idempotent hooks** — Hooks must be safe to run multiple times without side effects.
- **NFR-3: Performance** — Blocking hooks must complete within 1 second (`timeout: 1` in manifests). Non-blocking validation hooks within 3 seconds.
- **NFR-4: Graceful degradation** — If `archcore` CLI is not installed, the plugin informs the user and provides installation instructions. The push-mode context injection hook never blocks an edit on any internal error.
- **NFR-5: No template duplication** — Skills reference the template system; they don't embed template content that could drift from the CLI.
- **NFR-6: Multi-host parity** — Every hook, skill, and command works identically on Claude Code, Cursor, and Codex CLI.

## Delivered capabilities (current)

- **SessionStart index** — loads documents, tags, relation count on session start (JTBD #2).
- **SessionStart empty-state nudge** — on missing or functionally-empty `.archcore/`, emits a one-line pointer at `/archcore:init`; suppressible via `ARCHCORE_HIDE_EMPTY_NUDGE=1`.
- **PreToolUse guardrails** — `check-archcore-write` blocks direct `.archcore/*.md` writes; `check-code-alignment` injects applicable rules/ADRs/specs/cpats for source-file edits (JTBD #1 push mode).
- **PostToolUse validation** — `validate-archcore` + `check-cascade` + `check-precision` run on MCP mutations (JTBD #1/#3 back-pressure).
- **Intent skills (7)** — `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`. Routes natural-language intent into the right document type or workflow. `audit` covers dashboard, deep audit (`--deep`), and drift detection (`--drift`). `plan` covers single plan and all four cascades (product, sources, iso, feature) via on-demand references. `decide` covers ADR/RFC creation plus the standard cascade.
- **Universal agent** — `archcore-assistant` for complex multi-document tasks; `archcore-auditor` as a read-only reviewer.

See `development-roadmap.plan.md` for what remains.
