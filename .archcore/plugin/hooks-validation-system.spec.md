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

This specification covers all hook entries in `hooks/hooks.json`: the SessionStart hook (via `bin/session-start` wrapper with staleness check), two PreToolUse hooks on `Write|Edit` (blocking direct writes to `.archcore/*.md` and injecting context for source edits), the PostToolUse hook for validation after MCP document operations, the PostToolUse hook for cascade detection after document updates, and the PostToolUse hook for precision checks. It does not cover the MCP server itself, the Archcore CLI lifecycle (the CLI is installed by the user per https://docs.archcore.ai/cli/install/ and resolved via PATH), or the agent's tool restrictions.

## Authority

This specification is the authoritative reference for the plugin's hook configuration. The Always Use MCP Tools ADR provides the architectural rationale for the blocking behavior. The Actualize System ADR and Specification provide the rationale and contract for staleness detection (Layers 1 and 2). The Pre-Code Context Injection idea and its implementation plan provide the rationale for the source-edit context-injection hook.

## Subject

The hooks system consists of event handlers registered in `hooks/hooks.json` that respond to Claude Code lifecycle events. Three event types with six hook entries enforce quality, the MCP-only principle, documentation freshness, source-edit context alignment, and precision after document mutations.

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
        "matcher": "mcp__archcore__create_document|mcp__archcore__update_document|mcp__archcore__remove_document|mcp__archcore__add_relation|mcp__archcore__remove_relation",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore", "timeout": 3 }
        ]
      },
      {
        "matcher": "mcp__archcore__update_document",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-cascade", "timeout": 3 }
        ]
      },
      {
        "matcher": "mcp__archcore__create_document|mcp__archcore__update_document",
        "hooks": [
          { "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-precision", "timeout": 3 }
        ]
      }
    ]
  }
}
```

Historical note: a prior revision included a PostToolUse `Write|Edit` matcher invoking `validate-archcore` as defense-in-depth. The hook was dead in practice — PreToolUse blocks all Write/Edit to `.archcore/*.md` before they reach PostToolUse (PostToolUse fires only on success per Claude Code hooks semantics), and `.archcore/settings.json` / `.archcore/.sync-state.json` are allowlisted, so `validate-archcore` never had an edge case to handle through that path. It was removed to eliminate a per-Write/Edit shell fork across the entire repository. The MCP matcher below remains the single validation entry point.

The two PreToolUse entries on `Write|Edit` are deliberately coupled: `check-archcore-write` short-circuits on `.archcore/*.md` with exit 2 (blocks the write); `check-code-alignment` short-circuits on everything INSIDE `.archcore/` with exit 0 (silent). On any source path only the alignment hook does real work. The order matters for fast exit on blocks but does not affect correctness — exit codes from different hooks are combined per Claude Code semantics (any exit 2 blocks).

### Hook 1: SessionStart (Context Loading + Staleness Check)

**Event**: SessionStart (fires when a session begins or resumes)
**Matcher**: empty (matches all session sources: startup, resume, clear, compact)
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/session-start`
**Behavior**: three-phase pipeline:

1. **CLI availability check.** If `archcore` is not on PATH, emit an `additionalContext` install message pointing at https://docs.archcore.ai/cli/install/ and exit 0. No further phases run. Installing the CLI mid-session does NOT reconnect a Claude Code MCP server that failed to register at session start — users must restart the host after a fresh install.
2. **Project check.** If `.archcore/` does not exist, emit `additionalContext` instructing the agent to call `mcp__archcore__init_project` on first Archcore operation, then exit 0.
3. **Context loading + staleness.** If `.archcore/` exists, pipe stdin into `archcore hooks <host> session-start`; swallow any non-zero exit so SessionStart remains non-blocking. Then call `bin/check-staleness` to detect code-doc drift via git, emit findings via the info helper, and exit 0.

Staleness is additive — if it fails or produces no output, the preceding phases are unaffected.

**Input**: JSON on stdin with `session_id`, `cwd`, `hook_event_name`
**Output**: Structured `hookSpecificOutput.additionalContext` (Claude Code / Copilot) or plain text (other hosts)

### Hook 2: PreToolUse — Block Direct Writes

**Event**: PreToolUse (fires before a tool call executes)
**Matcher**: `Write|Edit` (only intercepts Write and Edit tool calls)
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/check-archcore-write`
**Timeout**: 1 second
**Input**: JSON on stdin containing the tool call details including `tool_input.file_path`

**Behavior**:

1. Extract `file_path` from the tool input (stdin JSON)
2. Check if the path matches `.archcore/**/*.md` (document files)
3. If NO match: exit 0 with empty output (allow the operation)
4. If MATCH: write blocking reason to **stderr** and **exit 2**

Per Claude Code documentation, exit code 2 is a blocking error — stderr is sent directly to the model as feedback, and the tool call is blocked.

**Stderr message when blocking**:

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

**Event**: PreToolUse (fires before a tool call executes)
**Matcher**: `Write|Edit` (shared with Hook 2)
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/check-code-alignment`
**Timeout**: 1 second
**Input**: JSON on stdin containing the tool call details including `tool_input.file_path`

