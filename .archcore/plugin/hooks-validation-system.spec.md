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

This specification covers all hook entries in `hooks/hooks.json`: the SessionStart hook (via `bin/session-start` wrapper with staleness check), two PreToolUse hooks on `Write|Edit` (blocking direct writes to `.archcore/*.md` and injecting context for source edits), the PostToolUse hook for validation after MCP document operations, and the PostToolUse hook for cascade detection after document updates. It does not cover the MCP server itself, the CLI launcher (see Bundled CLI Launcher ADR and Multi-Host Compatibility Layer Specification), or the agent's tool restrictions.

## Authority

This specification is the authoritative reference for the plugin's hook configuration. The Always Use MCP Tools ADR provides the architectural rationale for the blocking behavior. The Actualize System ADR and Specification provide the rationale and contract for staleness detection (Layers 1 and 2). The Bundled CLI Launcher ADR and Multi-Host Compatibility Layer Specification are authoritative for how hook scripts invoke the CLI binary (through `bin/archcore`). The Pre-Code Context Injection idea and its implementation plan provide the rationale for the source-edit context-injection hook.

## Subject

The hooks system consists of event handlers registered in `hooks/hooks.json` that respond to Claude Code lifecycle events. Three event types with five hook entries enforce quality, the MCP-only principle, documentation freshness, and source-edit context alignment.

## Contract Surface

### hooks/hooks.json Structure

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/bin/session-start"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-archcore-write",
            "timeout": 1
          },
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-code-alignment",
            "timeout": 1
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "mcp__archcore__create_document|mcp__archcore__update_document|mcp__archcore__remove_document|mcp__archcore__add_relation|mcp__archcore__remove_relation",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore",
            "timeout": 3
          }
        ]
      },
      {
        "matcher": "mcp__archcore__update_document",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-cascade",
            "timeout": 3
          }
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
**Behavior**: Three-phase pipeline:

1. **Project check**: if `.archcore/` does not exist, emit `additionalContext` instructing the agent to call `mcp__archcore__init_project` on first Archcore operation, then exit 0. No manual install/init required.
2. **Context loading**: if `.archcore/` exists, pipe stdin into `${SCRIPT_DIR}/archcore hooks <host> session-start` with `ARCHCORE_SKIP_DOWNLOAD=1` set. The local launcher resolves the CLI (from `$ARCHCORE_BIN`, `PATH`, or the plugin-managed cache); any failure (CLI missing, cache miss, launcher exit non-zero) is swallowed so session start remains non-blocking.
3. **Staleness check**: call `bin/check-staleness` to detect code-doc drift via git. Append findings to session context via the emit-info helper. Rate-limited to once per 24h (see `bin/check-staleness` requirements).

`ARCHCORE_SKIP_DOWNLOAD=1` prevents the launcher from downloading the CLI during SessionStart — the first MCP tool call triggers the download instead, keeping SessionStart fast and quiet on first use.

Phase 3 is additive — if it fails or produces no output, phases 1-2 are unaffected.

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

### Hook 4: PostToolUse — Validate After MCP Document Operations

**Event**: PostToolUse (fires after a tool call succeeds)
**Matcher**: `mcp__archcore__create_document|mcp__archcore__update_document|mcp__archcore__remove_document|mcp__archcore__add_relation|mcp__archcore__remove_relation`
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore`
**Timeout**: 3 seconds
**Input**: JSON on stdin containing the completed MCP tool call details

**Behavior**:

1. Extract `tool_name` from stdin JSON
2. Detect `mcp__archcore__*` prefix — run `archcore validate` via the launcher unconditionally
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

### PostToolUse Output Formats

**Validation (Hook 4)** — when issues found:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "Archcore validation found issues: <issues>. Run archcore validate --fix to auto-fix orphaned relations."
  }
}
```

**Cascade Detection (Hook 5)** — when cascade detected:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "[Archcore Cascade] Updated \"<document-title>\".\nDocuments that may need review:\n  → <path> (<relation-type> this document)\n  → <path> (<relation-type> this document)\nRun /archcore:actualize for detailed analysis."
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

