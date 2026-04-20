---
title: "Plugin Component Registry"
status: accepted
tags:
  - "plugin"
  - "reference"
---

## Overview

Reference document listing all components of the Archcore Plugin (multi-host: Claude Code, Cursor).

Note: Claude Code has merged commands into skills. All slash commands use `skills/<name>/SKILL.md`. The `commands/` directory is legacy and not used.

## Content

### Skills — Intent (8, user-only, disable-model-invocation: true)

Intent skills translate user intent into the correct document types, tracks, or analysis modes. They are the primary user entry points (Layer 1).

| Skill     | Directory           | User Intent                                                |
| --------- | ------------------- | ---------------------------------------------------------- |
| capture   | `skills/capture/`   | Document a module/component → routes to adr/spec/doc/guide |
| plan      | `skills/plan/`      | Plan a feature → routes to product-track or single plan    |
| decide    | `skills/decide/`    | Record a decision → creates adr, offers rule+guide         |
| standard  | `skills/standard/`  | Establish a standard → routes to standard-track            |
| review    | `skills/review/`    | Check documentation health → analysis + recommendations    |
| status    | `skills/status/`    | Show dashboard → counts, relations, issues                 |
| actualize | `skills/actualize/` | Detect stale docs → code drift, cascade, temporal analysis |
| help      | `skills/help/`      | Navigate the system → layer guide, onboarding              |

### Skills — Tracks (6, user-only, disable-model-invocation: true)

Track skills orchestrate complete multi-document flows, creating documents in sequence with proper relations. Descriptions prefixed "Advanced —" (Layer 2).

| Skill              | Directory                    | Flow                          |
| ------------------ | ---------------------------- | ----------------------------- |
| product-track      | `skills/product-track/`      | idea → prd → plan             |
| sources-track      | `skills/sources-track/`      | mrd → brd → urd               |
| iso-track          | `skills/iso-track/`          | brs → strs → syrs → srs       |
| architecture-track | `skills/architecture-track/` | adr → spec → plan             |
| standard-track     | `skills/standard-track/`     | adr → rule → guide            |
| feature-track      | `skills/feature-track/`      | prd → spec → plan → task-type |

### Skills — Document Types (18, model + user invoked)

Each teaches Claude about one document type. Model-invoked (auto-activate) and user-invokable via `/archcore:<type> <topic>`. Non-high-frequency types prefixed "Expert —" (Layer 3).

| Skill               | Type                          | Category   |
| ------------------- | ----------------------------- | ---------- |
| `skills/adr/`       | Architecture Decision Record  | knowledge  |
| `skills/rfc/`       | Request for Comments          | knowledge  |
| `skills/rule/`      | Team Standard                 | knowledge  |
| `skills/guide/`     | How-To Instructions           | knowledge  |
| `skills/doc/`       | Reference Material            | knowledge  |
| `skills/spec/`      | Technical Specification       | knowledge  |
| `skills/prd/`       | Product Requirements          | vision     |
| `skills/idea/`      | Product/Technical Concept     | vision     |
| `skills/plan/`      | Implementation Plan           | vision     |
| `skills/mrd/`       | Market Requirements           | vision     |
| `skills/brd/`       | Business Requirements         | vision     |
| `skills/urd/`       | User Requirements             | vision     |
| `skills/brs/`       | Business Requirements Spec    | vision     |
| `skills/strs/`      | Stakeholder Requirements Spec | vision     |
| `skills/syrs/`      | System Requirements Spec      | vision     |
| `skills/srs/`       | Software Requirements Spec    | vision     |
| `skills/task-type/` | Recurring Task Pattern        | experience |
| `skills/cpat/`      | Code Pattern Change           | experience |

### Skills — Utility (1, user-only, disable-model-invocation: true)

| Skill  | Directory        | Purpose                                                                        |
| ------ | ---------------- | ------------------------------------------------------------------------------ |
| verify | `skills/verify/` | Run plugin integrity checks — tests, lint, config validation, cross-references |

### Agents (2)

