---
title: "Hooks and Validation System Specification"
status: draft
tags:
  - "hooks"
  - "plugin"
  - "validation"
---

## Purpose

Define the contract for the hook-based validation and freshness detection layer that enforces the MCP-only principle, ensures `.archcore/` file integrity, and detects documentation staleness within the Archcore Claude Plugin.

## Scope

This specification covers all hook entries in `hooks/hooks.json`: the SessionStart hook (via `bin/session-start` wrapper with staleness check), the PreToolUse hook for blocking direct writes, the PostToolUse hooks for validation after both file writes and MCP document operations, and the PostToolUse hook for cascade detection after document updates. It does not cover the MCP server itself or the agent's tool restrictions.

## Authority

This specification is the authoritative reference for the plugin's hook configuration. The Always Use MCP Tools ADR provides the architectural rationale for the blocking behavior. The Actualize System ADR and Specification provide the rationale and contract for staleness detection (Layers 1 and 2).

## Subject

The hooks system consists of event handlers registered in `hooks/hooks.json` that respond to Claude Code lifecycle events. Three event types with five hook entries enforce quality, the MCP-only principle, and documentation freshness.

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
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore",
            "timeout": 3
          }
        ]
      },
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

### Hook 1: SessionStart (Context Loading + Staleness Check)

**Event**: SessionStart (fires when a session begins or resumes)
**Matcher**: empty (matches all session sources: startup, resume, clear, compact)
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/session-start`
**Behavior**: Three-phase pipeline:

1. **CLI check**: Verify `archcore` CLI is installed. If not, output install instructions and exit.
2. **Project check**: Verify `.archcore/` directory exists. If not, output init instructions and exit.
3. **Context loading**: Delegate to `archcore hooks claude-code session-start` to inject project context.
4. **Staleness check**: Call `bin/check-staleness` to detect code-doc drift via git. Append findings to session context.

Phase 4 is additive — if it fails or produces no output, phases 1-3 are unaffected.

**Input**: JSON on stdin with `session_id`, `cwd`, `hook_event_name`
**Output**: Plain text (injected as session additional context)

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

### Hook 3: PostToolUse — Validate After Write/Edit (defense-in-depth)

**Event**: PostToolUse (fires after a tool call succeeds)
**Matcher**: `Write|Edit`
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore`
**Timeout**: 3 seconds
**Input**: JSON on stdin containing the completed tool call details

**Behavior**:

1. Extract `tool_name` and `file_path` from stdin JSON
2. Check if the path is under `.archcore/`
3. If NO match: exit 0 with empty output (no validation needed)
4. If MATCH: run `archcore validate` and capture output
5. If validation passes: exit 0 with empty output
6. If validation fails: exit 0 with JSON output containing validation context

Note: In practice, the PreToolUse hook blocks most direct writes to `.archcore/*.md` before they execute. This PostToolUse entry serves as defense-in-depth for edge cases (e.g., writes to `.archcore/settings.json` or other non-.md files).

### Hook 4: PostToolUse — Validate After MCP Document Operations