| Hook | Exit 0 | Exit 2 | Other |
|------|--------|--------|-------|
| SessionStart | Always (output = context + staleness) | N/A | N/A |
| PreToolUse block (Hook 2, allow) | Empty output, operation proceeds | N/A | N/A |
| PreToolUse block (Hook 2, block) | N/A | stderr → model feedback, operation blocked | N/A |
| PreToolUse inject (Hook 3, no match) | Empty output, operation proceeds | N/A | N/A |
| PreToolUse inject (Hook 3, match) | JSON `additionalContext`, operation proceeds | N/A | N/A |
| PostToolUse validation (clean) | Empty output | N/A | N/A |
| PostToolUse validation (issues) | JSON with additionalContext | N/A | N/A |
| PostToolUse cascade (none) | Empty output | N/A | N/A |
| PostToolUse cascade (found) | JSON with additionalContext | N/A | N/A |

### bin/ Scripts

Five executable hook scripts in `bin/`, plus the normalizer library and the CLI launcher (launcher covered in the Bundled CLI Launcher ADR).

#### `bin/session-start`

Shell script that routes through the CLI launcher for context loading and runs the staleness check.

Requirements:

- Must be executable (`chmod +x`)
- Must source `bin/lib/normalize-stdin.sh`
- Must exit 0 in all cases
- Must emit `additionalContext` pointing at `mcp__archcore__init_project` when `.archcore/` is absent
- Must invoke the CLI via the local launcher with `ARCHCORE_SKIP_DOWNLOAD=1`, discarding non-zero exits
- Must call `bin/check-staleness` after the launcher call
- Must degrade gracefully — never error, just warn

#### `bin/check-archcore-write`

Shell script that reads stdin JSON, extracts `tool_input.file_path`, and decides whether to block.

Requirements:

- Must be executable (`chmod +x`)
- Must read JSON from stdin
- Must exit 0 when allowing, exit 2 when blocking
- Must write blocking reason to stderr (not stdout) when blocking
- Must complete within 1 second

#### `bin/check-code-alignment`

PreToolUse handler that injects applicable `.archcore/` context for source-file edits.

Requirements:

- Must be executable (`chmod +x`)
- Must source `bin/lib/normalize-stdin.sh`
- Must read JSON from stdin
- Must exit 0 in all cases — MUST NEVER return non-zero (injection is additive)
- Must short-circuit silently on `.archcore/*` paths (Hook 2 handles those)
- Must short-circuit silently on non-source-root paths
- Must honor `.archcore/settings.json` → `codeAlignment.sourceRoots` when configured; otherwise use the default root set
- Must honor the `ARCHCORE_DISABLE_INJECTION=1` escape hatch
- Must rank by specificity (longest directory prefix match) first, type priority (`rule > cpat > adr > spec > guide`) second
- Must consider only `rule`, `cpat`, `adr`, `spec`, `guide` document types
- Must emit at most 3 matches, capped at 2 KB total output
- Must complete within 1 second on a corpus of ≤50 `.archcore/*.md` documents (Phase 1 baseline; Phase 2 adds a path index for larger corpora)
- Must output host-normalized JSON — `hookSpecificOutput` for Claude Code / Copilot, flat `additional_context` for Cursor
- POSIX shell compatible

#### `bin/validate-archcore`

Shell script that reads stdin JSON, determines if validation is needed (by tool_name prefix), and runs `archcore validate` through the launcher.

Requirements:

- Must be executable (`chmod +x`)
- Must read JSON from stdin
- Must fire unconditionally for `mcp__archcore__*` tools; the legacy Write/Edit branch in the script is retained as defensive code but is never reached from the current hooks config
- Must invoke the CLI via `"$SCRIPT_DIR/archcore"` (launcher) rather than a bare `archcore` on `PATH`
- Must exit 0 in all cases, even if the launcher returns non-zero (silent skip on missing CLI)
- Must output valid JSON with `hookSpecificOutput` object when reporting issues, empty output when clean
- Must complete within 3 seconds

