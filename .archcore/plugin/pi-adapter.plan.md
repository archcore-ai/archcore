---
title: "Pi Adapter: TypeScript Bridge Package on the OpenCode Shape, pi Hook Dialect, npm Release, and CLI Delivery"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "multi-host"
  - "pi"
  - "plugin"
---

## Goal

Ship `@archcore/pi-plugin` so a Pi session gets the session recap, the `.archcore/` write guard, post-write validation, the `archcore` MCP server with direct exposure, and the skills. `archcore init`, `archcore plugin`, and `archcore update` deliver and refresh the package. The contract is `pi-adapter.spec`; the packaging decision is `pi-adapter-packaging.adr` (accepted 2026-11-04).

Host facts: Pi 1.5.0, `earendil-works/pi` commit `fcebc09`. No public user count was found on 2026-11-04.

## Tasks

### Phase 1 — Shared spawn and CLI dialect

1. Add a `host` parameter to `run` in `@plugin/plugins/opencode/src/spawn.ts`.
2. Pass `"opencode"` from every caller in `@plugin/plugins/opencode/src/bridge.ts` and `@plugin/plugins/opencode/src/v2.ts`.
3. Run `bun test` in `plugin/plugins/opencode/` and keep it green.
4. Add `AgentID` `pi` to `@cli/internal/agents/agents.go`.
5. Add the `pi` row to `@cli/cmd/hook_dialect.go`: plain-text envelopes, `denyExitTwo`, `preToolContext: true`.
6. Add `pi` to the `case` of `@plugin/plugins/archcore/bin/lib/normalize-stdin.sh`.
7. Add Go table tests for the `pi` leaf with fixtures under `@plugin/test/fixtures/stdin/pi/`.
8. Add `pi` cases to `@plugin/test/unit/hook-launchers.bats`.

### Phase 2 — Package and bridge

9. Create `@plugin/plugins/pi/package.json` per the spec's Surface.
10. Write `@plugin/plugins/pi/scripts/pack.sh` copying `bin/`, `skills/`, and `src/spawn.ts`.
11. Make the pack step build `dist/index.js` with `bun build --target node --format esm`.
12. Gitignore `bin/`, `skills/`, `dist/`, `src/spawn.ts`, and `node_modules/` under `plugin/plugins/pi/`.
13. Write `@plugin/plugins/pi/src/index.ts` registering MCP, the CLI gate, and three handlers.
14. Implement the per-session recap promise for `before_agent_start` (behaviors 4–6).
15. Implement `tool_call` block translation and held context by `toolCallId` (behaviors 7–9).
16. Implement `tool_result` appends that keep `structuredContent` (behaviors 10–13).
17. Wrap `before_agent_start` and `tool_result` so that no error escapes.
18. Write `bun test` files in `plugin/plugins/pi/` for behaviors 1–16 and Failure Behavior 1–6.
19. Add a parity structure test for spec behaviors 17–18 under `@plugin/test/structure/`.
20. Add a `test-pi` target to `@plugin/Makefile` and a Pi step to `@.github/workflows/test.yml`.
21. Add the `pi` row to `@plugin/test/structure/host-coverage-matrix.bats` and `@plugin/test/structure/probe-records.bats` in one commit.
22. Add the `pi` row to `host-probe-protocol.spec` as deferred until Phase 5.
23. Amend `stack-and-tooling.rule` items 4, 21, 22, and 30 for `plugins/pi/`.

### Phase 3 — Release

24. Exclude `plugins/pi/` from `@scripts/export-plugin.sh` and pin it in `@plugin/test/structure/release-blocklist.bats`.
25. Add an `npm pack --dry-run` check for `dist/index.js`, `bin/`, `skills/`, and no `test` directory.
26. Add a `publish-pi` job to `@.github/workflows/release.yml` that skips a version npm already has.

### Phase 4 — CLI delivery

27. Add `@cli/internal/agents/pi.go` and its test: detection `.pi/` directory, instructions `AGENTS.md`, no MCP writer.
28. Add `HostPi` to `hostOrder` and `agentToHost` in `@cli/internal/plugin/plugin.go`.
29. Add the row to `@cli/internal/plugin/hosts.go`: install `pi install npm:@archcore/pi-plugin`.
30. Set its update to `pi update npm:@archcore/pi-plugin` and its remove to `pi remove npm:@archcore/pi-plugin`.
31. Read evidence from `pi list` text or the `packages` key of `~/.pi/agent/settings.json` in `@cli/internal/plugin/evidence.go`.
32. Add row tests to `@cli/internal/plugin/hosts_test.go`, `plan_test.go`, `execute_test.go`, and `evidence_test.go`.
33. Correct the host-count comments in `@cli/internal/plugin/plugin.go`, `@cli/cmd/plugin_run.go`, and `@cli/cmd/init.go`.
34. Amend `plugin-delivery.spec` and `updating-the-plugin.spec` per the Declared Delta.
35. Add Pi to `supported-ai-agents.doc`, `cli-hooks-reference.doc`, and `@README.md`.
36. Report the roster change to the `archcore` global `supported-ai-hosts.doc` through its project.
37. Remove `pi` from the providers without an adapter in `wrapper-hosts-defer-to-inner-agent-adapter.adr`.

### Phase 5 — Live verification

38. Install Pi 1.5.0 or newer on the probe machine.
39. Run probes P0, A, B, and C per `host-probe-protocol.spec`.
40. Check that the Archcore MCP tools appear to the model with `exposure: "direct"`.
41. Check that `pi install` runs without a TTY; if it prompts, switch the CLI row to a printed command.
42. Capture live payloads and replace the synthetic fixtures.

## Acceptance Criteria

- `make verify` passes; `bun test` passes in `plugin/plugins/opencode/` and `plugin/plugins/pi/`.
- `pi install npm:@archcore/pi-plugin` loads the extension; the Archcore skills and MCP tools appear in a session.
- In a Pi session, a `write` on `.archcore/x.rule.md` returns a blocked tool result with the CLI reason.
- `archcore plugin install --agent pi` installs the package; a second run runs no install command.
- The `pi` probe row records `pass` on P0, A, B, and C with an evidence pointer.

## Dependencies

- Pi 1.5.0 or newer, Node 22.19.0 or newer, and Bun on the machine and on CI.
- npm publish rights on the `@archcore` scope and the `NPM_TOKEN` secret.
- A CLI release carrying the `pi` leaf before the package release; its version becomes the `bin/cli-gte` argument.

## Clarifications

- 2026-11-04: the maintainer chose CLI delivery as for OpenCode, and accepted the decisions this plan needs.

## Declared Delta

- create spec `pi-adapter` (draft, 2026-11-04).
- create adr `pi-adapter-packaging` (accepted, 2026-11-04).
- amend spec `opencode-adapter` — `spawn.ts` takes the host id as a parameter.
- amend spec `plugin-delivery` — Pi row; `pi` among the `--agent` hosts.
- amend spec `updating-the-plugin` — `pi update` for Pi.
- amend rule `stack-and-tooling` — TypeScript exception for `plugins/pi/`.
- amend doc `supported-ai-agents` and doc `cli-hooks-reference` — Pi rows.
- amend adr `wrapper-hosts-defer-to-inner-agent-adapter` — Pi gains an adapter.
