---
title: "OpenCode Adapter — @archcore/opencode-plugin Bridge, Bundled Skills and Agents, and Package Contract"
status: draft
tags:
  - "component:plugin"
  - "hooks"
  - "multi-host"
  - "opencode"
  - "plugin"
---

## Purpose & Scope

This spec is normative for the OpenCode adapter: the npm package `@archcore/opencode-plugin` developed under `@plugin/plugins/opencode/`, the hooks its `server` function returns, the spawn contract between those hooks and the launchers in `@plugin/plugins/archcore/bin/`, and the trees and agent definitions the package bundles. Depended on by OpenCode and Kilo Code sessions in a project that carries `.archcore/`, by the OpenCode row of `plugin-delivery.spec`, and by the bats and `bun test` suites that pin `@plugin/test/fixtures/stdin/opencode/`. Out of scope: the guard, validation, and context policy inside `archcore hooks opencode <leaf>` (`hook-runtime.spec`), the `archcore init` delivery (`plugin-delivery.spec`), and every other host.

## Surface

- Package: source `@plugin/plugins/opencode/src/` (`index.ts` exports the module, `bridge.ts` holds the hooks), no runtime dependency, manifest `@plugin/plugins/opencode/package.json`, pack step `@plugin/plugins/opencode/scripts/pack.sh`. `main` and `exports["./server"]` point at `dist/index.js` (`opencode-plugin-ships-built-javascript.adr`). `engines.opencode` is a `>=` range; Kilo Code compares the same key against its own 7.x version.
- Module shape: default export `{ id: "@archcore/opencode-plugin", server }`, the v1 form `readV1Plugin` accepts in `packages/opencode/src/plugin/shared.ts` (OpenCode v1.18.34). An unpinned `plugin` entry resolves to `@latest`.
- Bundled trees, written by the pack step and absent from git: `bin/` from `@plugin/plugins/archcore/bin/`; `skills/` from `@plugin/plugins/archcore/skills/`, each `SKILL.md` `name:` line renamed to `archcore-<name>` (`opencode-archcore-prefixed-skill-names.adr`); `agents/<name>.md` from `@plugin/plugins/archcore/agents/`.
- Agent definitions: `@plugin/plugins/opencode/src/agents.ts` maps `archcore-assistant` and `archcore-auditor` to `description`, `mode: "subagent"`, `steps`, and `permission`. The prompt is the body of the bundled `agents/<name>.md` below its frontmatter.
- Hooks `server()` returns (`packages/plugin/src/index.ts`): `config`, `tool.execute.before`, `tool.execute.after`, `experimental.chat.system.transform`.
- `tool.execute.after` output: a built-in tool passes `{title, output, metadata}`; an MCP tool passes the MCP `CallToolResult`, and OpenCode builds the model text from its `content[]` after the hook (`packages/opencode/src/session/tools.ts`).
- Spawn contract: env `ARCHCORE_HOST=opencode`; cwd = the plugin input's `directory`; stdin = one JSON object, then closed. Tool events send `{"tool_name": <input.tool>, "tool_input": <output.args>}`; session start sends `{"cwd": <directory>}` with no `session_id`. A deny is exit 2 with the reason on stderr; context is plain text on stdout (`agents.OpenCode` row of `@cli/cmd/hook_dialect.go`).
- Pre-mutation tool set: `write`, `edit`, `apply_patch`. Post-mutation tool set: `archcore_create_document`, `archcore_update_document`, `archcore_remove_document`, `archcore_add_relation`, `archcore_remove_relation` — the `<server>_<tool>` id of `packages/opencode/src/mcp/catalog.ts`, which `foldToolName` in `@cli/cmd/hook_payload.go` folds.
- CLI gate: `bin/cli-gte 0.10.0`. Release v0.10.0 is the first that carries the plain-text OpenCode session envelope and the `archcore_*` fold (commit `ea474ad`).
- Spawn budgets: CLI gate 2 s [assumption], pre-mutation 2 s, post-mutation 4 s (the `timeout` values of `@plugin/plugins/archcore/hooks/hooks.json`), session start 10 s [assumption].

## Normative Behavior

