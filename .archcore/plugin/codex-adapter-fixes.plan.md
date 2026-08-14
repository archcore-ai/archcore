---
title: "Codex Adapter Fixes — Detection Marker, Single-Document Stdout, Host Ask Wording"
status: draft
tags:
  - "codex"
  - "hooks"
  - "multi-host"
  - "plugin"
  - "testing"
---

## Goal

Bring the Codex adapter to the contract in `codex-adapter.spec` — detection from `CODEX_THREAD_ID`, a pinned host id inside hook processes, one document per hook event, and a host question that a desktop-app user recognizes — without adding a host id and without changing observable behavior on Claude Code, Cursor, or Copilot. Target: plugin 0.7.3.

## Tasks

1. **Done.** `@plugins/archcore/bin/detect-host` — the Codex branch reads `[ -n "${CODEX_THREAD_ID:-}" ] || [ -n "${CODEX_HOME:-}" ]`, and the header records why: `CODEX_THREAD_ID` is injected by `create_env` after filtering, `CODEX_HOME` is a user-set config-directory override, and the precedence cannot resolve a nested session in either direction.
2. **Done.** `@plugins/archcore/hooks/codex.hooks.json` — every command carries the `ARCHCORE_HOST=codex` prefix and quotes `"${PLUGIN_ROOT}"`. Codex offers no `env` field on a handler, and a Codex `SessionStart` payload has no `turn_id` for the stdin heuristic to key on. The command string is part of the hook's trust hash, so an installed plugin asks for re-approval under `/hooks` after this change.
3. **Done.** `@plugins/archcore/bin/session-start` — `_archcore_buffers_stdout()` returns true for `copilot` and `codex`; `_archcore_note` and the CLI capture branch key on it; `_archcore_copilot_flush` is now `_archcore_flush_stdout` with a `_archcore_flush_codex` arm that splices the buffered notes into `hookSpecificOutput.additionalContext` by anchoring on the literal that follows the value, so escaped quotes inside it are untouched.
4. **Done.** `@plugins/archcore/bin/session-start` — `codex` joins the `claude-code` arm of `_archcore_emit_info` and of the CLI-missing notice, and the no-payload flush arm emits a document rather than plain text, because `looks_like_json` fires on `[` and every message opens with `[Archcore]`. Unrecognized CLI payloads pass through unchanged with the advisory on stderr.
5. **Done.** `@plugins/archcore/skills/init/SKILL.md` — the host question offers "Codex (CLI or desktop app)" mapped to `codex-cli` and states that no `codex-desktop` id exists; the closing message names project trust and the `/hooks` approval as the two consents the wiring waits on.
6. **Partly done.** `component-registry.doc` carries the corrected `bin/detect-host` row, the host-id pin, the two-host single-document channel, and the new test files. `host-wiring-parity.adr` keeps its original Codex sentence as the record of what was decided then; `codex-adapter-conformance.adr` states the supersession and the relation graph links the two.
7. **Done.** `@test/fixtures/stdin/codex/session-start.json` and `@test/fixtures/stdin/codex/mcp-list-documents.json` hold payloads captured from Codex CLI 0.147.0 on 2026-08-14, with absolute paths generalized and `session_id` removed — `probe-wrapper.bats` replays every fixture twice through the real CLI, whose SessionStart dedup keys on that field and blanks the second run.
8. **Open.** The `codex` row of the probe records in `host-probe-protocol.spec` stays `deferred:not-yet-run`. The live runs below used a project hook pointing at the working tree, not the `mkprobe` tree that P0 is defined against, and that spec's item 10 forbids recording `pass` by analogy.
9. **Done.** Version 0.7.3 in all four manifests per `bump-plugin-version.cpat`, derived from the highest tag `v0.7.2` with a patch step.

## Test Plan

**Unit — `@test/unit/detect-host.bats`.** Added: `CODEX_THREAD_ID` alone prints `codex-cli`; a desktop-shaped environment (`CODEX_THREAD_ID` plus `CODEX_SANDBOX`) prints the same id; `CODEX_HOME` alone still prints `codex-cli`; `CLAUDECODE` or `CURSOR_TRACE_ID` alongside `CODEX_THREAD_ID` keeps the higher-precedence answer; `CODEX_SANDBOX` and `CODEX_COMPANION_SESSION_ID` alone print `__UNKNOWN__`. Fault injection performed: restoring the `CODEX_HOME`-only branch failed exactly the two `CODEX_THREAD_ID` cases and nothing else.

