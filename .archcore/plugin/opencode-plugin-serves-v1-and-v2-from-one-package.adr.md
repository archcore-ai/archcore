---
title: "@archcore/opencode-plugin Serves OpenCode 1.x and 2.x From One Default Export With server and setup"
status: accepted
tags:
  - "component:plugin"
  - "multi-host"
  - "opencode"
  - "plugin"
---

## Context

OpenCode 2.x (tags `v2.0.0`–`v2.1.4`, no GitHub Release yet; stable is `v1.19.4`, checked 2026-10-26) does not run V1 plugins: "V1 plugin implementations do not run in V2" (`services/www/src/docs/content/build/plugins/migrate-v1.mdx` in anomalyco/opencode). The bridge in `@plugin/plugins/opencode/src/bridge.ts` is a V1 plugin (`{ id, server }`), so on OpenCode 2.1.4 it loads nothing and guards nothing. OpenCode 2.x still reads the V1 `plugin` key and normalizes it to `plugins` (`packages/core/src/config/normalize.ts:185-190`, v2.1.4), so projects that `archcore init` wired under 1.x load the package on 2.x without a config change.

## Decision

The package keeps one default export that carries both the V1 `server` function and a V2 `setup(ctx)` function, as the V2 migration guide permits ("V1 calls `server()` and V2 calls `setup()`"), and the V2 half maps each V1 hook to its V2 destination while both halves keep spawning the same `bin/` launchers.

## Alternatives Considered

1. **Publish a separate `@archcore/opencode-plugin-v2` package** — rejected because `archcore init` would have to pick a package by host version, and projects already wired with the V1 package name would load nothing after an OpenCode upgrade.
2. **Drop V1 and ship a V2-only release** — rejected because OpenCode 1.19.4 is the stable release on 2026-10-26 and 2.x has no GitHub Release yet.
3. **Depend on `@opencode/plugin` for `Plugin.define`** — rejected because `define` returns its argument unchanged (`packages/plugin/src/promise/plugin.ts:59-66`), and a plugin that loads its own copy of `@opencode/plugin` fails under Bun (`packages/plugin/src/runtime.ts:3-33`).

## Consequences

- A project wired under OpenCode 1.x keeps working after an upgrade to 2.x, because 2.x reads the same `plugin` entry and calls `setup`.
- The guard needs no CLI change: CLI 0.13.1 denies the V2 `write` and `edit` payloads (`path` key) and the V2 `patch` payload (`patchText`), checked 2026-10-26.
- Tradeoff: V2 runs MCP tools inside Code Mode by default, so post-mutation validation text cannot ride on the MCP tool result; the V2 half delivers it on the next model request of the same session instead.
- Tradeoff: V2 does not turn skills into slash commands, so the V2 half registers `/archcore-init`, `/archcore-plan`, `/archcore-document`, and `/archcore-review` as commands next to the skills.
- Tradeoff: `opencode plugin add` writes only the global config on 2.x, so a 2.x install through the Archcore CLI is machine-scoped [expected].

## Superseded when

- OpenCode 2.x ships a stable release and the 1.x line stops receiving releases, which ends the V1 support window and removes the `server` half.
- OpenCode 2.x stops reading the V1 `plugin` key, which would require `archcore init` to write `plugins` for 2.x projects.