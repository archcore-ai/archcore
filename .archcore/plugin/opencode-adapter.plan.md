---
title: "OpenCode Adapter: Bridge Package, Shared-Fixture Tests, Release Export, and init Delivery"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "multi-host"
  - "opencode"
  - "plugin"
---

## Goal

Ship `@archcore/opencode-plugin` so an OpenCode session behaves like the four shipped hosts — recap at session start, direct `.archcore/` writes refused with the redirect reason, document tools and skills available, post-write validation appended — and let `archcore init` deliver the package from the host selection screen. Zero edits under `@plugin/plugins/archcore/`; the contract is `opencode-adapter.spec`.

## Tasks

### Phase 1 — Package scaffold and bridge

1. Create `@plugin/plugins/opencode/package.json`: name `@archcore/opencode-plugin`, `type` `module`, `engines.opencode`, `files` with `dist/`, `bin/`, `skills/`.
2. Add a pack script copying `@plugin/plugins/archcore/bin/` and `@plugin/plugins/archcore/skills/` into the package root; gitignore both copies in `@.gitignore`.
3. Write `@plugin/plugins/opencode/src/index.ts` exporting the plugin with the four hooks the spec names.
4. Write the spawn helper: `ARCHCORE_HOST=opencode`, cwd `directory`, stdin envelope, exit translation, per-launcher budget.
5. Implement `config`: set the MCP entry when absent; append the bundled `skills/` path to `cfg.skills.paths`.
6. Implement the per-session recap cache behind `experimental.chat.system.transform`, keyed by `sessionID`.
7. Implement the held pre-mutation context (spec behaviors 9–10) in `@plugin/plugins/opencode/src/index.ts`.

### Phase 2 — Tests on both sides of the stdin contract

