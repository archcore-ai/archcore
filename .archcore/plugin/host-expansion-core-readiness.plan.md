---
title: "Host Expansion Core Readiness — Shared Core Prepared for Copilot CLI and OpenCode"
status: accepted
tags:
  - "copilot"
  - "hooks"
  - "multi-host"
  - "opencode"
  - "plugin"
  - "roadmap"
---

## Goal

Bring the shared core (`plugins/archcore/bin/`) to a state where GitHub Copilot CLI and OpenCode adapters can be built on top of it without touching core logic — while the three shipped hosts (Claude Code, Cursor, Codex) keep byte-identical behavior. Scope decision (maintainer, 2026-07-06): **core only** — the adapter deliverables themselves (Copilot manifest + hooks config, OpenCode TS bridge, smoke test) are explicitly excluded and become follow-up contributor issues.

Trigger: `archcore_hook_block()` used `exit 2` for every host, but Copilot treats exit 2 as a warning — the write guard would not block there (@plugins/archcore/bin/lib/normalize-stdin.sh).

## Tasks

All landed 2026-07-06, sequenced so regression pins preceded every core edit:

1. **Regression pinning (three shipped hosts)** — codex stdin fixtures (`test/fixtures/stdin/codex/`), exact-output pins for `archcore_hook_block` / `archcore_hook_info` / `archcore_hook_pretool_info` across claude-code/cursor/codex, session-start emit-shape pins, and `test/structure/host-logic-locality.bats` (host branching allowed only in `normalize-stdin.sh` + `session-start`; no host markers in `skills/ agents/ commands/ rules/`).
2. **Copilot core branches** — stdin detection widened to native camelCase markers (`toolName`/`toolArgs`; legacy `hookEventName` kept as fallback; env override stays the primary channel for the future hooks config); extraction rewritten for `toolName` + escaped `toolArgs` (provisional key candidates until real captures); `archcore_hook_block` copilot arm emits `{"permissionDecision":"deny","permissionDecisionReason":…}` + exit 0; info/pretool helpers emit top-level `{"additionalContext":…}`; fixtures split into legacy + native-provisional (`test/fixtures/stdin/copilot/README.md` marks them).
3. **Copilot session-start** — `_archcore_emit_info` and the CLI-missing branch narrowed to `claude-code)` (byte-identical) plus a `copilot)` arm; plugin-dir guard extended with `.plugin/plugin.json`.
4. **OpenCode core** — explicit `opencode)` extraction case (load-bearing: the `*` fallback rewrites `ARCHCORE_HOST` to claude-code); plain-text arms in both info helpers; block stays exit 2 + stderr (bridge throws `Error(stderr)`); bridge contract documented in the `normalize-stdin.sh` header; `run_with_fixture_env` bats helper; `test/fixtures/stdin/opencode/` fixtures double as the future bridge-package contract fixtures.
5. **Test infrastructure** — `test/structure/host-coverage-matrix.bats` (loop-driven rows per hooks config: event set, session-start registration, write-guard + native matcher tokens, script-set parity; enrollment guard fails CI for any un-enrolled `hooks/*.json` or unknown fixture dir) and `test/unit/output-helpers-matrix.bats` (hosts × helpers completeness: non-empty, host-shaped output for all five core branches).

## Acceptance Criteria

- `make test` 360/360 green, `make lint` (shellcheck) clean, `make check-json` valid — verified 2026-07-06.
- Exact-output pins for the three shipped hosts pass unchanged through every core edit (invariant 1 proven mechanically).
- Enrollment guard verified negatively: a dummy `hooks/*.hooks.json` turns CI red.
- Copilot deny path: `{"permissionDecision":"deny"}` + exit 0 asserted; misdetection guards pin that snake_case payloads never route to copilot (deny-semantics asymmetry risk).
- OpenCode: `ARCHCORE_HOST=opencode` survives normalization (test fails on pre-change code).

## Dependencies

Follow-up work that consumes this core (tracked as separate issues, out of this plan's scope):

- **Copilot adapter (contributor):** `.plugin/plugin.json` with explicit `"hooks": "./hooks/copilot.hooks.json"` (Copilot's default discovery would otherwise pick up Claude's `hooks/hooks.json`), `hooks/copilot.hooks.json` (native camelCase, per-entry `env: {"ARCHCORE_HOST":"copilot"}`, preToolUse matcher `create|edit|str_replace_editor|apply_patch`), Makefile `JSON_FILES` enrollment, matrix row + structure tests — per `copilot-adapter-design.adr`.
- **Copilot smoke (maintainer, release gate):** capture real payloads (exact `toolArgs` keys, MCP `toolName` format, preToolUse `additionalContext` support, sessionStart output shape, `NAME.md` agent loading), replace provisional fixtures, prune extraction key candidates, add MCP-name normalization + tighten postToolUse matchers. Contingencies pre-wired: copilot arms of both info helpers emit the same shape (event move is config-only); sessionStart arm is a one-line flip.
- **OpenCode TS bridge (contributor):** `plugins/opencode/` npm package per `opencode-adapter-packaging.adr`, consuming the bridge contract pinned here.
- **Cross-repo (Archcore CLI, Go):** `archcore hooks <host> session-start` must accept `copilot` and `opencode` host names — `bin/session-start` already passes them through; failure degrades silently (no session context) until the CLI ships support.