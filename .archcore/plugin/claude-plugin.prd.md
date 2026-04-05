---
title: "Archcore Claude Plugin"
status: accepted
tags:
  - "plugin"
  - "vision"
---

## Vision

Make Archcore effortless in Claude Code. The plugin transforms the passive MCP+hook integration into a rich, guided experience where skills teach document types, commands accelerate workflows, a universal agent assists complex documentation tasks, and hooks enforce quality by blocking direct file writes.

Every interaction with the `.archcore/` knowledge base flows through MCP tools — ensuring validation, templates, relations, and sync manifest are always consistent.

## Problem Statement

The current Archcore Claude Plugin (v0.0.1) is a thin wrapper: it registers the MCP server and a SessionStart hook. This leaves significant gaps:

- **No guidance**: Claude doesn't know when or how to use each of the 18 document types. Users must manually instruct the agent about Archcore conventions.
- **No guardrails**: Nothing prevents the agent from writing `.archcore/` files directly via Write/Edit, bypassing validation, templates, and the sync manifest.
- **No workflows**: Common tasks (create an ADR, review documentation health, check status) require manual multi-step instructions every time.
- **No domain expertise**: Complex documentation tasks (requirements engineering, ISO 29148 cascades, multi-document planning) lack specialized assistance.

### Target Users

Anyone using Claude Code with Archcore — individual developers, team leads, architects, product managers. The plugin is tool-agnostic within Claude Code: it enhances the Archcore experience regardless of project type or team size.

## Goals and Success Metrics

### Goals

1. **Type-aware assistance**: Claude automatically applies the right document type, template, and best practices based on context
2. **Workflow acceleration**: Common documentation tasks reduced to single slash commands
3. **Quality enforcement**: Direct `.archcore/` file writes blocked at the hook level, redirected to MCP tools
4. **Expert assistance**: Universal agent handles complex multi-document tasks (requirements engineering, documentation review, relation management)

### Success Metrics

- All 18 document types have dedicated skills with guidance, best practices, and example workflows
- Slash commands cover the most common workflows (create, review, status, type shortcuts)
- PreToolUse hook intercepts 100% of direct Write/Edit attempts on `.archcore/` files
- Users never need to manually explain Archcore conventions to Claude

## Requirements

### Functional Requirements

#### FR-1: Skills (18 document type skills)

Each of the 18 Archcore document types gets a dedicated SKILL.md in `skills/<type-name>/SKILL.md`. Skills are model-invoked — Claude activates them automatically when the conversation context matches. Each skill covers: type overview, when to use, required sections, best practices, common mistakes, relation guidance, and an example MCP workflow.

#### FR-2: Slash Commands

User-invoked commands for common workflows:

- `/archcore:create` — Interactive document creation wizard (prompts for type, collects context, creates via MCP)
- `/archcore:review` — Review existing documentation for gaps, staleness, missing relations, orphaned documents
- `/archcore:status` — Dashboard showing document count by category/type/status, relation coverage, tag distribution
- Type shortcuts — `/archcore:adr`, `/archcore:prd`, `/archcore:rule`, etc. for quick single-type creation

#### FR-3: Universal Agent (archcore-assistant)

One subagent that covers all documentation scenarios:

- Full knowledge of all 18 document types and their templates
- Requirements engineering expertise (product track, sources track, ISO 29148 cascade)
- Relation pattern knowledge (implements, extends, depends_on, related)
- Tool restrictions: archcore MCP tools + Read + Grep + Glob (no Write/Edit on `.archcore/`)
- Invokable manually or automatically by Claude when complex documentation tasks arise

#### FR-4: Validation Hooks

- **PreToolUse (Write|Edit)**: If the target file matches `.archcore/**/*.md`, block the operation and return a message redirecting to the appropriate MCP tool
- **PostToolUse (Write|Edit)**: After any file write, if `.archcore/` files were affected, run `archcore validate` and report issues
- **SessionStart**: Existing hook — loads project context at session start

### Non-Functional Requirements

- **NFR-1: MCP-only operations** — All `.archcore/` document operations MUST go through MCP tools. No plugin component (skill, command, agent) should ever instruct direct file writes.
- **NFR-2: Idempotent hooks** — Hooks must be safe to run multiple times without side effects.
- **NFR-3: Performance** — Hooks must complete within 2 seconds. Skills and commands must not add perceptible latency.
- **NFR-4: Graceful degradation** — If `archcore` CLI is not installed, the plugin should inform the user and provide installation instructions rather than failing silently.
- **NFR-5: No template duplication** — Skills reference the template system; they don't embed template content that could drift from the CLI.
