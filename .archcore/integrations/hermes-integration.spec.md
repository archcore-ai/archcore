---
title: "Hermes Integration — hermes Hook Dialect, Root Resolution, and archcore init Wiring of the Hermes Profile"
status: draft
tags:
  - "component:cli"
  - "hermes"
  - "hooks"
  - "integrations"
  - "multi-host"
---

## Purpose & Scope

This spec is normative for the Hermes Agent integration in the CLI: the `hermes` hook dialect, how its leaves find the project, and what `archcore init --agent hermes` writes into the Hermes profile. Depended on by Hermes sessions that open a project with `.archcore/`, by the Go suites that pin `@plugin/test/fixtures/stdin/hermes/`, and by `supported-ai-agents.doc`. Out of scope: guard and validation policy (`hook-runtime.spec`), the Hermes skills tap, and the plugin catalog.

Host facts come from `NousResearch/hermes-agent` commit `cad0aaa` (release `v2026.9.24`): `agent/shell_hooks.py`, `model_tools.py`, `tools/file_tools.py`, `tools/mcp_tool_schema.py`, `website/docs/user-guide/features/{hooks,mcp,skills,context-files}.md`.

## Surface

- Profile file: `$HERMES_HOME/config.yaml`, default `~/.hermes/config.yaml`; consent file `~/.hermes/shell-hooks-allowlist.json` with `{"approvals":[{"event","command"}]}`.
- Hook entries written under `hooks:`: `pre_tool_call` with matcher `write_file|patch` and command `archcore hooks hermes pre-tool-use`; `pre_llm_call` with command `archcore hooks hermes session-start`; `timeout: 10` each (seconds, cap 300).
- MCP entry written under `mcp_servers:`: `archcore` with `command: archcore` and `args: [mcp]`. Hermes starts it in the session working directory when one is pinned.
- Hook stdin: `hook_event_name`, `tool_name`, `tool_input`, `session_id`, `cwd`, `extra`; `extra.is_first_turn` on `pre_llm_call`. MCP tools arrive as `mcp__archcore__<tool>`.
- Write tools: `write_file` (`path`), `patch` (`path`, or `patch` holding V4A markers `*** Add File:`, `*** Update File:`, `*** Delete File:`, `*** Move File:`).
- Dialect row `hermes` in `@cli/cmd/hook_dialect.go`: deny `denyExitTwo`; session and tool envelope a flat `{"context": <text>}`; `preToolContext: false`.
- Agent `@cli/internal/agents/hermes.go`: detection marker `.hermes/` directory or `HERMES.md`; instructions in `HERMES.md` when it exists, else `AGENTS.md`; hook writer in `@cli/internal/wiring/hooks_agents.go`.

## Normative Behavior

1. WHEN the `hermes` pre-tool leaf reads an absolute target path, the CLI MUST resolve the project from that path, not from payload `cwd`.
2. WHEN a target path is relative and holds a `.archcore` segment, the `hermes` pre-tool leaf MUST deny with the redirect-to-MCP reason.
3. WHEN a V4A patch names a file under a `.archcore` segment, the `hermes` pre-tool leaf MUST deny with the same reason.
4. WHEN the guard denies, the `hermes` leaf MUST write the reason to stderr and exit 2.
5. WHEN the `hermes` session leaf runs and `extra.is_first_turn` is not true, the leaf MUST print nothing.
6. WHEN the `hermes` session leaf runs on the first turn, the leaf MUST print the recap as `{"context": <text>}`.
7. WHEN the user runs `archcore init --agent hermes`, the CLI MUST merge the hook entries of Surface into the profile file.
8. WHEN the user runs `archcore init --agent hermes`, the CLI MUST merge the `mcp_servers.archcore` entry into the profile file.
9. WHEN the CLI writes a hook entry, the CLI MUST add the matching `{event, command}` approval to the consent file.
10. WHEN the CLI writes the profile or consent file, the CLI MUST print each file path it changed.
11. IF an `archcore` MCP entry already exists in the profile, THEN the CLI MUST leave it unchanged.
12. The CLI MUST preserve every profile key it does not own.
13. The CLI MUST NOT register a `post_tool_call` hook.
14. WHEN `HERMES.md` or `.hermes.md` exists in the project root, the CLI MUST write the managed block there instead of `AGENTS.md`.

## Constraints & Invariants

- Constraint: payload `cwd` is the Hermes process directory, which in gateway sessions differs from the session project (`hermes_cli/plugin_session.py`); behaviors 1–3 never use it.
- Constraint: `on_session_start` and `post_tool_call` outputs are discarded; `pre_llm_call` context is appended to the turn's user message.
- Constraint: hooks live in the user profile only; one install covers every project the profile opens.
- Constraint: non-interactive hosts (gateway, `hermes acp`) never prompt for consent; without behavior 9 the hooks do not run there.
- Constraint: Hermes loads one instruction file type, first match: `HERMES.md`, then `AGENTS.override.md`, then `AGENTS.md`.
- Constraint: a shell write through the `terminal` tool is outside the matcher and unguarded.
- Invariant: a second `archcore init --agent hermes` run leaves both files byte-identical.

## Failure Behavior

1. IF the profile file is not valid YAML, THEN the CLI MUST back it up and stop with an error naming the backup path.
2. IF the profile file is absent, THEN the CLI MUST create it with only the Archcore entries.
3. IF a hook timeout or spawn error occurs, THEN Hermes fails open; the CLI MUST document this in `supported-ai-agents.doc`.
4. IF stdin is not valid JSON, THEN the `hermes` leaf MUST print nothing and exit 0.

## Conformance

An implementation conforms when it satisfies behaviors 1–14, holds the invariants, degrades per Failure Behavior, and passes: Go table tests for the `hermes` dialect and root resolution in `@cli/cmd/` with fixtures under `@plugin/test/fixtures/stdin/hermes/`; YAML round-trip tests for behaviors 7–12 in `@cli/internal/agents/` or `@cli/internal/wiring/`; and the `hermes` row of `host-probe-protocol.spec` with `pass` on P0, A, and B.

Given a gateway session on `/repo` with `.archcore/`, when the agent calls `write_file` on `/repo/.archcore/x.rule.md`, then the hook exits 2 and the model receives the reason.
