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

Promote OpenAI Codex CLI from "P2 / Future / TBD" (as listed in the Multi-Host Compatibility Layer spec, Supported Hosts table) to a first-class implemented host with Codex-native packaging: plugin-shipped MCP, hooks config, skills, subagent TOML files, and marketplace install.

Codex CLI v0.117.0+ (March 2026) introduced a plugin system with near 1:1 surface to Claude Code:

- `.codex-plugin/plugin.json` manifest with component pointers (`skills`, `mcpServers`, `apps`, `hooks`)
- 6 hook events (SessionStart, PreToolUse, PermissionRequest, PostToolUse, UserPromptSubmit, Stop) — same names, same JSON shapes (snake_case), exit-code-2 blocking, `hookSpecificOutput.additionalContext` for context injection; runtime execution is gated by Codex's `plugin_hooks` feature (currently `under development, false`)
- MCP servers via `[mcp_servers.<name>]` in config or plugin-shipped `.mcp.json`
- Skills as `skills/<name>/SKILL.md` directories with `name`+`description` frontmatter — **already compatible with our SKILL.md files**
- Subagents in TOML format with `sandbox_mode` ("read-only" | "workspace-write"), `developer_instructions`, `mcp_servers`, `[[skills.config]]`
- Marketplace install: `codex plugin marketplace add archcore-ai/plugin` (GitHub shorthand)

Codex's resolution mechanisms differ between MCP and hooks (canonical record: `plugin/codex-plugin-spawn-semantics.adr.md`). For MCP, Codex does NOT substitute env-var placeholders in `command`/`args` — the resolution happens through a `cwd` rebase: `normalize_plugin_mcp_server_value` rebases a relative `cwd` field against the plugin install root. The plugin's `.codex.mcp.json` therefore sets `command: "./bin/archcore"` paired with `cwd: "."`. For hooks, Codex's hooks engine injects `PLUGIN_ROOT` (canonical, host-neutral) plus `CLAUDE_PLUGIN_ROOT` (a backward-compat alias for porting old Claude plugins) and folds `${KEY}` substitution at spawn time — so `${PLUGIN_ROOT}/bin/...` is the right form for `hooks/codex.hooks.json`. Note: `CODEX_PLUGIN_ROOT` does not exist in Codex.

This gives **Codex MCP parity with Claude Code** without requiring a `${CODEX_PLUGIN_ROOT}` env var. Cursor remains the multi-host outlier because it lacks an equivalent path-substitution or cwd-rebase mechanism for plugin-provided MCP.

## Value

**Audience reach.** Codex CLI is the third major AI coding host after Claude Code and Cursor. Adding it captures users who currently cannot install Archcore.

**Architectural ROI.** The Multi-Host Plugin Architecture ADR was designed exactly for this: shared core (skills, agents, bin/, launcher) + per-host adapter layer (manifest, hooks, MCP wiring). Codex reuses 100% of shared core. Per-host adapter cost is ~3 small files (manifest, hooks config, agent TOML conversion).

**Stronger than Cursor port.** Codex provides plugin-shipped MCP (parity with Claude Code) — Cursor does not. So Codex users get plugin-managed MCP without an external `codex mcp add` step.

**Validates the multi-host investment.** Phases 1–5 of the multi-host implementation plan paid off if adding the third host costs ~5 dev-days vs. weeks. Codex is the first real test of "low per-host cost" claim from the architecture ADR.

## Possible Implementation

Reuse the existing per-host adapter pattern. New components only:

1. **Manifest**: `.codex-plugin/plugin.json` — minimal name/version/description plus `interface{}` block for marketplace UI metadata; component pointers `skills: "./skills/"`, `mcpServers: "./.codex.mcp.json"`, `hooks: "./hooks/codex.hooks.json"`.