#### `bin/check-staleness`

Shell script called from `bin/session-start` after context loading. Detects code-document drift via git history comparison.

Requirements:

- Must be executable (`chmod +x`)
- Must exit 0 in all cases
- Must output plain text warning when drift detected, empty output when clean
- Output must not exceed 2KB
- Must complete within 3 seconds
- Must skip gracefully if git is unavailable, `.archcore/` has no commits, or project is not a git repo
- Must be rate-limited to emit at most once per 24h per project via a timestamp file at `$CLAUDE_PLUGIN_DATA/archcore/last-staleness` (falling back to `$XDG_DATA_HOME/archcore-plugin/last-staleness`, then `$HOME/.local/share/archcore-plugin/last-staleness`). Stamp is written only when a warning is actually emitted; corrupt or missing stamps are treated as "never emitted"
- Must emit a warning ONLY when matching documents are found — no generic "N files changed" fallback
- POSIX shell compatible

#### `bin/check-cascade`

PostToolUse handler for cascade detection after `update_document`. Queries relation graph for directional dependencies on the updated document.

Requirements:

- Must be executable (`chmod +x`)
- Must read JSON from stdin
- Must exit 0 in all cases
- Must output JSON with `hookSpecificOutput` when cascade detected, empty output otherwise
- Must invoke the CLI via `"$SCRIPT_DIR/archcore"` (launcher)
- Must complete within 3 seconds
- Must skip gracefully if the launcher / CLI is unavailable
- POSIX shell compatible

## Normative Behavior

- The PreToolUse block hook (Hook 2) MUST block all Write/Edit calls targeting `.archcore/**/*.md` files via exit code 2 with stderr message.
- The PreToolUse block hook MUST NOT block writes to `.archcore/settings.json` or `.archcore/.sync-state.json`.
- The PreToolUse block hook MUST NOT block writes to files outside `.archcore/`.
- The PreToolUse injection hook (Hook 3) MUST exit 0 on every code path and MUST NEVER block or fail an edit.
- The PreToolUse injection hook MUST short-circuit silently for paths inside `.archcore/`, paths outside configured source roots, and paths that produce no matches.
- The PreToolUse injection hook MUST rank matches by specificity first (longest matching directory prefix wins), type priority second, and MUST restrict eligible types to `rule`, `cpat`, `adr`, `spec`, `guide`.
- The PreToolUse injection hook MUST cap output at 3 documents and 2 KB.
- The PreToolUse injection hook MUST honor the `ARCHCORE_DISABLE_INJECTION=1` environment variable as an unconditional off-switch.
- The PostToolUse validation hook reports validation issues via `hookSpecificOutput.additionalContext` but does not block or revert operations.
- The PostToolUse MCP validation matcher MUST fire after all document mutation MCP tools.
- The hooks config MUST NOT register a Write/Edit matcher on PostToolUse. PreToolUse guarantees no Write/Edit reaches `.archcore/*.md` content, and revalidating on every non-archcore Write/Edit in the repo is wasted shell forks.
- The PostToolUse cascade hook MUST fire only after `update_document`, not after `create_document` or `remove_document`.
- The PostToolUse cascade hook MUST only flag documents connected via `implements`, `depends_on`, or `extends` (not `related`).
- The SessionStart staleness check MUST run after context loading, not before.
- The SessionStart staleness check output MUST NOT exceed 2KB.
- The SessionStart staleness check MUST rate-limit itself to once per 24h via a persistent timestamp file.
- Hook scripts that invoke the CLI MUST do so via the local launcher (`"$SCRIPT_DIR/archcore"`) so resolution order (`ARCHCORE_BIN` → `PATH` → cache → download) applies uniformly.
- `bin/session-start` MUST set `ARCHCORE_SKIP_DOWNLOAD=1` when invoking the launcher.
- All hooks MUST be idempotent — running them multiple times produces the same result.

## Constraints

