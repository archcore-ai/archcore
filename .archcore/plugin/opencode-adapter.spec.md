---
title: "OpenCode Adapter — @archcore/opencode-plugin Bridge, Config Registration, and Package Contract"
status: draft
tags:
  - "component:plugin"
  - "hooks"
  - "multi-host"
  - "opencode"
  - "plugin"
---

## Purpose & Scope

This spec is normative for the OpenCode adapter: the npm package `@archcore/opencode-plugin` developed under `@plugin/plugins/opencode/`, the four OpenCode hooks it exports, the spawn contract between those hooks and the three launchers in `@plugin/plugins/archcore/bin/`, and the trees the package bundles. Depended on by OpenCode sessions in a project that carries `.archcore/`, by the `archcore init` and `archcore plugin` delivery surface (`plugin-delivery.spec`), and by the bats and `bun test` suites that pin the shared stdin fixtures. Out of scope: the guard, validation, and context policy inside `archcore hooks opencode <leaf>` (`hook-runtime.spec`), the delivery surface itself, agent definitions for OpenCode, and every other host.

## Surface

- Package `@archcore/opencode-plugin`: source `@plugin/plugins/opencode/src/`, manifest `@plugin/plugins/opencode/package.json`; `engines.opencode` carries the lowest host version the live probes passed on. OpenCode installs it from the `plugin` array of `opencode.json` into `~/.cache/opencode/node_modules/` at startup (opencode.ai/docs/plugins, read 2026-10-09).
- Bundled trees `bin/` and `skills/`, copied at pack time from `@plugin/plugins/archcore/bin/` and `@plugin/plugins/archcore/skills/`; absent from git.
- Module shape: `export default { id: "@archcore/opencode-plugin", server }`, the v1 form `readV1Plugin` accepts (`packages/opencode/src/plugin/shared.ts`, OpenCode v1.18.34). An unpinned `plugin` entry resolves to `@latest`; `engines.opencode` is enforced at load.
- Hooks `server()` returns (`packages/plugin/src/index.ts`, v1.18.34): `config`, `tool.execute.before`, `tool.execute.after`, `experimental.chat.system.transform`.
- `tool.execute.after` output shape: a built-in tool passes `{title, output, metadata}`; an MCP tool passes the MCP `CallToolResult`, whose text lives in `content[]` (`packages/opencode/src/session/tools.ts`, v1.18.34).
- Spawn contract (header of `@plugin/plugins/archcore/bin/lib/normalize-stdin.sh`): env `ARCHCORE_HOST=opencode`; cwd = the plugin input's `directory`; stdin = one snake_case JSON object `{"tool_name": <input.tool>, "tool_input": <output.args>}` with the args verbatim. Captured shapes: `@plugin/test/fixtures/stdin/opencode/`.
- Host id `opencode` on both sides; the CLI dialect row is `agents.OpenCode` in `@cli/cmd/hook_dialect.go` — plain text on every event, deny by exit 2.
- Pre-mutation tool set: `write`, `edit`, `apply_patch`. Post-mutation tool set: `archcore_create_document`, `archcore_update_document`, `archcore_remove_document`, `archcore_add_relation`, `archcore_remove_relation` — the `archcore_<tool>` spelling `foldToolName` in `@cli/cmd/hook_payload.go` folds.
- Spawn budgets: pre-mutation 2 s, post-mutation 4 s (the `timeout` values of `@plugin/plugins/archcore/hooks/hooks.json`), session-start 10 s [assumption].

## Normative Behavior

1. WHEN OpenCode calls `config` and `cfg.mcp.archcore` is absent, the bridge MUST set it to `{"type":"local","command":["archcore","mcp"]}`.
2. IF `cfg.mcp.archcore` is present when `config` runs, THEN the bridge MUST leave that entry unchanged.
3. WHEN OpenCode calls `config`, the bridge MUST append the bundled `skills/` directory to `cfg.skills.paths`.
4. WHEN `tool.execute.before` fires for a tool in the pre-mutation set, the bridge MUST spawn `bin/pre-tool-use` under the spawn contract.
5. WHEN `tool.execute.after` fires for a tool in the post-mutation set, the bridge MUST spawn `bin/post-tool-use` under the spawn contract.
6. WHEN `experimental.chat.system.transform` fires with a `sessionID` that has no cached recap, the bridge MUST spawn `bin/session-start` once for that session.
7. WHILE a session holds a cached recap, the bridge MUST append that recap verbatim to `output.system` on every transform call.
18. IF `experimental.chat.system.transform` fires without a `sessionID`, THEN the bridge MUST leave `output.system` unchanged.
8. WHEN `bin/pre-tool-use` exits 2, the bridge MUST throw `Error(<stderr>)` from `tool.execute.before`.
9. WHEN `bin/pre-tool-use` exits 0 with stdout, the bridge MUST hold that text by `callID` for the matching `tool.execute.after`.
10. WHEN `tool.execute.after` fires for a `callID` with held text, the bridge MUST append that text to `output.output`.
11. WHEN `bin/post-tool-use` exits 0 with stdout, the bridge MUST push one `{type: "text", text: <stdout>}` item onto `output.content`.
12. The bridge MUST pass `output.args` through as `tool_input` without renaming a key.
13. The bridge MUST spawn every launcher with cwd set to the plugin input's `directory`.
14. The bridge MUST NOT decide from a tool name or a path whether a call is blocked.
15. The bridge MUST NOT parse launcher stdout as JSON.
16. The package MUST ship `bin/` and `skills/` byte-identical to `@plugin/plugins/archcore/bin/` and `@plugin/plugins/archcore/skills/` at the release tag.
17. The package MUST NOT include a directory named `test`.

