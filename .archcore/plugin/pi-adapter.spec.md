---
title: "Pi Adapter — @archcore/pi-plugin Extension Bridge, Bundled Skills, and pi Hook Dialect"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "hooks"
  - "multi-host"
  - "pi"
---

## Purpose & Scope

This spec is normative for the Pi adapter: the npm package `@archcore/pi-plugin` developed under `plugin/plugins/pi/`, the handlers its extension registers, the spawn contract between those handlers and `@plugin/plugins/archcore/bin/`, and the `pi` row of the CLI hook dialects. Depended on by Pi sessions in a project with `.archcore/`, by the Pi row of `plugin-delivery.spec`, and by the `bun test` and bats suites that pin `@plugin/test/fixtures/stdin/pi/`. Out of scope: guard and validation policy (`hook-runtime.spec`) and every other host.

Host facts come from `earendil-works/pi` commit `fcebc09` (Pi 1.5.0): `packages/coding-agent/docs/extensions.md`, `docs/packages.md`, `docs/mcp.md`, `src/core/extensions/types.ts`, `packages/agent/src/agent-loop.ts`.

## Surface

- Package manifest `plugin/plugins/pi/package.json`: name `@archcore/pi-plugin`, keyword `pi-package`, `pi.extensions: ["./dist/index.js"]`, `pi.skills: ["./skills"]`, `peerDependencies` `@earendil-works/pi-coding-agent: "*"`, no `dependencies` (`pi-adapter-packaging.adr`).
- Extension: default export factory receiving `ExtensionAPI` (`plugin/plugins/pi/src/index.ts`); spawn module `plugin/plugins/pi/src/spawn.ts` packed from `@plugin/plugins/opencode/src/spawn.ts`.
- Events used: `before_agent_start` (returns `message`), `tool_call` (returns `{block, reason}`), `tool_result` (returns a partial patch of `content`).
- Pre-mutation tool set: `write`, `edit` — both carry `path`. Post-mutation tool set: `mcp__archcore__create_document`, `mcp__archcore__update_document`, `mcp__archcore__remove_document`, `mcp__archcore__add_relation`, `mcp__archcore__remove_relation`.
- MCP registration: `pi.registerMcpServer("archcore", {command: "archcore", args: ["mcp"], exposure: "direct"})`; an `mcp.json` entry named `archcore` takes precedence (Pi behavior).
- Spawn contract: env `ARCHCORE_HOST=pi`; cwd `ctx.cwd`; stdin one JSON object then closed. Tool events send `{"tool_name", "tool_input"}`; session start sends `{"cwd", "session_id"}` with `ctx.sessionManager.getSessionId()`. Deny is exit 2 with the reason on stderr; context is plain text on stdout.
- CLI: `pi` row in `@cli/cmd/hook_dialect.go` with plain-text session and tool envelopes, `denyExitTwo`, `preToolContext: true`.
- Spawn budgets: pre-mutation 2 s, post-mutation 4 s, session start 10 s [assumption], as in `opencode-adapter.spec`.

## Normative Behavior

1. WHEN the extension loads and `pi.registerMcpServer` exists, the extension MUST register the `archcore` server of Surface.
2. WHEN the extension loads, the extension MUST run `bin/cli-gte` once with the first CLI version that carries the `pi` leaf.
3. WHILE the held gate result is `no`, the extension MUST NOT spawn a launcher.
4. WHEN `before_agent_start` fires for a session without a recap entry, the extension MUST store one `bin/session-start` spawn promise under its session id.
5. WHEN that promise resolves to non-empty text for the first time, the extension MUST return it as `message` with `display: false`.
6. WHEN `before_agent_start` fires for a session whose recap was returned, the extension MUST return nothing.
7. WHEN `tool_call` fires for a tool in the pre-mutation set, the extension MUST spawn `bin/pre-tool-use`.
8. WHEN `bin/pre-tool-use` exits 2, the extension MUST return `{block: true, reason}` with stderr minus its trailing newline.
9. WHEN `bin/pre-tool-use` exits 0 with stdout, the extension MUST hold that text by `toolCallId`.
10. WHEN `tool_result` fires for a `toolCallId` with held text, the extension MUST append one text item to `content`.
11. WHEN `tool_result` fires for a tool in the post-mutation set, the extension MUST spawn `bin/post-tool-use`.
12. WHEN `bin/post-tool-use` exits 0 with stdout, the extension MUST append one text item to `content`.
13. WHEN the extension returns a `content` patch, the extension MUST return the event's `structuredContent` unchanged.
14. The extension MUST pass `input` through as `tool_input` without renaming a key.
15. The extension MUST NOT decide from a tool name or a path whether a call is blocked.
16. The extension MUST NOT throw from `before_agent_start` or `tool_result`.
17. The package MUST ship `bin/` and `skills/` byte-identical to `@plugin/plugins/archcore/` at the release tag.
18. The package `src/spawn.ts` MUST be byte-identical to `@plugin/plugins/opencode/src/spawn.ts`.

## Constraints & Invariants

- Constraint: `session_start` cannot add context; `before_agent_start` `message` is stored in the session and sent to the model.
- Constraint: `tool_call` has no context slot, so pre-call advice rides `tool_result`, as on OpenCode.
- Constraint: Pi's default MCP exposure `codemode` hides tools from the model; `exposure: "direct"` is required.
- Constraint: `tool_call` fires for MCP calls made inside codemode scripts too, with `parentToolCallId` set.
- Constraint: `bash` is outside the pre-mutation set, so a shell write to `.archcore/` is not guarded.
- Constraint: Pi requires Node 22.19.0 or newer; the package ships built JavaScript in `dist/`.
- Invariant: no guard or validation decision executes in TypeScript (`stack-and-tooling.rule` items 5–6, extended by `pi-adapter-packaging.adr`).
- Invariant: every byte added to the model's context came from a launcher's stdout.

## Failure Behavior

1. IF `pi.registerMcpServer` is absent, THEN the extension MUST write one stderr line and skip MCP registration.
2. IF a launcher file is missing, THEN the extension MUST allow the call and write one stderr line naming the path.
3. IF a launcher exits with a code other than 0 or 2, THEN the extension MUST allow the call without context.
4. IF a launcher exceeds its budget, THEN the extension MUST kill the child process.
5. IF a launcher exceeds its budget, THEN the extension MUST allow the call without context.
6. IF the session-start spawn fails, THEN the extension MUST resolve that session's promise to an empty recap.

## Conformance

An implementation conforms when it satisfies behaviors 1–18, holds the invariants, degrades per Failure Behavior, and passes: `bun test` inside `plugin/plugins/pi/` driven by `@plugin/test/fixtures/stdin/pi/` against a stub launcher; a parity structure test for behaviors 17–18; Go table tests for the `pi` dialect; the `pi` cases of `@plugin/test/unit/hook-launchers.bats`; and the `pi` row of `host-probe-protocol.spec` with `pass` on P0, A, B, and C.

Given CLI support and a project with `.archcore/`, when the agent calls `write` on `.archcore/x.rule.md`, then `tool_call` returns `{block: true}` with the CLI reason and the file does not exist.
