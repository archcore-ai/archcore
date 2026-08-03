---
title: "Codex MCP CWD — Opt-In ARCHCORE_CWD Via Shell Wrapper"
status: rejected
tags:
  - "codex"
  - "multi-host"
  - "plugin"
---

## Idea

**Rejected and superseded by `remove-bundled-launcher-global-cli.idea`.** The proposal was an opt-in `ARCHCORE_CWD` environment variable, passed through the `env_vars` allowlist of `.codex.mcp.json` and consumed by the bundled `bin/archcore` launcher at its Step 0, which would `chdir` into the user's project before exec'ing the CLI. It required a user-side shell wrapper — `function codex; env ARCHCORE_CWD=$PWD command codex $argv; end` — because Codex spawned a plugin MCP from the plugin cache directory rather than from the user's project.

## Value

The value it would have delivered — an MCP server that reads the user's `.archcore/` rather than the plugin cache — is now delivered without any of the machinery. `.codex.mcp.json` is `{ "command": "archcore", "args": ["mcp"] }`, carrying no `cwd`, no `env_vars`, and needing no shell wrapper, so Codex resolves `archcore` from PATH like any other CLI-based MCP server and inherits the host process's working directory.

## Possible Implementation

None. The launcher this idea extended was removed in plugin v0.4.0, so there is no shell entry point left to host a Step 0. Git history holds the chdir and `env_vars` passthrough design if it is ever needed.

## Risks

- [assumption] Should a future Codex release reintroduce a plugin-cache working directory for MCP children, the problem returns without the launcher that this idea assumed. The countermeasure then would be `archcore mcp --project <path>`, which `cursor-mcp-architecture.adr` added for exactly this class of host behavior, rather than an environment variable and a user-side wrapper.