## Constraints & Invariants

- Constraint: the `config` hook is the only registration path a packaged plugin has. In v1.18.34 `packages/opencode/src/project/bootstrap.ts` runs `plugin.init()`, which calls every `config` hook on the cached config object, before any other service; the MCP and skill states read that object lazily afterwards. The ordering is source behavior, not documented API; probes P0 and C confirm it per release, and manual `opencode.json` wiring stays the fallback.
- Constraint: OpenCode has no session-start hook. `experimental.chat.system.transform` runs before every LLM request (`packages/opencode/src/session/llm/request.ts`) and also from agent generation without a `sessionID` (`packages/opencode/src/agent/agent.ts`); behaviors 6 and 18 follow from that.
- Constraint: the plugin input carries `$` only under Bun; the bridge spawns through `node:child_process`, which both runtimes provide.
- Constraint: a `config` hook that throws is logged and ignored by the host; a `tool.execute.before` that throws fails the tool call, and the host stores `errorMessage(error)` as the tool part's error (`packages/opencode/src/session/processor.ts`).
- Constraint: `@plugin/plugins/archcore/bin/lib/plugin-cache-guard.sh` and the manifest walk in `bin/session-start` key on cwd; behavior 13 keeps them from exiting silently from the package directory.
- Constraint: the launchers exit 0 without output below CLI 0.7.0 (`host-adapter-contract.spec`, Failure Behavior 7); the bridge adds no version check of its own.
- Constraint: behaviors 4 and 5 filter by tool name the way `hooks/*.json` matchers do on the other hosts; behavior 14 is what keeps that filter from becoming a guard.
- Invariant: no guard or validation decision executes in TypeScript (`stack-and-tooling.rule`, items 5–6).
- Invariant: every byte the bridge adds to the model's context came from a launcher's stdout.
- Invariant: skill content reaches OpenCode byte-identical to the other hosts; the pack-time copy is the delivery mechanism `opencode-adapter-packaging.adr` admits.

## Failure Behavior

1. IF a launcher file is missing from the package, THEN the bridge MUST write one line naming the path to stderr.
2. IF a launcher file is missing from the package, THEN the bridge MUST allow the call.
3. IF a launcher exits with a code other than 0 or 2, THEN the bridge MUST allow the call without context.
4. IF a launcher exceeds its spawn budget, THEN the bridge MUST kill the child process.
5. IF a launcher exceeds its spawn budget, THEN the bridge MUST allow the call without context.
6. IF a launcher's stdout is empty, THEN the bridge MUST add nothing to `output.system` or `output.output`.
7. IF `tool.execute.after` fires with no held text and no post-mutation match, THEN the bridge MUST leave `output.output` unchanged.

## Conformance

An implementation is conformant when it satisfies behaviors 1–18, holds every invariant, degrades per the failure rules, and passes: `bun test` inside `@plugin/plugins/opencode/` driven by `@plugin/test/fixtures/stdin/opencode/` against a stub launcher (behaviors 4–15); the `opencode` rows of `@plugin/test/structure/host-coverage-matrix.bats` and `@plugin/test/structure/probe-records.bats`; `@plugin/test/structure/host-logic-locality.bats` unchanged; a byte-parity structure test for behavior 16; `@scripts/export-plugin.sh` for behavior 17; and the `opencode` row of `host-probe-protocol.spec` with `pass` on P0, A, B, and C.

Given a project with `.archcore/`, when the agent calls `write` on `.archcore/x.rule.md`, then `tool.execute.before` throws with the CLI's redirect-to-MCP reason and the file does not exist.

## Clarifications

- Delivery through `archcore init` (asked 2026-10-09): yes — this package includes the CLI delivery; the delivery surface is amended and OpenCode joins the plugin host table.
- `mcp.archcore` ownership (asked 2026-10-09): the bridge yields to an existing entry; behaviors 1–2 above; rationale recorded as a decision.