**Unit — new `@test/unit/session-start-codex-stdout.bats` (10 cases).** One shared assertion, `assert_codex_single_document`, mirrors the host predicate: stdout either parses as exactly one JSON document (`jq -s 'length == 1'`) or starts with neither `{` nor `[`. Cases: byte-identical pass-through with no advisory; empty `.archcore/` spliced into `additionalContext`; pending CLI update spliced; both advisories in one run; escaped quotes and backslashes in the CLI payload surviving the splice; `systemMessage` surviving; unrecognized payload passing through with the advisory on stderr; empty payload wrapped in its own document; missing `.archcore/` as a document rather than a bare `[Archcore]` line; CLI absent from PATH as a document.

**Unit — `@test/unit/normalize-stdin.bats`.** The captured PostToolUse payload resolves to host `codex` with `tool_name` `mcp__archcore__list_documents`. The captured SessionStart payload is asserted to carry no `turn_id`, to resolve to `claude-code` on stdin alone, and to resolve to `codex` under the env pin — the three facts task 2 rests on.

**Unit — goldens and emit matrix.** `@test/unit/session-start-goldens.bats` and `@test/unit/session-start-emit-matrix.bats` moved `codex` from the plain-text group to the JSON group and gained a codex-specific empty-state golden; the claude-code, cursor, opencode, and copilot goldens are unchanged byte for byte, which is the non-regression gate for this change.

**Structure — `@test/structure/hooks.bats`.** Every codex hook command starts with `ARCHCORE_HOST=codex`; no other host config carries that pin.

**Structure — `@test/structure/codex-plugin.bats`.** The command assertion now requires the quoted `"${PLUGIN_ROOT}"/bin/` form, and its stale note about the removed `plugin_hooks` feature flag is corrected.

**Structure — `@test/structure/init-skill.bats`.** The host question names the desktop app and maps to `codex-cli`; the closing message names project trust and `/hooks`.

**Integration — `@test/integration/codex-plugin-smoke.bats`.** Re-run offline against Codex CLI 0.147.0: marketplace add, plugin list, plugin add, skill loading through `codex debug prompt-input`, and plugin-managed MCP registration — 5 of 5 green after the manifest and hook changes.

**Live — Codex CLI 0.147.0, 2026-08-14.** Five `codex exec` runs at roughly 8k tokens each established: the baseline failure (`hook: SessionStart Failed` on an empty `.archcore/`), the trust behavior of project hooks and project MCP config, the hook payload shapes now stored as fixtures, and the fix itself — a session where the corrected script reported `hook: SessionStart Completed` and the model quoted back both the corpus context and the empty-state advisory, while the installed 0.7.2 plugin failed in the same session.

## Acceptance Criteria

1. `make test` passes with the new cases, and each new assertion has been shown to fail under its own injected defect. Status: 489 tests green, 0 failures; the fault injection was performed for task 1.
2. `bin/detect-host` prints `codex-cli` inside a live Codex session on both surfaces. Status: unit-verified; the live desktop check is open.
3. A Codex session in a repository with an empty `.archcore/` reports `hook: SessionStart Completed` and receives the Archcore context. Status: verified live.
4. The goldens for Claude Code, Cursor, and Copilot are unchanged, byte for byte. Status: verified.
5. `shellcheck -s sh` stays clean for every modified `bin/` script. Status: verified — `make verify` reports JSON valid, permissions OK, ShellCheck clean.
6. No file outside `bin/`, `skills/init/SKILL.md`, `hooks/`, `test/`, the four manifests, and the documents listed in task 6 changes. Status: verified against `git status`.

## Dependencies

- The archcore CLI's `archcore hooks codex-cli session-start` keeps emitting the Claude-shaped document with `hookSpecificOutput`; a key-order change routes advisories to stderr until the splice pattern is updated.
- Codex CLI 0.147.0 or later for the live probe, plus a trusted scratch project and `--dangerously-bypass-hook-trust` for the non-interactive runs.
- `jq` in the test environment, already required by the existing emit-matrix tests.