**Event**: PostToolUse (fires after a tool call succeeds)
**Matcher**: `mcp__archcore__create_document|mcp__archcore__update_document|mcp__archcore__remove_document|mcp__archcore__add_relation|mcp__archcore__remove_relation`
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore`
**Timeout**: 3 seconds
**Input**: JSON on stdin containing the completed MCP tool call details

**Behavior**:

1. Extract `tool_name` from stdin JSON
2. Detect `mcp__archcore__*` prefix — run `archcore validate` unconditionally
3. If validation passes: exit 0 with empty output
4. If validation fails: exit 0 with JSON output containing validation context

This is the primary validation hook. MCP tools are the supported interface for document operations, so this hook fires after every document mutation to ensure consistency.

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

**Validation (Hooks 3 & 4)** — when issues found:

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

### Exit Code Semantics

| Hook | Exit 0 | Exit 2 | Other |
|------|--------|--------|-------|
| SessionStart | Always (output = context + staleness) | N/A | N/A |
| PreToolUse (allow) | Empty output, operation proceeds | N/A | N/A |
| PreToolUse (block) | N/A | stderr → model feedback, operation blocked | N/A |
| PostToolUse validation (clean) | Empty output | N/A | N/A |
| PostToolUse validation (issues) | JSON with additionalContext | N/A | N/A |
| PostToolUse cascade (none) | Empty output | N/A | N/A |
| PostToolUse cascade (found) | JSON with additionalContext | N/A | N/A |

### bin/ Scripts

Five executable scripts in the `bin/` directory:

#### `bin/session-start`

Shell script that checks for archcore CLI and project initialization, delegates context loading, and runs staleness check.

Requirements:

- Must be executable (`chmod +x`)
- Must exit 0 in all cases
- Must output human-readable warnings if CLI or init is missing
- Must call `bin/check-staleness` after successful context loading
- Must degrade gracefully — never error, just warn

#### `bin/check-archcore-write`

Shell script that reads stdin JSON, extracts `tool_input.file_path`, and decides whether to block.

Requirements:

- Must be executable (`chmod +x`)
- Must read JSON from stdin
- Must exit 0 when allowing, exit 2 when blocking
- Must write blocking reason to stderr (not stdout) when blocking
- Must complete within 1 second

#### `bin/validate-archcore`

Shell script that reads stdin JSON, determines if validation is needed (by tool_name or file_path), and runs `archcore validate`.

Requirements:

- Must be executable (`chmod +x`)
- Must read JSON from stdin
- Must handle both Write/Edit tools (check file_path) and MCP tools (validate unconditionally)
- Must exit 0 in all cases
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
- POSIX shell compatible

#### `bin/check-cascade`

PostToolUse handler for cascade detection after `update_document`. Queries relation graph for directional dependencies on the updated document.

Requirements:

- Must be executable (`chmod +x`)
- Must read JSON from stdin
- Must exit 0 in all cases
- Must output JSON with `hookSpecificOutput` when cascade detected, empty output otherwise
- Must complete within 3 seconds
- Must skip gracefully if `archcore` CLI is unavailable
- POSIX shell compatible

## Normative Behavior

- The PreToolUse hook MUST block all Write/Edit calls targeting `.archcore/**/*.md` files via exit code 2 with stderr message.
- The PreToolUse hook MUST NOT block writes to `.archcore/settings.json` or `.archcore/.sync-state.json`.
- The PreToolUse hook MUST NOT block writes to files outside `.archcore/`.
- The PostToolUse validation hooks report validation issues via `hookSpecificOutput.additionalContext` but do not block or revert operations.
- The PostToolUse MCP validation matcher MUST fire after all document mutation MCP tools.
- The PostToolUse cascade hook MUST fire only after `update_document`, not after `create_document` or `remove_document`.
- The PostToolUse cascade hook MUST only flag documents connected via `implements`, `depends_on`, or `extends` (not `related`).
- The SessionStart staleness check MUST run after context loading, not before.
- The SessionStart staleness check output MUST NOT exceed 2KB.
- All hooks MUST be idempotent — running them multiple times produces the same result.

## Constraints

- PreToolUse hook must complete within 1 second (enforced by `timeout: 1`).
- PostToolUse hooks must complete within 3 seconds (enforced by `timeout: 3`).
- SessionStart staleness check must complete within 3 seconds.
- Hooks must work without network access (no remote validation).
- Hooks must degrade gracefully if `archcore` CLI is not installed (skip validation/cascade, don't error).
- Bin scripts must be POSIX-compatible shell (no bash-specific features).

## Invariants

- The PreToolUse hook blocks 100% of direct Write/Edit to `.archcore/**/*.md` files.
- The PreToolUse hook never blocks writes outside `.archcore/`.
- The PostToolUse hooks never modify files — they only report.
- Hook 4 (validation) and Hook 5 (cascade) fire independently on `update_document` — neither depends on the other.
- SessionStart and PostToolUse hooks exit 0 regardless of outcome.
- PreToolUse exits 0 (allow) or 2 (block) — never other codes.
- SessionStart context loading succeeds even if staleness check fails.

## Error Handling

- If `archcore` CLI is not found: PostToolUse hooks skip validation/cascade silently. PreToolUse doesn't need the CLI — it only checks file paths. SessionStart outputs install instructions.
- If stdin JSON is malformed: exit 0 with empty output (fail open, don't break the session).
- If `archcore validate` hangs: enforced by `timeout` field in hooks.json (3 seconds).
- If git is unavailable for staleness check: skip silently, context loading continues.
- If relation graph is empty for cascade check: produce no output (no cascade possible).

## Conformance

The hooks system conforms to this specification if:

1. `hooks/hooks.json` contains all five hook entries (SessionStart, PreToolUse, three PostToolUse)
2. `bin/session-start` checks CLI availability, delegates context loading, and calls `bin/check-staleness`
3. `bin/check-archcore-write` blocks `.archcore/**/*.md` writes via exit 2 + stderr and allows everything else
4. `bin/validate-archcore` runs validation after `.archcore/` Write/Edit and after MCP document operations
5. `bin/check-staleness` detects code-doc drift via git and outputs warnings
6. `bin/check-cascade` detects relation cascade after `update_document` and outputs warnings
7. PreToolUse completes within 1 second
8. PostToolUse completes within 3 seconds
9. Output formats follow Claude Code hooks documentation (exit codes, hookSpecificOutput object)