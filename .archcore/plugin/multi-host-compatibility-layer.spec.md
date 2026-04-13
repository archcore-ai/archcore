---
title: "Multi-Host Compatibility Layer Specification"
status: draft
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Purpose

Define the contract for the multi-host compatibility layer that enables the Archcore plugin to run in Claude Code, Cursor, GitHub Copilot, and other AI coding tools from a single repository. This specification covers host detection, stdin normalization for hook scripts, per-host hook event mapping, manifest structure, and MCP configuration.

## Scope

The compatibility layer — specifically: `bin/lib/normalize-stdin.sh`, per-host `hooks/*.hooks.json` files, per-host plugin manifests, and MCP config placement. Does NOT cover the shared components (skills, agents, core bin scripts) which are already host-agnostic by design.

## Authority

This specification is authoritative for cross-host behavior. The Multi-Host Plugin Architecture ADR provides the architectural rationale. The Hooks and Validation System Specification remains authoritative for hook semantics (what each hook does); this spec defines how hooks adapt to different host runtimes.

## Subject

### System Overview

The plugin splits into a **shared core** (skills, agents, MCP server, bin script logic) and a **host adapter layer** (manifests, hooks configs, stdin normalization). The adapter layer is pure configuration with one small shell library for stdin format detection.

```
┌─────────────────────────────────────────────────────────┐
│                    Shared Core                           │
│                                                         │
│  skills/ (33)  agents/ (2)  bin/ (5 scripts)            │
│                                                         │
│  100% host-agnostic — uses only MCP tools + Read/Grep   │
├─────────────────────────────────────────────────────────┤
│              Host Adapter Layer                          │
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ Claude Code   │  │ Cursor       │  │ Copilot      │  │
│  │              │  │              │  │   (future)   │  │
│  │ .claude-     │  │ .cursor-     │  │ .github/     │  │
│  │  plugin/     │  │  plugin/     │  │              │  │
│  │ hooks.json   │  │ cursor.hooks │  │              │  │
│  │ .mcp.json    │  │ mcp.json     │  │              │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│                                                         │
│  bin/lib/normalize-stdin.sh — detects host, normalizes  │
└─────────────────────────────────────────────────────────┘
```

### Supported Hosts

| Host | Priority | Plugin Manifest | Hooks Config | MCP Config | Status |
|------|----------|----------------|--------------|------------|--------|
| Claude Code | P0 | `.claude-plugin/plugin.json` | `hooks/hooks.json` | `.mcp.json` | Production |
| Cursor | P1 | `.cursor-plugin/plugin.json` | `hooks/cursor.hooks.json` | `mcp.json` | Implemented |
| GitHub Copilot | P2 | TBD | TBD | `.vscode/mcp.json` | Future |
| Codex CLI | P2 | TBD | TBD | `config.toml` | Future |

## Contract Surface

### 1. Stdin Normalization (`bin/lib/normalize-stdin.sh`)

A POSIX shell library sourced by all bin scripts. Reads raw stdin JSON, detects the host, and exports normalized variables.

#### Canonical normalized variables

| Variable | Description | Source: Claude Code | Source: Cursor |
|----------|-------------|--------------------|-----------------------|
| `ARCHCORE_HOST` | Host identifier | `"claude-code"` (default) | `"cursor"` (from `conversation_id`) |
| `ARCHCORE_RAW_STDIN` | Unmodified stdin | Full stdin | Full stdin |
| `ARCHCORE_TOOL_NAME` | Normalized tool name | `tool_name` as-is | Prefixed `mcp__archcore__` for MCP events |
| `ARCHCORE_FILE_PATH` | Target file path | `tool_input.file_path` | `file_path` |
| `ARCHCORE_DOC_PATH` | Document path (MCP) | `tool_input.path` | Extracted from escaped `tool_input` string |

#### Host detection heuristic

Priority: `$ARCHCORE_HOST` env var (if set) > stdin detection > default.

```
if stdin contains "conversation_id"  → Cursor
if stdin contains "hookEventName"    → GitHub Copilot
else                                 → Claude Code (default/fallback)
```

Cursor includes `conversation_id` in all hook events. Claude Code does not send this field.

#### Tool name normalization

Cursor's `afterMCPExecution` event sends bare MCP tool names (`create_document`, `update_document`). The normalizer prefixes them with `mcp__archcore__` so bin scripts work unchanged:

```
Cursor afterMCPExecution: tool_name="create_document"
  → ARCHCORE_TOOL_NAME="mcp__archcore__create_document"

Cursor preToolUse: tool_name="Write"
  → ARCHCORE_TOOL_NAME="Write" (no change)

Claude Code: tool_name="mcp__archcore__create_document"
  → ARCHCORE_TOOL_NAME="mcp__archcore__create_document" (no change)
```

