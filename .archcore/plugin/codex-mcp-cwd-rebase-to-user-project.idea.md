---
title: "Codex MCP CWD — Opt-In ARCHCORE_CWD Via Shell Wrapper"
status: rejected
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Status: Rejected (Superseded by `remove-bundled-launcher-global-cli.idea`)

This idea proposed an opt-in `ARCHCORE_CWD` environment variable passed through `.codex.mcp.json`'s `env_vars` allowlist, consumed by the bundled `bin/archcore` launcher (Step 0) to chdir into the user's project before exec'ing the CLI. It required a user-side shell wrapper (`function codex; env ARCHCORE_CWD=$PWD command codex $argv; end`) because Codex spawned plugin MCPs from the plugin cache directory, not the user's project.

With the launcher removed in plugin v0.4.0, none of this applies. `.codex.mcp.json` is now `{ "command": "archcore", "args": ["mcp"] }` — no `cwd`, no `env_vars`, no shell wrapper. Codex resolves `archcore` from PATH like any other CLI-based MCP server.

Original idea body removed — git history holds the Step 0 chdir + env_vars passthrough design if needed.
