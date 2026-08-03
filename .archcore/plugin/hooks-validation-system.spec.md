---
title: "Hooks and Validation System Specification"
status: accepted
tags:
  - "hooks"
  - "plugin"
  - "validation"
---

## Purpose

Define the contract for the hook-based validation, freshness detection, and context-injection layer that enforces the MCP-only principle, ensures `.archcore/` file integrity, detects documentation staleness, and injects project-specific context before source-file edits within the Archcore Plugin.

## Scope

This specification covers all hook entries the plugin ships, across every host config (`hooks/hooks.json`, `hooks/cursor.hooks.json`, `hooks/codex.hooks.json`, `hooks/copilot.hooks.json`): the SessionStart hook (via `bin/session-start` wrapper with staleness check), the PreToolUse hooks on source mutations (blocking direct writes to `.archcore/*.md` and, on hosts whose pre-mutation event carries context, injecting context for source edits), the PostToolUse hook for validation after MCP document operations, the PostToolUse hook for cascade detection after document updates, and the PostToolUse hook for precision checks.

Claude Code's `hooks/hooks.json` is used below as the reference shape because it is the most explicit; per-host divergences are called out where they change behavior rather than only syntax. The canonical event/matcher matrix lives in `plugin-architecture.spec.md`; blocking-semantics translation per host lives in `host-adapter-contract.spec.md`.