8. Write `bun test` files beside `src/` in `@plugin/plugins/opencode/` feeding `@plugin/test/fixtures/stdin/opencode/` to a stub launcher.
9. Assert in those tests the thrown reason on exit 2, appended context on exit 0, and verbatim `tool_input`.
10. Add a bats structure test for byte parity between the bundled copies and their sources after the pack script runs.
11. Enroll `opencode` in `@plugin/test/structure/host-coverage-matrix.bats` with a bridge row read from the TypeScript tool sets.
12. Extend `enrolled_hosts()` in `@plugin/test/structure/probe-records.bats` so the bridge row yields `opencode`.
13. Add the `opencode` row to the records table of `host-probe-protocol.spec` as `deferred:bridge-not-released` (issue #21).
14. Widen the TypeScript-location assertion in `@plugin/test/structure/next-step-mod.bats` to allow `@plugin/plugins/opencode/`.
15. Add a `test-opencode` target to `@plugin/Makefile` and Bun setup plus `bun test` steps to `@.github/workflows/test.yml`.

### Phase 3 — Release export and publish

16. Run the pack script inside `@scripts/export-plugin.sh` before the copy, keeping test files out of any `test/` directory.
17. Extend the `.archcore`-reference grep in `@scripts/export-plugin.sh` to cover `plugins/opencode/`.
18. Add an npm publish job for the package to `@.github/workflows/release.yml`, gated on the release tag, using an `NPM_TOKEN` secret.

### Phase 4 — CLI delivery

19. Add `HostOpenCode` to `hostOrder` and `agentToHost` in `@cli/internal/plugin/plugin.go`.
20. Add an OpenCode row to `@cli/internal/plugin/hosts.go` whose install, remove, and status edit the `plugin` array of `opencode.json`.
21. Teach `@cli/internal/plugin/plan.go`, `@cli/internal/plugin/execute.go`, and `@cli/internal/plugin/evidence.go` a file-edit action and file evidence.
22. Reuse the JSON merge helpers of `@cli/internal/agents/mcp_helpers.go` for the array edit; back up invalid JSON first.
23. Mark OpenCode project-level on the init selection screen (`pluginSelectionNote` in `@cli/cmd/init.go`).
24. Name five hosts in the `--agent` rejection of `@cli/cmd/plugin_run.go`.
25. Pin the row in `@cli/internal/plugin/hosts_test.go`, `@cli/internal/plugin/plan_test.go`, and `@cli/internal/plugin/execute_test.go`.
26. Apply the recorded amendment to `plugin-delivery.spec` and `updating-the-plugin.spec` (Declared Delta below).
27. Update the OpenCode rows of `.archcore/integrations/supported-ai-agents.doc.md` and `@README.md`; report the roster change to the `archcore` global source (`supported-ai-hosts.doc`, read-only here).

### Phase 5 — Live verification

28. Run probes P0, A, A-d, B, C, D on OpenCode 1.18.34 per `host-probe-protocol.spec`; append the dated row.
29. Capture the host's hook payloads into `@plugin/test/fixtures/stdin/opencode/`, including a session-start shape.
30. Set `engines.opencode` in `@plugin/plugins/opencode/package.json` from the Phase 5 probe outcome.
31. If probe C shows `config` runs after MCP start, switch the guide to manual wiring.
32. If probe C shows `config` runs after MCP start, record the outcome in `opencode-mcp-entry-ownership.adr`.

## Acceptance Criteria

- `make verify` passes at the repository root with no diff under `plugin/plugins/archcore/`.
- `bun test` passes inside `plugin/plugins/opencode/`; `find plugin -name '*.ts'` lists only that directory and the Claude Code mod paths.
- `scripts/export-plugin.sh` succeeds and the exported `plugins/opencode/` carries `bin/` and `skills/` and no `test` directory.
- A second `archcore init --agent opencode` run over a wired project leaves `opencode.json` byte-identical.
- The `opencode` probe row records `pass` on P0, A, B, and C with an evidence pointer.

## Dependencies

- Archcore CLI 0.10.0 or newer on the probe machine — the `archcore hooks opencode <leaf>` leaves (`ea474ad`).
- Bun on CI runners and the developer machine — absent locally on 2026-10-09.
- npm publish rights on the `@archcore` scope.
- OpenCode 1.17.13 or newer for the verified hook set (`opencode-host-support.rnd`); 1.18.34 installed locally.
- `opencode-adapter-packaging.adr` (accepted) and `opencode-mcp-entry-ownership.adr` (draft) for the package shape and the `config` hook.
- Issue #21 for the live probe run on the four shipped hosts; the OpenCode row joins that run.

## Declared Delta

- Route: `capability` (size L); raised by M = stone (`host-adapter-contract.spec`, `stack-and-tooling.rule`, `opencode-adapter-packaging.adr` accepted) and R = `external-contract` (a published npm package against OpenCode's plugin API and a TS–shell stdin contract pinned by two suites).
- creates: `opencode-adapter` → `opencode-adapter.spec`.
- modifies: `plugin-delivery` → verdict `spec-wrong` against `plugin-delivery.spec` and `updating-the-plugin.spec`. Evidence: both state "OpenCode ships no plugin"; the maintainer's 2026-10-09 decision to deliver `@archcore/opencode-plugin` from `archcore init` makes that sentence stale. Amendment: an OpenCode row in the per-host install table (a file edit on the `plugin` array of `opencode.json`, no host command, project-level), `opencode` among the hosts `--agent` accepts, and the update step addressing OpenCode with no command [assumption: the host resolves an unpinned package at startup].
- retires: none.
- decision: `mcp-archcore-entry-ownership` → `opencode-mcp-entry-ownership.adr`.
- intent_gap: no — intent recorded in `opencode-adapter-packaging.adr` and `opencode-host-support.rnd`; the intent instrument was skipped on the compression path.
- Π: machine (the canon and code cited above), user (two questions, both answered 2026-10-09), empirical (the six probes of the rnd's Next Action, run through `host-probe-protocol.spec` as Phase 5; no spike document because the probes need the bridge to exist first).
- Illustrate: not engaged — no UI, skill, or new operator flow. Runbook: engaged — install and verify steps → `opencode-plugin-install.guide`.