**Behavior**:

1. Extract `file_path` via the normalized stdin layer.
2. Short-circuit (exit 0, empty output) if any of: no `file_path`, no `.archcore/` directory, path is inside `.archcore/`, env `ARCHCORE_DISABLE_INJECTION=1`.
3. Normalize to cwd-relative; exit 0 if path is absolute outside `$CWD`.
4. Enforce source-root filter: path must start with a configured source root. Default set: `src lib app pkg cmd internal apps packages modules components`. Override via `.archcore/settings.json` → `codeAlignment.sourceRoots` (JSON array). Exit 0 if not matched.
5. Generate candidate tokens — directory prefixes of the file path, longest first (capped at 5 levels).
6. Scan `.archcore/**/*.md` with `grep -rlF <token>` per token in longest-first order. Score each matched document by specificity (length of the longest matching token) combined with type priority: `rule=5, cpat=4, adr=3, spec=2, guide=1`. Only these five types are eligible — other types (prd, idea, plan, rfc, doc, task-type, etc.) are ignored as not enforceable or too high-level for line-of-code context.
7. Rank desc, take top 3.
8. Render a compact block:
   ```
   [Archcore Context] Before editing <relative-path>:
   - <type>: <title> [<short-doc-path>]
   ...
   ```
   Output capped at 2 KB.
9. Emit as PreToolUse `additionalContext`:
   - Claude Code / Copilot: `{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"..."}}`
   - Cursor: `{"additional_context":"..."}` (may be ignored by current Cursor — graceful degradation, documented limitation)

**Non-blocking by design**: exit code is always 0. Any error in the pipeline (missing tools, malformed JSON, empty matches) results in a silent pass. Injection is strictly additive and must never prevent a write.

**Escape hatch**: set environment variable `ARCHCORE_DISABLE_INJECTION=1` to disable injection globally for a session.

**Relationship to Hook 2**: both hooks fire on the same matcher. Hook 2 handles `.archcore/*.md` paths (blocks). Hook 3 handles source paths (injects). Their active path sets are disjoint by construction.

#### Sub-agent tool invocations (Task-dispatched)

PreToolUse hooks in Claude Code fire at the tool-execution boundary, not at the session boundary. Any Write or Edit tool call matches the `Write|Edit` matcher regardless of whether the call originates from the main conversation or from a sub-agent dispatched via the Task tool. Hook 2 and Hook 3 therefore cover Task-dispatched Write/Edit identically to main-session Write/Edit. Input stdin carries `tool_name` and `tool_input.file_path` in the same shape; the host does not annotate sub-agent origin in a way that the hooks need to consume or branch on.

Scope clarifications:

- **Archcore's own sub-agents** (`archcore-assistant`, `archcore-auditor`) do NOT have `Write` or `Edit` in their tools allowlist (see `agent-system.spec.md` Tool Access Matrix). They cannot trigger Hooks 2 or 3 by definition. The sub-agent coverage discussion concerns general-purpose and third-party Task agents dispatched by the user for code work.
- **Claude Code**: hook coverage for Task-dispatched Write/Edit holds by the host's PreToolUse contract. An empirical probe is recommended on major host releases but not required for the specification to stand.
- **Cursor**: the PreToolUse matcher in `cursor.hooks.json` is `Write` only, not `Write|Edit` — a pre-existing multi-host asymmetry, independent of the sub-agent question. Sub-agent-originated Edit calls on Cursor go unhooked for the same reason main-session Edit calls do.

### Hook 4: PostToolUse — Validate After MCP Document Operations