It does not cover the MCP server itself, the Archcore CLI lifecycle (the CLI is installed by the user per https://docs.archcore.ai/cli/install/ and resolved via PATH), or the agent's tool restrictions.

## Authority

This specification is the authoritative reference for the plugin's hook configuration. The Always Use MCP Tools ADR provides the architectural rationale for the blocking behavior. The Actualize System ADR and Specification provide the rationale and contract for staleness detection (Layers 1 and 2). The Pre-Code Context Injection idea and its implementation plan provide the rationale for the source-edit context-injection hook. The Host-Wiring Parity ADR governs the dual-naming matcher requirement and the SessionStart dedup/advisory additions. `copilot-mcp-architecture.adr.md` governs why Copilot has no plugin-shipped MCP and therefore needs the wiring advisory. `host-probe-protocol.spec.md` governs how the behaviors specified here are verified on a live host.

## Subject

The hooks system consists of event handlers registered in each host's hooks config that respond to that host's lifecycle events. Three event types enforce quality, the MCP-only principle, documentation freshness, source-edit context alignment, and precision after document mutations. The event *names* differ per host (PascalCase on Claude Code and Codex, camelCase on Cursor and Copilot) and so do the entry shapes. The behaviors are the same everywhere with one declared exception: Copilot registers five entries rather than six, because its pre-mutation event cannot carry context (Hook 3 below).

## Contract Surface

### hooks/hooks.json Structure

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

**Per-host shape divergence.** Cursor and Codex differ from the above only in event-name casing, plugin-root variable, and matcher contents. Copilot differs structurally: entries are flat objects (no nested `hooks[]` group), the command lives under `bash` rather than `command`, the budget under `timeoutSec` rather than `timeout`, each entry sets `cwd: "."` so the hook runs from the user's project and `env.ARCHCORE_HOST=copilot` so detection is deterministic, and its `postToolUse` entries carry **no matcher at all**. A config or a test written by copying another host's and swapping names will load without error and do nothing — which is why `test/structure/hooks.bats` extracts commands through a `.command // .bash` union and fails loudly when the extraction is empty.

**Copilot commands are not a single substitution.** Every other host's command is `${THAT_HOST_VARIABLE}/bin/<script>`. Copilot's is a candidate chain: `$COPILOT_PLUGIN_ROOT`, then `$PLUGIN_ROOT`, then `$CLAUDE_PLUGIN_ROOT`, each probed with `-x` for the script itself, exec'ing the first that holds it and otherwise warning on stderr and exiting 0. The variable this adapter shipped with is documented nowhere by GitHub, and unset it left the literal path `/bin/<script>` — see `copilot-adapter-design.adr.md`. Extraction in tests is therefore by `bin/<script>` token rather than by resolving one variable; a substituting `sed` would emit the surrounding shell as if it were a path.

**Dual tool naming (mandatory).** Every archcore tool in a PostToolUse matcher is listed under BOTH namings: `mcp__archcore__X` (the name a project-level `.mcp.json` server yields) and `mcp__plugin_archcore_archcore__X` (the name Claude Code gives tools from a plugin-bundled MCP server — `mcp__plugin_<plugin>_<server>__*`). Claude Code matchers without regex metacharacters are exact matches, so a single-naming matcher silently never fires in one of the two setups. Guarded by `test/structure/hooks.bats`; rationale in `host-wiring-parity.adr.md`.

Matchers and scripts solve different halves of the same problem, and the split matters. A matcher decides *whether the script runs at all*; the script decides *what to do*. `bin/lib/normalize-stdin.sh` therefore folds all three namings — including Copilot's flat `archcore-<tool>`, where the host joins server and tool with a hyphen — into the canonical `mcp__archcore__*` before any guard inspects a tool name. Without that fold a guard fires and then silently falls through its own filter, which is the worst of both: cost paid, protection absent. On Copilot, where `postToolUse` takes no matcher, that filtering *is* the whole selection mechanism.

Historical note: a prior revision included a PostToolUse `Write|Edit` matcher invoking `validate-archcore` as defense-in-depth. The hook was dead in practice — PreToolUse blocks all Write/Edit to `.archcore/*.md` before they reach PostToolUse (PostToolUse fires only on success per Claude Code hooks semantics), and `.archcore/settings.json` / `.archcore/.sync-state.json` are allowlisted, so `validate-archcore` never had an edge case to handle through that path. It was removed to eliminate a per-Write/Edit shell fork across the entire repository. The MCP matcher below remains the single validation entry point.

The two PreToolUse entries on `Write|Edit` are deliberately coupled: `check-archcore-write` short-circuits on `.archcore/*.md` and denies; `check-code-alignment` short-circuits on everything INSIDE `.archcore/` with exit 0 (silent). On any source path only the alignment hook does real work. The order matters for fast exit on blocks but does not affect correctness — the two run as independent entries with independent budgets on every host that registers both.

### The Copilot output channel

Copilot's stdout contract is not "the host reads what it recognizes and ignores the rest". Per the [hooks reference](https://docs.github.com/en/copilot/reference/hooks-reference), a line that is a single complete JSON object with `"type":"progress"` is consumed as a progress event and **removed** from the stream; every other line — blank lines, plain text, and JSON objects that are not progress messages — is preserved verbatim. On exit the preserved lines are concatenated, trimmed, and parsed with **one** `JSON.parse`; if that fails, the hook is treated as having produced **no output at all**.

Two consequences drive the design of every Copilot emission in this specification:

1. **A hook on this host may emit at most one JSON document.** Plain text appended after a JSON payload does not add a note — it discards the payload, the ~9 KB of Archcore context included, along with the note itself. `bin/session-start` therefore buffers every advisory and folds them into the CLI hook's document at the end (phase 7). Emitting them as trailing plain text was the behavior through plugin 0.6.1 and cost the whole session's context whenever any advisory fired.
2. **Output fields are per-event, and `additionalContext` is not universal.** `sessionStart`, `postToolUse`, `postToolUseFailure`, `notification` and `subagentStart` accept it; `preToolUse` accepts only `permissionDecision`, `permissionDecisionReason` and `modifiedArgs`. A context-only hook on `preToolUse` can therefore never deliver anything on this host — see Hook 3.

Hosts other than Copilot are unaffected by both points, and their byte-level output is pinned by `test/unit/session-start-goldens.bats` so that a change made for Copilot cannot move theirs.

### Hook 1: SessionStart (Context Loading + Staleness Check)

**Event**: SessionStart / `sessionStart` (fires when a session begins or resumes)
**Matcher**: empty (matches all session sources: startup, resume, clear, compact)
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/session-start` (per-host plugin-root variable; on Copilot, the candidate chain)
**Behavior**: pipeline of phases, in this order:

0. **Plugin-install-dir guard.** Exit 0 silently when `$PWD` contains an install-cache path fragment (`.cursor/plugins/`, `.claude/plugins/`, `.codex/plugins/`, `.copilot/installed-plugins/`, `plugins/cache/`) or when a bounded upward walk finds a `.cursor-plugin/`, `.claude-plugin/`, `.codex-plugin/`, or `.plugin/` manifest — a cwd misrouted into a plugin install (at any depth) must never surface the plugin's own bundled files as the user's knowledge base (`cursor-mcp-architecture.adr.md`, extended per `host-wiring-parity.adr.md`).
1. **CLI availability check.** If `archcore` is not on PATH, emit an install message pointing at https://docs.archcore.ai/cli/install/ and exit 0 (on Copilot the message additionally names the mandatory next step — `archcore init --agent copilot --project "$PWD"` — because installing the CLI alone still leaves that host without document tools). No further phases run. Installing the CLI mid-session does NOT reconnect a Claude Code MCP server that failed to register at session start — users must restart the host after a fresh install.
2. **Project check.** If `.archcore/` does not exist, emit context instructing the agent to call `mcp__archcore__init_project` on first Archcore operation, then exit 0. On Copilot that tool does not exist day-one (no plugin-shipped MCP — `copilot-mcp-architecture.adr.md`), so the instruction body is forked to the CLI wiring command `archcore init --agent copilot --project "$PWD"`; the `/archcore:init` suffix sentence stays shared byte-for-byte across hosts.
3. **Context loading.** If `.archcore/` exists, pipe stdin into `archcore hooks <host> session-start`; swallow any non-zero exit so SessionStart remains non-blocking. The CLI-side handler dedupes duplicate SessionStart emissions per `session_id`+`source` (Cursor: `conversation_id`) via short-window XDG-state stamps, fail-open — so a project-level hook installed by `archcore init --agent` coexisting with this plugin hook emits context once, for any plugin/CLI version combination. It also emits the response in that host's shape (CLI ≥ v0.6.4 knows Copilot's bare `additionalContext`). On Copilot this payload is **captured rather than streamed**, because phase 7 must join it with anything the phases below produce; on every other host it streams straight through as before.
4. **Empty-state nudge.** When `.archcore/` exists but carries no substantive documents, emit a nudge pointing at `/archcore:init`. Suppress with `ARCHCORE_HIDE_EMPTY_NUDGE=1`.
5. **Copilot wiring advisory.** Copilot only: when the CLI is present, `.archcore/` exists, and no archcore server is wired, emit a nudge naming `archcore init --agent copilot --project "$PWD"`. Detection mirrors the host's own discovery, measured on Copilot CLI 1.0.76: a config is read from **every** directory between the working directory and the git root — not merely those two ends — and within each directory `.mcp.json` wins, with `.github/mcp.json` consulted only where the first is absent. User-level `${COPILOT_HOME:-~/.copilot}/mcp-config.json` also counts. Checking fewer places nags a project that works; treating the two filenames as a union stays silent for a project that does not, which is the failure this advisory exists to prevent. Rate-limited per project (stamp keyed by `cksum` of the project root — a global stamp would let one repo silence another's mandatory step); suppress with `ARCHCORE_HIDE_WIRING_NUDGE=1`. Detection is pure stat/grep plus one `git rev-parse`; the other hosts never reach this phase.
6. **Staleness check.** Call `bin/check-staleness` to detect code-doc drift via git and emit findings.
7. **Outdated-CLI advisory.** Run `archcore update --check` (24h-cached, ~500ms-bounded, silent on any failure, exit 0 always; an older CLI without the flag degrades silently). When it reports a newer version and the advisory's own 24h rate-limit stamp is due, emit a one-line nudge naming `archcore update`.
8. **Copilot flush.** Copilot only: everything phases 4–7 buffered leaves as a single JSON document, spliced into the payload captured in phase 3. The splice fires only on the exact `{"additionalContext":"…"}` shape it can take apart; an unrecognized payload passes through untouched and the advisories fall back to `progress` lines, which the host strips before parsing. Both branches satisfy the one-document rule; corrupting the host's context payload would be a worse failure than delivering an advisory through a lesser channel. A no-op on every other host, where phases 4–7 have already printed their lines.

Phases 4–7 are additive — if any fails or produces no output, the preceding phases are unaffected.

**Input**: JSON on stdin with `session_id`, `cwd`, `hook_event_name` (host-specific field names, normalized by `bin/lib/normalize-stdin.sh`)
**Output**: the host's context envelope — `hookSpecificOutput.additionalContext` (Claude Code, Codex), top-level `additionalContext` (Copilot, exactly one document), `additional_context` (Cursor), plain text (OpenCode)

### Hook 2: PreToolUse — Block Direct Writes

**Event**: PreToolUse / `preToolUse` (fires before a tool call executes)
**Matcher**: the host's file-mutation tools — `Write|Edit` (Claude Code), `Write` (Cursor, which exposes no Edit tool), `Write|Edit|apply_patch` (Codex), `create|edit|str_replace_editor|apply_patch` (Copilot)
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/check-archcore-write`
**Timeout**: 1 second
**Input**: JSON on stdin containing the tool call details including the target path

**Behavior**:

1. Extract the target path from the tool input (stdin JSON, via the normalizer — on Copilot the path lives inside an escaped JSON string under `toolArgs`)
2. Check if the path matches `.archcore/**/*.md` (document files)
3. If NO match: exit 0 with empty output (allow the operation)
4. If MATCH: deny, using the mechanism the host honors

**Deny mechanism is host-specific.** On Claude Code, Codex and Cursor, exit code 2 is a blocking error — stderr goes to the model as feedback and the tool call is blocked. On Copilot the guard writes `{"permissionDecision":"deny","permissionDecisionReason":"…"}` to stdout with exit 0 instead, and the reason is why: **every** non-zero exit denies there — exit 2 explicitly (its stdout JSON merged with the deny), any other non-zero exit as `Denied by preToolUse hook (hook errored)` — but only the JSON form carries reason text back to the user (hooks-reference, re-read 2026-07-27). Those two fields are also among the only three the event accepts, which is what makes this hook viable on Copilot where Hook 3 is not. The branch lives in the shared script keyed on `ARCHCORE_HOST`; see `host-adapter-contract.spec.md` for the full translation table.

**On that host the failure mode inverts.** Where exit 2 is the deny channel, a guard that cannot start degrades to no enforcement. Where every non-zero exit denies, it degrades to refusing every matched tool call — and the host cannot tell the two apart. This is why hook bootstrap on Copilot is specified below as a correctness requirement rather than left to the adapter, and why `hooks/copilot.hooks.json` exits 0 when it cannot locate a script.

**Reason message when blocking**:

```
Direct writes to .archcore/ documents are not allowed. Use Archcore MCP tools instead:
- create_document: create a new document
- update_document: modify an existing document
- remove_document: delete a document
This ensures validation, templates, and the sync manifest stay consistent.
```

**Exceptions** (paths that are NOT blocked):

- `.archcore/settings.json` — configuration file, not a document
- `.archcore/.sync-state.json` — managed by MCP tools internally

### Hook 3: PreToolUse — Inject Context for Source Edits

**Event**: PreToolUse / `preToolUse`
**Matcher**: same as Hook 2, per host
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/check-code-alignment`
**Timeout**: 1 second
**Registered on**: Claude Code, Cursor, Codex. **Not registered on Copilot** — that host's `preToolUse` accepts only `permissionDecision`, `permissionDecisionReason` and `modifiedArgs`, so a hook whose entire product is context could only fork a process per edit and emit into a channel the host discards. The omission is declared, not silent: `test/structure/copilot-plugin.bats` asserts its absence there, `test/structure/hooks.bats` and `test/structure/host-coverage-matrix.bats` carve it out by name so that no other host can drop any other script quietly, and a negative-control test asserts it is still present in the other three configs. If Copilot adds context to `preToolUse`, re-registering the entry is the whole change.
**Input**: JSON on stdin containing the tool call details including the target path

**Behavior**:

1. Extract `file_path` via the normalized stdin layer.
2. Short-circuit (exit 0, empty output) if any of: no `file_path`, no `.archcore/` directory, path is inside `.archcore/`, env `ARCHCORE_DISABLE_INJECTION=1`.
3. Normalize to cwd-relative; exit 0 if path is absolute outside `$CWD`.
4. Enforce source-root filter: path must start with a configured source root. Default set: `src lib app pkg cmd internal apps packages modules components`. Override via `.archcore/settings.json` → `codeAlignment.sourceRoots` (JSON array). Exit 0 if not matched.
5. Generate candidate tokens — directory prefixes of the file path, longest first (capped at 5 levels).
6. Scan `.archcore/**/*.md` with `grep -rlF <token>` per token in longest-first order — one grep per token, not per match. Score each matched document by specificity (length of the longest matching token) combined with type priority: `rule=5, cpat=4, adr=3, spec=2, guide=1`. Only these five types are eligible — other types (prd, idea, plan, rfc, doc, task-type, etc.) are ignored as not enforceable or too high-level for line-of-code context.
7. Rank desc, take top 3.
8. Render a compact block:
   ```
   [Archcore Context] Before editing <relative-path>:
   - <type>: <title> [<short-doc-path>]
   ...
   ```
   Output capped at 2 KB.
9. Emit the host's context envelope:
   - Claude Code / Codex: `{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"..."}}`
   - Cursor: `{"additional_context":"..."}` (may be ignored by current Cursor — graceful degradation, documented limitation)
   - Copilot: the helper retains a top-level `{"additionalContext":"..."}` arm so no host falls through the emit matrix silently, but on that host the arm is unreachable in practice because no Copilot hook registers this script.

**Non-blocking by design**: exit code is always 0. Any error in the pipeline (missing tools, malformed JSON, empty matches) results in a silent pass. Injection is strictly additive and must never prevent a write.

**That additivity is exactly why its cost is a correctness concern.** A timeout here raises nothing: the write proceeds and no context arrives, with nothing in any log. The hook once cost roughly 6 ms per *matching* document — two process spawns each, per token — so a knowledge base where ~170 documents mentioned a common source root exceeded the 1 s budget outright, and push-mode stopped working on precisely the repositories with the most context to give. Deduplication now happens in a single pass, so process spawns scale with the number of tokens rather than the number of matches. `test/unit/hook-latency.bats` pins both the absolute budget and the independence from match count.

**Escape hatch**: set environment variable `ARCHCORE_DISABLE_INJECTION=1` to disable injection globally for a session.

**Relationship to Hook 2**: where both are registered, both fire on the same matcher. Hook 2 handles `.archcore/*.md` paths (blocks). Hook 3 handles source paths (injects). Their active path sets are disjoint by construction. They are separate hook entries with separate budgets, so a slow Hook 3 cannot consume Hook 2's.

#### Sub-agent tool invocations (delegated)

PreToolUse hooks fire at the tool-execution boundary, not at the session boundary. Any file-mutation tool call matches the host's matcher regardless of whether the call originates from the main conversation or from a delegated sub-agent. Hook 2 and Hook 3 therefore cover delegated Write/Edit identically to main-session Write/Edit. Input stdin carries the tool name and path in the same shape; the host does not annotate sub-agent origin in a way that the hooks need to consume or branch on.

Scope clarifications:

- **Archcore's own sub-agents** (`archcore-assistant`, `archcore-auditor`) do NOT have `Write` or `Edit` in their tools allowlist (see `agent-system.spec.md` Tool Access Matrix). They cannot trigger Hooks 2 or 3 by definition. The sub-agent coverage discussion concerns general-purpose and third-party agents dispatched by the user for code work.
- **Claude Code**: hook coverage for delegated Write/Edit holds by the host's PreToolUse contract.
- **Copilot**: has a delegation surface (`subagentStart` / `subagentStop`), so the same question is answerable there. Note that `subagentStart` does accept `additionalContext`, unlike `preToolUse`.
- **Cursor**: the PreToolUse matcher in `cursor.hooks.json` is `Write` only, not `Write|Edit` — a pre-existing multi-host asymmetry, independent of the sub-agent question. Sub-agent-originated Edit calls on Cursor go unhooked for the same reason main-session Edit calls do.

This is probe A-d in `host-probe-protocol.spec.md`, which is where the per-host result is recorded. A specification claim about delegated coverage is not evidence; the record is.

### Hook 4: PostToolUse — Validate After MCP Document Operations

**Event**: PostToolUse / `postToolUse` (fires after a tool call succeeds)
**Matcher**: the five document-mutation tools, each under both namings (`mcp__archcore__X|mcp__plugin_archcore_archcore__X` for `create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`). Copilot registers no matcher; the script filters on the normalized tool name instead.
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore`
**Timeout**: 3 seconds
**Input**: JSON on stdin containing the completed MCP tool call details

**Behavior**:

1. Extract `tool_name` from stdin JSON (normalized to the canonical naming)
2. Detect the archcore MCP prefix — run `archcore doctor` directly (resolved via PATH, wrapped in `timeout 2` and `|| true`)
3. If validation passes: exit 0 with empty output
4. If validation fails: exit 0 with the host's context envelope containing validation context

This is the sole validation hook. Because PreToolUse blocks all direct Write/Edit to `.archcore/*.md` and MCP tools are the supported interface for document operations, this single matcher fires after every document mutation that can actually touch the knowledge base.

### Hook 5: PostToolUse — Cascade Detection After Document Updates

**Event**: PostToolUse / `postToolUse`
**Matcher**: `mcp__archcore__update_document|mcp__plugin_archcore_archcore__update_document` (Copilot: none — script-side filtering)
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/check-cascade`
**Timeout**: 3 seconds
**Input**: JSON on stdin containing the completed `update_document` tool call details

**Behavior**:

1. Extract updated document path from `tool_input.path` in stdin JSON
2. Query relation graph for documents where the updated document is the **target** of `implements`, `depends_on`, or `extends` relations
3. If no such relations found: exit 0 with empty output (no cascade)
4. If cascade found: exit 0 with the host's context envelope containing the affected document list

This hook fires **in addition to** Hook 4 (validation). Both hooks fire independently on `update_document` — Hook 4 validates structural integrity, Hook 5 detects cascade staleness. Neither depends on the other.

**Fires only on `update_document`**: New documents (`create_document`) cannot cause cascade because nothing depends on them yet. Removed documents (`remove_document`) are intentional deletions.

**Excludes `related` relations**: Only `implements`, `depends_on`, and `extends` indicate directional dependency where cascade staleness is meaningful.

### Hook 6: PostToolUse — Precision Check

**Event**: PostToolUse / `postToolUse`
**Matcher**: `create_document` and `update_document`, each under both namings (Copilot: none — script-side filtering)
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/check-precision`
**Timeout**: 3 seconds

Phase 1 of the Precision Initiative (see `precision-over-coverage.adr`). Reads the resulting file from disk and runs four checks: forbidden vagueness lexicon, mandatory sections by type (adr/rule/spec/guide/rfc), frontmatter title+status presence, body length ≥ 200 chars. Emits soft warnings as additional context. Always exits 0; never blocks.

### PostToolUse Output Formats

**Validation (Hook 4)** — when issues found:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "Archcore validation found issues: <issues>. Run archcore doctor --fix to auto-fix orphaned relations."
  }
}
```

**Cascade Detection (Hook 5)** — when cascade detected:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "[Archcore Cascade] Updated \"<document-title>\".\nDocuments that may need review:\n  → <path> (<relation-type> this document)\nRun /archcore:audit --drift for detailed analysis."
  }
}
```

### PreToolUse Injection Output Format (Hook 3)

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "additionalContext": "[Archcore Context] Before editing <relative-path>:\n- <type>: <title> [<short-doc-path>]\n..."
  }
}
```

### Output envelope per host

The scripts never build these by hand; the output helpers in `bin/lib/normalize-stdin.sh` select the shape from `ARCHCORE_HOST`, which is why a new host costs one branch in one file.

| Host | Context envelope | Pre-mutation deny |
|---|---|---|
| Claude Code | `hookSpecificOutput.additionalContext` | exit 2 + stderr |
| Codex CLI | `hookSpecificOutput.additionalContext` | exit 2 + stderr |
| Cursor | `additional_context` (flat) | exit 2 + stderr |
| GitHub Copilot CLI | `additionalContext` (flat, top level) — at most one document per hook invocation | stdout `{"permissionDecision":"deny",…}`, exit 0 |
| OpenCode | plain message, no JSON | bridge throws `Error(reason)` |

On Copilot the envelope is accepted on `sessionStart`, `postToolUse`, `postToolUseFailure`, `notification` and `subagentStart`, and **not** on `preToolUse`. A script whose only product is context must therefore be registered on an event that carries it, or not registered at all.

### Exit Code Semantics

| Hook | Exit 0 | Exit 2 |
|------|--------|--------|
| SessionStart | Always (output = install msg / init msg / context + nudges + staleness + advisories) | N/A |
| PreToolUse block (Hook 2, allow) | Empty output, operation proceeds | N/A |
| PreToolUse block (Hook 2, block) | Copilot only: deny JSON on stdout | Claude Code / Codex / Cursor: stderr → model feedback, operation blocked |
| PreToolUse inject (Hook 3) | Always (no match → empty; match → context) | N/A |
| PostToolUse validation (Hook 4) | Always (clean → empty; issues → context) | N/A |
| PostToolUse cascade (Hook 5) | Always (no cascade → empty; cascade → context) | N/A |
| PostToolUse precision (Hook 6) | Always (clean → empty; warnings → context) | N/A |

The exit-2 column is Claude Code's convention, adopted by Codex and Cursor. Copilot is the one host where a non-zero exit is not the *reason-carrying* deny channel — it denies on any non-zero exit, including exit 2, but only the stdout JSON tells the user why. See Hook 2.

### bin/ Scripts

Six executable hook scripts in `bin/`, plus the stdin normalization library. The plugin does **not** bundle the Archcore CLI binary or any launcher wrapper — every script that invokes the CLI calls `archcore <subcmd>` directly, resolved via PATH. If `archcore` is missing, hooks degrade silently (timeout + `|| true`).

#### `bin/session-start`

Shell script that handles the SessionStart pipeline (install-dir guard, CLI check, project check, context loading, empty-state nudge, Copilot wiring advisory, staleness, update advisory, Copilot flush).

Requirements:

- Executable (`chmod +x`); `#!/bin/sh`
- Sources `bin/lib/normalize-stdin.sh`
- Exits 0 in all cases
- Exits silently when run from inside a plugin install — cache path fragments in `$PWD` or a manifest found by the bounded upward walk
- When `archcore` is not on PATH: emits an install message pointing at https://docs.archcore.ai/cli/install/ and exits 0. On Copilot the message additionally names `archcore init --agent copilot --project "$PWD"`, because the CLI alone does not give that host document tools
- When `.archcore/` is absent: emits context pointing at `mcp__archcore__init_project` (copilot: at `archcore init --agent copilot`, since no plugin MCP exists there)
- Otherwise, in this order: invokes `archcore hooks <host> session-start` and discards any non-zero exit; emits the empty-state nudge when `.archcore/` holds nothing substantive; on Copilot only, emits the wiring advisory when no archcore server is discoverable from any directory between cwd and the git root (either `.mcp.json` or, where that is absent, `.github/mcp.json`) nor at user level; calls `bin/check-staleness`; runs the `archcore update --check`-backed advisory (own 24h rate-limit stamp; silent when the CLI is current, the flag is unsupported, or the network is unavailable)
- On Copilot: emits **exactly one JSON document** on stdout. The CLI payload is captured rather than streamed, the four advisories above are buffered, and the flush folds them together — with an unrecognized payload passing through untouched and its advisories degrading to `progress` lines. On every other host the same advisories print as plain text, byte-identically to prior releases
- Keeps `"$PWD"` literal in any command it prints, so no path or environment value leaks into hook output
- Invokes only allowlisted CLI subcommands: `hooks`, `update` (as `update --check` only), `--version`
- Degrades gracefully — never errors, just warns

#### `bin/check-archcore-write`

Shell script that reads stdin JSON, extracts the target path, and decides whether to block.

Requirements: executable; `#!/bin/sh`; reads JSON from stdin; exit 0 when allowing; denies through the mechanism the host honors — exit 2 + stderr on Claude Code / Codex / Cursor, stdout `{"permissionDecision":"deny","permissionDecisionReason":…}` with exit 0 on Copilot, where any non-zero exit would deny without carrying the reason; completes within 1 second.

#### `bin/check-code-alignment`

PreToolUse handler that injects applicable `.archcore/` context for source-file edits.

Requirements:

- Executable; `#!/bin/sh`; sources `bin/lib/normalize-stdin.sh`; reads JSON from stdin
- Exits 0 in all cases — MUST NEVER return non-zero (injection is additive)
- Short-circuits silently on `.archcore/*` paths (Hook 2 handles those) and on non-source-root paths
- Honors `.archcore/settings.json` → `codeAlignment.sourceRoots` when configured; otherwise uses the default root set
- Honors `ARCHCORE_DISABLE_INJECTION=1` escape hatch
- Ranks by specificity first, type priority (`rule > cpat > adr > spec > guide`) second
- Considers only `rule`, `cpat`, `adr`, `spec`, `guide` document types
- Emits at most 3 matches, capped at 2 KB total output
- Completes within 1 second, and its cost MUST NOT scale with the number of matching documents — process spawns are bounded by the token count, not the match count
- Emits the host's context envelope (see the table above)
- Is registered only on hosts whose pre-mutation event carries context; see Hook 3

#### `bin/validate-archcore`

Shell script that reads stdin JSON, determines if validation is needed (by tool_name prefix), and runs `archcore doctor` directly via PATH.

Requirements:

- Executable; `#!/bin/sh`; reads JSON from stdin
- Fires unconditionally for archcore MCP tools under every naming, which the normalizer has already folded to one; the legacy Write/Edit branch is retained as defensive code but is never reached from the current hooks config
- Invokes `archcore doctor` directly (`timeout 2 archcore doctor 2>&1`), no wrapper script
- Exits 0 in all cases — silent skip when `archcore` is unavailable
- Outputs the host's context envelope when reporting issues, empty output when clean
- Completes within 3 seconds

##### Test Contract

The script's CLI subcommand invocation is locked at two test layers, so a phantom subcommand (e.g. an accidental return to the historical `archcore validate`) cannot reach production:

- **Allowlist guard (README references)** — `test/structure/readme-cli-references.bats` extracts every backtick-quoted `archcore <subcmd>` reference in `README.md` and asserts each is a member of the canonical CLI surface: `config doctor help hooks init mcp status update`.
- **Invocation assertion (unit)** — `test/unit/validate-archcore.bats` runs the script under a logging mock (`mock_archcore_logging` + `MOCK_ARCHCORE_LOG`) and asserts `doctor` was invoked. Two tests cover this: `validate-archcore calls archcore doctor (not validate)` (positive + negative assertion) and `validate-archcore invokes only allowlisted subcommands` (allowlist-guard mirror).

When the upstream CLI surface changes (subcommand added/renamed/removed), update `ARCHCORE_SUBCOMMANDS` in `readme-cli-references.bats` and add an invocation-log assertion for any new subcommand a bin/ script starts using. The `cli-integration-tests.rule.md` rule mandates this contract for every change that touches plugin scripts, hook configs, MCP configs, or skill/agent prose that prescribes CLI usage.

#### `bin/check-staleness`

Shell script called from `bin/session-start` after context loading. Detects code-document drift via git history comparison.

Requirements: executable; `#!/bin/sh`; exit 0 in all cases; output ≤ 2 KB plain text or empty; completes within 3 seconds; skips gracefully if git unavailable / `.archcore/` has no commits / not a git repo; rate-limited to one emission per 24 h per project via a timestamp file (`$CLAUDE_PLUGIN_DATA/archcore/last-staleness` → `$XDG_DATA_HOME/...` → `$HOME/.local/share/...`); emits ONLY when matching documents are found (no generic "N files changed" fallback).

#### `bin/check-cascade`

PostToolUse handler for cascade detection after `update_document`.

Requirements: executable; `#!/bin/sh`; reads JSON from stdin; exit 0 in all cases; outputs the host's context envelope when cascade detected, empty otherwise; invokes `archcore` directly via PATH; completes within 3 seconds; skips gracefully if `archcore` is unavailable.

#### `bin/check-precision`

PostToolUse handler running the precision lexicon, mandatory-sections, frontmatter, and length checks. Always exits 0; reads files from disk (no CLI shell-out); ≤ 3 seconds.

## Normative Behavior

- The PreToolUse block hook (Hook 2) MUST block all Write/Edit calls targeting `.archcore/**/*.md` files on every host, using that host's honored deny mechanism.
- WHERE a host does not carry reason text on a non-zero exit, the block hook MUST emit that host's deny payload instead; a guard whose reason never reaches the user is a guard the user cannot act on.
- The PreToolUse block hook MUST NOT block writes to `.archcore/settings.json` or `.archcore/.sync-state.json`.
- The PreToolUse block hook MUST NOT block writes to files outside `.archcore/`.
- WHERE a host treats every non-zero exit as a deny, each hook command MUST resolve its script before invoking it and MUST exit 0 when it cannot — a guard that fails to start on such a host blocks the user's work instead of merely going unenforced.
- WHERE a hook command cannot resolve its script, it MUST report that on stderr rather than exiting silently.
- The PreToolUse injection hook (Hook 3) MUST exit 0 on every code path and MUST NEVER block or fail an edit.
- The PreToolUse injection hook MUST short-circuit silently for paths inside `.archcore/`, paths outside configured source roots, and paths that produce no matches.
- The PreToolUse injection hook MUST rank matches by specificity first (longest matching directory prefix wins), type priority second, and MUST restrict eligible types to `rule`, `cpat`, `adr`, `spec`, `guide`.
- The PreToolUse injection hook MUST cap output at 3 documents and 2 KB.
- The PreToolUse injection hook MUST honor the `ARCHCORE_DISABLE_INJECTION=1` environment variable as an unconditional off-switch.
- WHERE a host's pre-mutation event does not accept a context field, a context-only hook MUST NOT be registered on it, and its absence MUST be asserted by name rather than left to a loosened parity check.
- Both PreToolUse hooks MUST complete far enough inside their 1 s budget that a timeout is unreachable on realistic knowledge bases. On Copilot a `preToolUse` timeout fails OPEN, which turns latency into a correctness property rather than a comfort one.
- The PreToolUse hooks MUST treat delegated Write/Edit tool calls identically to main-session calls — no special-casing, no skipping.
- Every PostToolUse matcher MUST list each archcore tool under both namings (`mcp__archcore__X|mcp__plugin_archcore_archcore__X`) — Claude Code matchers are exact-match, and the two MCP registration paths (project `.mcp.json` vs plugin-bundled server) yield different tool names.
- `bin/lib/normalize-stdin.sh` MUST fold every host's MCP tool naming to the canonical `mcp__archcore__*` before any guard script inspects a tool name, so that a guard which fires always also acts.
- WHERE a host's post-mutation event accepts no matcher, the handling script MUST filter to the same tool set the matcher would have selected.
- The PostToolUse validation hook reports validation issues as additional context but does not block or revert operations.
- The PostToolUse MCP validation matcher MUST fire after all document mutation MCP tools.
- The hooks config MUST NOT register a Write/Edit matcher on PostToolUse.
- The PostToolUse cascade hook MUST fire only after `update_document`, not after `create_document` or `remove_document`.
- The PostToolUse cascade hook MUST only flag documents connected via `implements`, `depends_on`, or `extends` (not `related`).
- The SessionStart hook MUST exit silently when run from inside a plugin install (cache path fragments or upward-walk manifest hit).
- The SessionStart hook MUST emit the install message when `archcore` is not on PATH and MUST NOT block the session in that case.
- The SessionStart staleness check MUST run after context loading, and after the empty-state and wiring nudges.
- The SessionStart staleness check output MUST NOT exceed 2 KB.
- The SessionStart staleness check MUST rate-limit itself to once per 24h via a persistent timestamp file.
- The SessionStart update advisory MUST be backed by `archcore update --check`, MUST rate-limit itself to once per 24h via its own stamp, and MUST stay silent on any failure (including an older CLI without the flag).
- WHERE a host parses a hook's entire stdout with a single JSON parse, SessionStart MUST emit exactly one JSON document — every nudge, advisory and finding folded into it, never appended after it. A payload the plugin cannot safely rewrite MUST be passed through unmodified, with its accompanying messages routed to a channel the host strips before parsing.
- The Copilot wiring advisory MUST mirror the host's discovery: every directory from the working directory to the git root, `.mcp.json` per directory with `.github/mcp.json` only where the first is absent, plus the user-level config. It MUST NOT treat the two filenames as a union.
- The Copilot wiring advisory MUST be rate-limited per project, keyed on the project root, and MUST be suppressible by `ARCHCORE_HIDE_WIRING_NUDGE=1`.
- Changes made for one host MUST NOT alter another host's byte-level output; `test/unit/session-start-goldens.bats` is the pin.
- Hook scripts that invoke the CLI MUST call `archcore <subcmd>` directly (resolved via PATH); the plugin does NOT ship any launcher wrapper, version pin, or cache directory. Reintroducing a `bin/archcore*` launcher or `bin/CLI_VERSION` requires a fresh ADR per `stack-and-tooling.rule`.
- Hook scripts that invoke the CLI MUST only pass subcommands in the canonical surface (`config|doctor|help|hooks|init|mcp|status|update`); the contract is enforced by `test/structure/readme-cli-references.bats` and per-script invocation-log assertions.
- All hooks MUST be idempotent.

## Constraints

- PreToolUse hooks (Hook 2 and Hook 3) must each complete within 1 second.
- PostToolUse hooks must complete within 3 seconds.
- SessionStart staleness check must complete within 3 seconds.
- Hooks must work without network access in steady state. The plugin never downloads anything — CLI lifecycle is the user's responsibility via the official installer. (The update advisory's `update --check` probe is bounded to ~500ms and silent offline; it checks freshness, it never downloads a binary.)
- Hooks must degrade gracefully if the Archcore CLI is missing (skip validation/cascade silently; SessionStart prints install guidance and exits 0).
- The injection hook MUST degrade gracefully for large corpora — either by completing in time at lower fidelity or by short-circuiting cleanly; it MUST NOT time out in a way that costs a write its context.
- Bin scripts must be POSIX-compatible shell (no bash-specific features).
- Hook configs carry no decision logic. Resolving the path to a script is the sole exception, granted by `host-adapter-contract.spec.md` item 3 and limited to probing candidate plugin roots.

## Invariants

- The PreToolUse block hook blocks 100% of direct Write/Edit to `.archcore/**/*.md` files, on every host that supports pre-mutation hooks.
- The PreToolUse block hook never blocks writes outside `.archcore/`.
- The PreToolUse injection hook never blocks any edit, regardless of result or error mode.
- The PreToolUse injection hook and the PreToolUse block hook act on disjoint path sets — the injection hook is silent for every path the block hook acts on.
- No hook is registered on an event that cannot carry its only output.
- Delegated Write/Edit tool calls are subject to the same PreToolUse behavior as main-session calls; there is no dispatcher-based bypass.
- The PostToolUse hooks never modify files — they only report.
- Every PostToolUse matcher covers both archcore tool namings — no hook silently dies when the MCP registration path changes.
- Every guard that fires also acts: no naming reaches a script that the script cannot recognize.
- No hook denies a tool call for a reason unrelated to its own verdict. On a host where any non-zero exit denies, an unlocatable script yields exit 0 and a stderr warning.
- Hook 4 (validation) and Hook 5 (cascade) fire independently on `update_document` — neither depends on the other.
- SessionStart and PostToolUse hooks exit 0 regardless of outcome.
- On a single-parse host, SessionStart's stdout is exactly one JSON document after progress lines are removed — in every combination of nudges and advisories, including all of them at once.
- The PreToolUse block hook exits 0 (allow, or Copilot deny-JSON) or 2 (block) — never other codes.
- The PreToolUse injection hook exits 0 — never other codes.
- SessionStart never initiates a binary download (the plugin no longer has download logic; CLI lifecycle is the user's responsibility).
- SessionStart emits the staleness warning at most once per 24h per project, the update advisory at most once per 24h, and the wiring advisory at most once per 24h per project.

## Error Handling

- If `archcore` is not on PATH: SessionStart emits the install message and exits 0; PostToolUse hooks skip validation/cascade silently. PreToolUse hooks (Hooks 2 and 3) do not depend on the CLI — Hook 2 only inspects file paths; Hook 3 scans `.archcore/` via shell grep.
- If stdin JSON is malformed: exit 0 with empty output (fail open, don't break the session).
- If `archcore doctor` hangs: enforced by `timeout 2` inside the script plus the hook's 3-second envelope.
- If git is unavailable for staleness check: skip silently, context loading continues.
- If relation graph is empty for cascade check: produce no output (no cascade possible).
- If the staleness timestamp file is missing, empty, or contains non-numeric data: treat as "never emitted" and run the check normally.
- If `archcore update --check` fails, is unsupported, or the network is down: the advisory stays silent; no retry, no error surface.
- If the CLI session-start payload is empty on a single-parse host: the buffered advisories become the document on their own.
- If that payload is present but in a shape the flush cannot take apart: emit it unchanged and route the advisories to `progress` lines. Losing an advisory is recoverable; corrupting the session's context is not.
- If the injection hook encounters any error (grep failure, malformed frontmatter, I/O error): exit 0 with empty output.
- If a hook command cannot locate its script under any candidate plugin root: exit 0 with a stderr warning naming the script. Enforcement is off for that session, and the warning is the only thing distinguishing it from a clean one.
- If a host's pre-mutation hook times out: on Copilot the write proceeds (fail-open) — mitigated by keeping both guards far inside budget rather than by relying on the host. Elsewhere the host's own timeout semantics apply. Observed per host as probe D in `host-probe-protocol.spec.md`.

## Conformance

The hooks system conforms to this specification if:

1. Every host hooks config (`hooks.json`, `cursor.hooks.json`, `codex.hooks.json`, `copilot.hooks.json`) registers the behaviors its host's event set supports, with dual tool naming wherever the host uses matchers. Documented per-host gaps (Cursor's `Write`-only matcher and absent postToolUse; Copilot's omitted context-only preToolUse entry) are declared and asserted by name, not silent.
2. `bin/session-start` guards against plugin-install cwd at any depth, emits an install message when `archcore` is missing, emits init guidance when `.archcore/` is missing, otherwise delegates to `archcore hooks` and then runs, in order, the empty-state nudge, the Copilot wiring advisory, `bin/check-staleness`, the rate-limited update advisory, and the Copilot flush.
3. `bin/check-archcore-write` blocks `.archcore/**/*.md` writes through each host's honored deny mechanism and allows everything else.
4. `bin/check-code-alignment` injects top-ranked `.archcore/` context for source-file edits inside configured source roots, exits 0 on every code path, honors the `ARCHCORE_DISABLE_INJECTION=1` escape hatch, and keeps its cost independent of the match count.
5. `bin/validate-archcore` runs `archcore doctor` directly (no launcher wrapper) for archcore MCP tool calls and is covered by the Test Contract above.
6. `bin/check-staleness` detects code-doc drift via git, emits only when matching documents are found, and is rate-limited to once per 24h.
7. `bin/check-cascade` detects relation cascade after `update_document` and outputs warnings.
8. `bin/check-precision` runs the precision checks after `create_document` and `update_document`.
9. Both PreToolUse hooks complete within 1 second, pinned by `test/unit/hook-latency.bats`.
10. PostToolUse hooks complete within 3 seconds.
11. SessionStart never initiates a binary download — the plugin contains no fetcher.
12. Output formats follow each host's documented shape — `hookSpecificOutput` for Claude Code and Codex, flat `additional_context` for Cursor, top-level `additionalContext` and `permissionDecision` deny JSON for Copilot, plain text for OpenCode — all produced by the shared output helpers rather than by per-script string building.
13. On Copilot, SessionStart stdout parses as one JSON document after progress-line removal in every advisory combination, pinned by `test/unit/session-start-emit-matrix.bats`; non-copilot output is unchanged byte for byte, pinned by `test/unit/session-start-goldens.bats`.
14. Delegated tool invocations are covered by Hooks 2 and 3 identically to main-session calls; no committed code contains a probe line (the harness in `test/probe/` wraps a copy — see `host-probe-protocol.spec.md`).
15. Every script that invokes `archcore` passes only allowlisted subcommands; the contract is enforced by `test/structure/readme-cli-references.bats` and per-script invocation-log assertions.
16. Each host's hook commands resolve their scripts under that host's documented load paths, verified by executing the command rather than by inspecting it — `test/structure/copilot-plugin.bats` runs Copilot's under `env -u` for the unresolved, single-candidate, and dead-candidate cases.
