---
title: "Plugin Development Roadmap"
status: accepted
tags:
  - "plugin"
  - "roadmap"
---

## Goal

Deliver the complete Archcore Claude Plugin feature set across four development phases, transforming the current thin MCP+hook wrapper into a rich, guided Archcore experience in Claude Code.

## Tasks

### Phase 1: Documentation (current)

Create comprehensive project documentation using Archcore's own document types (dogfooding):

- [x] PRD defining the plugin vision, problem, goals, and requirements
- [x] ADRs for core architectural decisions (MCP-only, component architecture, universal agent)
- [x] Development roadmap (this document)
- [ ] Component specifications (skills, commands, agent, hooks)
- [ ] Development standards (rules) and how-to guides
- [ ] Component registry (reference document)

### Phase 2: Skills

Build SKILL.md files for all 18 Archcore document types:

- [ ] Knowledge types: adr, rfc, rule, guide, doc, spec (6 skills)
- [ ] Vision types: prd, idea, plan, mrd, brd, urd, brs, strs, syrs, srs (10 skills)
- [ ] Experience types: task-type, cpat (2 skills)
- [ ] Each skill follows the standard structure: Overview, When to Use, Required Sections, Best Practices, Common Mistakes, Relation Guidance, Example Workflow
- [ ] All skills reference MCP tools by exact name, never instruct direct file writes

### Phase 3: Commands and Agent

Build user-invoked commands and the universal subagent:

- [ ] `/archcore:create` — Interactive document creation wizard
- [ ] `/archcore:review` — Documentation health review (gaps, staleness, missing relations)
- [ ] `/archcore:status` — Dashboard of documents by category, status, relations
- [ ] Type shortcut commands — `/archcore:adr`, `/archcore:prd`, `/archcore:rule`, etc.
- [ ] `archcore-assistant` agent — universal subagent with full type knowledge and MCP-only tool restrictions

### Phase 4: Hooks and Validation

Build the enforcement layer:

- [ ] PreToolUse hook (Write|Edit) — block direct `.archcore/` writes, redirect to MCP tools
- [ ] PostToolUse hook (Write|Edit) — validate `.archcore/` files after changes via `archcore validate`
- [ ] Ensure hooks are idempotent and complete within 2 seconds
- [ ] Test hook behavior with edge cases (non-.archcore/ files, settings.json, .sync-state.json)

## Acceptance Criteria

- All 18 document types have dedicated skills with complete guidance
- At least 4 slash commands operational (create, review, status, + 1 type shortcut)
- Universal agent (archcore-assistant) handles complex multi-document tasks
- PreToolUse hook blocks 100% of direct Write/Edit attempts on `.archcore/*.md` files
- PostToolUse hook reports validation issues after `.archcore/` file changes
- All plugin components use MCP tools exclusively — zero direct file writes
- Plugin passes `claude plugin validate .`

## Dependencies

- Archcore CLI installed and in PATH (provides MCP server and validation)
- Claude Code plugin system supports: skills/, commands/, agents/, hooks/, bin/
- MCP tools available: create_document, update_document, list_documents, get_document, add_relation, remove_relation, list_relations, remove_document
- ADR: Always Use MCP Tools (architectural constraint)
- ADR: Plugin Component Architecture (component mapping)
- ADR: Single Universal Agent (agent design)