| Agent                | File                           | Role                            | Model  | Tools                       |
| -------------------- | ------------------------------ | ------------------------------- | ------ | --------------------------- |
| `archcore-assistant` | `agents/archcore-assistant.md` | Read/write documentation agent  | sonnet | All 8 MCP + Read/Grep/Glob  |
| `archcore-auditor`   | `agents/archcore-auditor.md`   | Read-only documentation auditor | sonnet | 3 read MCP + Read/Grep/Glob |

**archcore-assistant** — complex multi-document tasks: creation, requirements engineering, relation management. Foreground, blue, max 20 turns.

**archcore-auditor** — documentation health checks: coverage gaps, orphaned docs, stale statuses, code-document correlation (cross-references document path mentions with git history to flag drift). Background, yellow, max 15 turns.

### Hooks (5 entries across 3 events)

| #   | Event        | Matcher                                                                                           | Handler                    | Timeout |
| --- | ------------ | ------------------------------------------------------------------------------------------------- | -------------------------- | ------- |
| 1   | SessionStart | (all)                                                                                             | `bin/session-start`        | —       |
| 2   | PreToolUse   | `Write\|Edit`                                                                                     | `bin/check-archcore-write` | 1s      |
| 3   | PostToolUse  | `Write\|Edit`                                                                                     | `bin/validate-archcore`    | 3s      |
| 4   | PostToolUse  | `mcp__archcore__create_document\|update_document\|remove_document\|add_relation\|remove_relation` | `bin/validate-archcore`    | 3s      |
| 5   | PostToolUse  | `mcp__archcore__update_document`                                                                  | `bin/check-cascade`        | 3s      |

Hook configs: `hooks/hooks.json` (Claude Code, PascalCase events), `hooks/cursor.hooks.json` (Cursor, camelCase events + `afterMCPExecution`).

### Bin Scripts (6: 5 hooks + 1 library)

| Script                       | Hook Event                                   | Purpose                                                                                                                                                                                                                                                                                                                                        |
| ---------------------------- | -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `bin/lib/normalize-stdin.sh` | (library)                                    | Multi-host stdin normalization. Detects host (Claude Code/Cursor/Copilot), extracts fields (tool_name, file_path, path), normalizes MCP tool names, provides output helpers (archcore_hook_block, archcore_hook_info, archcore_hook_allow). Sourced by all hook scripts except check-staleness.                                                |
| `bin/session-start`          | SessionStart                                 | Checks CLI availability and project init, delegates context loading via `archcore hooks <host> session-start`, then calls `bin/check-staleness` for drift detection. Emits structured `hookSpecificOutput.additionalContext` with install/init instructions if prerequisites missing (plain text on non-Claude/Copilot hosts). Always exits 0. |
| `bin/check-archcore-write`   | PreToolUse                                   | Blocks direct Write/Edit to `.archcore/**/*.md` with exit 2 + stderr message redirecting to MCP tools. Allows `.archcore/settings.json` and `.archcore/.sync-state.json`. Allows all paths outside `.archcore/`.                                                                                                                               |
| `bin/validate-archcore`      | PostToolUse                                  | Runs `archcore validate` after `.archcore/` file changes (Write/Edit by path check) or MCP document operations (by tool_name prefix). Outputs JSON `hookSpecificOutput` when issues found, empty otherwise. Silently exits 0 if CLI is missing. Always exits 0.                                                                                |
| `bin/check-staleness`        | SessionStart (called by `bin/session-start`) | Detects code-document drift via git: finds source files changed since the last `.archcore/` commit, cross-references with documents that mention affected directories. Outputs plain text warning (max 2KB) or empty. Always exits 0.                                                                                                          |
| `bin/check-cascade`          | PostToolUse                                  | After `update_document`, queries `.sync-state.json` relation graph for documents connected via `implements`, `depends_on`, or `extends` to the updated document. Outputs JSON `hookSpecificOutput` listing potentially stale dependents, or empty if no cascade. Always exits 0.                                                               |

### Test Suite

