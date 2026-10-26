---
title: "OpenCode Adapter: Bridge Package, Skills and Agents, Shared-Fixture Tests, Release Export, and init Delivery"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "multi-host"
  - "opencode"
  - "plugin"
---

## Goal

Ship `@archcore/opencode-plugin` so an OpenCode session behaves like the four shipped hosts. It gets the recap at session start and has direct `.archcore/` writes refused with the redirect reason. It has the document tools, the `/archcore-init`, `/archcore-plan`, `/archcore-document`, and `/archcore-review` commands, and both Archcore agents, and it appends post-write validation. Then let `archcore init` deliver the package from the host selection screen. The plan makes zero edits under `@plugin/plugins/archcore/`; the contract is `opencode-adapter.spec`.

Status: Phases 1-4 shipped in v0.14.0-v0.14.2. OpenCode 2.x support and the global install followed, the global install since v0.14.2. Phase 5 remains open.

## Tasks

### Phase 1 — Package scaffold and bridge

1. Create `@plugin/plugins/opencode/package.json`: name `@archcore/opencode-plugin`, `type` `module`, `main` and `exports["./server"]` `dist/index.js`.
2. Set `engines.opencode` to `>=1.18.34` and `files` to `dist/`, `bin/`, `skills/`, `agents/`.
3. Declare no dependency; type the four hooks locally in `@plugin/plugins/opencode/src/bridge.ts`.
4. Write `@plugin/plugins/opencode/scripts/pack.sh` copying `bin/` and `agents/*.md` from `@plugin/plugins/archcore/`.
5. Make the pack step copy `skills/` and rewrite each `SKILL.md` `name:` line to `archcore-<name>`.
6. Make the pack step run `bun build src/index.ts --outdir dist --target node --format esm`.
7. Gitignore `bin/`, `skills/`, `agents/`, `dist/`, and `node_modules/` under `plugin/plugins/opencode/` in `@.gitignore`.
8. Write `@plugin/plugins/opencode/src/spawn.ts`: env, cwd `directory`, closed stdin, exit translation, per-launcher budget, kill on timeout.
9. Write `@plugin/plugins/opencode/src/index.ts` exporting `{ id, server, setup }`; `server` returns the four hooks the spec names and `setup` serves OpenCode 2.x.
10. Run the CLI gate `bin/cli-gte 0.10.0` once inside `server()` and disable every spawn below it.
11. Implement `config`: MCP entry when absent, `skills.paths` append with `skills` created, agent keys when absent.
12. Write `@plugin/plugins/opencode/src/agents.ts` with both agent definitions and the prompt read from bundled `agents/<name>.md`.
13. Implement the per-session recap promise behind `experimental.chat.system.transform`, keyed by `sessionID`.
14. Implement the held pre-mutation context by `callID` and the post-mutation `content[]` push.
15. Wrap `config`, `tool.execute.after`, and the transform so that no error escapes them.

### Phase 2 — Tests on both sides of the stdin contract

