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

Implement OpenAI Codex CLI as the third first-class host for the Archcore plugin, with Codex-native packaging for the slash-command wrappers, skills, plugin-managed MCP, hooks config, and subagent TOML files. Marketplace registration happens through `codex plugin marketplace add archcore-ai/plugin`, no manual `codex mcp add` is required, and Claude Code and Cursor see zero regression.

## Outcome

Shipped. The current Codex packaging contract lives in `component-registry.doc` for the per-host config table, `codex-local-plugin-testing.guide` for the end-to-end test recipe, and `codex-host-support.prd` for the functional requirements, whose F6 is marked obsolete after the launcher removal.

The plan ran in three rounds.

1. **A Phase 0 spike of 1–2 days** verified Codex plugin-relative path resolution, the plugin-shipped MCP schema, subagent TOML packaging, skill invocation namespacing, per-subagent `disabled_tools[]` enforcement, and `SKILL.md` frontmatter compatibility. All resolved cleanly, and the risks were captured in the PRD.
2. **Implementation across phases 1–7, roughly 4–5 days**, delivered the manifest, the marketplace entry, the command wrappers, `hooks/codex.hooks.json`, the `codex` branch in `bin/lib/normalize-stdin.sh`, and the TOML subagent variants. MCP was initially wired through the bundled launcher with `cwd: "."` and an `env_vars` allowlist, per the ADR that was live at the time.
3. **The launcher rollback on 2026-05-12, in v0.4.0**, removed the bundled launcher entirely, simplified `.codex.mcp.json` to `command: "archcore"` resolved through PATH, dropped the `$CODEX_PLUGIN_DATA` cache extension, and removed the shell-wrapper requirement.

The shared-core and per-host-adapter split of `multi-host-plugin-architecture.adr` proved correct: adding Codex required no change to any skill, agent, or hook script body — only a new branch in `normalize-stdin.sh` and the per-host adapter files. The rollback simplified the packaging further.

## Acceptance Criteria

- [x] `.codex-plugin/plugin.json` exists with synchronized metadata, valid component pointers, and an `interface{}` marketplace block.
- [x] Plugin-shipped MCP works in Codex through `.codex.mcp.json` with `command: "archcore"`, needing no external `codex mcp add`.
- [x] A `commands/<name>.md` wrapper exists for every user-facing skill, and the parity tests in `@test/structure/codex-plugin.bats` pass.
- [x] `hooks/codex.hooks.json` maps the active hook functions with the correct matchers and timeouts.
- [x] `bin/lib/normalize-stdin.sh` carries explicit `codex` host detection and field extraction.
- [x] `agents/archcore-auditor.toml` and `agents/archcore-assistant.toml` exist, with the auditor held read-only through `sandbox_mode` and `disabled_tools[]`.
- [x] Every existing Claude Code test passes unchanged.
- [x] The Cursor manual smoke test passes unchanged.

## Dependencies

- `multi-host-plugin-architecture.adr` as the architectural authority.
- `multi-host-implementation.plan` as the predecessor.
- `hooks-validation-system.spec` for the hook semantics.
- Codex CLI v0.117.0 or later, available locally for testing.
- The bundled CLI launcher ADR was a dependency at the time and is now rejected, replaced by `remove-bundled-launcher-global-cli.idea`. The current MCP wiring resolves `archcore` through PATH, which removed the original F6 launcher cache extension entirely.
