---
title: "Codex MCP and Hooks Path Resolution"
status: rejected
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Status: Rejected (Superseded by `remove-bundled-launcher-global-cli.idea`)

This ADR documented Codex 0.130.0's three path-resolution quirks affecting plugin-shipped MCP/hooks under the bundled-launcher architecture: (1) no `${CODEX_PLUGIN_ROOT}` substitution in MCP `command`/`args`; (2) `.env_clear()` stripping inherited env vars at MCP spawn; (3) plugin-cache CWD instead of user-project CWD. The shipped workaround was `.codex.mcp.json` with `command: "./bin/archcore"`, `cwd: "."`, and `env_vars: ["ARCHCORE_CWD"]` — relying on Codex's `normalize_plugin_mcp_server_value` rebasing `"."` to the plugin install root.

With the bundled launcher removed in plugin v0.4.0, the relative-path workaround disappears. `.codex.mcp.json` is now `{ "command": "archcore", "args": ["mcp"] }` — Codex resolves `archcore` from PATH using the host process's PATH, with no plugin-relative resolution needed.

The three Codex quirks themselves may still exist upstream, but they no longer matter for this plugin because it no longer ships anything Codex needs to resolve relative to the plugin install root.

Original ADR body removed — git history holds the `normalize_plugin_mcp_server_value` analysis, the `.env_clear()` trace, and the chdir Step 0 design if needed.
