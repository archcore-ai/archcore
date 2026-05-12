---
title: "Codex MCP and Hooks Path Resolution"
status: rejected
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Status: Rejected (Superseded)

This decision described a complex workaround for the bundled launcher architecture. With the global CLI approach, path resolution is trivial.

**New architecture (as of 2026-05-12):**
- `.codex.mcp.json`: `{ "command": "archcore", "args": ["mcp"] }` — no relative paths, no `cwd`, no `env_vars`
- Codex resolves `archcore` on PATH like any standard CLI tool
- No Step 0 launcher logic, no ARCHCORE_CWD shell wrapper, no guard

See: `remove-bundled-launcher-global-cli.idea.md` for context.

---

## Original Decision (Historical, Superseded)

[Original content preserved below for reference only]

### Context

Codex 0.130.0 resolves paths in plugin MCP and hooks configs differently from Claude Code. We hit three ENOENT-or-equivalent failures porting the plugin to Codex:

1. **MCP command resolution** — Codex does **not** substitute `${CODEX_PLUGIN_ROOT}` or `${CLAUDE_PLUGIN_ROOT}` in `command`/`args`. The only plugin-aware rewrite is in `core-plugins/src/loader.rs::normalize_plugin_mcp_server_value`...

[Full original content removed for brevity — see git history if needed]

---

## Why Rejected

- **Launcher removed.** The bundled `bin/archcore` and all its resolution logic have been deleted.
- **No more relative-path workarounds.** The global CLI on PATH works with standard MCP mechanisms.
- **Simplified hook resolution.** Hooks directly call `archcore` on PATH (via `bin/session-start`, `bin/validate-archcore`, etc.).
- **Tests removed.** Tests for Step 0 chdir, Step 0b guard, `ARCHCORE_CWD` passthrough were deleted with `test/unit/launcher.bats`.

The global CLI approach completely eliminates the path resolution complexity that required this decision. Codex now treats archcore like any other CLI-based MCP server.

## Legacy Context: Why This Was Complex

Codex's MCP spawn pipeline called `.env_clear()` and didn't substitute plugin-root env vars, making it hard to:
- Point MCP to a relative `./bin/archcore` in the plugin directory
- Pass through a custom `ARCHCORE_CWD` variable to override cwd inside the launcher

This decision provided a workaround using a shell launcher Step 0 that checked for `ARCHCORE_CWD` and a Step 0b guard that refused to operate from the plugin cache without a shell wrapper. The complexity was necessary _given the launcher architecture_. With the launcher removed, all of this is moot.
