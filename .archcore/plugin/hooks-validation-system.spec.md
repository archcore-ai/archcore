---
title: "Hooks and Validation System Specification"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "validation"
---

## Purpose & Scope

This spec defines the hook layer that enforces the MCP-only principle, keeps `.archcore/` files consistent, detects documentation staleness, and injects project context before source-file edits. Normative for every hook entry the plugin ships across all four host configs (`hooks/hooks.json`, `hooks/cursor.hooks.json`, `hooks/codex.hooks.json`, `hooks/copilot.hooks.json`) and for the seven scripts under `plugins/archcore/bin/`. Depended on by every host adapter and by `plugin-architecture.spec`, which owns the canonical event matrix; `host-adapter-contract.spec` owns the blocking-semantics translation, and `host-probe-protocol.spec` owns how these behaviors are verified on a live host. Out of scope: the MCP server itself, the Archcore CLI lifecycle (the user installs the CLI per https://docs.archcore.ai/cli/install/ and the plugin resolves it via PATH), and agent tool restrictions.

Claude Code's `hooks/hooks.json` is the reference shape below because it is the most explicit. Per-host divergence is called out where it changes behavior rather than only syntax. Event names differ per host — PascalCase on Claude Code and Codex, camelCase on Cursor and Copilot — and so do entry shapes. The behaviors are identical everywhere with one declared exception: Copilot registers five entries rather than six, because its pre-mutation event cannot carry context (Hook 3).

`always-use-mcp-tools.adr` records the rationale for the blocking behavior, `actualize-system.adr` for staleness detection, `pre-code-context-injection.idea` for the injection hook, `host-wiring-parity.adr` for dual naming and the SessionStart dedup and advisory additions, and `copilot-mcp-architecture.adr` for why Copilot needs the wiring advisory.

## Surface

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/session-start" }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-archcore-write", "timeout": 1 },
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-code-alignment", "timeout": 1 }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "mcp__archcore__create_document|mcp__plugin_archcore_archcore__create_document|mcp__archcore__update_document|mcp__plugin_archcore_archcore__update_document|mcp__archcore__remove_document|mcp__plugin_archcore_archcore__remove_document|mcp__archcore__add_relation|mcp__plugin_archcore_archcore__add_relation|mcp__archcore__remove_relation|mcp__plugin_archcore_archcore__remove_relation",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore", "timeout": 3 }
        ]
      },
      {
        "matcher": "mcp__archcore__update_document|mcp__plugin_archcore_archcore__update_document",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-cascade", "timeout": 3 }
        ]
      },
      {
        "matcher": "mcp__archcore__create_document|mcp__plugin_archcore_archcore__create_document|mcp__archcore__update_document|mcp__plugin_archcore_archcore__update_document",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-precision", "timeout": 3 }
        ]
      }
    ]
  }
}
```

**Per-host shape divergence.** Cursor and Codex differ only in event-name casing, plugin-root variable, and matcher contents. Copilot differs structurally: entries are flat objects with no nested `hooks[]` group, the command lives under `bash` rather than `command`, the budget under `timeoutSec` rather than `timeout`, each entry sets `cwd: "."` so the hook runs from the user's project and `env.ARCHCORE_HOST=copilot` so detection is deterministic, and its `postToolUse` entries carry **no matcher**. A config or a test written by copying another host's and swapping names loads without error and does nothing, which is why `@test/structure/hooks.bats` extracts commands through a `.command // .bash` union and fails loudly when the extraction is empty.

**Copilot commands are a candidate chain, not a substitution.** Every other host's command is `${THAT_HOST_VARIABLE}/bin/<script>`. Copilot's probes `$COPILOT_PLUGIN_ROOT`, then `$PLUGIN_ROOT`, then `$CLAUDE_PLUGIN_ROOT`, each with `-x` for the script itself, execs the first that holds it, and otherwise warns on stderr and exits 0. The variable this adapter shipped with is documented nowhere by GitHub, and unset it left the literal path `/bin/<script>` (`copilot-adapter-design.adr`). Test extraction is therefore by `bin/<script>` token; a substituting `sed` would emit the surrounding shell as if it were a path.

