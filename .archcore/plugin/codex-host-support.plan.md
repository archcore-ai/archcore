---
title: "Codex CLI Host Support Implementation Plan"
status: accepted
tags:
  - "codex"
  - "multi-host"
  - "plugin"
  - "roadmap"
---

## Goal

Implement OpenAI Codex CLI as the third first-class host for the Archcore plugin with Codex-native packaging for slash command wrappers, skills, plugin-managed MCP, hooks config, and subagent TOML files. Marketplace registration via `codex plugin marketplace add archcore-ai/plugin`, no manual `codex mcp add`, zero regression for Claude Code/Cursor.

## Outcome

Shipped. The current Codex packaging contract is documented in:

- `component-registry.doc` — per-host config table (`.codex-plugin/plugin.json`, `.codex.mcp.json`, `hooks/codex.hooks.json`, `.agents/plugins/marketplace.json`, `commands/*.md`, `agents/*.toml`)
- `codex-local-plugin-testing.guide` — end-to-end test recipe
- `codex-host-support.prd` — functional requirements F1–F10 (F6 marked obsolete after launcher removal)

The plan went through three rounds:

1. **Phase 0 spike** (1–2 days) — verified Codex plugin-relative path resolution, plugin-shipped `.mcp.json` schema, subagent TOML packaging, skill invocation namespacing, per-subagent `disabled_tools[]` enforcement, and SKILL.md frontmatter compatibility. All resolved cleanly; risks captured in the PRD.
2. **Phases 1–7 implementation** (~4–5 days) — manifest + marketplace + 16 `commands/*.md` wrappers + `hooks/codex.hooks.json` + `bin/lib/normalize-stdin.sh` `codex` branch + TOML subagent variants. Initially wired MCP via `command: "./bin/archcore"` with `cwd: "."` + `env_vars: ["ARCHCORE_CWD"]` per the (at the time live) bundled-launcher ADR.
3. **Launcher rollback** (2026-05-12, v0.4.0) — bundled launcher removed entirely; `.codex.mcp.json` simplified to `command: "archcore"` resolved via PATH; `$CODEX_PLUGIN_DATA` cache extension dropped; ARCHCORE_CWD shell wrapper requirement gone. See `remove-bundled-launcher-global-cli.idea`.

The shared-core / per-host-adapter split from `multi-host-plugin-architecture.adr` proved correct: adding Codex did not require any change to skills, agents, or hook script bodies — only a new `codex` branch in `normalize-stdin.sh` and the per-host adapter files. The launcher rollback simplified Codex packaging further (no more `cwd: "."` / `env_vars` / shell wrapper).

## Acceptance Criteria

- [x] `.codex-plugin/plugin.json` exists with synchronized metadata, valid component pointers, and `interface{}` marketplace block
- [x] Plugin-shipped MCP works in Codex (`.codex.mcp.json` with `command: "archcore"`, no external `codex mcp add` step)
- [x] `commands/<name>.md` wrappers exist for all 16 user-facing skills; parity tests in `test/structure/codex-plugin.bats` pass
- [x] `hooks/codex.hooks.json` maps the active hook functions with correct matchers and timeouts
- [x] `bin/lib/normalize-stdin.sh` has explicit `codex` host detection and field extraction
- [x] `agents/archcore-auditor.toml` and `agents/archcore-assistant.toml` exist; auditor enforced read-only via `sandbox_mode` and `disabled_tools[]`
- [x] All existing Claude Code tests pass unchanged
- [x] Cursor manual smoke test passes unchanged

## Dependencies

- Multi-Host Plugin Architecture ADR (architectural authority).
- Multi-Host Implementation Plan (predecessor).
- Hooks and Validation System Specification (hook semantics).
- Codex CLI v0.117.0+ available locally for testing.
- ~~Bundled CLI Launcher ADR~~ — rejected; replaced by `remove-bundled-launcher-global-cli.idea`. The current MCP wiring (`command: "archcore"` via PATH) eliminates the original F6 launcher cache extension entirely.
