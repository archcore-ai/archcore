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

Promote OpenAI Codex CLI from "P2 / Future / TBD" to a first-class implemented host with Codex-native packaging: plugin-shipped MCP, hooks config, skills, slash command wrappers, and read-only auditor TOML.

Codex CLI v0.117.0+ (March 2026) introduced a plugin system with near 1:1 surface to Claude Code:

- `.codex-plugin/plugin.json` manifest with component pointers (`skills`, `mcpServers`, `hooks`)
- 6 hook events (SessionStart, PreToolUse, PermissionRequest, PostToolUse, UserPromptSubmit, Stop) — runtime execution gated by `[features].codex_hooks = true`
- MCP servers via plugin-shipped `.codex.mcp.json` referenced from the manifest (`mcpServers: "./.codex.mcp.json"`)
- Skills as `skills/<name>/SKILL.md` directories — **already compatible with our SKILL.md files**
- Subagents in TOML format with `sandbox_mode`, `developer_instructions`, `disabled_tools[]`
- Marketplace install via `.agents/plugins/marketplace.json`

## Value

**Audience reach.** Codex CLI is the third major AI coding host. Adding it captures users who otherwise cannot install Archcore.

**Architectural ROI.** The Multi-Host Plugin Architecture ADR was designed for exactly this: shared core (skills, agents, bin/) + per-host adapter layer (manifest, hooks, MCP). Codex reuses 100% of shared core; per-host adapter cost was ~5 small config files plus 16 thin `commands/*.md` slash command wrappers.

**Validates the multi-host investment.** Phases 1–5 of `multi-host-implementation.plan` paid off if adding the third host costs ~5 dev-days vs. weeks. Codex was the first real test of "low per-host cost" — port shipped in roughly that envelope.

## Outcome

Shipped. Current Codex packaging is documented in:

- `codex-host-support.prd` — functional requirements (F1–F10)
- `codex-host-support.plan` — implementation phases
- `codex-local-plugin-testing.guide` — current contract for `.codex.mcp.json`, `.codex-plugin/plugin.json`, `hooks/codex.hooks.json`, `commands/*.md`, marketplace
- `component-registry.doc` — current per-host config table

**CLI lifecycle note.** Codex MCP wiring went through two iterations: first via the bundled launcher (`./bin/archcore` with `cwd: "."` + `env_vars: ["ARCHCORE_CWD"]`), then simplified to `command: "archcore"` resolved via PATH when the launcher was removed in v0.4.0 (see `remove-bundled-launcher-global-cli.idea`). The current shape is the simpler PATH resolution — no `$CODEX_PLUGIN_DATA` cache extension, no plugin-relative paths, no shell wrapper.
