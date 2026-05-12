---
title: "Plugin Development Roadmap"
status: accepted
tags:
  - "plugin"
  - "roadmap"
---

## Goal

Deliver the complete Archcore Claude Plugin feature set, transforming the current thin MCP+hook wrapper into a rich, guided Archcore experience across Claude Code, Cursor, and Codex CLI.

## Tasks

### Phase 1: Documentation — DONE

Created comprehensive project documentation using Archcore's own document types (dogfooding):

- [x] PRD defining the plugin vision, problem, goals, and requirements
- [x] ADRs for core architectural decisions (MCP-only, component architecture, universal agent)
- [x] Development roadmap (this document)
- [x] Component specifications (skills, commands, agent, hooks, plugin architecture)
- [x] Development standards (rules) and how-to guides
- [x] Component registry (reference document)

### Phase 2: Skills — DONE (post-type-skill-removal, post-status/graph-merge)

Built skills across the current 3-group hierarchy (intent, track, utility). Historical evolution: a Layer 3 of 17 per-type skills existed between the initial build and the type-skill removal decision (`remove-document-type-skills.adr.md`); the `status` and `graph` intents existed until the inspection-skill consolidation (`merge-review-status-remove-graph.adr.md`). Per-type elicitation now lives inline in intent and track skills; documentation health (counts and audit) lives in `/archcore:review`.

- [x] Intent skills (9): bootstrap, capture, plan, decide, standard, review, actualize, help, context
- [x] Track skills (6): product-track, sources-track, iso-track, architecture-track, standard-track, feature-track
- [x] Utility skill (1): verify
- [x] Each skill follows the structure defined in skills-system.spec.md
- [x] All skills reference MCP tools by exact name, never instruct direct file writes
- [x] Tier prefix applied: "Advanced —" for tracks; intent and utility use clean descriptions
- [x] Inverted Invocation Policy applied (intent + track auto-invocable; utility user-only)

### Phase 3: Commands and Agents — DONE

Built user-invoked command surface and subagents:

- [x] 9 intent skills as primary user entry points
- [x] 6 track skills for advanced multi-document flows
- [x] 1 utility skill (`/archcore:verify`) for plugin developers
- [x] Every Archcore document type reachable via intent/track skill or direct MCP (`create_document(type=<any>)`)
- [x] `archcore-assistant` agent — read/write agent with full MCP tool access
- [x] `archcore-auditor` agent — read-only auditor with code-document correlation

### Phase 4: Hooks and Validation — DONE

Built the enforcement and freshness detection layer (6 entries in `hooks/hooks.json` for Claude Code; analogous configs for Cursor and Codex):

- [x] SessionStart hook (`bin/session-start`) — CLI availability check, project check, context loading, staleness check via `bin/check-staleness`
- [x] PreToolUse hook (`bin/check-archcore-write`) — blocks direct `.archcore/**/*.md` writes
- [x] PreToolUse hook (`bin/check-code-alignment`) — injects context for source-file edits
- [x] PostToolUse hook (`bin/validate-archcore`) — validates after MCP document mutations
- [x] PostToolUse hook (`bin/check-cascade`) — detects cascade staleness after `update_document`
- [x] PostToolUse hook (`bin/check-precision`) — precision warnings after create/update
- [x] All hooks idempotent, PreToolUse within 1 s, PostToolUse within 3 s
- [x] No PostToolUse `Write|Edit` validate-archcore entry; anti-regression test guards against re-introduction

### Phase 5: Multi-Host Support — DONE

Extended the plugin to Cursor and Codex CLI:

- [x] Cursor adapter layer (`.cursor-plugin/`, `hooks/cursor.hooks.json`, `rules/`, `cursor.mcp.json` reference template)
- [x] Codex CLI adapter layer (`.codex-plugin/plugin.json`, `.agents/plugins/marketplace.json`, `hooks/codex.hooks.json`, `commands/*.md` slash command wrappers, `.codex.mcp.json`, TOML subagent variants)
- [x] Stdin normalization library (`bin/lib/normalize-stdin.sh`) — cross-host detection for Claude Code, Cursor, Copilot, Codex
- [x] Plugin-shipped MCP for Claude Code (`.mcp.json`) and Codex CLI (`.codex.mcp.json`) — both name `archcore` on PATH
- [x] Cursor MCP setup documented as a one-time copy of `cursor.mcp.json` (Cursor does not auto-register plugin MCP)

#### Phase 5a: Bundled CLI Launcher — Reverted