**Event**: PostToolUse (fires after a tool call succeeds)
**Matcher**: `mcp__archcore__create_document|mcp__archcore__update_document|mcp__archcore__remove_document|mcp__archcore__add_relation|mcp__archcore__remove_relation`
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore`
**Timeout**: 3 seconds
**Input**: JSON on stdin containing the completed MCP tool call details

**Behavior**:

1. Extract `tool_name` from stdin JSON
2. Detect `mcp__archcore__*` prefix — run `archcore doctor` directly (resolved via PATH, wrapped in `timeout 2` and `|| true`)
3. If validation passes: exit 0 with empty output
4. If validation fails: exit 0 with JSON output containing validation context

This is the sole validation hook. Because PreToolUse blocks all direct Write/Edit to `.archcore/*.md` and MCP tools are the supported interface for document operations, this single matcher fires after every document mutation that can actually touch the knowledge base.

### Hook 5: PostToolUse — Cascade Detection After Document Updates

**Event**: PostToolUse (fires after a tool call succeeds)
**Matcher**: `mcp__archcore__update_document`
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/check-cascade`
**Timeout**: 3 seconds
**Input**: JSON on stdin containing the completed `update_document` tool call details

**Behavior**:

1. Extract updated document path from `tool_input.path` in stdin JSON
2. Query relation graph for documents where the updated document is the **target** of `implements`, `depends_on`, or `extends` relations
3. If no such relations found: exit 0 with empty output (no cascade)
4. If cascade found: exit 0 with JSON output containing affected document list

This hook fires **in addition to** Hook 4 (validation). Both hooks fire independently on `update_document` — Hook 4 validates structural integrity, Hook 5 detects cascade staleness. Neither depends on the other.

**Fires only on `update_document`**: New documents (`create_document`) cannot cause cascade because nothing depends on them yet. Removed documents (`remove_document`) are intentional deletions.

**Excludes `related` relations**: Only `implements`, `depends_on`, and `extends` indicate directional dependency where cascade staleness is meaningful.

### Hook 6: PostToolUse — Precision Check

**Event**: PostToolUse
**Matcher**: `mcp__archcore__create_document|mcp__archcore__update_document`
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/check-precision`
**Timeout**: 3 seconds

Phase 1 of the Precision Initiative (see `precision-over-coverage.adr`). Reads the resulting file from disk and runs four checks: forbidden vagueness lexicon, mandatory sections by type (adr/rule/spec/guide/rfc), frontmatter title+status presence, body length ≥ 200 chars. Emits soft warnings via `additionalContext`. Always exits 0; never blocks.

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

Cursor host uses the flat `{"additional_context": "..."}` shape.

### Exit Code Semantics

| Hook | Exit 0 | Exit 2 |
|------|--------|--------|
| SessionStart | Always (output = install msg / init msg / context + staleness) | N/A |
| PreToolUse block (Hook 2, allow) | Empty output, operation proceeds | N/A |
| PreToolUse block (Hook 2, block) | N/A | stderr → model feedback, operation blocked |
| PreToolUse inject (Hook 3) | Always (no match → empty; match → `additionalContext`) | N/A |
| PostToolUse validation (Hook 4) | Always (clean → empty; issues → `additionalContext`) | N/A |
| PostToolUse cascade (Hook 5) | Always (no cascade → empty; cascade → `additionalContext`) | N/A |
| PostToolUse precision (Hook 6) | Always (clean → empty; warnings → `additionalContext`) | N/A |

### bin/ Scripts

Six executable hook scripts in `bin/`, plus the stdin normalization library. The plugin does **not** bundle the Archcore CLI binary or any launcher wrapper — every script that invokes the CLI calls `archcore <subcmd>` directly, resolved via PATH. If `archcore` is missing, hooks degrade silently (timeout + `|| true`).

#### `bin/session-start`

Shell script that handles SessionStart pipeline (CLI check + project check + context loading + staleness).

Requirements:

- Executable (`chmod +x`); `#!/bin/sh`
- Sources `bin/lib/normalize-stdin.sh`
- Exits 0 in all cases
- When `archcore` is not on PATH: emits an install message pointing at https://docs.archcore.ai/cli/install/ and exits 0
- When `.archcore/` is absent: emits `additionalContext` pointing at `mcp__archcore__init_project`
- Otherwise: invokes `archcore hooks <host> session-start` and discards any non-zero exit, then calls `bin/check-staleness`
- Degrades gracefully — never errors, just warns

#### `bin/check-archcore-write`

Shell script that reads stdin JSON, extracts `tool_input.file_path`, and decides whether to block.

Requirements: executable; `#!/bin/sh`; reads JSON from stdin; exit 0 when allowing, exit 2 when blocking; writes blocking reason to stderr; completes within 1 second.

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
- Completes within 1 second on a corpus of ≤ 50 `.archcore/*.md` documents
- Outputs host-normalized JSON — `hookSpecificOutput` for Claude Code / Copilot, flat `additional_context` for Cursor

#### `bin/validate-archcore`

Shell script that reads stdin JSON, determines if validation is needed (by tool_name prefix), and runs `archcore doctor` directly via PATH.

Requirements:

- Executable; `#!/bin/sh`; reads JSON from stdin
- Fires unconditionally for `mcp__archcore__*` tools; the legacy Write/Edit branch is retained as defensive code but is never reached from the current hooks config
- Invokes `archcore doctor` directly (`timeout 2 archcore doctor 2>&1`), no wrapper script
- Exits 0 in all cases — silent skip when `archcore` is unavailable
- Outputs valid JSON with `hookSpecificOutput` when reporting issues, empty output when clean
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

Requirements: executable; `#!/bin/sh`; reads JSON from stdin; exit 0 in all cases; outputs JSON `hookSpecificOutput` when cascade detected, empty otherwise; invokes `archcore` directly via PATH; completes within 3 seconds; skips gracefully if `archcore` is unavailable.

#### `bin/check-precision`

PostToolUse handler running the precision lexicon, mandatory-sections, frontmatter, and length checks. Always exits 0; reads files from disk (no CLI shell-out); ≤ 3 seconds.

## Normative Behavior

- The PreToolUse block hook (Hook 2) MUST block all Write/Edit calls targeting `.archcore/**/*.md` files via exit code 2 with stderr message.
- The PreToolUse block hook MUST NOT block writes to `.archcore/settings.json` or `.archcore/.sync-state.json`.
- The PreToolUse block hook MUST NOT block writes to files outside `.archcore/`.
- The PreToolUse injection hook (Hook 3) MUST exit 0 on every code path and MUST NEVER block or fail an edit.
- The PreToolUse injection hook MUST short-circuit silently for paths inside `.archcore/`, paths outside configured source roots, and paths that produce no matches.
- The PreToolUse injection hook MUST rank matches by specificity first (longest matching directory prefix wins), type priority second, and MUST restrict eligible types to `rule`, `cpat`, `adr`, `spec`, `guide`.
- The PreToolUse injection hook MUST cap output at 3 documents and 2 KB.
- The PreToolUse injection hook MUST honor the `ARCHCORE_DISABLE_INJECTION=1` environment variable as an unconditional off-switch.
- The PreToolUse hooks MUST treat Task-dispatched Write/Edit tool calls identically to main-session calls — no special-casing, no skipping.
- The PostToolUse validation hook reports validation issues via `hookSpecificOutput.additionalContext` but does not block or revert operations.
- The PostToolUse MCP validation matcher MUST fire after all document mutation MCP tools.
- The hooks config MUST NOT register a Write/Edit matcher on PostToolUse.
- The PostToolUse cascade hook MUST fire only after `update_document`, not after `create_document` or `remove_document`.
- The PostToolUse cascade hook MUST only flag documents connected via `implements`, `depends_on`, or `extends` (not `related`).
- The SessionStart hook MUST emit the install message when `archcore` is not on PATH and MUST NOT block the session in that case.
- The SessionStart staleness check MUST run after context loading, not before.
- The SessionStart staleness check output MUST NOT exceed 2 KB.
- The SessionStart staleness check MUST rate-limit itself to once per 24h via a persistent timestamp file.
- Hook scripts that invoke the CLI MUST call `archcore <subcmd>` directly (resolved via PATH); the plugin does NOT ship any launcher wrapper, version pin, or cache directory. Reintroducing a `bin/archcore*` launcher or `bin/CLI_VERSION` requires a fresh ADR per `stack-and-tooling.rule`.
- Hook scripts that invoke the CLI MUST only pass subcommands in the canonical surface (`config|doctor|help|hooks|init|mcp|status|update`); the contract is enforced by `test/structure/readme-cli-references.bats` and per-script invocation-log assertions.
- All hooks MUST be idempotent.

## Constraints

- PreToolUse hooks (Hook 2 and Hook 3) must each complete within 1 second.
- PostToolUse hooks must complete within 3 seconds.
- SessionStart staleness check must complete within 3 seconds.
- Hooks must work without network access in steady state. The plugin never downloads anything — CLI lifecycle is the user's responsibility via the official installer.
- Hooks must degrade gracefully if the Archcore CLI is missing (skip validation/cascade silently; SessionStart prints install guidance and exits 0).
- The injection hook MUST degrade gracefully for corpora larger than the Phase 1 baseline — either by completing in time at lower fidelity or by short-circuiting cleanly; it MUST NOT time out in a way that blocks Write/Edit.
- Bin scripts must be POSIX-compatible shell (no bash-specific features).

## Invariants

- The PreToolUse block hook blocks 100% of direct Write/Edit to `.archcore/**/*.md` files.
- The PreToolUse block hook never blocks writes outside `.archcore/`.
- The PreToolUse injection hook never blocks any edit, regardless of result or error mode.
- The PreToolUse injection hook and the PreToolUse block hook act on disjoint path sets — the injection hook is silent for every path the block hook acts on.
- Task-dispatched Write/Edit tool calls are subject to the same PreToolUse behavior as main-session calls; there is no dispatcher-based bypass.
- The PostToolUse hooks never modify files — they only report.
- Hook 4 (validation) and Hook 5 (cascade) fire independently on `update_document` — neither depends on the other.
- SessionStart and PostToolUse hooks exit 0 regardless of outcome.
- The PreToolUse block hook exits 0 (allow) or 2 (block) — never other codes.
- The PreToolUse injection hook exits 0 — never other codes.
- SessionStart never initiates a network download (the plugin no longer has download logic; CLI is the user's responsibility).
- SessionStart emits the staleness warning at most once per 24h per project.

## Error Handling

- If `archcore` is not on PATH: SessionStart emits the install message and exits 0; PostToolUse hooks skip validation/cascade silently. PreToolUse hooks (Hooks 2 and 3) do not depend on the CLI — Hook 2 only inspects file paths; Hook 3 scans `.archcore/` via shell grep.
- If stdin JSON is malformed: exit 0 with empty output (fail open, don't break the session).
- If `archcore doctor` hangs: enforced by `timeout 2` inside the script plus the hook's `timeout: 3` envelope.
- If git is unavailable for staleness check: skip silently, context loading continues.
- If relation graph is empty for cascade check: produce no output (no cascade possible).
- If the staleness timestamp file is missing, empty, or contains non-numeric data: treat as "never emitted" and run the check normally.
- If the injection hook encounters any error (grep failure, malformed frontmatter, I/O error): exit 0 with empty output.

## Conformance

The hooks system conforms to this specification if:

1. `hooks/hooks.json` contains all six hook entries (SessionStart, two PreToolUse on `Write|Edit`, three PostToolUse on MCP matchers).
2. `bin/session-start` emits an install message when `archcore` is missing, emits init guidance when `.archcore/` is missing, otherwise delegates to `archcore hooks` and then calls `bin/check-staleness`.
3. `bin/check-archcore-write` blocks `.archcore/**/*.md` writes via exit 2 + stderr and allows everything else.
4. `bin/check-code-alignment` injects top-ranked `.archcore/` context for source-file edits inside configured source roots, exits 0 on every code path, and honors the `ARCHCORE_DISABLE_INJECTION=1` escape hatch.
5. `bin/validate-archcore` runs `archcore doctor` directly (no launcher wrapper) for `mcp__archcore__*` tool calls and is covered by the Test Contract above.
6. `bin/check-staleness` detects code-doc drift via git, emits only when matching documents are found, and is rate-limited to once per 24h.
7. `bin/check-cascade` detects relation cascade after `update_document` and outputs warnings.
8. `bin/check-precision` runs the precision checks after `create_document` and `update_document`.
9. Both PreToolUse hooks complete within 1 second.
10. PostToolUse hooks complete within 3 seconds.
11. SessionStart never initiates a network download — the plugin contains no fetcher.
12. Output formats follow Claude Code hooks documentation (exit codes, hookSpecificOutput object) with host-normalized Cursor shape where applicable.
13. Sub-agent tool invocations (Task-dispatched Write/Edit) are covered by Hooks 2 and 3 identically to main-session calls; no committed code contains a probe line.
14. Every script that invokes `archcore` passes only allowlisted subcommands; the contract is enforced by `test/structure/readme-cli-references.bats` and per-script invocation-log assertions.
15. No `bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION`, or any download/cache logic exists in the repo.
