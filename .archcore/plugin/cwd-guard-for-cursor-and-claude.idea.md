---
title: "Cross-Host CWD Sanity Guard for Cursor and Claude Code MCP"
status: rejected
tags:
  - "claude-code"
  - "cursor"
  - "multi-host"
  - "plugin"
---

## Idea

**Rejected and superseded by `remove-bundled-launcher-global-cli.idea`.** The proposal was to extend the bundled `bin/archcore` launcher with a cross-host sanity check, Step 0c, that refused to start the MCP server when the working directory did not look like a user project root, and to ship a `cursor.mcp.json` template requiring `cwd: "${workspaceFolder}"`.

## Value

The guard the idea wanted exists today, in a better place. Removing the launcher in plugin v0.4.0 left no shell entry point to host a Step 0c, and the protection moved into the CLI itself: `resolveProjectRoot` refuses any project root containing a plugin-install cache fragment and exits with an error naming `--project`, per `cursor-mcp-architecture.adr` and `host-wiring-parity.adr`, while `bin/session-start` applies the same rule to hooks.

The template half of the idea was also wrong on its facts. Cursor's MCP stdio schema has no `cwd` field at all, so a template requiring one would have been silently ignored. The shipped template is `docs/cursor.mcp.example.json`, which carries the workspace path in `args` as `--project ${workspaceFolder}` and sits outside the plugin root, off every host's auto-discovery list.

## Possible Implementation

None as proposed. Any future work on this problem belongs in the CLI's `resolveProjectRoot` and in `bin/session-start`, which already share one fragment list. Git history holds the Step 0, 0b, and 0c design discussion.

## Risks

- [assumption] A user project whose absolute path contains a literal plugin-cache fragment is silenced by the surviving guard. `host-wiring-parity.adr` records that tradeoff as accepted and pinned by tests.
