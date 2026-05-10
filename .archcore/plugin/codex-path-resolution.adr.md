---
title: "Codex MCP and Hooks Path Resolution"
status: accepted
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Context

Codex 0.130.0 resolves paths in plugin MCP and hooks configs differently from Claude Code. We hit two ENOENT failures porting the plugin to Codex:

1. **MCP servers** — Codex spawns plugin MCP servers from the **user's project CWD**, not the plugin install dir. It does **not** substitute `${CODEX_PLUGIN_ROOT}` or `${CLAUDE_PLUGIN_ROOT}` in `command`/`args`. The only plugin-aware rewrite is in `core-plugins/src/loader.rs::normalize_plugin_mcp_server_value`, which rebases a relative `cwd` field against the plugin install root.

2. **Hooks** — Codex's hooks engine (`codex-rs/hooks/src/engine/discovery.rs`) injects two env vars before spawn: a canonical host-neutral `PLUGIN_ROOT` and a `CLAUDE_PLUGIN_ROOT` compat shim for porting old Claude plugins. It does **not** treat `./...` as plugin-relative.

A previous config used `./bin/archcore` for MCP and `./bin/...` for hooks. Both broke under Codex.

See: <https://github.com/openai/codex/issues/19582>.

## Decision

- `.codex.mcp.json`: keep `command: "./bin/archcore"` and add `cwd: "."`. Codex's normalizer rebases the relative `cwd` against the plugin install root, which then resolves the relative command correctly.
- `hooks/codex.hooks.json`: use `${PLUGIN_ROOT}/bin/...` for every command. We use the canonical host-neutral name, not the `CLAUDE_PLUGIN_ROOT` compat alias.
- `.mcp.json` (Claude) stays unchanged (`${CLAUDE_PLUGIN_ROOT}/bin/archcore`); `hooks/hooks.json` stays unchanged (`${CLAUDE_PLUGIN_ROOT}/bin/...`).

Contract is enforced by `test/structure/codex-plugin.bats` (cwd required, no env vars in MCP; hooks must use `${PLUGIN_ROOT}`) and `test/structure/hooks.bats` (path resolver expands `${PLUGIN_ROOT}`).

## Alternatives Considered

- **Use `${CODEX_PLUGIN_ROOT}` in MCP `command`** — rejected. Codex does no env substitution in `command`/`args` for MCP servers. Would silently fail.
- **Use `${CLAUDE_PLUGIN_ROOT}` in Codex hooks** — rejected. It works (compat shim exists) but borrowing another host's name in a Codex-native config is misleading; `PLUGIN_ROOT` is the canonical name.
- **Absolute paths** — rejected. Plugin install path is host-controlled and not stable.
- **Drop `cwd` and rely on env substitution in MCP** — rejected. There is no env substitution for MCP. `cwd` is the only knob Codex exposes.

## Consequences

- Codex and Claude diverge in config style: Claude uses `${CLAUDE_PLUGIN_ROOT}`-substituted commands; Codex uses `cwd`-rebased relative paths for MCP and `${PLUGIN_ROOT}` substitution for hooks. Each host's quirks are isolated to its own config file.
- Tests now hard-pin both contracts. Adding a new bin script requires updating both hook configs and stays cross-checked by `hooks.bats` ("codex hook config references the same script set as Claude Code").
- Codex plugin hooks require `codex features enable plugin_hooks` (currently `under development, false` in 0.130.0). The contract is in place ahead of GA.
