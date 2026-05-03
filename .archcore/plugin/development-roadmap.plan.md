---
title: "Plugin Development Roadmap"
status: accepted
tags:
  - "plugin"
  - "roadmap"
---

## Goal

Deliver the complete Archcore Claude Plugin feature set, transforming the current thin MCP+hook wrapper into a rich, guided Archcore experience in Claude Code.

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
- [x] Each skill follows the structure defined in skills-system.spec.md (Intent: 5 sections, Track: sequential steps)
- [x] All skills reference MCP tools by exact name, never instruct direct file writes
- [x] Tier prefix applied: "Advanced —" for tracks. Intent and utility use clean descriptions.
- [x] Inverted Invocation Policy applied (intent + track auto-invocable; utility user-only). Type-skill portion of that policy is now moot — type skills have been removed.

### Phase 3: Commands and Agents — DONE

Built user-invoked command surface and subagents:

- [x] 9 intent skills as primary user entry points (`/archcore:bootstrap`, `/archcore:capture`, `/archcore:plan`, `/archcore:decide`, `/archcore:standard`, `/archcore:review`, `/archcore:actualize`, `/archcore:help`, `/archcore:context`)
- [x] 6 track skills for advanced multi-document flows (`/archcore:product-track`, etc.)
- [x] 1 utility skill (`/archcore:verify`) for plugin developers
- [x] Every Archcore document type reachable via intent/track skill or direct MCP (`create_document(type=<any>)`)
- [x] `archcore-assistant` agent — read/write agent with full MCP tool access
- [x] `archcore-auditor` agent — read-only auditor with code-document correlation

### Phase 4: Hooks and Validation — DONE

Built the enforcement and freshness detection layer (4 active entries in `hooks/hooks.json` for Claude Code; 3 events in `hooks/cursor.hooks.json` for Cursor):

- [x] SessionStart hook (`bin/session-start`) — CLI check, project check, context loading, staleness check via `bin/check-staleness`
- [x] PreToolUse hook (`bin/check-archcore-write`) — blocks direct `.archcore/**/*.md` writes, redirects to MCP tools
- [x] PostToolUse hook (`bin/validate-archcore`) — validates after MCP document mutations (create_document, update_document, remove_document, add_relation, remove_relation)
- [x] PostToolUse hook (`bin/check-cascade`) — detects cascade staleness after `update_document` via relation graph
- [x] All hooks idempotent, PreToolUse within 1s, PostToolUse within 3s
- [x] No PostToolUse `Write|Edit` validate-archcore entry: PreToolUse already blocks `.archcore/*.md` writes (PostToolUse only fires on success), so a Write/Edit PostToolUse entry would fork a shell on every write anywhere in the repo for no benefit. Anti-regression test guards against re-introduction.

### Phase 5: Multi-Host Support and Bundled Launcher — DONE

Extended to Cursor and shipped a bundled CLI launcher:

