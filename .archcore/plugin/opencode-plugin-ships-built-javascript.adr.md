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

`opencode-adapter.spec` lists `dist/` in the package `files`, but nothing produced it. Every host that loads the package imports a `.ts` entry: OpenCode and Kilo CLI run plugins under a Bun-compiled binary through `await import(entry)` (`packages/opencode/src/plugin/loader.ts` in both repositories), and Pi loads extensions through jiti. Plain Node refuses to strip types for files under `node_modules` (`ERR_UNSUPPORTED_NODE_MODULES_TYPE_STRIPPING`, nodejs.org/api/typescript.html, read 2026-10-18). Published OpenCode plugins, including `@opencode-ai/plugin` itself, ship built JavaScript.

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