1. WHEN OpenCode calls `config` and `cfg.mcp.archcore` is absent, the bridge MUST set it to `{"type":"local","command":["archcore","mcp"]}`.
2. IF `cfg.mcp.archcore` is present when `config` runs, THEN the bridge MUST leave that entry unchanged.
3. WHEN OpenCode calls `config`, the bridge MUST append the bundled `skills/` path to `cfg.skills.paths`.
4. WHEN OpenCode calls `config` and `cfg.skills` is absent, the bridge MUST create it with an empty `paths` array first.
5. WHEN OpenCode calls `config` and `cfg.agent` lacks an Archcore agent key, the bridge MUST set that key from its agent definition.
6. IF `cfg.agent` already holds an Archcore agent key, THEN the bridge MUST leave that entry unchanged.
7. WHEN `server()` runs, the bridge MUST run the CLI gate once and hold its result.
8. WHILE the held CLI gate result is `no`, the bridge MUST NOT spawn a launcher.
9. WHEN `tool.execute.before` fires for a tool in the pre-mutation set, the bridge MUST spawn `bin/pre-tool-use`.
10. WHEN `bin/pre-tool-use` exits 2, the bridge MUST throw an `Error` whose message is stderr without its trailing newline.
11. WHEN `bin/pre-tool-use` exits 0 with stdout, the bridge MUST hold that text by `callID`.
12. WHEN `tool.execute.after` fires for a `callID` with held text, the bridge MUST append that text to `output.output`.
13. WHEN `tool.execute.after` fires for a tool in the post-mutation set, the bridge MUST spawn `bin/post-tool-use`.
14. WHEN `bin/post-tool-use` exits 0 with stdout, the bridge MUST push one `{type: "text", text: <stdout>}` item onto `output.content`.
15. WHEN `experimental.chat.system.transform` fires with a `sessionID` that has no recap entry, the bridge MUST store one `bin/session-start` spawn promise under it.
16. WHEN `experimental.chat.system.transform` fires with a `sessionID`, the bridge MUST await that session's stored promise.
17. WHEN the awaited recap is not empty, the bridge MUST append the recap verbatim as one element of `output.system`.
18. IF `experimental.chat.system.transform` fires without a `sessionID`, THEN the bridge MUST leave `output.system` unchanged.
19. The bridge MUST pass the call's args through as `tool_input` without renaming a key.
20. The bridge MUST spawn every launcher with cwd set to the plugin input's `directory`.
21. The bridge MUST NOT decide from a tool name or a path whether a call is blocked.
22. The bridge MUST NOT parse launcher stdout as JSON.
23. The bridge MUST NOT throw from `config`, `tool.execute.after`, or `experimental.chat.system.transform`.
24. The package MUST ship `bin/` byte-identical to `@plugin/plugins/archcore/bin/` at the release tag.
25. The package MUST ship every `skills/` file byte-identical to its source, except the renamed `name:` line of each `SKILL.md`.
26. The package MUST ship every `agents/<name>.md` byte-identical to its source.
27. The bridge MUST take each agent's `description` and `steps` from the source frontmatter `description` and `maxTurns`.
28. The bridge MUST allow an agent exactly the read tools and `archcore_<tool>` tools its source `tools:` list names.
29. The npm tarball MUST NOT contain a directory named `test`.
30. WHILE the held CLI gate result is `__NO_CLI__`, the bridge MUST spawn no launcher except `bin/session-start`.
31. WHEN the bridge allows the `read` tool, the bridge MUST keep `ask` for `*.env` and `*.env.*` paths.

## Constraints & Invariants