#### Escaped JSON extraction

Cursor's `afterMCPExecution` sends `tool_input` as a JSON string (double-escaped). The normalizer provides `_archcore_json_val_unescaped()` to extract fields from escaped strings. Used as fallback when direct extraction fails for `ARCHCORE_DOC_PATH`.

#### Usage in bin scripts

```sh
#!/bin/sh
SCRIPT_DIR=$(dirname "$0")
. "$SCRIPT_DIR/lib/normalize-stdin.sh"

# Now use normalized variables:
# $ARCHCORE_HOST, $ARCHCORE_FILE_PATH, $ARCHCORE_TOOL_NAME, etc.
```

#### Output helpers

**`archcore_hook_block "reason"`** — Block the operation and exit. Uses `exit 2` with stderr message for all hosts. Exit code 2 is the universal blocking signal recognized by both Claude Code and Cursor.

**`archcore_hook_info "message"`** — Emit informational message to the agent. Format varies by host:

| Host | Output format |
|------|--------------|
| Claude Code | `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"..."}}` |
| Cursor | `{"additional_context":"..."}` |

**`archcore_hook_allow`** — Allow the operation silently. `exit 0` for all hosts.

### 2. Hook Event Mapping

| Plugin Hook | Claude Code Event | Cursor Event | Notes |
|-------------|------------------|--------------|-------|
| Session context load | `SessionStart` | `sessionStart` | Both hosts support this event |
| Block .archcore/ writes | `PreToolUse` (Write\|Edit) | `preToolUse` (Write) | Cursor has no Edit tool |
| Validate after file write | `PostToolUse` (Write\|Edit) | `postToolUse` (Write) | Defense-in-depth |
| Validate after MCP ops | `PostToolUse` (mcp__archcore__*) | `afterMCPExecution` | Cursor has dedicated MCP event |
| Cascade detection | `PostToolUse` (update_document) | `afterMCPExecution` | Script filters for update internally |

Key differences:
- **Event naming**: Claude Code uses PascalCase (`PreToolUse`), Cursor uses camelCase (`preToolUse`)
- **MCP hooks**: Claude Code uses `PostToolUse` with MCP tool matcher; Cursor has `afterMCPExecution` — a dedicated event for all MCP operations
- **Cascade filtering**: Claude Code matcher filters for `update_document` only; Cursor's `afterMCPExecution` fires for all MCP tools — `check-cascade` script exits early when `ARCHCORE_DOC_PATH` is empty

### 3. Per-Host Hooks Configuration

#### Claude Code (`hooks/hooks.json`)

```json
{
  "hooks": {
    "SessionStart": [{ "matcher": "", "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/session-start" }] }],
    "PreToolUse": [{ "matcher": "Write|Edit", "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-archcore-write", "timeout": 1 }] }],
    "PostToolUse": [
      { "matcher": "Write|Edit", "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore", "timeout": 3 }] },
      { "matcher": "mcp__archcore__create_document|mcp__archcore__update_document|mcp__archcore__remove_document|mcp__archcore__add_relation|mcp__archcore__remove_relation", "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore", "timeout": 3 }] },
      { "matcher": "mcp__archcore__update_document", "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-cascade", "timeout": 3 }] }
    ]
  }
}
```

