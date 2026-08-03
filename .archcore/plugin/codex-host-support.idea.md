---
title: "Codex CLI Host Support — Promote from P2 Future to Implemented"
status: accepted
tags:
  - "architecture"
  - "codex"
  - "multi-host"
  - "plugin"
---

## Idea

Promote OpenAI Codex CLI from "P2 / future / TBD" to a first-class implemented host with Codex-native packaging: plugin-shipped MCP, a hooks config, skills, slash-command wrappers, and a read-only auditor in TOML.

Codex CLI v0.117.0, released in March 2026, introduced a plugin system with a near one-to-one surface to Claude Code: a `.codex-plugin/plugin.json` manifest with `skills`, `mcpServers`, and `hooks` pointers; six hook events — SessionStart, PreToolUse, PermissionRequest, PostToolUse, UserPromptSubmit, and Stop — whose runtime execution uses `[features].hooks`, with the older `codex_hooks` key surviving only as a deprecated alias, and whose plugin hooks require user trust; MCP servers through a plugin-shipped `.codex.mcp.json` referenced from the manifest; skills as `skills/<name>/SKILL.md` directories, already compatible with the plugin's existing files; subagents in TOML with `sandbox_mode`, `developer_instructions`, and `disabled_tools[]`; and marketplace install through `.agents/plugins/marketplace.json`.

## Value

**Audience reach.** Codex CLI was the third major AI coding host, so adding it captured users who otherwise could not install Archcore.

**Architectural return.** `multi-host-plugin-architecture.adr` was designed for exactly this shape — a shared core of skills, agents, and `bin/` plus a per-host adapter of manifest, hooks, and MCP. Codex reused the entire shared core, and the adapter cost was about five small config files plus the thin `commands/*.md` wrappers.

**Validation of the multi-host investment.** The earlier phases of `multi-host-implementation.plan` paid off only if a third host cost about 5 developer-days rather than weeks. Codex was the first real test of the low-per-host-cost claim, and the port shipped inside that envelope.

## Possible Implementation

Shipped. The current packaging is documented in `codex-host-support.prd` for the functional requirements, `codex-host-support.plan` for the implementation phases, `codex-local-plugin-testing.guide` for the current contract covering `.codex.mcp.json`, `.codex-plugin/plugin.json`, `hooks/codex.hooks.json`, the command wrappers, and the marketplace, and `component-registry.doc` for the per-host config table.

Two facts have moved since this idea was written. The wrapper count was 16 at the time and is 7 now, after `skill-surface-collapse.adr` collapsed the palette. And the MCP wiring went through two iterations: first the bundled launcher, with `./bin/archcore`, `cwd: "."`, and `env_vars: ["ARCHCORE_CWD"]`, then the simpler `command: "archcore"` resolved from PATH once the launcher was removed in v0.4.0. The current shape needs no `$CODEX_PLUGIN_DATA` cache extension, no plugin-relative path, and no shell wrapper.

## Risks

- Codex plugin hooks require per-hook user trust, so a hook can be present and silently inactive. `codex-local-plugin-testing.guide` documents `codex features enable plugin_hooks` as the diagnosis step.
- [assumption] The near one-to-one surface with Claude Code could diverge in a future Codex release, which would raise the per-host adapter cost that this idea used to justify the port.
