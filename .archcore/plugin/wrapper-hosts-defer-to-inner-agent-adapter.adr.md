---
title: "Wrapper Editors Defer to the Inner Agent's Archcore Adapter and Contribute Only Wrapper-Level Surfaces"
status: draft
tags:
  - "architecture"
  - "component:plugin"
  - "multi-host"
  - "plugin"
---

## Context

Wrapper editors such as bb, Conductor, and Superset run Claude Code, Codex, Pi, or OpenCode as the inner agent, and the inner agent's own plugins keep loading there: a bb session on 2026-10-18 carried the Archcore Claude Code plugin's MCP tools and SessionStart hook beside bb's own skills and `bb-bridge` MCP server. bb stages every enabled plugin's `skills/` into every provider session, and its only hook, `message.dispatch`, runs before a message is sent rather than before a tool call, so a wrapper-level plugin cannot run the `.archcore/` write guard that `@plugin/plugins/archcore/bin/pre-tool-use` provides. A wrapper plugin that also registered the `archcore` MCP server, the four skills, or the managed instruction block would put a second copy of each beside the inner agent's copy in the same session.

## Decision

Every Archcore surface the agent consumes — the `archcore` MCP server, the skills under `@plugin/plugins/archcore/skills/`, the SessionStart, PreToolUse, and PostToolUse hooks, and the managed `AGENTS.md`/`CLAUDE.md` block — is delivered by the inner agent's adapter, and an Archcore plugin for a wrapper editor (first case: a bb plugin on `@get-bb/plugin-sdk`) adds only wrapper-level surfaces (UI panels, wrapper CLI commands, worktree setup) and selects MCP or skills only for a provider with no Archcore adapter, through bb's `bb.agents.configure(context)` keyed on `context.provider.id`.

## Alternatives Considered

1. A full wrapper-level adapter that registers MCP, skills, and instructions for every provider — rejected because the inner Claude Code, Codex, and OpenCode adapters already load in the same session, so each tool and skill would appear twice, and the wrapper still could not run the write guard.
2. No wrapper plugin at all, relying only on inner adapters — rejected because wrapper-only surfaces (a context panel for the open file, a `bb archcore` command, worktree readiness through `.bb-env-setup.sh`) have no inner-agent equivalent, and the bb store's "Memory & Context" category is a listing channel Archcore would leave to bb's own Memory plugin.
3. Asking users to disable the inner plugin when the wrapper plugin is active, as bb's Memory plugin does for provider-native memory — rejected because it removes the write guard and the post-write validation, which run only in the inner agent's hooks.

## Consequences

- Positive: one copy of each Archcore tool and skill per session, independent of the wrapper.
- Positive: the write guard and post-write validation keep one owner, the inner adapter, on every wrapper.
- Positive: [expected] a new wrapper costs a verification run and a recipe page unless it offers a wrapper-only surface worth a plugin.
- Tradeoff: a provider with no Archcore adapter (in bb on 2026-10-18: `pi`, `acp-cursor`) receives MCP and skills from the wrapper plugin without the write guard, and the plugin listing has to say so.
- Tradeoff: the wrapper plugin has to keep a list of providers that carry an Archcore adapter; adding an adapter (Pi, for example) requires a change in the wrapper plugin as well.
- Tradeoff: [assumption] ACP-hosted agents in Zed, JetBrains, and Devin Desktop may not load inner plugins through `claude-agent-acp`; this decision does not cover them until a probe shows how they behave.

## Superseded when

- A wrapper editor ships a pre-tool-call hook that can block a tool call, which would let a wrapper-level adapter carry the write guard and reopen alternative 1 for that wrapper.
- A wrapper starts its inner agent with the user's agent plugins disabled, which removes the inner adapter from the session.
- A wrapper exposes a way to detect an installed inner Archcore plugin at runtime, which would replace the static provider list with detection.
