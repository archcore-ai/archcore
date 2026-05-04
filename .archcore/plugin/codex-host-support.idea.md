---
title: "Codex CLI Host Support — Promote from P2 Future to Implemented"
status: draft
tags:
  - "architecture"
  - "codex"
  - "multi-host"
  - "plugin"
---

## Idea

Promote OpenAI Codex CLI from "P2 / Future / TBD" (as listed in the Multi-Host Compatibility Layer spec, Supported Hosts table) to a first-class implemented host with Codex-native packaging: plugin-shipped MCP, hooks config, skills, subagent TOML files, and marketplace install.

Codex CLI v0.117.0+ (March 2026) introduced a plugin system with near 1:1 surface to Claude Code:

- `.codex-plugin/plugin.json` manifest with component pointers (`skills`, `mcpServers`, `apps`, `hooks`)
- 6 hook events (SessionStart, PreToolUse, PermissionRequest, PostToolUse, UserPromptSubmit, Stop) — same names, same JSON shapes (snake_case), exit-code-2 blocking, `hookSpecificOutput.additionalContext` for context injection; runtime execution is gated by Codex's `codex_hooks` feature and current plugin-local hook discovery behavior
- MCP servers via `[mcp_servers.<name>]` in config or plugin-shipped `.mcp.json`
- Skills as `skills/<name>/SKILL.md` directories with `name`+`description` frontmatter — **already compatible with our SKILL.md files**
- Subagents in TOML format with `sandbox_mode` ("read-only" | "workspace-write"), `developer_instructions`, `mcp_servers`, `[[skills.config]]`
- Marketplace install: `codex plugin marketplace add archcore-ai/plugin` (GitHub shorthand)

Unlike Cursor (which lacks `${CLAUDE_PLUGIN_ROOT}`-equivalent path substitution for plugin-provided MCP), Codex supports plugin-relative paths from the manifest, which means **Codex gets MCP parity with Claude Code** without depending on a `${CODEX_PLUGIN_ROOT}` env var. The plugin ships a Codex-specific plugin-root `.codex.mcp.json` whose command is `./bin/archcore`.

## Value

**Audience reach.** Codex CLI is the third major AI coding host after Claude Code and Cursor. Adding it captures users who currently cannot install Archcore.

**Architectural ROI.** The Multi-Host Plugin Architecture ADR was designed exactly for this: shared core (skills, agents, bin/, launcher) + per-host adapter layer (manifest, hooks, MCP wiring). Codex reuses 100% of shared core. Per-host adapter cost is ~3 small files (manifest, hooks config, agent TOML conversion).

**Stronger than Cursor port.** Codex provides plugin-shipped MCP (parity with Claude Code) — Cursor does not. So Codex users get plugin-managed MCP without an external `codex mcp add` step.

**Validates the multi-host investment.** Phases 1–5 of the multi-host implementation plan paid off if adding the third host costs ~5 dev-days vs. weeks. Codex is the first real test of "low per-host cost" claim from the architecture ADR.

## Possible Implementation

Reuse the existing per-host adapter pattern. New components only:

1. **Manifest**: `.codex-plugin/plugin.json` — minimal name/version/description plus `interface{}` block for marketplace UI metadata; component pointers `skills: "./skills/"`, `mcpServers: "./.codex.mcp.json"`, `hooks: "./hooks/codex.hooks.json"`.

2. **Hooks**: `hooks/codex.hooks.json` — clone of `hooks/hooks.json` with plugin-relative `./bin/...` commands and `apply_patch` added to the edit matcher. Events PascalCase (same as Claude). Block via exit 2 (already supported by `archcore_hook_block`). PostToolUse `additionalContext` via `hookSpecificOutput` (already emitted by `archcore_hook_info`). Runtime execution requires Codex hooks support and `[features].codex_hooks = true`.

3. **MCP wiring**: ship plugin-root `.codex.mcp.json` using the public Codex plugin examples' `{"mcpServers": {...}}` wrapper. Do not reuse Claude's `.mcp.json`, because it contains `${CLAUDE_PLUGIN_ROOT}`.

4. **`bin/lib/normalize-stdin.sh`**: add explicit `codex` host detection branch. Codex sends snake_case `hook_event_name` like Claude, so the existing claude-code branch is a working fallback, but explicit detection (e.g., presence of `turn_id` without `conversation_id`/`hookEventName`) gives cleaner separation and future-proofing.

5. **Launcher cache**: extend `bin/archcore` resolution step 3 to check `$CODEX_PLUGIN_DATA/archcore/cli` before XDG fallback. Mirror in `bin/archcore.ps1` for Windows (`$env:CODEX_PLUGIN_DATA`).

6. **Subagents**: convert `agents/archcore-auditor.md` (YAML frontmatter MD) to `agents/archcore-auditor.toml` with `sandbox_mode = "read-only"`. Keep the original `.md` for Claude Code/Cursor (they read the MD format). If Codex doesn't pick up plugin-bundled subagents, ship an `install-codex-agents.sh` helper or document the manual install path.

7. **Marketplace**: `codex plugin marketplace add archcore-ai/plugin` should work with the existing GitHub repo. The Codex marketplace descriptor lives at `.agents/plugins/marketplace.json`; do not add legacy `.codex-plugin/marketplace.json`.

8. **Docs**: README install section for Codex CLI; promote the Multi-Host Compatibility Layer spec's "Codex CLI" row from TBD to actual values.

## Risks and Constraints

**Codex plugin-local hooks.** The official docs describe plugin-bundled lifecycle config, but hooks are behind `[features].codex_hooks = true` and upstream runtime behavior has been in flux. Mitigation: ship the documented hook config and keep end-to-end hook execution as a smoke-test requirement rather than assuming it from static packaging.

**Plugin-bundled subagents not confirmed.** Codex docs describe subagents in `~/.codex/agents/` (user) and `.codex/agents/` (project), but don't explicitly state plugins can ship `agents/*.toml`. If unsupported, auditor degrades to manual install.

**Auditor MCP whitelist coarsening.** Current `archcore-auditor.md` whitelists 8 specific MCP read tools via `tools: [...]`. Codex's `sandbox_mode = "read-only"` blocks file writes but does NOT filter MCP tools. To prevent auditor from calling mutating MCP tools (`create_document`, `update_document`, etc.), need either: (a) `disabled_tools[]` per subagent if Codex supports it, (b) `developer_instructions` enforcement (soft), or (c) a separate read-only MCP server invocation (e.g., `bin/archcore mcp --read-only`).

**Skill namespacing.** Codex skill invocation via `@` — unclear if `@archcore/decide` or flat `@decide`. Flat namespace risks collisions with other plugins' similarly-named skills. Spike should confirm.

**`.mcp.json` schema divergence.** Resolved for current Codex examples: use `{"mcpServers": {...}}` in a Codex-specific plugin-root `.codex.mcp.json`. Existing `.mcp.json` remains Claude-only because it relies on `${CLAUDE_PLUGIN_ROOT}`.

**Cursor ADR side-effect.** `bundled-cli-launcher.adr.md` notes Cursor cannot use plugin-shipped MCP because of missing path substitution. If Codex CAN use it, the ADR's "Multi-host divergence risk" Negative consequence shifts: Cursor remains the outlier, Codex joins Claude Code in zero-setup install.

**Codex versioning.** Plugin system is recent (v0.117.0, March 2026). API may evolve; users on older Codex versions will hit incompatibility. Document minimum Codex version in README and check at session-start where possible.
