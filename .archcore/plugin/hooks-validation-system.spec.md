---
title: "Hooks and Validation System Specification"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "validation"
---

## Purpose & Scope

This spec defines the plugin's hook layer after the v0.7.0 repatriation: the hook POLICY — MCP-only write enforcement, code-alignment context injection, post-write validation, cascade and precision reporting, staleness, and the session recap — executes inside the archcore CLI (`archcore hooks <host> {session-start,pre-tool-use,post-tool-use}`), per `cli-owns-layers-4-5.adr`, and is verified by the CLI repository's own test suite. The plugin ships only host glue: hook entries in four host configs (`hooks/hooks.json`, `hooks/cursor.hooks.json`, `hooks/codex.hooks.json`, `hooks/copilot.hooks.json`) and three launcher scripts under `plugins/archcore/bin/` backed by two library files. Normative for those configs and scripts. Depended on by every host adapter and by `plugin-architecture.spec`, which owns the canonical event matrix; `host-adapter-contract.spec` owns the blocking-semantics translation, and `host-probe-protocol.spec` owns how these behaviors are verified on a live host. Out of scope: the MCP server, the CLI-side hook handlers and their budgets (CLI repository), and agent tool restrictions.

`always-use-mcp-tools.adr` records the rationale for the blocking behavior, `actualize-system.adr` for staleness detection, `pre-code-context-injection.idea` for the injection hook, `host-wiring-parity.adr` for dual naming, the version gate, and the SessionStart dedup, and `copilot-mcp-architecture.adr` for why Copilot needs the wiring advisory.

## Surface

