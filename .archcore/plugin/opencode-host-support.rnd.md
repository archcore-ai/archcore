---
title: "OpenCode Host Support — Verified Plugin API and Adapter Design Context"
status: accepted
tags:
  - "hooks"
  - "multi-host"
  - "opencode"
  - "plugin"
  - "roadmap"
---

## Goal

Build the decision context for an OpenCode host adapter: verify the plugin API, skills discovery, hook semantics, and MCP registration against the official documentation and source, and update the adapter design ahead of the mandatory ADR. `stack-and-tooling.rule` requires that ADR before any code, because a JS or TS adapter introduces a new language.

## Questions

1. How does an OpenCode plugin load, and what hook points exist?
2. Can the guard scripts stay authoritative, with the bridge shelling out to `bin/check-*`, and how does a deny reach the model?
3. Can the Claude-style skills and markdown agents be reused without copying?
4. How is the MCP server registered, and can the plugin do it programmatically?
5. How are OpenCode plugins distributed?

## Approach

A documentation and source sweep on 2026-07-05 over opencode.ai/docs — plugins, skills, agents, mcp-servers, rules, config, and ecosystem — plus `packages/plugin/src/index.ts` for the full Hooks interface, `skill/index.ts`, `session/tools.ts`, the v1 config schema, and the first-party `customize-opencode` skill. Repository note: `sst/opencode` now redirects to `anomalyco/opencode`, and the plugin types package `@opencode-ai/plugin` is at 1.17.13.

## Findings

**Plugin loading.** A local file in `.opencode/plugins/` for a project, or `~/.config/opencode/plugins/` globally, auto-loads at startup. An npm package loads through the `plugin` array in `opencode.json`, is auto-installed by Bun into `~/.cache/opencode/node_modules/`, is pinnable as `name@x.y.z`, and is gateable through `"engines": {"opencode": "<range>"}`. A plugin exports `Plugin = (input, options?) => Promise<Hooks>`, where `input` carries `directory`, `worktree`, `client`, and `$`, the Bun shell.

**Hook points.** `tool.execute.before` and `tool.execute.after` exist exactly as assumed, carrying `{tool, sessionID, callID}` with `{args}` and `{title, output, metadata}` respectively. Also available are `event` for all bus events including `session.created` and `session.idle`; `config`, called once at init with the live merged config, which the first-party skill describes as the place to mutate fields; `permission.ask`, which can force a deny or an allow; `chat.message`; `shell.env`; and custom `tool` definitions. There is no literal session-start hook — the equivalent is plugin init plus the `session.created` event.

**Deny semantics.** Throwing from `tool.execute.before` blocks the call. Source reading confirms the model receives a failed tool result, an `output-error` whose `errorText` is the thrown message, and the session continues. A hook mutates `output` in place and returns void.

**The shell-out bridge is an endorsed pattern.** A plugin receives Bun's `$` shell explicitly, and the official examples shell out. The bridge can spawn a `bin/check-*` script with the canonical stdin JSON and translate a blocking exit into a thrown error carrying the reason from stderr, which keeps decision logic out of TypeScript per `host-adapter-contract.spec`. One guard applies: `$` is undefined in a non-Bun embedding.

**Skills reuse — a corrected decision.** OpenCode natively reads project and global `.claude/skills/**/SKILL.md` and `.agents/skills/**/SKILL.md` and ignores unknown frontmatter, tolerating `allowed-tools`, so a Claude-authored skill loads unmodified. But those are user-project paths, and the plugin's skills ship inside the package rather than in the user's repository, so the compatibility paths alone do not deliver them. The zero-copy route for a packaged adapter is the `skills.paths` config key, scanned recursively for `**/SKILL.md`, pointed at the package's own `skills/` directory from the plugin's `config` hook. The ordering guarantee — that the config hook runs before skill discovery — is undocumented and needs a probe. On a duplicate skill name the later load overwrites with a warning and the order is nondeterministic, so duplicates must be avoided entirely.

**MCP registration.** The `opencode.json` `mcp` schema for stdio takes `{"archcore": {"type": "local", "command": ["archcore", "mcp"], "environment": {…}, "enabled": true}}`, where the key is `environment` rather than `env`, `command` is an array, and the timeout defaults to 5000 ms. Programmatic registration from the plugin's `config` hook is the community-standard mechanism and is supported by source reading, but the init-ordering contract is undocumented and needs a probe. MCP tool names arrive prefixed as `archcore_*`, which matters for matching post-MCP validation in `tool.execute.after`.

**Agents.** They live in `.opencode/agents/*.md`, with the singular `agent/` also working; the filename becomes the agent name; `description` is required; `mode` is `primary`, `subagent`, or `all`; and permissions come from the `permission` map, with the `tools` field deprecated. Unknown frontmatter is routed into `options`. Both Archcore agents port as markdown with permission maps replacing the tool lists.

**Instructions.** OpenCode walks up for `AGENTS.md` and also reads a global one, falling back to `CLAUDE.md` only where no `AGENTS.md` exists, with the first match winning per category. The `instructions` config key accepts globs and URLs, and there are no native `@file` imports.

**Distribution.** The ecosystem convention favors unscoped `opencode-*` names, which dominate the roughly 37 listed plugins, though a scoped `@org/name` is supported. A user installs by adding the package name to the `plugin` array, with no separate install step, and the config is not hot-reloaded, so a restart is required. A community template exists at `zenobi-us/opencode-plugin-template`; there is no first-party template.

## Recommendation

The updated adapter design, pending the maintainer ADR:

1. **Package it on npm.** The name decision for the ADR is `opencode-archcore`, which is more discoverable in the ecosystem list, against `@archcore/opencode-plugin`, which is consistent with the namespace. Ship an `engines` gate and document version pinning.
2. **Decide the repository location in the ADR.** A separate repository keeps this one shell-only and needs no stack-rule exception; `plugins/opencode/` here gives single-repo releases but requires amending `stack-and-tooling.rule`.
3. **Bridge the hooks.** Route `tool.execute.before` to `bin/check-archcore-write` and `bin/check-code-alignment`, and `tool.execute.after` to `bin/validate-archcore` plus the cascade and precision checks. A deny is a thrown error carrying the reason, and session start is plugin init plus `session.created` routed to `bin/session-start`.
4. **Bundle `skills/` inside the package and register it through `cfg.skills.paths` in the `config` hook**, with a documented manual `skills.paths` entry as the fallback. Do not sync or copy into a user project.
5. **Register MCP in the `config` hook**, with `type: "local"`, `command: ["archcore", "mcp"]`, and the `environment` key.
6. **Ship both agents as markdown** with `mode: subagent` and permission maps.

## Next Action

The maintainer ADR first, covering the package name, the repository location, and whether the stack rule gains an exception. Then six live probes: the `config`-hook mutation ordering against MCP init and skill discovery; whether `cfg.skills.paths` is honored when set from the hook; how a thrown `tool.execute.before` renders in the transcript; whether the duplicate-skill-name winner is deterministic; whether `$` is available in a packaged install; and the exact `input.tool` string for an MCP tool in both hooks. Then adapter implementation per `host-adapter-contract.spec`.