#### Cursor (`hooks/cursor.hooks.json`)

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [{ "matcher": "", "hooks": [{ "type": "command", "command": "${CURSOR_PLUGIN_ROOT}/bin/session-start" }] }],
    "preToolUse": [{ "matcher": "Write", "hooks": [{ "type": "command", "command": "${CURSOR_PLUGIN_ROOT}/bin/check-archcore-write", "timeout": 1 }] }],
    "postToolUse": [{ "matcher": "Write", "hooks": [{ "type": "command", "command": "${CURSOR_PLUGIN_ROOT}/bin/validate-archcore", "timeout": 3 }] }],
    "afterMCPExecution": [{ "matcher": "", "hooks": [
      { "type": "command", "command": "${CURSOR_PLUGIN_ROOT}/bin/validate-archcore", "timeout": 3 },
      { "type": "command", "command": "${CURSOR_PLUGIN_ROOT}/bin/check-cascade", "timeout": 3 }
    ] }]
  }
}
```

### 4. Plugin Manifests

#### Claude Code (`.claude-plugin/plugin.json`)

```json
{
  "name": "archcore",
  "description": "Git-native context for AI coding agents",
  "version": "0.0.1",
  "author": { "name": "Archcore" },
  "license": "Apache-2.0"
}
```

Claude Code discovers skills, agents, hooks, and MCP by convention (fixed directory names). Manifest contains only metadata.

#### Cursor (`.cursor-plugin/plugin.json`)

```json
{
  "name": "archcore",
  "description": "Git-native context for AI coding agents",
  "version": "0.0.1",
  "author": { "name": "Archcore" },
  "license": "Apache-2.0",
  "repository": "https://github.com/archcore-ai/archcore-plugin",
  "keywords": ["documentation", "architecture", "knowledge-base", "mcp"],
  "skills": "skills/",
  "agents": "agents/",
  "hooks": "hooks/cursor.hooks.json",
  "mcpServers": "mcp.json",
  "rules": "rules/"
}
```

Cursor requires explicit paths to components. Only `name` is required; all other fields are optional but recommended.

### 5. MCP Configuration

Identical content, different file locations per host convention:

| Host | File | Notes |
|------|------|-------|
| Claude Code | `.mcp.json` | Dot-prefixed, root level |
| Cursor | `mcp.json` | Referenced from plugin manifest `mcpServers` field |

Content:
```json
{
  "mcpServers": {
    "archcore": {
      "command": "archcore",
      "args": ["mcp"]
    }
  }
}
```

### 6. Cursor Rules

Rules in `rules/` provide context injection. Two files:

**`rules/archcore-context.mdc`** — `alwaysApply: true`. Injected into every session. Contains: document type reference, MCP tool names, MCP-only principle.

**`rules/archcore-files.mdc`** — `globs: ".archcore/**"`. Injected when `.archcore/` files are in context. Reminds about MCP-only operations.

## Normative Behavior

- All bin scripts MUST source `bin/lib/normalize-stdin.sh` before processing stdin.
- The normalizer MUST detect the host and export `ARCHCORE_HOST` correctly.
- The normalizer MUST normalize MCP tool names to `mcp__archcore__` prefix for Cursor's `afterMCPExecution` events.
- The normalizer MUST handle escaped JSON strings in Cursor's `tool_input` field.
- `archcore_hook_block` MUST use exit code 2 for all hosts (universally recognized).
- `archcore_hook_info` MUST emit the correct JSON format per host.
- Per-host hooks config files MUST map all five hook functions.
- Plugin manifests MUST use identical `name`, `description`, and `version` across all hosts.
- MCP server config content MUST be identical across hosts — only the file location varies.
- Adding a new host MUST NOT require changes to skills, agents, or core bin script logic.

## Constraints

- `bin/lib/normalize-stdin.sh` MUST be POSIX shell compatible (no bashisms).
- Host detection MUST work without external dependencies (no `jq`, only `grep`/`sed`).
- Stdin normalization MUST complete within 100ms (included in hook timeout budget).
- Plugin root variable name varies by host (`${CLAUDE_PLUGIN_ROOT}`, `${CURSOR_PLUGIN_ROOT}`) — hooks configs MUST use the host-specific variable name. Note: Cursor also recognizes `${CLAUDE_PLUGIN_ROOT}` as an alias.

## Invariants

- Shared core components (skills, agents, MCP config, core bin logic) are identical across all hosts.
- A change to a skill or agent file benefits all hosts simultaneously.
- Per-host adapter files contain no business logic — only configuration and format mapping.
- The normalizer always falls back to Claude Code format if host detection fails (backward compatible).
- Hook semantics (what gets blocked, what gets validated) are identical across hosts — only the wire format differs.
- Exit code 2 blocks operations universally across all supported hosts.

## Error Handling

- **Unknown host detected**: Fall back to Claude Code format. Log warning to stderr.
- **Stdin JSON missing expected fields**: Export empty variables. Bin script logic handles missing fields gracefully.
- **Escaped JSON extraction fails**: `ARCHCORE_DOC_PATH` remains empty. `check-cascade` exits early (no cascade possible).
- **Plugin root variable not set**: Bin scripts use `$(dirname "$0")` for relative paths.

## Conformance

The multi-host compatibility layer conforms to this specification if:

1. All bin scripts source `bin/lib/normalize-stdin.sh` and use normalized variables
2. Host detection correctly identifies Claude Code and Cursor (and any additional hosts)
3. MCP tool names are normalized to `mcp__archcore__` prefix regardless of host
4. Output helpers emit correct format per detected host
5. Each supported host has a complete hooks config mapping all five hook functions
6. Each supported host has a valid plugin manifest
7. Shared components (skills, agents, MCP) contain zero host-specific references
8. Adding a new host requires only new config files, not changes to shared components