16. Rewrite `@plugin/test/fixtures/stdin/opencode/mcp-update.json` to the `archcore_update_document` tool name.
17. Rewrite both write fixtures to OpenCode's `filePath` argument key.
18. Add `edit`, `apply_patch`, and session-start fixtures in the spec's envelope.
19. Write `bun test` files beside `src/` driving those fixtures through a stub launcher.
20. Assert the thrown reason without trailing newline, held and appended context, verbatim `tool_input`, and one session spawn.
21. Assert agent `description`, `steps`, and permission parity against `@plugin/plugins/archcore/agents/*.md` frontmatter.
22. Add a bats structure test for `bin/`, `skills/`, and `agents/` parity, masking the renamed `name:` line.
23. Make that parity test skip with a message when the pack output is absent, and fail when `CI` is set.
24. Add an `opencode` case to `@plugin/test/unit/hook-launchers.bats`.
25. Give `@plugin/test/structure/host-coverage-matrix.bats` a bridge row type read from the TypeScript tool sets.
26. Extend `enrolled_hosts()` in `@plugin/test/structure/probe-records.bats` so the bridge row yields `opencode`.
27. Add the `opencode` row to `host-probe-protocol.spec` as `deferred:bridge-not-released` (issue #21).
28. Land tasks 25, 26, and 27 in one commit, because the row count must equal the enrolled host count.
29. Add a `test-opencode` target to `@plugin/Makefile` that skips with a message when `bun` is missing, and fails when `CI` is set.
30. Add Bun setup, the pack step, and `bun test` to `@.github/workflows/test.yml`.

### Phase 3 — Release export and publish

31. Remove `plugins/opencode/` from the tree `@scripts/export-plugin.sh` writes, because npm is its channel.
32. Pin that exclusion in `@plugin/test/structure/release-blocklist.bats`.
33. Add an `npm pack --dry-run` check for `dist/index.js` and for no `test` directory.
34. Add a `publish-opencode` job to `@.github/workflows/release.yml` after `publish-plugin` that writes the tag version and publishes with `NPM_TOKEN`, and that skips a version npm already has.

### Phase 4 — CLI delivery

35. Add `HostOpenCode` to `hostOrder` and `agentToHost` in `@cli/internal/plugin/plugin.go`.
36. Add an OpenCode row to `@cli/internal/plugin/hosts.go`: CLI `opencode`, no listing, with a `Since2` row for OpenCode 2.x.
37. Set its install to `opencode plugin -g @archcore/opencode-plugin` on 1.x and `opencode plugin add @archcore/opencode-plugin` on 2.x, with `Command.Dir` set to the project root.
38. Set its remove to a printed note naming the `plugin` array of the global `opencode.json` on 1.x, and to `opencode plugin remove @archcore/opencode-plugin` on 2.x.
39. Set its update to `opencode plugin update @archcore/opencode-plugin` on 2.x and leave it empty on 1.x.
40. Pass the project root into `CollectEvidence` in `@cli/internal/plugin/evidence.go`.
41. Read OpenCode evidence for the package name under the keys `plugin` and `plugins` from the global config, and from `.opencode/opencode.json[c]` and `opencode.json[c]` in the project root and each parent up to the git working-tree root. Read the major version from `opencode --version`.
42. Skip OpenCode in `archcore update` on 1.x and in `archcore plugin` runs without a resolved project root.
43. Update the seven tests that pin "OpenCode ships no plugin" in `@cli/cmd/` and `@cli/internal/plugin/`.
44. Add row tests to `@cli/internal/plugin/hosts_test.go`, `plan_test.go`, `execute_test.go`, and `evidence_test.go`.
45. Correct the host-count comments in `@cli/internal/plugin/plugin.go`, `@cli/cmd/plugin_run.go`, and `@cli/cmd/init.go`.
46. Apply the amendment to `plugin-delivery.spec` and `updating-the-plugin.spec` (Declared Delta below).
47. Update the OpenCode statements of `supported-ai-agents.doc` and `agent-hooks-integration.doc` that call the plugin undelivered.
48. Update the OpenCode row of `@README.md`; report the roster change to the `archcore` global `supported-ai-hosts.doc`.

### Phase 5 — Live verification

49. Remove or override the global `mcp.archcore` entry in `~/.config/opencode/opencode.json` on the probe machine.
50. Run probes P0, A, A-d, B, C, D on OpenCode 1.18.34 and 1.19.3 per `host-probe-protocol.spec`.
51. Check that `/archcore-plan` runs the skill and that both agents appear as subagents.
52. Capture the live hook payloads into `@plugin/test/fixtures/stdin/opencode/`, replacing the synthetic ones.
53. Set `engines.opencode` from the lowest version that passed.
54. Load the package once under Kilo CLI and record the result in `agent-host-landscape.rnd`.
55. If probe C shows `config` runs after MCP start, switch the guide to manual wiring.
56. If probe C shows `config` runs after MCP start, record the outcome in `opencode-mcp-entry-ownership.adr`.
57. Restart OpenCode after a newer package release and record whether the unpinned entry refreshes.

## Acceptance Criteria

- `make verify` passes at the repository root with no diff under `plugin/plugins/archcore/`.
- `bun test` passes inside `plugin/plugins/opencode/`; outside that directory the set of tracked `.ts` files is unchanged.
- `scripts/export-plugin.sh` succeeds without `plugins/opencode/`, and `npm pack --dry-run` lists `dist/index.js`, `bin/`, `skills/`, `agents/`, and no `test` directory.
- In an OpenCode session, `/archcore-init`, `/archcore-plan`, `/archcore-document`, and `/archcore-review` run the Archcore skills, and `/init` and `/review` stay the OpenCode built-ins.
- A second `archcore init --agent opencode` run over a wired project leaves `.opencode/opencode.json` byte-identical.
- The `opencode` probe row records `pass` on P0, A, B, and C with an evidence pointer.

## Dependencies

- Archcore CLI 0.10.0 or newer — the first release with the corrected OpenCode leaves (`ea474ad`); 0.13.1 installed locally.
- Bun 1.4.3, installed locally and on CI (`@.github/workflows/test.yml`).
- npm publish rights on the `@archcore` scope and an `NPM_TOKEN` repository secret.
- OpenCode 1.18.34 installed locally; 1.19.3 is the latest release on 2026-10-18.
- `opencode-adapter-packaging.adr`, `opencode-archcore-prefixed-skill-names.adr`, and `opencode-plugin-ships-built-javascript.adr` (accepted, the last two on 2026-10-18); `opencode-mcp-entry-ownership.adr` (draft).
- Issue #21 for the live probe run on the four shipped hosts; the OpenCode row joins that run.

## Declared Delta

- Route: `capability` (size L); raised by M = stone (`host-adapter-contract.spec`, `stack-and-tooling.rule`, `opencode-adapter-packaging.adr` accepted) and R = `external-contract` (a published npm package against OpenCode's plugin API and a TS–shell stdin contract pinned by two suites).
- creates: `opencode-adapter` → `opencode-adapter.spec`.
- modifies: `plugin-delivery` → verdict `spec-wrong` against `plugin-delivery.spec` and `updating-the-plugin.spec`. Evidence: both state "OpenCode ships no plugin"; the maintainer's 2026-10-09 decision to deliver `@archcore/opencode-plugin` from `archcore init` makes that sentence stale once Phase 4 ships. Amendment: an OpenCode row in the per-host install table (`opencode plugin -g @archcore/opencode-plugin` on 1.x and `opencode plugin add @archcore/opencode-plugin` on 2.x, which write the global config; global scope; remove as a printed note on 1.x and `opencode plugin remove` on 2.x), `opencode` among the hosts `--agent` accepts, and the update step running `opencode plugin update @archcore/opencode-plugin` on 2.x and no command on 1.x.
- retires: none.
- decision: `mcp-archcore-entry-ownership` → `opencode-mcp-entry-ownership.adr`; `opencode-command-names` → `opencode-archcore-prefixed-skill-names.adr`; `opencode-package-build` → `opencode-plugin-ships-built-javascript.adr`.
- intent_gap: no — intent recorded in `opencode-adapter-packaging.adr` and `opencode-host-support.rnd`.
- Π: machine (OpenCode v1.18.34 source at commit `aec0b9a6`, the CLI and launcher code cited above), user (questions answered 2026-10-09 and 2026-10-18), empirical (the probes of Phase 5).
- Illustrate: not engaged — no new operator flow beyond the install guide. Runbook: engaged → `opencode-plugin-install.guide`.