Shipped briefly under `bundled-cli-launcher.adr` (download-on-first-use shell/PowerShell launcher with platform-specific cache and `bin/CLI_VERSION` pin), then removed entirely in plugin v0.4.0 (2026-05-12, commit `2f99997`). Eight bug classes (offline failures, version coupling, cache pollution, enterprise friction, etc.) made the "zero-setup install" framing a net loss. The plugin no longer bundles, downloads, or caches the CLI; users install it globally via the official installer at https://docs.archcore.ai/cli/install/. See `remove-bundled-launcher-global-cli.idea` for the rollback decision.

### Phase 6: Zero-Content Onboarding — DONE

Seeded first-session experience for repos with empty `.archcore/`:

- [x] SessionStart empty-state helper (`bin/lib/empty-state.sh`) — 200-byte `.md` body floor detection
- [x] SessionStart advisory hook — emits `/archcore:bootstrap` nudge on missing or functionally-empty `.archcore/`; suppressible via `ARCHCORE_HIDE_EMPTY_NUDGE=1`
- [x] `/archcore:bootstrap` intent skill — three confirmable steps: stack rule, run-the-app guide, opt-in agent-instruction-file import
- [x] Skill support libraries under `skills/bootstrap/lib/`
- [x] Tag + body source convention for imported documents (`imported` + `source:<slug>` tags; body first line `> Imported from <path> on <date>.`)
- [x] Idempotent re-runs via `list_documents(tags=['imported'])` lookup
- [x] CLI availability pre-flight uses the official installer (https://docs.archcore.ai/cli/install/); brew/go-install paths are explicitly forbidden as unsupported

### Phase 7: Type Skill Removal — DONE

Collapsed the per-document-type skill layer:

- [x] RFC elicitation absorbed into `/archcore:decide`
- [x] CPAT elicitation absorbed into `/archcore:standard-track`
- [x] 17 type-skill directories deleted
- [x] Count invariants updated across README, tests, and `.archcore/` docs
- [x] Reversal recorded in `remove-document-type-skills.adr.md`

### Phase 8: Inspection Skill Consolidation — DONE

Merged secondary inspection intents into a single skill:

- [x] `/archcore:status` absorbed into `/archcore:review` (default short mode)
- [x] `/archcore:review --deep` runs the full audit
- [x] `/archcore:graph` removed entirely (near-zero usage)
- [x] Count invariants updated everywhere
- [x] Decision recorded in `merge-review-status-remove-graph.adr.md`
- [x] Visible `/` palette: **16 commands** (9 intent + 6 track + 1 utility)

## Acceptance Criteria

- All 17 Archcore document types are covered through intent/track skills or direct MCP (`create_document(type=<any>)`)
- 9 intent skills operational as primary user surface
- 6 track skills for multi-document flows
- 1 utility skill (verify)
- Total skills on disk: 16. All visible in `/`; no hidden surface.
- Two agents: archcore-assistant (read/write) and archcore-auditor (read-only)
- PreToolUse hook blocks 100% of direct Write/Edit attempts on `.archcore/*.md` files
- PostToolUse hooks report validation issues, cascade staleness, and precision warnings
- SessionStart hook prints CLI install guidance when `archcore` is missing, loads context when present, and nudges users to `/archcore:bootstrap` on empty `.archcore/`
- All plugin components use MCP tools exclusively — zero direct file writes
- Plugin runs identically in Claude Code, Cursor, and Codex CLI

## Dependencies

- **Archcore CLI installed globally on PATH** per https://docs.archcore.ai/cli/install/ — the plugin does NOT bundle, download, or cache the CLI
- Claude Code plugin system supports: skills/, agents/, hooks/, bin/, plugin-shipped `.mcp.json`
- Cursor plugin system supports: skills/, agents/, hooks/, rules/ (MCP registered externally)
- Codex CLI plugin system supports: skills/, commands/, agents/*.toml, hooks/, plugin-shipped MCP
- MCP tools: create_document, update_document, list_documents, get_document, add_relation, remove_relation, list_relations, remove_document, search_documents, init_project
- Key ADRs: Always Use MCP Tools, Plugin Component Architecture, Single Universal Agent (+ Add Read-Only Auditor Agent), Intent-Based Skill Architecture, Inverted Invocation Policy, Remove Document Type Skills, Merge `/archcore:status` into `/archcore:review`, Actualize System, Multi-Host Plugin Architecture
- Key superseded ADR: Bundled CLI Launcher (`bundled-cli-launcher.adr`, rejected — see `remove-bundled-launcher-global-cli.idea` for the global-CLI replacement)
