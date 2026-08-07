---
title: "Host Expansion Core Readiness — Shared Core Prepared for Copilot CLI and OpenCode"
status: rejected
tags:
  - "copilot"
  - "hooks"
  - "multi-host"
  - "opencode"
  - "plugin"
  - "roadmap"
---

## Goal

Bring the shared core under `plugins/archcore/bin/` to a state where a GitHub Copilot CLI adapter and an OpenCode adapter can be built on top of it without touching core logic, while the three shipped hosts — Claude Code, Cursor, and Codex — keep byte-identical behavior.

Scope decision by the maintainer on 2026-07-06: **core only**. The adapter deliverables themselves — the Copilot manifest and hooks config, the OpenCode TypeScript bridge, and the smoke test — are excluded and become follow-up contributor issues.

The trigger was that `archcore_hook_block()` used `exit 2` for every host, while Copilot treated exit 2 as a warning, so the write guard would not have blocked there (@plugins/archcore/bin/lib/normalize-stdin.sh).

## Tasks

All landed on 2026-07-06, sequenced so that the regression pins preceded every core edit.

1. **Pin the regressions for the three shipped hosts.** Add codex stdin fixtures under `test/fixtures/stdin/codex/`; add exact-output pins for `archcore_hook_block`, `archcore_hook_info`, and `archcore_hook_pretool_info` across claude-code, cursor, and codex; add session-start emit-shape pins; and add `test/structure/host-logic-locality.bats`, which permits host branching only in `normalize-stdin.sh` and `session-start` and forbids a host marker in `skills/`, `agents/`, `commands/`, or `rules/`.
2. **Add the Copilot core branches.** Widen stdin detection to the native camelCase markers `toolName` and `toolArgs`, keeping legacy `hookEventName` as a fallback while the environment override stays the primary channel; rewrite extraction for `toolName` plus escaped `toolArgs`, with provisional key candidates until real captures exist; give `archcore_hook_block` a copilot arm emitting the deny JSON with exit 0; give the info and pretool helpers a top-level `additionalContext` arm; and split the fixtures into legacy and native-provisional sets, marked as such in their README.
3. **Add the Copilot session-start arm.** Narrow `_archcore_emit_info` and the CLI-missing branch to `claude-code` byte-identically, add a `copilot` arm, and extend the plugin-dir guard with `.plugin/plugin.json`.
4. **Add the OpenCode core branches.** Add an explicit `opencode` extraction case, which is load-bearing because the wildcard fallback rewrites `ARCHCORE_HOST` to claude-code; add plain-text arms to both info helpers; keep the block path at exit 2 with stderr, since the bridge throws an error carrying that text; document the bridge contract in the `normalize-stdin.sh` header; add the `run_with_fixture_env` bats helper; and add `test/fixtures/stdin/opencode/` fixtures, which double as the future bridge-package contract fixtures.
5. **Add the test infrastructure.** Add `test/structure/host-coverage-matrix.bats`, whose loop-driven rows cover per hooks config the event set, the session-start registration, the write-guard and native matcher tokens, and script-set parity, with an enrollment guard that fails CI for any un-enrolled `hooks/*.json` or unknown fixture directory. Add `test/unit/output-helpers-matrix.bats`, covering hosts against helpers for completeness, asserting non-empty host-shaped output for all five core branches.

## Acceptance Criteria

- `make test` reports 360 of 360 green, `make lint` is shellcheck-clean, and `make check-json` reports valid — verified 2026-07-06.
- The exact-output pins for the three shipped hosts pass unchanged through every core edit, which proves the byte-identical-behavior invariant mechanically.
- The enrollment guard is verified negatively: a dummy `hooks/*.hooks.json` turns CI red.
- The Copilot deny path asserts the deny JSON with exit 0, and misdetection guards pin that a snake_case payload never routes to copilot, which is the deny-semantics asymmetry risk.
- `ARCHCORE_HOST=opencode` survives normalization, with the test failing on pre-change code.

## Dependencies

Follow-up work that consumes this core, tracked as separate issues outside this plan's scope:

- **The Copilot adapter, for a contributor.** A `.plugin/plugin.json` with an explicit `"hooks": "./hooks/copilot.hooks.json"`, since Copilot's default discovery would otherwise pick up the Claude config; a `hooks/copilot.hooks.json` in native camelCase with a per-entry `env` setting `ARCHCORE_HOST`, and a pre-mutation matcher of `create|edit|str_replace_editor|apply_patch`; Makefile `JSON_FILES` enrollment; and the matrix row plus structure tests, per `copilot-adapter-design.adr`.
- **The Copilot smoke test, for the maintainer, as a release gate.** Capture the real payloads — the exact `toolArgs` keys, the MCP `toolName` format, whether pre-mutation supports a context field, the session-start output shape, and whether a plain `NAME.md` agent loads — then replace the provisional fixtures, prune the extraction key candidates, add MCP-name normalization, and tighten the post-mutation matchers. Two contingencies are pre-wired: both copilot info-helper arms emit the same shape, so moving the event is config-only, and the session-start arm is a one-line flip.
- **The OpenCode TypeScript bridge, for a contributor.** A `plugins/opencode/` npm package per `opencode-adapter-packaging.adr`, consuming the bridge contract pinned here.
- **Cross-repository work in the Archcore CLI.** `archcore hooks <host> session-start` must accept the `copilot` and `opencode` host names. `bin/session-start` already passes them through, and the failure degrades silently to no session context until the CLI ships support.
