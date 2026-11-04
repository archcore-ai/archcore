---
title: "OpenClaw Gets One archcore Skill on ClawHub and Native MCP Through openclaw mcp add; Coding Sessions Rely on a Project-Scoped Inner Plugin"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "integrations"
  - "multi-host"
  - "openclaw"
---

## Context

OpenClaw 2026.10.10 loads AgentSkills-format `SKILL.md` folders from `<workspace>/skills`, `.agents/skills`, and `~/.openclaw/skills`, and installs ClawHub skills with `openclaw skills install @owner/<slug>` (`openclaw` `docs/tools/skills.md`, commit `37d5c24f`). ClawHub publishes with `clawhub skill publish` and licenses every skill `MIT-0` (`clawhub` `docs/publishing.md`, commit `ea85448f`). OpenClaw has a native MCP client configured by `openclaw mcp add` under `mcp.servers` (`docs/tools/mcp.md`); `mcporter` is a separate CLI whose stdio `cwd` defaults to the config file directory (`mcporter` `docs/config.md`). nanobot reads the same skill metadata; PicoClaw reads only `name` and `description`.

Coding sessions run through ACP: acpx 0.21.0, pinned by OpenClaw, passes `settingSources: ["project", "local"]` to `claude-agent-acp` unless `ACPX_CLAUDE_INCLUDE_USER_SETTINGS=1` is set (`acpx` `src/acp/agent-command.ts`), so a user-scope Claude Code plugin does not load there. `stack-and-tooling.rule` item 24 requires an accepted ADR before a new distribution mechanism.

## Decision

Archcore publishes one skill, slug `archcore`, on ClawHub. The skill declares `metadata.openclaw.requires.bins: ["archcore"]` and directs the agent to the Archcore MCP tools. The MCP server is registered with OpenClaw's native client (`openclaw mcp add archcore --command archcore --arg mcp --arg --project --arg <root>`), not through `mcporter`. For ACP coding sessions, the integration recipe installs the Archcore Claude Code plugin at project scope (`archcore plugin install --scope project`), which `settingSources: ["project", "local"]` loads.

## Alternatives Considered

1. MCP through `mcporter`, chosen first on 2026-11-04 — replaced because OpenClaw's native client needs no second CLI, and `mcporter`'s stdio `cwd` default points at the config directory instead of the project.
2. An OpenClaw code plugin with a `before_tool_call` guard — deferred; the maintainer scoped this work to the skill, MCP, and the ACP recipe.
3. Ask users to set `ACPX_CLAUDE_INCLUDE_USER_SETTINGS=1` — kept as the documented alternative, not the default, because it also loads every other user plugin, which acpx excludes on purpose.

## Consequences

- Positive: one `SKILL.md` serves OpenClaw and nanobot.
- Positive: ACP coding sessions get the full Claude Code guard with no OpenClaw-side code.
- Tradeoff: OpenClaw's own agent gets no write guard; a direct `.archcore/` write from it is not blocked.
- Tradeoff: the project-scope install writes the marketplace entry into the repository's `.claude/settings.json`, which reaches every teammate.
- Tradeoff: ClawHub publishes the skill under `MIT-0`.

## Superseded when

- acpx stops excluding user settings, or OpenClaw sets `ACPX_CLAUDE_INCLUDE_USER_SETTINGS`.
- The maintainer approves an OpenClaw code plugin with a blocking `before_tool_call` guard.
