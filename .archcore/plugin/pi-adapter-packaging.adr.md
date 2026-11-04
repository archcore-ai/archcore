---
title: "Pi Adapter Is a TypeScript Package Under plugins/pi, Published to npm, Sharing the OpenCode Spawn Module by a Pack-Step Copy"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "multi-host"
  - "pi"
  - "plugin"
---

## Context

Pi 1.5.0 (`earendil-works/pi` commit `fcebc09`) loads extensions as TypeScript through jiti from a Pi package declared by `package.json` (`pi.extensions`, `pi.skills`, keyword `pi-package`), and installs packages with `pi install npm:<pkg>` (`packages/coding-agent/docs/packages.md`). Its hooks exist only as extension code: `session_start`, `tool_call` (returns `{block, reason}`), `tool_result`, and `before_agent_start` (`src/core/extensions/types.ts`). No declarative hook config exists.

`stack-and-tooling.rule` item 4 allows TypeScript only under `plugins/opencode/`; items 24 and 25 require an accepted ADR before a new language scope or distribution mechanism. `@plugin/plugins/opencode/src/spawn.ts` (53 lines) holds the launcher spawn with `ARCHCORE_HOST: "opencode"` fixed in its env.

## Decision

The Pi adapter is the npm package `@archcore/pi-plugin` under `plugin/plugins/pi/`, written in TypeScript and tested with `bun test` inside that directory. `@plugin/plugins/opencode/src/spawn.ts` takes the host id as a parameter, and the Pi pack step copies that file byte-identical into `plugin/plugins/pi/src/spawn.ts`. The package ships `bin/` and `skills/` copied by the pack step and no agents, because Pi core has no agent-definition format.

## Alternatives Considered

1. A third shared TypeScript directory imported by both packages — rejected because both packages publish independently to npm, and a shared source directory needs a workspace build that neither package has.
2. A separate spawn implementation in the Pi package — rejected because two copies of the timeout and exit-translation code drift; the parity test keeps one source.
3. A Pi package with skills and MCP only, no extension code — rejected because the guard then never runs, although `tool_call` can block.

## Consequences

- Positive: guard and validation logic stay in the CLI; the bridge translates only.
- Positive: one spawn source for both TypeScript hosts.
- Tradeoff: `stack-and-tooling.rule` items 4, 21, 22, and 30 extend to `plugins/pi/`.
- Tradeoff: the OpenCode package changes once, to take the host id as a parameter.

## Superseded when

- Pi ships a declarative hook config that runs a command.
- A third TypeScript host arrives, which justifies a shared workspace package.
