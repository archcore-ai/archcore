---
title: "Antigravity Plugin Ships as a Dedicated Root plugins/antigravity Built by a Pack Step, With Its Own hooks.json, mcp_config.json, and Agent Variants"
status: accepted
tags:
  - "antigravity"
  - "architecture"
  - "component:plugin"
  - "multi-host"
  - "plugin"
---

## Context

Antigravity (`agy` CLI v1.4.12, 2026-10-30) loads a plugin from a root that holds `plugin.json` or `.antigravity-plugin/plugin.json`, and finds components only at fixed paths under that root: `skills/<name>/SKILL.md`, `agents/<name>.md`, `rules/`, `mcp_config.json`, and `hooks.json` (antigravity.google/docs/build-plugins, fetched 2026-11-04). It does not read `hooks/hooks.json`, and `agy plugin validate` warns about that file (changelog v1.4.3). Its agent frontmatter (`name`, `description`, `tools`, `model`, `mainAgent`, `subagent`) differs from the Claude Code frontmatter in `@plugin/plugins/archcore/agents/`. Hook commands run with the plugin root as the working directory.

Pointing an Antigravity manifest at `@plugin/plugins/archcore/` would make Antigravity load the Claude-format `agents/*.md` from its fixed path. `stack-and-tooling.rule` item 28 requires an accepted ADR before a new manifest format lands, and `host-adapter-contract.spec` item 18 forbids a per-host copy of a skill.

## Decision

The Antigravity plugin is a separate plugin root, `plugin/plugins/antigravity/`, whose tracked files are `plugin.json`, `hooks.json`, `mcp_config.json`, and the Antigravity agent variants under `agents/`. A pack step copies `skills/` and `bin/` byte-identical from `@plugin/plugins/archcore/` into that root before export, and the release tree carries the packed root. Users install it with `agy plugin install <github-url-of-the-subpath>` before a Marketplace listing exists, and by name after.

## Alternatives Considered

1. `.antigravity-plugin/plugin.json` inside `@plugin/plugins/archcore/` — rejected because Antigravity then loads the Claude-format agents from the shared `agents/` directory, and the shared root would carry a second hooks file.
2. Users run `agy plugin import claude` on the installed Claude Code plugin — rejected because the import skips project-scoped installs (changelog v1.4.11), requires Claude Code on the machine, and keeps the Claude hook payload expectations that Antigravity does not send.

## Consequences

- Positive: each host loads only files in its own format; the shared root stays unchanged.
- Positive: skills and launchers stay single-sourced through the pack step, as for the OpenCode package.
- Tradeoff: the agent variants are a format variant exempt from contract item 18 only under a byte-parity test of their prompt bodies.
- Tradeoff: a GitHub-subpath install URL form is undocumented; the release must verify one form before the CLI row uses it.
- Tradeoff: `stack-and-tooling.rule` item 14 gains `plugin.json`, `hooks.json`, and `mcp_config.json` under `plugins/antigravity/` as allowed declarative shapes.

## Superseded when

- Antigravity reads Claude Code agent frontmatter or a per-host agents directory.
- Antigravity documents a manifest key that names a component directory, removing the fixed-path collision.