2. **Hooks**: `hooks/codex.hooks.json` — clone of `hooks/hooks.json` with `${PLUGIN_ROOT}/bin/...` substitution (Codex's canonical, host-neutral env var; not the `${CLAUDE_PLUGIN_ROOT}` compat alias) and `apply_patch` added to the edit matcher. Events PascalCase (same as Claude). Block via exit 2 (already supported by `archcore_hook_block`). PostToolUse `additionalContext` via `hookSpecificOutput` (already emitted by `archcore_hook_info`). Runtime execution requires `codex features enable plugin_hooks` (the `plugin_hooks` flag is `under development, false` by default in Codex 0.130.0).

3. **MCP wiring**: ship plugin-root `.codex.mcp.json` with `command: "./bin/archcore"`, `args: ["mcp"]`, and `cwd: "."`. The `cwd: "."` is essential — Codex's `normalize_plugin_mcp_server_value` rebases the relative cwd to the plugin install root so the relative command resolves correctly. Without `cwd`, the spawn happens from the user's project CWD and fails with ENOENT. Do NOT reuse Claude's `.mcp.json`, because it contains `${CLAUDE_PLUGIN_ROOT}` which Codex does not substitute.

4. **`bin/lib/normalize-stdin.sh`**: add explicit `codex` host detection branch. Codex sends snake_case `hook_event_name` like Claude, so the existing claude-code branch is a working fallback, but explicit detection (e.g., presence of `turn_id` without `conversation_id`/`hookEventName`) gives cleaner separation and future-proofing.

5. **Launcher cache**: extend `bin/archcore` resolution step 3 to check `$CODEX_PLUGIN_DATA/archcore/cli` before XDG fallback. Mirror in `bin/archcore.ps1` for Windows (`$env:CODEX_PLUGIN_DATA`).

6. **Subagents**: convert `agents/archcore-auditor.md` (YAML frontmatter MD) to `agents/archcore-auditor.toml` with `sandbox_mode = "read-only"`. Keep the original `.md` for Claude Code/Cursor (they read the MD format). If Codex doesn't pick up plugin-bundled subagents, ship an `install-codex-agents.sh` helper or document the manual install path.

7. **Marketplace**: `codex plugin marketplace add archcore-ai/plugin` should work with the existing GitHub repo. The Codex marketplace descriptor lives at `.agents/plugins/marketplace.json`; do not add legacy `.codex-plugin/marketplace.json`.

8. **Docs**: README install section for Codex CLI; promote the Multi-Host Compatibility Layer spec's "Codex CLI" row from TBD to actual values.

## Risks and Constraints

**Codex plugin-local hooks.** The official docs describe plugin-bundled lifecycle config, but hooks are behind the `plugin_hooks` feature (`under development, false` by default in Codex 0.130.0) and upstream runtime behavior has been in flux. Mitigation: ship the documented hook config and keep end-to-end hook execution as a smoke-test requirement rather than assuming it from static packaging.

**Plugin-bundled subagents not confirmed.** Codex docs describe subagents in `~/.codex/agents/` (user) and `.codex/agents/` (project), but don't explicitly state plugins can ship `agents/*.toml`. If unsupported, auditor degrades to manual install.

**Auditor MCP whitelist coarsening.** Current `archcore-auditor.md` whitelists 8 specific MCP read tools via `tools: [...]`. Codex's `sandbox_mode = "read-only"` blocks file writes but does NOT filter MCP tools. To prevent auditor from calling mutating MCP tools (`create_document`, `update_document`, etc.), need either: (a) `disabled_tools[]` per subagent if Codex supports it, (b) `developer_instructions` enforcement (soft), or (c) a separate read-only MCP server invocation (e.g., `bin/archcore mcp --read-only`).

**Skill namespacing.** Codex skill invocation via `@` — unclear if `@archcore/decide` or flat `@decide`. Flat namespace risks collisions with other plugins' similarly-named skills. Spike should confirm.

**`.mcp.json` schema divergence.** Resolved for current Codex examples: use `{"mcpServers": {...}}` in a Codex-specific plugin-root `.codex.mcp.json` with `cwd: "."` companion. Existing `.mcp.json` remains Claude-only because it relies on `${CLAUDE_PLUGIN_ROOT}`, which Codex does not substitute.

**Cursor ADR side-effect.** `bundled-cli-launcher.adr.md` notes Cursor cannot use plugin-shipped MCP because of missing path substitution. Codex CAN — via the `cwd` rebase mechanism — so the ADR's "Multi-host divergence risk" Negative consequence shifts: Cursor remains the outlier, Codex joins Claude Code in zero-setup install.

**Codex versioning.** Plugin system is recent (v0.117.0, March 2026). API may evolve; users on older Codex versions will hit incompatibility. Document minimum Codex version in README and check at session-start where possible. The `plugin_hooks` feature flag in particular is unstable — track its promotion to `stable` upstream.
