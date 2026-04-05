---
title: "Plugin Component Registry"
status: draft
tags:
  - "plugin"
  - "reference"
---

## Overview

Reference document listing all components of the Archcore Claude Plugin.

Note: Claude Code has merged commands into skills. All slash commands use `skills/<name>/SKILL.md`. The `commands/` directory is legacy and not used.

## Content

### Skills — Document Types (18, model + user invoked)

Each serves dual purpose: model-invoked knowledge when Claude auto-activates, and user-invoked quick-create via `/archcore:<type> <topic>`.

| Skill | Type | Category |
|-------|------|----------|
| `skills/adr/` | Architecture Decision Record | knowledge |
| `skills/rfc/` | Request for Comments | knowledge |
| `skills/rule/` | Team Standard | knowledge |
| `skills/guide/` | How-To Instructions | knowledge |
| `skills/doc/` | Reference Material | knowledge |
| `skills/spec/` | Technical Specification | knowledge |
| `skills/prd/` | Product Requirements | vision |
| `skills/idea/` | Product/Technical Concept | vision |
| `skills/plan/` | Implementation Plan | vision |
| `skills/mrd/` | Market Requirements | vision |
| `skills/brd/` | Business Requirements | vision |
| `skills/urd/` | User Requirements | vision |
| `skills/brs/` | Business Requirements Spec | vision |
| `skills/strs/` | Stakeholder Requirements Spec | vision |
| `skills/syrs/` | System Requirements Spec | vision |
| `skills/srs/` | Software Requirements Spec | vision |
| `skills/task-type/` | Recurring Task Pattern | experience |
| `skills/cpat/` | Code Pattern Change | experience |

### Skills — Tracks (3, user-only, disable-model-invocation: true)

Each orchestrates a complete requirements flow, creating multiple documents in sequence with proper relations.

| Skill | Command | Flow |
|-------|---------|------|
| `skills/product-track/` | `/archcore:product-track` | idea → prd → plan |
| `skills/sources-track/` | `/archcore:sources-track` | mrd → brd → urd |
| `skills/iso-track/` | `/archcore:iso-track` | brs → strs → syrs → srs |

### Skills — Workflows (3, user-only, disable-model-invocation: true)

| Skill | Command | Purpose |
|-------|---------|---------| 
| `skills/create/` | `/archcore:create` | Interactive creation wizard |
| `skills/review/` | `/archcore:review` | Documentation health review |
| `skills/status/` | `/archcore:status` | Documentation dashboard |

### Agents (2)

| Agent | File | Role | Model | Tools |
|-------|------|------|-------|-------|
| `archcore-assistant` | `agents/archcore-assistant.md` | Read/write documentation agent | sonnet | All 8 MCP + Read/Grep/Glob |
| `archcore-auditor` | `agents/archcore-auditor.md` | Read-only documentation auditor | sonnet | 3 read MCP + Read/Grep/Glob |

**archcore-assistant** — complex multi-document tasks: creation, requirements engineering, relation management. Foreground, blue, max 20 turns.

**archcore-auditor** — documentation health checks: coverage gaps, orphaned docs, stale statuses. Background, yellow, max 15 turns.

### Hooks (3)

| Event | Matcher | Handler |
|-------|---------|---------| 
| SessionStart | (all) | `archcore hooks claude-code session-start` |
| PreToolUse | `Write\|Edit` | `bin/check-archcore-write` |
| PostToolUse | `Write\|Edit` | `bin/validate-archcore` |

### Bin Scripts (2)

| Script | Purpose |
|--------|---------| 
| `bin/check-archcore-write` | PreToolUse: block direct .archcore/ writes |
| `bin/validate-archcore` | PostToolUse: run archcore validate |

### MCP Server

| Server | Command |
|--------|---------|
| `archcore` | `archcore mcp` |

## Examples

### All skills available as slash commands

```
/archcore:create          — interactive wizard
/archcore:review          — documentation health check
/archcore:status          — dashboard
/archcore:product-track   — full Product Track flow
/archcore:sources-track   — full Sources Track flow
/archcore:iso-track       — full ISO 29148 Track flow
/archcore:adr <topic>     — quick ADR creation
/archcore:prd <topic>     — quick PRD creation
/archcore:rule <topic>    — quick rule creation
... (all 18 types available)
```