- Constraint: the `config` hook is the only registration path a packaged plugin has. In v1.18.34 and v1.19.3, `packages/opencode/src/project/bootstrap.ts` runs every `config` hook on the cached config object before MCP, skill, command, and agent state read it. The ordering is source behavior, not documented API; probes P0 and C confirm it per release.
- Constraint: OpenCode has no session-start hook. `experimental.chat.system.transform` runs before every LLM request (`packages/opencode/src/session/llm/request.ts`), including the title request of the same session, which can run beside the first main request; behaviors 15–16 keep the spawn single. The recap also reaches title requests; that token cost is accepted.
- Constraint: the transform also runs from agent generation without a `sessionID` (`packages/opencode/src/agent/agent.ts`); behavior 18 follows.
- Constraint: session start sends no `session_id`, so the CLI dedup key is empty and dedup is off (`deriveDedupKey` in `@cli/cmd/hook_command.go`). The per-session promise is the dedup; a restarted OpenCode process recaps a resumed session again.
- Constraint: CLI releases 0.7.0–0.9.x answer the OpenCode leaves with a JSON session envelope and without the `archcore_*` fold, so the guard would deny the document tools (`plugin-cli-compatibility.rule`, requirement 9). `bin/cli-gte 0.7.0` inside the launchers cannot tell those releases apart; behaviors 7–8 can.
- Constraint: a `config` hook that throws is logged and ignored; any other hook that throws fails the call, and the host stores `errorMessage(error)` as the tool error (`packages/opencode/src/session/processor.ts`).
- Constraint: OpenCode gives `gpt-` models `apply_patch` instead of `edit` and `write` (`packages/opencode/src/tool/registry.ts`); the pre-mutation set covers both families. The `bash` tool is outside the set, so a shell write to `.archcore/` is not guarded.
- Constraint: OpenCode applies the last matching permission rule (`packages/opencode/src/permission/index.ts`), and an agent's own rules follow the defaults. A plain `read: allow` after the defaults would drop their `ask` on env files; behavior 31 keeps it. Without a CLI, `bin/session-start` prints the install instructions, which is why behavior 30 still runs it.
- Constraint: an `engines.opencode` mismatch makes OpenCode skip the plugin with a session error event (`shared.ts`); a `>=` range keeps newer OpenCode and Kilo Code releases loading it.
- Constraint: `bin/session-start` and `bin/lib/plugin-cache-guard.sh` key on cwd; behavior 20 points them at the project, not at the package directory.
- Constraint: behaviors 9 and 13 filter by tool name the way `hooks/*.json` matchers do on the other hosts; behavior 21 keeps that filter from becoming a guard.
- Invariant: no guard or validation decision executes in TypeScript (`stack-and-tooling.rule`, items 5–6).
- Invariant: every byte the bridge adds to the model's context came from a launcher's stdout or a bundled file.

## Failure Behavior

1. IF a launcher file is missing from the package, THEN the bridge MUST write one line naming the path to stderr.
2. IF a launcher file is missing from the package, THEN the bridge MUST allow the call.
3. IF a launcher exits with a code other than 0 or 2, THEN the bridge MUST allow the call without context.
4. IF a launcher exceeds its spawn budget, THEN the bridge MUST kill the child process.
5. IF a launcher exceeds its spawn budget, THEN the bridge MUST allow the call without context.
6. IF a launcher's stdout is empty, THEN the bridge MUST add nothing to `output.system` or `output.output`.
7. IF `tool.execute.after` fires with no held text and no post-mutation match, THEN the bridge MUST leave `output` unchanged.
8. IF `output.content` is not an array when behavior 14 applies, THEN the bridge MUST leave `output` unchanged.
9. IF the session-start spawn fails, THEN the bridge MUST resolve that session's promise to an empty recap.
10. IF the CLI gate cannot run, THEN the bridge MUST hold the result `no` and write one line to stderr.

## Conformance

An implementation is conformant when it satisfies behaviors 1–31, holds every invariant, degrades per the failure rules, and passes: `bun test` inside `@plugin/plugins/opencode/` driven by `@plugin/test/fixtures/stdin/opencode/` against a stub launcher (behaviors 1–23, 27–28, and 30–31); the `opencode` rows of `@plugin/test/structure/host-coverage-matrix.bats`, `@plugin/test/structure/probe-records.bats`, and `@plugin/test/structure/hook-launchers.bats`; `@plugin/test/structure/host-logic-locality.bats` unchanged; a byte-parity structure test after the pack step (behaviors 24–26); an `npm pack --dry-run` listing check (behavior 29 and `dist/index.js`); and the `opencode` row of `host-probe-protocol.spec` with `pass` on P0, A, B, and C.

Given a project with `.archcore/` and CLI 0.10.0 or newer, when the agent calls `write` on `.archcore/x.rule.md`, then `tool.execute.before` throws with the CLI's redirect-to-MCP reason and the file does not exist.

## Clarifications

- Delivery through `archcore init` (asked 2026-10-09): yes, through `plugin-delivery.spec`.
- `mcp.archcore` ownership (asked 2026-10-09): the bridge yields to an existing entry (`opencode-mcp-entry-ownership.adr`).
- Command names, agents, and build (asked 2026-10-18): `/archcore-*` commands, both agents in this adapter, and a built `dist/`.
