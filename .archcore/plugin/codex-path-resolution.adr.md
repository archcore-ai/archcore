---
title: "Codex MCP and Hooks Path Resolution"
status: rejected
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Context

Codex 0.130.0 carried three path-resolution quirks that affected plugin-shipped MCP and hooks under the bundled-launcher architecture: it performed no `${CODEX_PLUGIN_ROOT}` substitution in an MCP `command` or `args`; its `.env_clear()` stripped inherited environment variables at MCP spawn; and it used the plugin-cache working directory rather than the user-project one. The shipped workaround was a `.codex.mcp.json` carrying `command: "./bin/archcore"`, `cwd: "."`, and `env_vars: ["ARCHCORE_CWD"]`, which relied on Codex's `normalize_plugin_mcp_server_value` rebasing `"."` to the plugin install root.

## Decision

Rejected and superseded. Plugin v0.4.0 removed the bundled launcher, which removed the relative path the workaround depended on; `.codex.mcp.json` is now `{ "command": "archcore", "args": ["mcp"] }`, and Codex resolves `archcore` from the host process's PATH with no plugin-relative resolution.

## Alternatives Considered

1. **Keep the `cwd: "."` plus `env_vars` workaround after the launcher removal** — ruled out because it rebased to the plugin install root, which is precisely the directory the plugin must not serve documents from; without a plugin-relative binary to reach, the rebase bought nothing and reintroduced the install-directory failure mode.
2. **Ship a Python trampoline that reads `$PWD` and calls `chdir`** — rejected because it added a third language to a two-language plugin, which `stack-and-tooling.rule` forbids without its own ADR. The rejected companion record is `codex-mcp-cwd-rebase-to-user-project.idea`.

## Consequences

- The three Codex quirks may still exist upstream, but they no longer affect this plugin, because it ships nothing Codex must resolve relative to the plugin install root.
- Tradeoff: the plugin now depends on Codex inheriting a PATH that contains `archcore`. When it does not, MCP startup fails with `command not found`, which `codex-local-plugin-testing.guide` documents as a diagnosis step.
- The original ADR body was removed from this document. Git history holds the `normalize_plugin_mcp_server_value` analysis, the `.env_clear()` trace, and the chdir design.

## Superseded when

- Codex introduces a documented plugin-root substitution for MCP `command` and `args`, which would make a plugin-relative binary viable again.
- A measured Codex release spawns plugin MCP children in the user's project directory, which would remove the reason the `cwd` rebase was dangerous.