- [x] Cursor adapter layer (`.cursor-plugin/`, `hooks/cursor.hooks.json`, `rules/`)
- [x] Stdin normalization library (`bin/lib/normalize-stdin.sh`) for cross-host bin scripts
- [x] Cross-platform CLI launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION`)
- [x] Plugin-shipped `.mcp.json` for Claude Code wired to the launcher
- [x] Cursor users still register MCP externally (no host-side path substitution available yet)

### Phase 6: Zero-Content Onboarding — DONE

Seeded first-session experience for repos with empty `.archcore/`:

- [x] SessionStart empty-state helper (`bin/lib/empty-state.sh`) — 200-byte `.md` body floor detection
- [x] SessionStart advisory hook (`bin/session-start`) — emits `/archcore:bootstrap` nudge on missing or functionally-empty `.archcore/`; suppressible via `ARCHCORE_HIDE_EMPTY_NUDGE=1`
- [x] `/archcore:bootstrap` intent skill — three independently-confirmable steps: stack rule, run-the-app guide, opt-in agent-instruction-file import (link / extract / skip per file with cost warning)
- [x] Skill support libraries: `skills/bootstrap/lib/detect-stack.md`, `lib/extract-run-instructions.md`, `lib/agent-files.md`, `lib/extract-routing.md`
- [x] Tag + body source convention for imported documents (`imported` + `source:<slug>` tags; body first line `> Imported from <path> on <date>.`) — survives the CLI's frontmatter strip
- [x] Idempotent re-runs via `list_documents(tags=['imported'])` lookup, per-source-slug skip semantics
- [x] Documentation sync: README (skill counts + bootstrap discovery line + Intent-commands table), PRD (FR-5 empty-state nudge + FR-6 bootstrap skill), multi-host spec (`ARCHCORE_HIDE_EMPTY_NUDGE` env var), this roadmap

### Phase 7: Type Skill Removal — DONE

Collapsed the per-document-type skill layer:

- [x] RFC elicitation absorbed into `/archcore:decide` (open-proposal branch alongside the finalized-decision ADR branch)
- [x] CPAT elicitation absorbed into `/archcore:standard-track` (optional step between ADR and rule)
- [x] 17 type-skill directories deleted (10 mainstream + 7 niche)
- [x] Count invariants updated in README, bats structure test, skills-system spec, plugin-architecture spec, component-registry, commands-system spec, skill-file-structure rule
- [x] Obsolete lifecycle docs removed (`adding-document-type-skill.guide.md`, `creating-skill-batch.task-type.md`, `keep-document-type-skills.adr.md`)
- [x] Reversal recorded in `remove-document-type-skills.adr.md`
- [x] Visible `/` palette at end of phase: 18 commands (11 intent + 6 track + 1 utility)

### Phase 8: Inspection Skill Consolidation — DONE

Merged the secondary inspection intents into a single skill, removed the unused graph visualizer:

- [x] `/archcore:status` (compact dashboard) absorbed into `/archcore:review` as the default short mode — counts by category/status/type, relation totals, one-line issues summary
- [x] `/archcore:review --deep` (or any non-flag filter argument) runs the full audit — coverage gaps, staleness, orphans, prioritized recommendations
- [x] `/archcore:graph` removed entirely — observed near-zero usage; orphan list and relation counts are already in the short dashboard
- [x] `skills/status/` and `skills/graph/` directories deleted
- [x] Sibling-skill anti-trigger lines updated in `skills/actualize/`, `skills/bootstrap/`, `skills/context/` (and `skills/help/` Quick Start)
- [x] Count invariants updated: README (18 → 16, 11 intent → 9 intent), `test/structure/skills.bats` (`>= 18` → `>= 16`), and every `.archcore/plugin/` document referencing skill counts (skills-system spec, commands-system spec, plugin-architecture spec, component-registry doc, claude-plugin PRD, plugin-component-architecture ADR, inverted-invocation-policy ADR, precision-over-coverage ADR)
- [x] Decision recorded in `merge-review-status-remove-graph.adr.md`
- [x] Visible `/` palette: **16 commands** (9 intent + 6 track + 1 utility)

## Acceptance Criteria

- All 17 Archcore document types are covered through intent/track skills or direct MCP (`create_document(type=<any>)`)
- 9 intent skills operational as primary user surface (bootstrap, capture, context, plan, decide, standard, review, actualize, help)
- 6 track skills for multi-document flows
- 1 utility skill (verify)
- Total skills on disk: 16. All visible in `/`; no hidden surface.
- Two agents: archcore-assistant (read/write) and archcore-auditor (read-only)
- PreToolUse hook blocks 100% of direct Write/Edit attempts on `.archcore/*.md` files
- PostToolUse hooks report validation issues and cascade staleness
- SessionStart hook loads context, detects code-document drift, and nudges users to `/archcore:bootstrap` on empty `.archcore/`
- All plugin components use MCP tools exclusively — zero direct file writes
- Plugin runs identically in Claude Code and Cursor

## Dependencies

- Archcore CLI installed and in PATH OR auto-resolved by the bundled launcher
- Claude Code plugin system supports: skills/, agents/, hooks/, bin/
- Cursor plugin system supports: skills/, agents/, hooks/, rules/
- MCP tools available: create_document, update_document, list_documents, get_document, add_relation, remove_relation, list_relations, remove_document, search_documents, init_project
- ADR: Always Use MCP Tools (architectural constraint)
- ADR: Plugin Component Architecture (component mapping)
- ADR: Single Universal Agent → extended by Add Read-Only Auditor Agent
- ADR: Intent-Based Skill Architecture (layer hierarchy; Layer 3 later collapsed)
- ADR: Inverted Invocation Policy (per-class invocation flags; type-skill portion superseded)
- ADR: Remove Document Type Skills (type-skill layer removal)
- ADR: Merge `/archcore:status` into `/archcore:review` and Remove `/archcore:graph` (inspection-skill consolidation)
- ADR: Actualize System (freshness detection)
- ADR: Multi-Host Plugin Architecture (shared core + per-host adapters)
- ADR: Bundled CLI Launcher (auto-resolve CLI; plugin-owned MCP for Claude Code)
