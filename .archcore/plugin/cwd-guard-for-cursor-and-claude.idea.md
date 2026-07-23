---
title: "Cross-Host CWD Sanity Guard for Cursor and Claude Code MCP"
status: rejected
tags:
  - "claude-code"
  - "cursor"
  - "multi-host"
  - "plugin"
---

## Status: Rejected (Superseded by `remove-bundled-launcher-global-cli.idea`)

This idea proposed extending the bundled `bin/archcore` launcher with a cross-host sanity check (Step 0c) that refused to start the MCP server when CWD did not look like a user project root, plus shipping a `cursor.mcp.json` template requiring `cwd: "${workspaceFolder}"`.

With the launcher removed in plugin v0.4.0, there is no shell entry point to host a guard. CWD handling now relies entirely on each host's native `cwd` field (Cursor: `${workspaceFolder}`; Claude Code and Codex CLI: spawn cwd of the host process). The `cursor.mcp.json` template at the plugin root still uses `cwd: "${workspaceFolder}"`, but as a standard MCP convention rather than as defensive armor against a CWD-rebase bug.

Original idea body removed — git history holds the Step 0/0b/0c design discussion if needed.
