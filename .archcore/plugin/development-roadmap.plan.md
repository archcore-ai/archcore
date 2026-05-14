---
title: "Plugin Development Roadmap"
status: accepted
tags:
  - "plugin"
  - "roadmap"
---

## Goal

Deliver the complete Archcore Plugin feature set, transforming the original thin MCP+hook wrapper into a rich, guided Archcore experience across Claude Code, Cursor, and Codex CLI.

## Tasks

### Phase 1: Documentation — DONE

Created comprehensive project documentation using Archcore's own document types (dogfooding).

### Phase 2: Skills — DONE, then collapsed

Built skills across the multi-tier hierarchy as planned, then progressively consolidated per a series of ADRs. See **Phase 9** for the final state.

Historical evolution:

- Initial build: intent (Layer 1) + track (Layer 2) + type (Layer 3) + utility tiers; up to 34 skill directories at peak.
- `remove-document-type-skills.adr.md`: Layer 3 removed → 18 skills.
- `merge-review-status-remove-graph.adr.md`: `status` merged into `review`, `graph` removed → 16 skills.
- `skill-surface-collapse.adr.md`: track tier removed, `actualize` merged into `audit`, `bootstrap` renamed to `init`, `standard` merged into `decide`, `verify` removed → **7 skills (current)**.

### Phase 3: Commands and Agents — DONE

Built user-invoked command surface and subagents:

- [x] Intent skills as primary user entry points.
- [x] Every Archcore document type reachable via an intent skill or direct MCP (`create_document(type=<any>)`).
- [x] `archcore-assistant` agent — read/write agent with full MCP tool access.
- [x] `archcore-auditor` agent — read-only auditor with code-document correlation.

### Phase 4: Hooks and Validation — DONE

Built the enforcement and freshness detection layer (6 hook entries per host):

- [x] SessionStart hook (`bin/session-start`) — CLI availability check, project check, context loading, staleness check via `bin/check-staleness`.
- [x] PreToolUse hook (`bin/check-archcore-write`) — blocks direct `.archcore/**/*.md` writes.
- [x] PreToolUse hook (`bin/check-code-alignment`) — injects context for source-file edits.
- [x] PostToolUse hook (`bin/validate-archcore`) — validates after MCP document mutations.
- [x] PostToolUse hook (`bin/check-cascade`) — detects cascade staleness after `update_document`.
- [x] PostToolUse hook (`bin/check-precision`) — precision warnings after create/update.
- [x] All hooks idempotent, PreToolUse within 1 s, PostToolUse within 3 s.
- [x] No PostToolUse `Write|Edit` validate-archcore entry; anti-regression test guards against re-introduction.

### Phase 5: Multi-Host Support — DONE

Extended the plugin to Cursor and Codex CLI:

- [x] Cursor adapter layer (`.cursor-plugin/`, `hooks/cursor.hooks.json`, `rules/`, `docs/cursor.mcp.example.json`).
- [x] Codex CLI adapter layer (`.codex-plugin/plugin.json`, `.agents/plugins/marketplace.json`, `hooks/codex.hooks.json`, `commands/*.md` slash command wrappers, `.codex.mcp.json`, TOML subagent variants).
- [x] Stdin normalization library (`bin/lib/normalize-stdin.sh`) — cross-host detection for Claude Code, Cursor, Copilot, Codex.
- [x] Plugin-shipped MCP for Claude Code (`.mcp.json`) and Codex CLI (`.codex.mcp.json`) — both name `archcore` on PATH.
- [x] Cursor MCP setup documented as a one-time copy of `docs/cursor.mcp.example.json`.

#### Phase 5a: Bundled CLI Launcher — Reverted

Shipped briefly under `bundled-cli-launcher.adr` (download-on-first-use shell/PowerShell launcher), then removed entirely in plugin v0.4.0 (2026-05-12). Eight bug classes (offline failures, version coupling, cache pollution, enterprise friction, etc.) made the "zero-setup install" framing a net loss. The plugin no longer bundles, downloads, or caches the CLI; users install it globally via the official installer at https://docs.archcore.ai/cli/install/. See `remove-bundled-launcher-global-cli.idea` for the rollback decision.

### Phase 6: Zero-Content Onboarding — DONE

Seeded first-session experience for repos with empty `.archcore/`:

- [x] SessionStart empty-state helper (`bin/lib/empty-state.sh`) — 200-byte `.md` body floor detection.
- [x] SessionStart advisory hook — emits `/archcore:init` nudge on missing or functionally-empty `.archcore/`; suppressible via `ARCHCORE_HIDE_EMPTY_NUDGE=1`.
- [x] `/archcore:init` intent skill — three confirmable steps: stack rule, run-the-app guide, opt-in agent-instruction-file import. Originally shipped as `/archcore:bootstrap`; renamed to `init` in Phase 9.
- [x] Skill support libraries under `skills/init/lib/`.
- [x] Tag + body source convention for imported documents (`imported` + `source:<slug>` tags; body first line `> Imported from <path> on <date>.`).
- [x] Idempotent re-runs via `list_documents(tags=['imported'])` lookup.
- [x] CLI availability pre-flight uses the official installer (https://docs.archcore.ai/cli/install/).

### Phase 7: Type Skill Removal — DONE

Collapsed the per-document-type skill layer:

- [x] RFC elicitation absorbed into `/archcore:decide`.
- [x] CPAT elicitation absorbed into the standard cascade (later folded into `/archcore:decide` continuations in Phase 9).
- [x] 17 type-skill directories deleted.
- [x] Count invariants updated across README, tests, and `.archcore/` docs.
- [x] Decision recorded in `remove-document-type-skills.adr.md`.

### Phase 8: Inspection Skill Consolidation — DONE

Merged secondary inspection intents into a single skill:

- [x] `/archcore:status` absorbed into `/archcore:review` (default short mode).
- [x] `/archcore:review --deep` runs the full audit.
- [x] `/archcore:graph` removed entirely (near-zero usage).
- [x] Decision recorded in `merge-review-status-remove-graph.adr.md`.

### Phase 9: Skill Surface Collapse — DONE

Final consolidation to a 7-skill surface (per `skill-surface-collapse.adr.md`):

- [x] `bootstrap` renamed to `init` (and `/archcore:bootstrap` → `/archcore:init`).
- [x] `review` and `actualize` merged into `audit`. New flag surface: `[--deep] [--drift] [filter]`. Drift-detection protocol moved to `skills/audit/lib/drift-detection.md`.
- [x] Six track skills (`product-track`, `sources-track`, `iso-track`, `architecture-track`, `standard-track`, `feature-track`) removed; their flow logic moved to `skills/plan/references/*.md` (for the four cascades) and `skills/decide/references/continuations.md` (for the ADR-driven standard and architecture chains).
- [x] `standard` skill removed; the ADR + rule + guide cascade lives inside `/archcore:decide` as continuations.
- [x] `verify` utility skill removed; `make verify` is the canonical way to run plugin integrity checks.
- [x] Codex `commands/*.md` wrappers reduced to 7 (`init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`).
- [x] Count invariants updated across README, structure tests, and `.archcore/` docs.

## Acceptance Criteria

- All 18 Archcore document types are covered through one of the 7 intent skills or direct MCP (`create_document(type=<any>)`).
- 7 intent skills operational: `init`, `capture`, `decide`, `plan`, `audit`, `context`, `help`.
- Total skills on disk: **7**. All visible in `/`; no hidden surface.
- Two agents: archcore-assistant (read/write) and archcore-auditor (read-only).
- PreToolUse hook blocks 100% of direct Write/Edit attempts on `.archcore/*.md` files.
- PostToolUse hooks report validation issues, cascade staleness, and precision warnings.
- SessionStart hook prints CLI install guidance when `archcore` is missing, loads context when present, and nudges users to `/archcore:init` on empty `.archcore/`.
- All plugin components use MCP tools exclusively — zero direct file writes.
- Plugin runs identically in Claude Code, Cursor, and Codex CLI.

## Dependencies

- **Archcore CLI installed globally on PATH** per https://docs.archcore.ai/cli/install/ — the plugin does NOT bundle, download, or cache the CLI.
- Claude Code plugin system supports: skills/, agents/, hooks/, bin/, plugin-shipped `.mcp.json`.
- Cursor plugin system supports: skills/, agents/, hooks/, rules/ (MCP registered externally).
- Codex CLI plugin system supports: skills/, commands/, agents/*.toml, hooks/, plugin-shipped MCP.
- MCP tools: create_document, update_document, list_documents, get_document, add_relation, remove_relation, list_relations, remove_document, search_documents, init_project.
- Key ADRs: Always Use MCP Tools, Plugin Component Architecture, Single Universal Agent (+ Add Read-Only Auditor Agent), Intent-Based Skill Architecture, Inverted Invocation Policy, Remove Document Type Skills, Merge `/archcore:status` into `/archcore:review`, **Skill Surface Collapse**, Actualize System, Multi-Host Plugin Architecture.
- Key superseded ADR: Bundled CLI Launcher (`bundled-cli-launcher.adr`, rejected — see `remove-bundled-launcher-global-cli.idea`).