- PreToolUse hooks (Hook 2 and Hook 3) must each complete within 1 second (enforced by `timeout: 1`).
- PostToolUse hooks must complete within 3 seconds (enforced by `timeout: 3`).
- SessionStart staleness check must complete within 3 seconds.
- Hooks must work without network access in the steady state (downloads are gated to first-ever CLI resolution and never occur inside a hook timeout budget because SessionStart sets `ARCHCORE_SKIP_DOWNLOAD=1`).
- Hooks must degrade gracefully if the launcher cannot resolve a CLI (skip validation/cascade, don't error).
- The injection hook (Hook 3) MUST degrade gracefully for corpora larger than the Phase 1 baseline — either by completing in time at lower fidelity or by short-circuiting cleanly; it MUST NOT time out in a way that blocks Write/Edit.
- Bin scripts must be POSIX-compatible shell (no bash-specific features).

## Invariants

- The PreToolUse block hook blocks 100% of direct Write/Edit to `.archcore/**/*.md` files.
- The PreToolUse block hook never blocks writes outside `.archcore/`.
- The PreToolUse injection hook never blocks any edit, regardless of result or error mode.
- The PreToolUse injection hook and the PreToolUse block hook act on disjoint path sets — the injection hook is silent for every path the block hook acts on.
- The PostToolUse hooks never modify files — they only report.
- Hook 4 (validation) and Hook 5 (cascade) fire independently on `update_document` — neither depends on the other.
- SessionStart and PostToolUse hooks exit 0 regardless of outcome.
- The PreToolUse block hook exits 0 (allow) or 2 (block) — never other codes.
- The PreToolUse injection hook exits 0 — never other codes.
- SessionStart context loading never initiates a network download.
- SessionStart emits the staleness warning at most once per 24h per project.

## Error Handling

- If the launcher cannot resolve a CLI binary: PostToolUse hooks skip validation/cascade silently. PreToolUse hooks do not depend on the CLI — Hook 2 only inspects file paths; Hook 3 scans `.archcore/` via shell grep. SessionStart emits init guidance only when `.archcore/` is absent; it otherwise swallows launcher failures.
- If stdin JSON is malformed: exit 0 with empty output (fail open, don't break the session).
- If `archcore validate` hangs: enforced by `timeout` field in hooks.json (3 seconds).
- If git is unavailable for staleness check: skip silently, context loading continues.
- If relation graph is empty for cascade check: produce no output (no cascade possible).
- If the staleness timestamp file is missing, empty, or contains non-numeric data: treat as "never emitted" and run the check normally.
- If the injection hook encounters any error (grep failure, malformed frontmatter, I/O error): exit 0 with empty output.

## Conformance

The hooks system conforms to this specification if:

1. `hooks/hooks.json` contains all five hook entries (SessionStart, two PreToolUse on `Write|Edit`, two PostToolUse on MCP matchers).
2. `bin/session-start` emits init guidance when `.archcore/` is missing, otherwise delegates context loading through the launcher with `ARCHCORE_SKIP_DOWNLOAD=1` and then calls `bin/check-staleness`.
3. `bin/check-archcore-write` blocks `.archcore/**/*.md` writes via exit 2 + stderr and allows everything else.
4. `bin/check-code-alignment` injects top-ranked `.archcore/` context for source-file edits inside configured source roots, exits 0 on every code path, and honors the `ARCHCORE_DISABLE_INJECTION=1` escape hatch.
5. `bin/validate-archcore` runs validation via the launcher for `mcp__archcore__*` tool calls.
6. `bin/check-staleness` detects code-doc drift via git, emits only when matching documents are found, and is rate-limited to once per 24h via a persistent timestamp file.
7. `bin/check-cascade` detects relation cascade after `update_document` via the launcher and outputs warnings.
8. Both PreToolUse hooks complete within 1 second.
9. PostToolUse completes within 3 seconds.
10. SessionStart never initiates a network download (launcher called with `ARCHCORE_SKIP_DOWNLOAD=1`).
11. Output formats follow Claude Code hooks documentation (exit codes, hookSpecificOutput object) with host-normalized Cursor shape where applicable.
