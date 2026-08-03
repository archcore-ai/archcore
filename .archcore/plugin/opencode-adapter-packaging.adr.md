---
title: "OpenCode Adapter — TypeScript Package @archcore/opencode-plugin in plugins/opencode/"
status: accepted
tags:
  - "architecture"
  - "multi-host"
  - "opencode"
  - "plugin"
---

## Context

OpenCode's plugin runtime executes JS and TS modules under Bun and exposes hooks only programmatically — `tool.execute.before`, `tool.execute.after`, `config`, and `event`, per `@opencode-ai/plugin` 1.17.13 and opencode.ai/docs/plugins — so an OpenCode host adapter cannot be pure declarative configuration like the three existing adapters. `stack-and-tooling.rule` restricts executable code in this repository to POSIX shell under `@plugins/archcore/bin/` and requires an accepted ADR before any new language lands. This document is that exception record, decided by the maintainer on 2026-07-05.

## Decision

Ship the OpenCode host adapter as the TypeScript npm package **`@archcore/opencode-plugin`**, developed in this repository under **`plugins/opencode/`** with `bun test` scoped to that directory, whose hooks are thin bridges shelling out to the shared `plugins/archcore/bin/` scripts — a deny being `throw Error(<reason>)` from `tool.execute.before` — and whose MCP server and package-bundled `skills/` are registered in the plugin's `config` hook as `cfg.mcp.archcore = {type:"local", command:["archcore","mcp"]}` and `cfg.skills.paths`.

Duplication of skill content is permitted only where a delivery mechanism requires it, and each such mechanism is specified in a spec and covered by tests before release.

## Alternatives Considered

1. **A separate repository, `archcore-ai/opencode-plugin`** — rejected because it splits the release cycle and the test harness for content that must stay byte-aligned with `plugins/archcore/skills/` and `bin/`; every core change would need a cross-repo sync pull request, and the maintainer chose single-repo releases on 2026-07-05.
2. **Local-file distribution, copying `.opencode/plugins/archcore.ts` into user projects** — rejected because it has no version or update channel: users hand-copy a file that drifts from the core silently, whereas OpenCode's npm path auto-installs at startup, supports pinning as `name@x.y.z`, and honors an `engines: {"opencode": <range>}` compatibility gate.
3. **Reimplementing guard and validation logic natively in TypeScript** — ruled out because it forks enforcement semantics away from the `bin/check-*` scripts the other three hosts execute, so parity would then depend on manually porting every guard change into TypeScript.
4. **The unscoped package name `opencode-archcore`** — rejected because the maintainer prioritized `@archcore/*` namespace consistency over the ecosystem list's unscoped convention on 2026-07-05, and OpenCode supports scoped packages.

## Consequences

- [expected] Skills, agents, and `bin/` scripts ship to OpenCode from the same repository and release, so a core guard fix reaches all four hosts in one pull request.
- npm distribution provides version pinning and an `engines` gate, and the install is one line in the user's `opencode.json` `plugin` array.
- Tradeoff: `stack-and-tooling.rule` gains a scoped exception — TypeScript and `bun test` inside `plugins/opencode/` only, with `plugins/archcore/` and repo-root tooling staying POSIX shell and bats. The rule text is amended alongside this decision.
- Tradeoff: the supply-chain surface grows. npm publishing credentials, the `@opencode-ai/plugin` dependency, and Bun runtime behavior — a failed module resolution is cached per process — become operational concerns.
- Tradeoff: the `config`-hook registration path for MCP and `skills.paths` relies on an init-ordering contract that OpenCode does not document. A live probe MUST pass before the first release, with documented manual `opencode.json` wiring as the fallback.
- [expected] Tradeoff: the repository gains a second plugin root, so structure tests, release synthesis, and the `.archcore`-reference grep guard must each include or exclude `plugins/opencode/` explicitly.

## Superseded when

- OpenCode ships a declarative JSON hook config comparable to the Claude and Codex `hooks.json`, which would make a pure-configuration adapter possible and argue for dropping the TypeScript package.
- The `config`-hook ordering probe fails and no supported programmatic registration mechanism lands within one OpenCode minor release cycle, which would argue for documented manual `opencode.json` wiring.
- The adapter needs more than 2 hotfix releases per month decoupled from plugin releases, sustained over a quarter, which would reopen the separate-repository alternative.