Claude Code's `hooks/hooks.json` is the reference shape because it is the most explicit:

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "", "hooks": [ { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/session-start" } ] }
    ],
    "PreToolUse": [
      { "matcher": "Write|Edit", "hooks": [ { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/pre-tool-use", "timeout": 2 } ] }
    ],
    "PostToolUse": [
      { "matcher": "<five document-mutation tools under both namings>", "hooks": [ { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/post-tool-use", "timeout": 4 } ] }
    ]
  }
}
```

**Per-host shape divergence.** Cursor and Codex differ in event-name casing, plugin-root variable, and matcher contents; Cursor's post event is `afterMCPExecution` with no matcher. Copilot differs structurally: three flat entries with the command under `bash` rather than `command`, the budget under `timeoutSec` rather than `timeout`, `cwd: "."`, and `env.ARCHCORE_HOST=copilot` for deterministic detection; its `postToolUse` carries no matcher. A config or a test written by copying another host's and swapping names loads without error and does nothing, which is why `@test/structure/hooks.bats` extracts commands through a `.command // .bash` union and fails loudly when the extraction is empty.

**Copilot commands are a candidate chain, not a substitution.** Every other host's command is `${THAT_HOST_VARIABLE}/bin/<script>`. Copilot's probes `$COPILOT_PLUGIN_ROOT`, then `$PLUGIN_ROOT`, then `$CLAUDE_PLUGIN_ROOT`, each with `-x` for the script itself, execs the first that holds it, and otherwise warns on stderr and exits 0 — on this host every non-zero exit denies the tool call and discards the reason (`copilot-adapter-design.adr`).

**Dual tool naming.** Every archcore tool in a PostToolUse matcher appears under both `mcp__archcore__X` (project `.mcp.json` server) and `mcp__plugin_archcore_archcore__X` (plugin-bundled server on Claude Code). Matchers without regex metacharacters are exact matches, so a single-naming matcher silently never fires in one of the two setups. The CLI additionally recognizes Copilot's flat `archcore-<tool>` naming when it gates on the tool name; on hosts whose post event takes no matcher, that CLI-side gate is the whole selection mechanism.

**Launcher contract.** `bin/pre-tool-use` and `bin/post-tool-use` are the same four steps: (1) source `bin/lib/normalize-stdin.sh` for the raw payload and host id, and `bin/lib/plugin-cache-guard.sh` for the misrouted-cwd guard — exit 0 silently from a plugin install directory; (2) fail OPEN when `bin/cli-gte 0.7.0` does not print `yes` — a pre-0.7 CLI has no pre/post-tool-use leaves, and its usage error would deny every matched call on Copilot; (3) map the shell host id `codex` to the CLI agent id `codex-cli` (all other ids match the CLI dialect names); (4) pipe the raw payload to `archcore hooks <cli-host> <event>` and propagate stdout, stderr, and the exit code unchanged. Exit 2 plus stderr IS the deny protocol on claude-compatible hosts; Copilot's deny is a `{"permissionDecision":"deny",…}` document with exit 0, emitted by the CLI's Copilot dialect. Transparency is pinned by `@test/unit/hook-launchers.bats`.

**What the CLI does behind each event.** PreToolUse: the write guard (deny direct writes to `.archcore/` documents through the same predicate the MCP write tools consult; `.archcore/settings.json` and `.archcore/.sync-state.json` stay writable) and the code-alignment advisory (path-token ranking over local, non-rejected documents; skipped for Copilot, whose preToolUse carries no context field). PostToolUse: in-process doctor validation capped at five findings, the cascade notice for documents whose `implements`/`depends_on`/`extends` relations target the updated document, and the precision scan — the forbidden lexicon, the sections each type owes, a heading another type owns, and the line form the prose canon assigns the type, reported as at most twelve findings with a final line counting what the cap dropped — always exit 0. SessionStart: the ranked recap (corpus line, branch, drafts by mtime, recent accepted, staleness marks, tags, relation count) with `rejected` documents excluded from injection and recap.

**Hook 1 — SessionStart, `bin/session-start`.** Matcher empty, so it matches startup, resume, clear, and compact. Phases in order: (0) plugin-install-dir guard — the same two layers `bin/lib/plugin-cache-guard.sh` gives the launchers; (1) CLI availability check, emitting the install message and stopping when `archcore` is off PATH, on Copilot additionally naming `archcore init --agent copilot --project "$PWD"`; (2) project check, emitting `mcp__archcore__init_project` guidance when `.archcore/` is absent, forked on Copilot to the CLI wiring command; (3) context loading through `archcore hooks <cli-host> session-start` — non-zero exit swallowed, deduped CLI-side per `session_id`+`source`+host via short-window XDG stamps, captured rather than streamed on Copilot; (4) empty-state nudge pointing at `/archcore:init`, suppressible with `ARCHCORE_HIDE_EMPTY_NUDGE=1`; (5) Copilot wiring advisory, rate-limited per project; (6) outdated-CLI advisory backed by `archcore update --check`, rate-limited 24h; (7) Copilot flush folding buffered advisories into the one JSON document that host parses. Staleness needs no phase: it arrives inside the CLI recap of phase 3.

**The Copilot output channel.** Per the hooks reference, single-line `{"type":"progress"}` objects are stripped from stdout, every remaining line is concatenated, and ONE `JSON.parse` runs over the result; on failure the hook is treated as having produced no output. A hook on this host emits at most one JSON document — trailing plain text would discard the CLI context payload along with the note. Output fields are per-event: `sessionStart` and `postToolUse` accept `additionalContext`; `preToolUse` accepts only `permissionDecision`, `permissionDecisionReason`, and `modifiedArgs`, which is why the CLI skips pre-tool context there. Non-Copilot output is pinned byte-level by `@test/unit/session-start-goldens.bats`; the single-document rule by `@test/unit/session-start-emit-matrix.bats`.

**Scripts.** Three executables plus two library files. The plugin bundles no CLI binary and no policy scripts; every CLI invocation resolves `archcore` through PATH.

| Script | Role |
|---|---|
| `@plugins/archcore/bin/session-start` | SessionStart pipeline, phases 0–7 above |
| `@plugins/archcore/bin/pre-tool-use` | Launcher — CLI write guard + code-alignment context |
| `@plugins/archcore/bin/post-tool-use` | Launcher — CLI validation + cascade + precision |
| `@plugins/archcore/bin/lib/normalize-stdin.sh` | Host detection, stdin capture and field extraction, output helpers |
| `@plugins/archcore/bin/lib/plugin-cache-guard.sh` | Shared misrouted-cwd guard |

## Normative Behavior

1. The PreToolUse entry MUST match the host's file-mutation tool set: `Write|Edit` on Claude Code, `Write` on Cursor, `Write|Edit|apply_patch` on Codex, `create|edit|str_replace_editor|apply_patch` on Copilot.
2. Every PostToolUse matcher MUST list each archcore document-mutation tool under both namings.
3. Each launcher MUST exit 0 silently when the installed CLI is absent or `bin/cli-gte 0.7.0` prints anything but `yes`.
4. Each launcher MUST exit 0 silently when the working directory is inside a plugin install, detected by a cache path fragment or an upward-walk manifest hit.
5. Each launcher MUST pass the raw stdin payload to the CLI unchanged.
6. Each launcher MUST propagate the CLI's stdout, stderr, and exit code unchanged.
7. Each launcher and `bin/session-start` MUST map the shell host id `codex` to the CLI agent id `codex-cli`, passing every other host id as-is.
8. A launcher MUST NOT shape output per host or inspect the tool name — both live in the CLI.
9. IF a host treats every non-zero exit as a deny, THEN each hook command MUST resolve its script before invoking it.
10. IF such a command cannot resolve its script, THEN it MUST exit 0 and name the script in a stderr warning.
11. Hook 1 MUST exit silently when it runs from inside a plugin install.
12. WHEN `archcore` is off PATH, Hook 1 MUST emit the install message without blocking the session.
13. Hook 1 MUST back the update advisory with `archcore update --check`, rate-limited to once per 24 hours and silent on any failure.
14. Hook 1 MUST rate-limit the Copilot wiring advisory per project, keyed on the project root and suppressed under `ARCHCORE_HIDE_WIRING_NUDGE=1`.
15. IF a host parses a hook's entire stdout with a single JSON parse, THEN Hook 1 MUST emit exactly one JSON document, with every nudge and advisory folded into it.
16. IF the captured payload cannot be rewritten safely, THEN Hook 1 MUST pass it through unmodified and route its own messages to a channel the host strips before parsing.
17. The author MUST keep every other host's byte-level output unchanged when changing behavior for one host.
18. A hook script that invokes the CLI MUST call `archcore <subcmd>` directly, resolved through PATH, with a subcommand from the canonical surface `config|doctor|help|hooks|init|mcp|status|update`.
19. Every bin script MUST be POSIX shell.
20. Every bin script that reads host stdin MUST source `bin/lib/normalize-stdin.sh`.
21. Hook 1 MUST keep `"$PWD"` literal in any command it prints, so no path or environment value leaks into hook output.

## Constraints & Invariants

- Constraint: the PreToolUse entry's host timeout is 2 seconds and the PostToolUse entry's is 4 seconds — one second of launcher and process-spawn headroom above the CLI's internal 1 s / 3 s budgets, so the CLI's within-budget graceful output is never cut off by the host.
- Constraint: the hooks MUST work without network access in steady state; `update --check` is bounded to roughly 500 ms and stays silent offline.
- Constraint: a hook config carries no decision logic. Resolving the path to a script is the sole exception, granted by `host-adapter-contract.spec`.
- Invariant: the write guard blocks every direct Write or Edit to a `.archcore/` document on every host that supports pre-mutation hooks, and blocks nothing outside `.archcore/` except declared external global sources.
- Invariant: a launcher failure mode is allow, never deny — deny can originate only in the CLI's verdict.
- Invariant: a delegated Write or Edit call is subject to the same PreToolUse behavior as a main-session call; no dispatcher-based bypass exists.
- Invariant: PostToolUse never modifies a file and exits 0 whatever the outcome.
- Invariant: on a single-parse host, Hook 1's stdout is exactly one JSON document after progress lines are removed, in every combination of nudges and advisories.
- Invariant: Hook 1 never initiates a binary download; CLI lifecycle is the user's responsibility.

## Failure Behavior

1. IF `archcore` is off PATH, THEN Hook 1 MUST emit the install message and exit 0.
2. IF `archcore` is off PATH, THEN the launchers MUST exit 0 silently.
3. IF the installed CLI predates 0.7.0, THEN the launchers MUST exit 0 silently — enforcement is off for that session and the session-start update advisory is the recovery channel.
4. IF stdin JSON is malformed, THEN the hook MUST exit 0 with empty output (the CLI fails open on undecodable payloads).
5. IF git is unavailable, THEN Hook 1 MUST continue context loading.
6. IF a hook command cannot locate its script under any candidate plugin root, THEN it MUST exit 0 and name the script in a stderr warning.
7. IF a pre-mutation hook times out on Copilot, THEN the plugin MUST NOT rely on the host to block the write; the CLI holds its guard far inside budget instead, observed per host as probe D.

## Conformance

The hooks system is conformant when:

1. Every host config registers SessionStart, its pre-mutation event, and its post-mutation event, routed to the three launchers, with dual tool naming wherever the host uses matchers. Documented gaps — Cursor's `Write`-only matcher — are declared and asserted by name in `@test/structure/hooks.bats` and `@test/structure/host-coverage-matrix.bats`.
2. `bin/session-start` guards against a plugin-install cwd at any depth, emits an install message when `archcore` is missing, emits init guidance when `.archcore/` is missing, and otherwise delegates to `archcore hooks` and runs, in order, the empty-state nudge, the Copilot wiring advisory, the rate-limited update advisory, and the Copilot flush.
3. The launchers are byte-transparent to the CLI — same stdout, stderr, and exit status — pinned by `@test/unit/hook-launchers.bats` and, for the probe harness copy, `@test/unit/probe-wrapper.bats`.
4. Output shapes follow each host's documented form and are produced by the CLI dialects, not by per-script string building.
5. On Copilot, Hook 1's stdout parses as one JSON document after progress-line removal in every advisory combination, pinned by `@test/unit/session-start-emit-matrix.bats`; non-Copilot output is pinned byte for byte by `@test/unit/session-start-goldens.bats`.
6. Every script that invokes `archcore` passes only allowlisted subcommands, guarded by `@test/structure/readme-cli-references.bats`.
7. Each host's hook commands resolve their scripts under that host's documented load paths, verified by executing the command rather than by inspecting it; `@test/structure/copilot-plugin.bats` runs Copilot's under `env -u` for the unresolved, single-candidate, and dead-candidate cases.