| Component       | Location                     | Tests    | Description                                                                                             |
| --------------- | ---------------------------- | -------- | ------------------------------------------------------------------------------------------------------- |
| Unit tests      | `test/unit/`                 | 69       | Test each bin script: stdin parsing, host detection, exit codes, output format, edge cases              |
| Structure tests | `test/structure/`            | 45       | Validate JSON configs, skill frontmatter, agent frontmatter, hook references, script permissions, rules |
| Fixtures        | `test/fixtures/stdin/`       | 12 files | Mock stdin JSON for Claude Code, Cursor, Copilot, and malformed inputs                                  |
| Helpers         | `test/helpers/`              | —        | common.bash (setup, mocks, timeout shim), bats-support, bats-assert (git submodules)                    |
| Makefile        | `Makefile`                   | —        | Targets: `test`, `test-unit`, `test-structure`, `lint`, `check-json`, `check-perms`, `verify`           |
| CI              | `.github/workflows/test.yml` | —        | GitHub Actions: macOS + Linux matrix, bats + shellcheck                                                 |

Run `make verify` for full check. Run `make test` for tests only. See `plugin-testing.guide.md` for details.

### MCP Server

The plugin does **not** ship an MCP server configuration. MCP tools (`mcp__archcore__*`) are provided by the externally-installed Archcore CLI (`archcore mcp`).

Users register the MCP server via one of:

| Mechanism                                                                                                    | Scope                 | Use when                                        |
| ------------------------------------------------------------------------------------------------------------ | --------------------- | ----------------------------------------------- |
| `claude mcp add archcore archcore mcp -s user`                                                               | User-global           | Standalone usage, no repo `.mcp.json`           |
| Project-root `.mcp.json` with `{ "mcpServers": { "archcore": { "command": "archcore", "args": ["mcp"] } } }` | Project (team-shared) | Repo used by multiple AI agents or team members |

Rationale: shipping MCP in the plugin triggers Claude Code's duplicate-suppression warning when a repo already declares `archcore` in `.mcp.json` or a user has registered it globally. Decoupling keeps the plugin focused on skills/agents/hooks; MCP lifecycle is owned by the CLI install.

### Plugin Configs

| File                              | Host        | Purpose                                                                |
| --------------------------------- | ----------- | ---------------------------------------------------------------------- |
| `.claude-plugin/plugin.json`      | Claude Code | Plugin manifest                                                        |
| `.cursor-plugin/plugin.json`      | Cursor      | Plugin manifest (with explicit component paths; no `mcpServers` field) |
| `.claude-plugin/marketplace.json` | Claude Code | Marketplace metadata                                                   |
| `.cursor-plugin/marketplace.json` | Cursor      | Marketplace metadata                                                   |
| `hooks/hooks.json`                | Claude Code | Hook event config (PascalCase)                                         |
| `hooks/cursor.hooks.json`         | Cursor      | Hook event config (camelCase + afterMCPExecution)                      |
| `rules/archcore-context.mdc`      | Cursor      | Always-apply context rule                                              |
| `rules/archcore-files.mdc`        | Cursor      | .archcore/ glob-triggered MCP-only rule                                |

## Examples

### All skills available as slash commands

```
## Primary (intent skills)
/archcore:capture          — document a module or component
/archcore:plan             — plan a feature end-to-end
/archcore:decide           — record a decision
/archcore:standard         — establish a team standard
/archcore:review           — documentation health check
/archcore:status           — dashboard
/archcore:actualize        — detect stale docs, suggest updates
/archcore:help             — system guide

## Utility
/archcore:verify           — run plugin integrity checks

## Advanced (track skills)
/archcore:product-track    — idea → prd → plan
/archcore:sources-track    — mrd → brd → urd
/archcore:iso-track        — brs → strs → syrs → srs
/archcore:architecture-track — adr → spec → plan
/archcore:standard-track   — adr → rule → guide
/archcore:feature-track    — prd → spec → plan → task-type

## Expert (type skills)
/archcore:adr <topic>      — quick ADR creation
/archcore:prd <topic>      — quick PRD creation
/archcore:rule <topic>     — quick rule creation
... (all 18 types available)
```
