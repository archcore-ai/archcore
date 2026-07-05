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

OpenCode's plugin runtime executes JS/TS modules under Bun and exposes hooks only programmatically (`tool.execute.before`/`tool.execute.after`, `config`, `event` — `@opencode-ai/plugin` 1.17.13, opencode.ai/docs/plugins), so an OpenCode host adapter cannot be pure declarative configuration like the three existing host adapters. `stack-and-tooling.rule` restricts executable code in this repository to POSIX shell (@plugins/archcore/bin/) and requires an accepted ADR before any new language lands; this ADR is that exception record, decided by the maintainer on 2026-07-05.

## Decision

Ship the OpenCode host adapter as a TypeScript npm package **`@archcore/opencode-plugin`**, developed in this repository under **`plugins/opencode/`** (with `bun test` as its test runner, scoped to that directory), whose hooks are thin bridges shelling out to the shared `plugins/archcore/bin/` scripts (deny = `throw Error(<reason>)` from `tool.execute.before`), whose MCP server and package-bundled `skills/` are registered in the plugin's `config` hook (`cfg.mcp.archcore = {type:"local", command:["archcore","mcp"]}`; `cfg.skills.paths`), and where duplication of skill content is permitted only when a delivery mechanism requires it — each such mechanism specified in a spec and covered by tests before release.

## Alternatives Considered

1. **Separate repository (`archcore-ai/opencode-plugin`)** — rejected because it splits the release cycle and test harness for content that must stay byte-aligned with `plugins/archcore/skills/` and `bin/`; every core change would require a cross-repo sync PR, and the maintainer chose single-repo releases (2026-07-05).
2. **Local-file distribution (`.opencode/plugins/archcore.ts` copied into user projects)** — rejected because it has no version or update channel: users hand-copy a file that silently drifts from the core, while OpenCode's npm path auto-installs at startup, supports pinning (`name@x.y.z`), and honors an `engines: {"opencode": <range>}` compatibility gate.
3. **Reimplementing guard/validation logic natively in TypeScript** — ruled out because it forks enforcement semantics away from the `bin/check-*` scripts the other three hosts execute; parity would then depend on manually porting every guard change into TS.
4. **Unscoped package name `opencode-archcore`** — rejected because the maintainer prioritized `@archcore/*` namespace consistency over the ecosystem list's unscoped naming convention (2026-07-05); OpenCode supports scoped packages.

## Consequences

- Skills, agents, and `bin/` scripts ship to OpenCode from the same repo and release; a core guard fix reaches all four hosts in one PR. [expected]
- npm distribution gives version pinning and an `engines` gate; install UX is one line in the user's `opencode.json` `plugin` array.
- `stack-and-tooling.rule` gains a scoped exception: TypeScript + `bun test` are allowed inside `plugins/opencode/` only; `plugins/archcore/` and repo-root tooling stay POSIX shell + bats. The rule text is amended alongside this ADR.
- Supply-chain surface grows: npm publishing credentials, the `@opencode-ai/plugin` dependency, and Bun runtime behaviors (failed module resolution is cached per process) become operational concerns.
- The `config`-hook registration path (MCP + `skills.paths`) relies on an init-ordering contract that OpenCode does not document; a live probe MUST pass before first release, with documented manual `opencode.json` wiring as the fallback.
- The repo gains a second plugin root: structure tests, release synthesis, and the `.archcore`-reference grep guard must explicitly include or exclude `plugins/opencode/`. [expected]

## Superseded when

- OpenCode ships a declarative JSON hook config comparable to Claude/Codex `hooks.json`, making a pure-configuration adapter possible — re-decide toward dropping the TS package.
- The `config`-hook ordering probe fails and no supported programmatic registration mechanism lands within one OpenCode minor release cycle — re-decide toward documented manual `opencode.json` wiring.
- The adapter needs more than 2 hotfix releases per month decoupled from plugin releases (sustained over a quarter) — revisit the separate-repo alternative.