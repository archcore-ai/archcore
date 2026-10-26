---
title: "@archcore/opencode-plugin Publishes Bundled JavaScript in dist/, Built by bun build at Pack Time"
status: accepted
tags:
  - "component:plugin"
  - "multi-host"
  - "opencode"
  - "plugin"
  - "release"
---

## Context

`opencode-adapter.spec` listed `dist/` in the package `files`, while no task of `opencode-adapter.plan` produced it. Every host that loads the package today imports a `.ts` entry: OpenCode 1.18.34 and Kilo CLI 7.8.14 run plugins under a Bun-compiled binary through `await import(entry)` (`packages/opencode/src/plugin/loader.ts` in both repositories), and Pi 1.3.0 loads extensions through jiti (`packages/coding-agent/src/core/extensions/loader.ts`). Plain Node refuses to strip types for files under `node_modules` (`ERR_UNSUPPORTED_NODE_MODULES_TYPE_STRIPPING`, nodejs.org/api/typescript.html, read 2026-10-18), and the OpenCode plugins `oh-my-opencode` 5.1.35, `opencode-wakatime` 1.4.0, `opencode-gemini-auth` 2.0.1, plus `@opencode-ai/plugin` itself, all publish built JavaScript.

## Decision

The pack step of `@plugin/plugins/opencode/` runs `bun build src/index.ts --outdir dist --target node --format esm`, and `package.json` points `main` and `exports["./server"]` at `dist/index.js`, so the published tarball carries JavaScript and no `.ts` file is imported at runtime.

## Alternatives Considered

1. **Publish `src/*.ts` with `main` pointing at `src/index.ts`** — rejected because the package would then load only under Bun or jiti, and Node's documented refusal blocks a later reuse of the bridge under a Node runtime.
2. **Compile with `tsc` to `dist/`** — rejected because it adds `typescript` as a build dependency and emits one file per module, while `bun build` is already required for `bun test` and emits one bundle.

## Consequences

- The package loads under Bun, jiti, and Node alike, which keeps the bridge reusable for the Pi adapter that `agent-host-landscape.rnd` ranks third.
- `--target node` limits the bridge to Node built-ins such as `node:child_process`; the Bun shell `$` stays unused, as the spec already requires.
- Tradeoff: `dist/` is a build artifact absent from git, so a structure check runs `npm pack --dry-run` and fails when `dist/index.js` is missing from the tarball listing.
- Tradeoff: the pack step gains one command and Bun becomes a release-time dependency as well as a test-time one.

## Superseded when

- Node accepts type stripping under `node_modules`, which would make a `.ts` entry portable to every JavaScript runtime.
- OpenCode documents a plugin build format other than an ESM entry from `exports["./server"]` or `main`.
