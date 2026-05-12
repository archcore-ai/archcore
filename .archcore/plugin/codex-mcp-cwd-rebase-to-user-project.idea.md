---
title: "Codex MCP CWD — Opt-In ARCHCORE_CWD Via Shell Wrapper"
status: rejected
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Status: Rejected (Superseded)

This idea was based on the bundled CLI launcher with a custom shell-wrapper workaround. The new global CLI architecture eliminates the need for this approach.

**New architecture (as of 2026-05-12):**
- Users install the Archcore CLI globally
- Codex MCP config: `{ "command": "archcore", "args": ["mcp"] }` — no `cwd`, no `env_vars`
- No shell wrapper required
- No `ARCHCORE_CWD` environment variable
- Standard Codex MCP resolution handles everything

See: `remove-bundled-launcher-global-cli.idea.md` for the full transition.

---

## Original Idea (Historical, Superseded)

[Original content preserved below for reference only]

### Idea

Make Codex-spawned archcore MCP servers operate in the user's project directory by combining three mechanisms — entirely within the existing shell launcher, no new language dependency:

1. **Manifest passthrough.** `.codex.mcp.json` declares `env_vars: ["ARCHCORE_CWD"]`. Codex spawns MCP children with `.env_clear()` plus a fixed allowlist...

[Full original content removed for brevity — see git history if needed]

---

## Why Rejected

- **Launcher removed.** The bundled `bin/archcore` shell launcher and all its Step 0 logic for `ARCHCORE_CWD` chdir have been deleted.
- **Shell wrapper no longer needed.** The global CLI approach works with standard Codex MCP mechanisms; no user-side wrapper (`function codex; env ARCHCORE_CWD=$PWD ...`) is required.
- **Tests removed.** Tests validating `ARCHCORE_CWD` chdir and guard behavior were part of `test/unit/launcher.bats` (deleted).
- **Simpler architecture.** Codex now just resolves `archcore` on PATH like any other CLI-based MCP server.

The global CLI approach is more maintainable and eliminates custom environment variables and wrapper logic.
