---
title: "Hermes Agent Runs the Archcore Guards Through Shell Hooks That Call archcore hooks hermes, Not a Python Plugin"
status: accepted
tags:
  - "architecture"
  - "component:cli"
  - "component:plugin"
  - "hermes"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Context

Hermes Agent (`NousResearch/hermes-agent` commit `cad0aaa`, release `v2026.9.24`) runs shell hooks declared in the `hooks:` block of the profile's `~/.hermes/config.yaml` (`website/docs/user-guide/features/hooks.md`; `agent/shell_hooks.py`). A shell hook receives one JSON object on stdin with `hook_event_name`, `tool_name`, `tool_input`, `session_id`, `cwd`, and `extra`. On `pre_tool_call`, exit 2 blocks the call and the first 400 characters of stderr reach the model as the tool error (`shell_hooks.py`; `model_tools.py`). Only `pre_llm_call` can add context; the return values of `on_session_start` and `post_tool_call` are ignored. The payload `cwd` is the Hermes process directory, which in gateway-hosted sessions is not the session's project (`hermes_cli/plugin_session.py`). Each `(event, command)` pair needs a one-time consent stored in `~/.hermes/shell-hooks-allowlist.json`; non-interactive hosts never prompt.

The Archcore guard and validation run in the CLI as `archcore hooks <host> <event>`. `stack-and-tooling.rule` item 3 forbids a `python3` shebang for plugin code; item 24 requires an accepted ADR before a new language. The maintainer chose the shell-hook route on 2026-11-04.

## Decision

The CLI wires Hermes through shell hooks that call `archcore hooks hermes <leaf>`: `pre_tool_call` runs the guard, and `pre_llm_call` delivers the session recap on the first turn. A new `hermes` row in `@cli/cmd/hook_dialect.go` uses exit 2 with the reason on stderr and a flat `{"context": ...}` envelope. `archcore init --agent hermes` writes the hook entries, the `mcp_servers.archcore` entry, and the matching consent entries into the user's Hermes profile. The adapter contains no Python, and Archcore registers no `post_tool_call` hook, because Hermes discards its output.

## Alternatives Considered

1. A Python plugin for the Hermes plugin catalog — rejected because it adds Python to the plugin stack (`stack-and-tooling.rule` items 3 and 24), and the guard logic already lives in the CLI.
2. Skills and MCP only, with no hooks — rejected because Hermes then gets no write guard although `pre_tool_call` can block.

## Consequences

- Positive: guard logic keeps one owner, the CLI.
- Positive: no new language in the plugin tree.
- Tradeoff: hooks live in the user profile, so they run for every project that profile opens; the leaf allows silently outside a project with `.archcore/`.
- Tradeoff: the payload `cwd` cannot locate the project in gateway sessions, so the `hermes` guard denies a write whose path has a `.archcore` segment without resolving a root.
- Tradeoff: Hermes gets no post-write validation text, because no shell-hook event returns context after a tool call.
- Tradeoff: writing consent entries on the user's behalf is part of the install; `archcore init` names them in its output.

## Superseded when

- Hermes passes the session working directory in the shell-hook payload.
- Hermes lets `post_tool_call` or `transform_tool_result` return context from a shell hook.
- Hermes removes shell hooks in favor of Python plugins.
