---
title: "Hooks and Validation System Specification"
status: draft
tags:
  - "hooks"
  - "plugin"
  - "validation"
---

## Purpose

Define the contract for the hook-based validation layer that enforces the MCP-only principle and ensures `.archcore/` file integrity within the Archcore Claude Plugin.

## Scope

This specification covers all hook entries in `hooks/hooks.json`: the SessionStart hook (via `bin/session-start` wrapper), the PreToolUse hook for blocking direct writes, and the PostToolUse hooks for validation after both file writes and MCP document operations. It does not cover the MCP server itself or the agent's tool restrictions.

## Authority

This specification is the authoritative reference for the plugin's hook configuration. The Always Use MCP Tools ADR provides the architectural rationale for the blocking behavior.

## Subject

The hooks system consists of event handlers registered in `hooks/hooks.json` that respond to Claude Code lifecycle events. Three event types with four hook entries enforce quality and the MCP-only principle.

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
      }
    ]
  }
}
```

### Hook 1: SessionStart

**Event**: SessionStart (fires when a session begins or resumes)
**Matcher**: empty (matches all session sources: startup, resume, clear, compact)
**Handler**: `${CLAUDE_PLUGIN_ROOT}/bin/session-start`
**Behavior**: Checks if archcore CLI is installed and project is initialized. If not, outputs a warning with install/init instructions and exits. If available, delegates to `archcore hooks claude-code session-start` which reads project context from `.archcore/` and injects it into the session as additional context.
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

### PostToolUse Output Format

When validation issues are found, both PostToolUse hooks output:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "Archcore validation found issues: <issues>. Run archcore validate --fix to auto-fix orphaned relations."
  }
}
```

### Exit Code Semantics

| Hook | Exit 0 | Exit 2 | Other |
|------|--------|--------|-------|
| SessionStart | Always (output = context) | N/A | N/A |
| PreToolUse (allow) | Empty output, operation proceeds | N/A | N/A |
| PreToolUse (block) | N/A | stderr → model feedback, operation blocked | N/A |
| PostToolUse (clean) | Empty output | N/A | N/A |
| PostToolUse (issues) | JSON with additionalContext | N/A | N/A |

### bin/ Scripts

Three executable scripts in the `bin/` directory:

#### `bin/session-start`

Shell script that checks for archcore CLI and project initialization before delegating to `archcore hooks claude-code session-start`.

Requirements:

- Must be executable (`chmod +x`)
- Must exit 0 in all cases
- Must output human-readable warnings if CLI or init is missing
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

## Normative Behavior

- The PreToolUse hook MUST block all Write/Edit calls targeting `.archcore/**/*.md` files via exit code 2 with stderr message.
- The PreToolUse hook MUST NOT block writes to `.archcore/settings.json` or `.archcore/.sync-state.json`.
- The PreToolUse hook MUST NOT block writes to files outside `.archcore/`.
- The PostToolUse hooks report validation issues via `hookSpecificOutput.additionalContext` but do not block or revert operations.
- The PostToolUse MCP matcher MUST fire after all document mutation MCP tools.
- All hooks MUST be idempotent — running them multiple times produces the same result.

## Constraints

- PreToolUse hook must complete within 1 second (enforced by `timeout: 1`).
- PostToolUse hooks must complete within 3 seconds (enforced by `timeout: 3`).
- Hooks must work without network access (no remote validation).
- Hooks must degrade gracefully if `archcore` CLI is not installed (skip validation, don't error).
- Bin scripts must be POSIX-compatible shell (no bash-specific features).

## Invariants

- The PreToolUse hook blocks 100% of direct Write/Edit to `.archcore/**/*.md` files.
- The PreToolUse hook never blocks writes outside `.archcore/`.
- The PostToolUse hooks never modify files — they only report.
- SessionStart and PostToolUse hooks exit 0 regardless of outcome.
- PreToolUse exits 0 (allow) or 2 (block) — never other codes.

## Error Handling

- If `archcore` CLI is not found: PostToolUse hooks skip validation silently. PreToolUse doesn't need the CLI — it only checks file paths. SessionStart outputs install instructions.
- If stdin JSON is malformed: exit 0 with empty output (fail open, don't break the session).
- If `archcore validate` hangs: enforced by `timeout` field in hooks.json (3 seconds).

## Conformance

The hooks system conforms to this specification if:

1. `hooks/hooks.json` contains all four hook entries (SessionStart, PreToolUse, two PostToolUse)
2. `bin/session-start` checks CLI availability and delegates or warns
3. `bin/check-archcore-write` blocks `.archcore/**/*.md` writes via exit 2 + stderr and allows everything else
4. `bin/validate-archcore` runs validation after `.archcore/` Write/Edit and after MCP document operations
5. PreToolUse completes within 1 second
6. PostToolUse completes within 3 seconds
7. Output formats follow Claude Code hooks documentation (exit codes, hookSpecificOutput object)