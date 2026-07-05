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

Build the decision context for an OpenCode host adapter: verify the plugin API, skills discovery, hook semantics, and MCP registration against official docs and source, and update the adapter design decisions ahead of the mandatory ADR (`stack-and-tooling.rule` — a JS/TS adapter introduces a new language and requires an ADR before any code).

## Questions

1. How does an OpenCode plugin load, and what hook points exist?
2. Can our guard scripts stay authoritative (bridge shells out to `bin/check-*`), and how does a deny reach the model?
3. Can our Claude-style skills and markdown agents be reused without copying?
4. How is the MCP server registered — and can the plugin do it programmatically?
5. How are OpenCode plugins distributed?

## Approach

Documentation and source sweep, 2026-07-05: opencode.ai/docs (plugins, skills, agents, mcp-servers, rules, config, ecosystem), `packages/plugin/src/index.ts` (full Hooks interface), `skill/index.ts`, `session/tools.ts`, the v1 config schema, and the first-party `customize-opencode` skill. Repo note: `sst/opencode` now redirects to `anomalyco/opencode`; plugin types package `@opencode-ai/plugin` at 1.17.13.

## Findings

**Plugin loading.** Local files in `.opencode/plugins/` (project) and `~/.config/opencode/plugins/` (global) auto-load at startup; npm packages load via the `plugin` array in `opencode.json`, auto-installed by Bun into `~/.cache/opencode/node_modules/`, pinnable as `name@x.y.z`, gateable via `"engines": {"opencode": "<range>"}`. A plugin exports `Plugin = (input, options?) => Promise<Hooks>`; `input` includes `directory`, `worktree`, `client`, and `$` (Bun shell).

**Hook points.** `tool.execute.before ({tool, sessionID, callID}, {args})` and `tool.execute.after ({tool, sessionID, callID, args}, {title, output, metadata})` exist exactly as assumed. Also available: `event` (all bus events, incl. `session.created` / `session.idle`), `config` (called once at init with the live merged config — "mutate fields here" per the first-party skill), `permission.ask` (can force `deny`/`allow`), `chat.message`, `shell.env`, and custom `tool` definitions. There is **no literal session-start hook** — the equivalent is plugin-init plus `event: session.created`.

**Deny semantics.** Throwing from `tool.execute.before` blocks the call; verified through source that the model receives a failed tool result (`output-error` with `errorText` = the thrown message) and the session continues. Hooks mutate `output` in place and return void.

**Shell-out bridge — endorsed pattern.** Plugins receive Bun's `$` shell explicitly and official examples shell out. The hook bridge can spawn `bin/check-*` with the canonical stdin JSON and translate a blocking exit into `throw Error(<reason from stderr>)` — zero decision logic in TS, per `host-adapter-contract.spec`. Guard: `$` is `undefined` in non-Bun embeddings.

**Skills reuse — corrected decision.** OpenCode natively reads project/global `.claude/skills/**/SKILL.md` and `.agents/skills/**/SKILL.md` and *ignores unknown frontmatter* (`allowed-tools` is tolerated), so Claude-authored skills load unmodified. **But those are user-project paths** — our skills ship inside the plugin package, not in the user's repo, so the compatibility paths alone do not deliver them. The zero-copy route for a packaged adapter is the `skills.paths` config key (scanned recursively for `**/SKILL.md`), pointed at the npm package's own `skills/` directory from the plugin's `config` hook (`cfg.skills.paths.push(<pkg>/skills)`). Ordering guarantee (config hook before skill discovery) is undocumented → probe. Duplicate skill names: later-loaded overwrites with a warning and the order is nondeterministic — avoid duplicates entirely.

**MCP registration.** `opencode.json` `mcp` schema for stdio: `{"archcore": {"type": "local", "command": ["archcore", "mcp"], "environment": {…}, "enabled": true}}` — the key is `environment` (not `env`), `command` is an array, timeout defaults to 5000 ms. Programmatic registration from the plugin's `config` hook (`cfg.mcp.archcore = …`) is the community-standard mechanism and is supported by source reading, but the init-ordering contract is undocumented → probe. MCP tool names get prefixed (`archcore_*`) — relevant for matching post-MCP validation in `tool.execute.after`.

**Agents.** `.opencode/agents/*.md` (singular `agent/` also works); the filename becomes the agent name; `description` is required; `mode: primary|subagent|all`; permissions via the `permission` map — the `tools` field is deprecated. Unknown frontmatter is routed into `options`. Our two agents port as markdown with permission maps replacing tool lists.

**Instructions.** `AGENTS.md` walk-up plus global; `CLAUDE.md` is read when no `AGENTS.md` exists (first match wins per category); the `instructions` config key accepts globs and URLs. No native `@file` imports.

**Distribution.** Ecosystem convention: unscoped `opencode-*` names dominate (~37 plugins listed); scoped `@org/name` is supported. Users install by adding the package name to `plugin: []` — no separate install step; config is not hot-reloaded (restart required). Community template exists (`zenobi-us/opencode-plugin-template`); no first-party template.

## Recommendation

Updated adapter design, pending the maintainer ADR:

1. **Packaging: npm package.** Name decision for the ADR: `opencode-archcore` (better ecosystem-list discoverability) vs `@archcore/opencode-plugin` (namespace consistency). Ship an `engines` gate and document version pinning.
2. **Repo-location decision for the ADR:** a separate repo keeps this repo shell-only (no stack-rule exception needed); `plugins/opencode/` in this repo gives single-repo releases but requires amending `stack-and-tooling.rule`.
3. **Hook bridge:** `tool.execute.before` → `bin/check-archcore-write` / `bin/check-code-alignment`; `tool.execute.after` → `bin/validate-archcore` (+ cascade/precision); deny = `throw Error(<reason>)`; session-start = plugin init + `session.created` → `bin/session-start`.
4. **Skills: bundle `skills/` inside the package and register via `cfg.skills.paths` in the `config` hook**; fallback = a documented manual `skills.paths` entry. Do NOT sync/copy into user projects.
5. **MCP: register in the `config` hook** (`type: "local"`, `command: ["archcore", "mcp"]`, `environment` key).
6. **Agents: ship both as markdown** with `mode: subagent` and permission maps.

## Next Action

Maintainer ADR (package name, repo location, stack-rule exception or non-exception), then live probes: (1) `config`-hook mutation ordering vs MCP init and skill discovery; (2) `cfg.skills.paths` honored when set from the hook; (3) transcript rendering of a `tool.execute.before` throw; (4) duplicate-skill-name winner determinism; (5) `$` availability in packaged installs; (6) the exact `input.tool` string for MCP tools (`archcore_*`) in `tool.execute.before/after`. Then adapter implementation per `host-adapter-contract.spec`.