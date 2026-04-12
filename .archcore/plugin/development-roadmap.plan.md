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

### Phase 2: Skills — DONE

Built skills across the 4-layer hierarchy:

- [x] Intent skills (8): capture, plan, decide, standard, review, status, actualize, help
- [x] Track skills (6): product-track, sources-track, iso-track, architecture-track, standard-track, feature-track
- [x] Type skills (18): adr, rfc, rule, guide, doc, spec, prd, idea, plan, mrd, brd, urd, brs, strs, syrs, srs, task-type, cpat
- [x] Each skill follows the structure defined in skills-system.spec.md (Intent: 5 sections, Track: sequential steps, Type: 3 sections)
- [x] All skills reference MCP tools by exact name, never instruct direct file writes
- [x] Tier prefixes applied: "Advanced —" for tracks, "Expert —" for non-high-frequency types

### Phase 3: Commands and Agents — DONE

Built user-invoked command surface and subagents:

- [x] 8 intent skills as primary user entry points (`/archcore:capture`, `/archcore:plan`, etc.)
- [x] 6 track skills for advanced multi-document flows (`/archcore:product-track`, etc.)
- [x] 18 type skills for expert quick-create (`/archcore:adr`, `/archcore:prd`, etc.)
- [x] `archcore-assistant` agent — read/write agent with full MCP tool access
- [x] `archcore-auditor` agent — read-only auditor with code-document correlation

### Phase 4: Hooks and Validation — DONE

Built the enforcement and freshness detection layer:

- [x] SessionStart hook (`bin/session-start`) — CLI check, project check, context loading, staleness check via `bin/check-staleness`
- [x] PreToolUse hook (`bin/check-archcore-write`) — blocks direct `.archcore/**/*.md` writes, redirects to MCP tools
- [x] PostToolUse hook (`bin/validate-archcore`) — validates after Write/Edit to `.archcore/` paths
- [x] PostToolUse hook (`bin/validate-archcore`) — validates after MCP document mutations
- [x] PostToolUse hook (`bin/check-cascade`) — detects cascade staleness after `update_document` via relation graph
- [x] All hooks idempotent, PreToolUse within 1s, PostToolUse within 3s

## Acceptance Criteria

- All 18 document types have dedicated type skills with complete guidance
- 8 intent skills operational as primary user surface
- 6 track skills for multi-document flows
- Two agents: archcore-assistant (read/write) and archcore-auditor (read-only)
- PreToolUse hook blocks 100% of direct Write/Edit attempts on `.archcore/*.md` files
- PostToolUse hooks report validation issues and cascade staleness
- SessionStart hook loads context and detects code-document drift
- All plugin components use MCP tools exclusively — zero direct file writes

## Dependencies

- Archcore CLI installed and in PATH (provides MCP server and validation)
- Claude Code plugin system supports: skills/, agents/, hooks/, bin/
- MCP tools available: create_document, update_document, list_documents, get_document, add_relation, remove_relation, list_relations, remove_document
- ADR: Always Use MCP Tools (architectural constraint)
- ADR: Plugin Component Architecture (component mapping)
- ADR: Single Universal Agent → extended by Add Read-Only Auditor Agent
- ADR: Intent-Based Skill Architecture (4-layer hierarchy)
- ADR: Actualize System (freshness detection)