**Dual tool naming.** Every archcore tool in a PostToolUse matcher appears under both `mcp__archcore__X` (the name a project-level `.mcp.json` server yields) and `mcp__plugin_archcore_archcore__X` (the name Claude Code gives a plugin-bundled server). Claude Code matchers without regex metacharacters are exact matches, so a single-naming matcher silently never fires in one of the two setups.

Matchers and scripts solve different halves of one problem: a matcher decides *whether the script runs*, and the script decides *what to do*. `@plugins/archcore/bin/lib/normalize-stdin.sh` folds all three namings — including Copilot's flat `archcore-<tool>`, where the host joins server and tool with a hyphen — into the canonical `mcp__archcore__*` before any guard inspects a tool name. Without that fold a guard fires and then falls through its own filter: cost paid, protection absent. On Copilot, where `postToolUse` takes no matcher, that filtering is the whole selection mechanism.

**The Copilot output channel.** Per the [hooks reference](https://docs.github.com/en/copilot/reference/hooks-reference), a line that is a single complete JSON object with `"type":"progress"` is consumed as a progress event and removed from the stream; every other line — blank lines, plain text, and non-progress JSON — is preserved verbatim. On exit the preserved lines are concatenated, trimmed, and parsed with **one** `JSON.parse`; if that fails, the hook is treated as having produced no output at all. Two consequences follow. First, a hook on this host may emit at most one JSON document: plain text appended after a JSON payload discards the payload, including the roughly 9 KB of Archcore context, along with the note itself. Emitting advisories as trailing plain text was the behavior through plugin 0.6.1 and cost the whole session's context whenever any advisory fired. Second, output fields are per-event: `sessionStart`, `postToolUse`, `postToolUseFailure`, `notification`, and `subagentStart` accept `additionalContext`, while `preToolUse` accepts only `permissionDecision`, `permissionDecisionReason`, and `modifiedArgs`. Other hosts are unaffected, and their byte-level output is pinned by `@test/unit/session-start-goldens.bats`.

**Hook 1 — SessionStart, `bin/session-start`.** Matcher empty, so it matches startup, resume, clear, and compact. It runs eight phases in order: (0) plugin-install-dir guard — exit 0 silently when `$PWD` holds an install-cache fragment (`.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/`, `.copilot/installed-plugins/`, `plugins/cache/`) or a bounded upward walk finds a `.cursor-plugin/`, `.claude-plugin/`, `.codex-plugin/`, or `.plugin/` manifest; (1) CLI availability check, emitting the install message and stopping when `archcore` is off PATH, and on Copilot additionally naming `archcore init --agent copilot --project "$PWD"`; (2) project check, emitting `mcp__archcore__init_project` guidance when `.archcore/` is absent, forked on Copilot to the CLI wiring command because no plugin-shipped MCP exists there, with the `/archcore:init` suffix sentence shared byte for byte; (3) context loading through `archcore hooks <host> session-start`, whose non-zero exit is swallowed, deduped CLI-side per `session_id`+`source` (Cursor: `conversation_id`) via short-window XDG stamps, fail-open, and captured rather than streamed on Copilot; (4) empty-state nudge pointing at `/archcore:init`, suppressible with `ARCHCORE_HIDE_EMPTY_NUDGE=1`; (5) Copilot wiring advisory; (6) staleness check through `bin/check-staleness`; (7) outdated-CLI advisory backed by `archcore update --check`; (8) Copilot flush. Phases 4–7 are additive: a failure in one leaves the preceding phases unaffected.

The wiring advisory's detection mirrors Copilot's own discovery, measured on Copilot CLI 1.0.76: a config is read from **every** directory between the working directory and the git root — not merely the two ends — and within each directory `.mcp.json` wins, with `.github/mcp.json` consulted only where the first is absent. User-level `${COPILOT_HOME:-~/.copilot}/mcp-config.json` also counts. Checking fewer places nags a project that works; treating the two filenames as a union stays silent for a project that does not, which is the failure the advisory exists to prevent.

The Copilot flush splices the buffered advisories into the payload captured in phase 3, and fires only on the exact `{"additionalContext":"…"}` shape it can take apart. An unrecognized payload passes through untouched and the advisories fall back to `progress` lines, which the host strips before parsing. Both branches satisfy the one-document rule.

**Hook 2 — PreToolUse block, `bin/check-archcore-write`.** Matcher is the host's file-mutation tool set: `Write|Edit` on Claude Code, `Write` on Cursor (which exposes no Edit tool), `Write|Edit|apply_patch` on Codex, `create|edit|str_replace_editor|apply_patch` on Copilot. Budget 1 second. It extracts the target path through the normalizer — on Copilot the path lives inside an escaped JSON string under `toolArgs` — matches it against `.archcore/**/*.md`, allows on no match, and denies on a match through the mechanism the host honors. `.archcore/settings.json` and `.archcore/.sync-state.json` are exempt: the first is configuration and the second is managed by the MCP tools. The deny reason text is:

```
Direct writes to .archcore/ documents are not allowed. Use Archcore MCP tools instead:
- create_document: create a new document
- update_document: modify an existing document
- remove_document: delete a document
This ensures validation, templates, and the sync manifest stay consistent.
```

On Claude Code, Codex, and Cursor, exit 2 is the blocking error and stderr reaches the model. On Copilot the guard writes `{"permissionDecision":"deny","permissionDecisionReason":"…"}` to stdout with exit 0, because **every** non-zero exit denies there — exit 2 explicitly, any other non-zero as `Denied by preToolUse hook (hook errored)` — and only the JSON form carries reason text to the user (hooks-reference, re-read 2026-07-27). Those two fields are also among the only three that event accepts, which is what makes Hook 2 viable on Copilot where Hook 3 is not. On that host the failure mode inverts: where exit 2 is the deny channel, a guard that cannot start degrades to no enforcement; where every non-zero exit denies, it degrades to refusing every matched tool call, and the host cannot tell the two apart.

**Hook 3 — PreToolUse injection, `bin/check-code-alignment`.** Same matcher per host, budget 1 second, registered on Claude Code, Cursor, and Codex and **not on Copilot**, whose `preToolUse` accepts no context field. The omission is declared rather than silent: `@test/structure/copilot-plugin.bats` asserts its absence there, `@test/structure/hooks.bats` and `@test/structure/host-coverage-matrix.bats` carve it out by name so no other host can drop any other script quietly, and a negative-control test asserts it is still present in the other three configs. Re-registering the entry is the whole change if Copilot adds context to `preToolUse`.

It extracts `file_path` through the normalized stdin layer; short-circuits on a missing `file_path`, an absent `.archcore/` directory, a path inside `.archcore/`, or `ARCHCORE_DISABLE_INJECTION=1`; normalizes to cwd-relative and exits on an absolute path outside `$CWD`; requires the path to start with a configured source root, defaulting to `src lib app pkg cmd internal apps packages modules components` and overridable through `.archcore/settings.json` → `codeAlignment.sourceRoots`; generates directory-prefix tokens longest first, capped at 5 levels; scans with one `grep -rlF` per token rather than per match; scores by specificity combined with type priority `rule=5, cpat=4, adr=3, spec=2, guide=1`; takes the top 3; and renders a block capped at 2 KB.

```
[Archcore Context] Before editing <relative-path>:
- <type>: <title> [<short-doc-path>]
...
```

Its additivity is why its cost is a correctness concern: a timeout raises nothing — the write proceeds, no context arrives, and no log records it. The hook once cost roughly 6 ms per *matching* document, two process spawns each per token, so a knowledge base where about 170 documents mentioned a common source root exceeded the 1 s budget outright, and push-mode stopped working on exactly the repositories with the most context to give. Deduplication now happens in a single pass, so spawns scale with token count rather than match count. `@test/unit/hook-latency.bats` pins both the absolute budget and the independence from match count.

Hooks 2 and 3 are deliberately coupled where both are registered: Hook 2 short-circuits on `.archcore/*.md` and denies, Hook 3 short-circuits on everything inside `.archcore/` with a silent exit 0. Their active path sets are disjoint by construction, and they run as independent entries with independent budgets.

**Delegated calls.** PreToolUse hooks fire at the tool-execution boundary, not the session boundary, so a file-mutation call matches the host's matcher whether it comes from the main conversation or from a delegated sub-agent, in the same stdin shape and with no sub-agent annotation to branch on. Archcore's own sub-agents hold no `Write` or `Edit` tool and cannot trigger Hooks 2 or 3 at all; the coverage question concerns general-purpose and third-party agents dispatched for code work. Cursor's matcher is `Write` only, so sub-agent Edit calls go unhooked there for the same reason main-session Edit calls do — a pre-existing asymmetry independent of delegation. This is probe A-d in `host-probe-protocol.spec`, which is where the per-host result is recorded; a specification claim about delegated coverage is not evidence.

**Hooks 4–6 — PostToolUse.** Hook 4 (`bin/validate-archcore`, budget 3 s) matches the five document-mutation tools under both namings and runs `archcore doctor` through `timeout 2` with `|| true`. It is the sole validation entry point: PreToolUse already blocks direct writes, so this matcher covers every mutation that can reach the knowledge base. Hook 5 (`bin/check-cascade`, budget 3 s) matches `update_document` only, queries the relation graph for documents that are the *source* of an `implements`, `depends_on`, or `extends` relation whose target was updated, and excludes `related`. It fires in addition to Hook 4, independently. Hook 6 (`bin/check-precision`, budget 3 s) matches `create_document` and `update_document`, reads the resulting file from disk, and emits soft warnings. Copilot registers no matcher for any of the three; the scripts self-filter.

A prior revision also registered a PostToolUse `Write|Edit` matcher invoking `validate-archcore` as defense in depth. It was dead in practice — PreToolUse blocks every Write and Edit to `.archcore/*.md` before PostToolUse fires, and the two allowlisted JSON files gave it no edge case — so it was removed to eliminate a per-write shell fork across the whole repository.

**Output envelopes.** The scripts never build these by hand; the output helpers in `bin/lib/normalize-stdin.sh` select the shape from `ARCHCORE_HOST`, which is why a new host costs one branch in one file.

| Host | Context envelope | Pre-mutation deny |
|---|---|---|
| Claude Code | `hookSpecificOutput.additionalContext` | exit 2 + stderr |
| Codex CLI | `hookSpecificOutput.additionalContext` | exit 2 + stderr |
| Cursor | `additional_context` (flat) | exit 2 + stderr |
| GitHub Copilot CLI | `additionalContext` (flat, top level) — at most one document per hook invocation | stdout `{"permissionDecision":"deny",…}`, exit 0 |
| OpenCode | plain message, no JSON | bridge throws `Error(reason)` |

| Hook | Exit 0 | Exit 2 |
|------|--------|--------|
| SessionStart | Always (install msg / init msg / context + nudges + staleness + advisories) | N/A |
| Hook 2, allow | Empty output, operation proceeds | N/A |
| Hook 2, block | Copilot only: deny JSON on stdout | Claude Code / Codex / Cursor: stderr → model feedback, blocked |
| Hook 3 | Always (no match → empty; match → context) | N/A |
| Hook 4 | Always (clean → empty; issues → context) | N/A |
| Hook 5 | Always (no cascade → empty; cascade → context) | N/A |
| Hook 6 | Always (clean → empty; warnings → context) | N/A |

**Scripts.** Seven executables plus the stdin normalization library. The plugin bundles no CLI binary and no launcher wrapper; every script that invokes the CLI calls `archcore <subcmd>` directly through PATH.

| Script | Role |
|---|---|
| `@plugins/archcore/bin/session-start` | SessionStart pipeline, phases 0–8 above |
| `@plugins/archcore/bin/check-archcore-write` | Hook 2 — deny direct `.archcore/` document writes |
| `@plugins/archcore/bin/check-code-alignment` | Hook 3 — inject context for source edits |
| `@plugins/archcore/bin/validate-archcore` | Hook 4 — `archcore doctor` after MCP mutations |
| `@plugins/archcore/bin/check-staleness` | Layer 1 staleness detection, called by `session-start` |
| `@plugins/archcore/bin/check-cascade` | Hook 5 — relation-graph cascade detection |
| `@plugins/archcore/bin/check-precision` | Hook 6 — lexicon, sections, frontmatter, length |
| `@plugins/archcore/bin/lib/normalize-stdin.sh` | Host detection, stdin field extraction, output helpers |

## Normative Behavior

1. Hook 2 MUST block every Write or Edit call targeting a `.archcore/**/*.md` file on every host, through that host's honored deny mechanism.
2. IF a host does not carry reason text on a non-zero exit, THEN Hook 2 MUST emit that host's deny payload instead.
3. Hook 2 MUST NOT block a write to `.archcore/settings.json`.
4. Hook 2 MUST NOT block a write to `.archcore/.sync-state.json`.
5. Hook 2 MUST NOT block a write to a file outside `.archcore/`.
6. Hook 2 MUST exit 0 to allow a call and 2 to block it.
7. Hook 2 MUST NOT use any other exit code.
8. IF a host treats every non-zero exit as a deny, THEN each hook command MUST resolve its script before invoking it.
9. IF such a command cannot resolve its script, THEN the command MUST exit 0.
10. IF a hook command cannot resolve its script, THEN the command MUST report that on stderr.
11. Hook 3 MUST exit 0 on every code path.
12. Hook 3 MUST NOT block or fail an edit.
13. Hook 3 MUST short-circuit silently for a path inside `.archcore/`, a path outside the configured source roots, and a path that produces no match.
14. Hook 3 MUST rank matches by specificity first, where the longest matching directory prefix wins.
15. Hook 3 MUST break a specificity tie by type priority `rule > cpat > adr > spec > guide`.
16. Hook 3 MUST restrict eligible types to `rule`, `cpat`, `adr`, `spec`, and `guide`.
17. Hook 3 MUST cap its output at 3 documents and 2 KB.
18. Hook 3 MUST honor `ARCHCORE_DISABLE_INJECTION=1` as an unconditional off-switch.
19. Hook 3 MUST honor `.archcore/settings.json` → `codeAlignment.sourceRoots` when it is configured.
20. IF a host's pre-mutation event accepts no context field, THEN the adapter MUST NOT register a context-only hook on that event.
21. WHEN an adapter omits a context-only hook from a host, a structure test MUST assert that absence by name rather than by a loosened parity check.
22. Each PreToolUse hook MUST complete far enough inside its 1 second budget that a timeout is unreachable on a realistic knowledge base.
23. Hook 3's cost MUST NOT scale with the number of matching documents.
24. The PreToolUse hooks MUST treat a delegated Write or Edit call identically to a main-session call.
25. Every PostToolUse matcher MUST list each archcore tool under both namings.
26. `bin/lib/normalize-stdin.sh` MUST fold every host's MCP tool naming to the canonical `mcp__archcore__*` before any guard inspects a tool name.
27. IF a host's post-mutation event accepts no matcher, THEN the handling script MUST filter to the same tool set the matcher would have selected.
28. Hook 4 MUST fire after every document-mutation MCP tool call.
29. Hook 4 MUST report validation issues as additional context.
30. Hook 4 MUST NOT block or revert an operation.
31. The hooks config MUST NOT register a `Write|Edit` matcher on PostToolUse.
32. Hook 5 MUST fire only after `update_document`.
33. Hook 5 MUST flag only documents connected by `implements`, `depends_on`, or `extends`.
34. Hook 1 MUST exit silently when it runs from inside a plugin install, detected by a cache path fragment or by an upward-walk manifest hit.
35. WHEN `archcore` is off PATH, Hook 1 MUST emit the install message.
36. WHEN `archcore` is off PATH, Hook 1 MUST NOT block the session.
37. Hook 1 MUST run the staleness check after context loading and after the empty-state and wiring nudges.
38. The staleness check MUST keep its output at or below 2 KB.
39. The staleness check MUST rate-limit itself to once per 24 hours through a persistent timestamp file.
40. Hook 1 MUST back the update advisory with `archcore update --check`.
41. Hook 1 MUST rate-limit the update advisory to once per 24 hours through its own stamp.
42. Hook 1 MUST keep the update advisory silent on any failure, including an older CLI that lacks the flag.
43. IF a host parses a hook's entire stdout with a single JSON parse, THEN Hook 1 MUST emit exactly one JSON document, with every nudge, advisory, and finding folded into it.
44. IF the captured payload cannot be rewritten safely, THEN Hook 1 MUST pass it through unmodified and route its own messages to a channel the host strips before parsing.
45. The Copilot wiring advisory MUST mirror the host's discovery: every directory from the working directory to the git root, `.mcp.json` per directory with `.github/mcp.json` only where the first is absent, plus the user-level config.
46. The Copilot wiring advisory MUST NOT treat the two filenames as a union.
47. Hook 1 MUST rate-limit the Copilot wiring advisory per project, keyed on the project root.
48. WHEN `ARCHCORE_HIDE_WIRING_NUDGE=1` is set, Hook 1 MUST suppress the Copilot wiring advisory.
49. The author MUST keep every other host's byte-level output unchanged when changing behavior for one host.
50. A hook script that invokes the CLI MUST call `archcore <subcmd>` directly, resolved through PATH.
51. A hook script that invokes the CLI MUST pass only a subcommand in the canonical surface `config|doctor|help|hooks|init|mcp|status|update`.
52. Every hook MUST be idempotent.
53. Every bin script MUST be POSIX shell.
54. Every bin script that reads host stdin MUST source `bin/lib/normalize-stdin.sh`.
55. Hook 1 MUST keep `"$PWD"` literal in any command it prints, so no path or environment value leaks into hook output.

## Constraints & Invariants

- Constraint: each PreToolUse hook MUST complete within 1 second, which is its declared budget.
- Constraint: each PostToolUse hook MUST complete within 3 seconds.
- Constraint: the staleness check MUST complete within 3 seconds.
- Constraint: the hooks MUST work without network access in steady state. The update advisory's `update --check` probe is bounded to roughly 500 ms and stays silent offline; it checks freshness and never downloads a binary.
- Constraint: a hook config carries no decision logic. Resolving the path to a script is the sole exception, granted by `host-adapter-contract.spec` and limited to probing candidate plugin roots.
- Invariant: Hook 2 blocks every direct Write or Edit to a `.archcore/**/*.md` file on every host that supports pre-mutation hooks, and blocks nothing outside `.archcore/`.
- Invariant: Hook 3 never blocks an edit, whatever its result or error mode.
- Invariant: Hooks 2 and 3 act on disjoint path sets; Hook 3 is silent for every path Hook 2 acts on.
- Invariant: no hook is registered on an event that cannot carry its only output.
- Invariant: a delegated Write or Edit call is subject to the same PreToolUse behavior as a main-session call; no dispatcher-based bypass exists.
- Invariant: the PostToolUse hooks never modify a file; they only report.
- Invariant: every PostToolUse matcher covers both archcore tool namings, so no hook silently dies when the MCP registration path changes.
- Invariant: every guard that fires also acts — no naming reaches a script that the script cannot recognize.
- Invariant: no hook denies a tool call for a reason unrelated to its own verdict. On a host where any non-zero exit denies, an unlocatable script yields exit 0 and a stderr warning.
- Invariant: Hooks 4 and 5 fire independently on `update_document`; neither depends on the other.
- Invariant: Hook 1 and the PostToolUse hooks exit 0 whatever the outcome.
- Invariant: on a single-parse host, Hook 1's stdout is exactly one JSON document after progress lines are removed, in every combination of nudges and advisories.
- Invariant: Hook 1 never initiates a binary download; the plugin contains no fetcher, and CLI lifecycle is the user's responsibility.
- Invariant: Hook 1 emits the staleness warning at most once per 24 hours per project, the update advisory at most once per 24 hours, and the wiring advisory at most once per 24 hours per project.

## Failure Behavior

1. IF `archcore` is off PATH, THEN Hook 1 MUST emit the install message and exit 0.
2. IF `archcore` is off PATH, THEN Hooks 4, 5, and 6 MUST skip silently. Hooks 2 and 3 do not depend on the CLI: Hook 2 inspects file paths, and Hook 3 scans `.archcore/` with shell grep.
3. IF stdin JSON is malformed, THEN the hook MUST exit 0 with empty output.
4. IF `archcore doctor` hangs, THEN the `timeout 2` inside the script MUST bound it, inside the hook's 3-second envelope.
5. IF git is unavailable, THEN the staleness check MUST skip silently.
6. IF git is unavailable, THEN Hook 1 MUST continue context loading.
7. IF the relation graph is empty, THEN Hook 5 MUST produce no output.
8. IF the staleness timestamp file is missing, empty, or non-numeric, THEN the check MUST treat it as never emitted and run normally.
9. IF `archcore update --check` fails, is unsupported, or the network is down, THEN the advisory MUST stay silent, with no retry and no error surface.
10. IF the CLI session-start payload is empty on a single-parse host, THEN the buffered advisories MUST become the document on their own.
11. IF that payload cannot be taken apart, THEN Hook 1 MUST emit it unchanged and route the advisories to `progress` lines. Losing an advisory is recoverable; corrupting the session's context is not.
12. IF Hook 3 hits any error — grep failure, malformed frontmatter, I/O error — THEN it MUST exit 0 with empty output.
13. IF a hook command cannot locate its script under any candidate plugin root, THEN it MUST exit 0 and name the script in a stderr warning. Enforcement is off for that session, and the warning is the only thing distinguishing it from a clean one.
14. IF a pre-mutation hook times out on Copilot, THEN the plugin MUST NOT rely on the host to block the write. The write proceeds there, both guards are held far inside budget instead, and the path is observed per host as probe D.

## Conformance

The hooks system is conformant when:

1. Every host config registers the behaviors its event set supports, with dual tool naming wherever the host uses matchers. Documented gaps — Cursor's `Write`-only matcher and absent postToolUse, Copilot's omitted context-only preToolUse entry — are declared and asserted by name.
2. `bin/session-start` guards against a plugin-install cwd at any depth, emits an install message when `archcore` is missing, emits init guidance when `.archcore/` is missing, and otherwise delegates to `archcore hooks` and then runs, in order, the empty-state nudge, the Copilot wiring advisory, `bin/check-staleness`, the rate-limited update advisory, and the Copilot flush.
3. `bin/check-archcore-write` blocks `.archcore/**/*.md` writes through each host's honored deny mechanism and allows everything else.
4. `bin/check-code-alignment` injects top-ranked context for source edits inside configured source roots, exits 0 on every path, honors the escape hatch, and keeps its cost independent of match count.
5. `bin/validate-archcore` runs `archcore doctor` directly and is covered by the invocation-log assertions in `@test/unit/validate-archcore.bats` and the allowlist guard in `@test/structure/readme-cli-references.bats`.
6. `bin/check-staleness` detects code-doc drift through git, emits only when matching documents are found, and rate-limits to once per 24 hours.
7. `bin/check-cascade` detects relation cascade after `update_document`.
8. `bin/check-precision` runs after `create_document` and `update_document`.
9. Both PreToolUse hooks complete within 1 second, pinned by `@test/unit/hook-latency.bats`.
10. The PostToolUse hooks complete within 3 seconds.
11. Hook 1 never initiates a binary download.
12. Output shapes follow each host's documented form and are produced by the shared output helpers rather than by per-script string building.
13. On Copilot, Hook 1's stdout parses as one JSON document after progress-line removal in every advisory combination, pinned by `@test/unit/session-start-emit-matrix.bats`; non-Copilot output is unchanged byte for byte, pinned by `@test/unit/session-start-goldens.bats`.
14. Delegated tool invocations are covered by Hooks 2 and 3 identically to main-session calls, and no committed file contains a probe line — the harness in `test/probe/` wraps a copy.
15. Every script that invokes `archcore` passes only allowlisted subcommands.
16. Each host's hook commands resolve their scripts under that host's documented load paths, verified by executing the command rather than by inspecting it. `@test/structure/copilot-plugin.bats` runs Copilot's under `env -u` for the unresolved, single-candidate, and dead-candidate cases.
