---
title: "Multi-Host Compatibility Layer Specification"
status: accepted
tags:
  - "architecture"
  - "hooks"
  - "multi-host"
  - "plugin"
---

## Purpose

Define the contract for the multi-host compatibility layer that enables the Archcore plugin to run in Claude Code, Cursor, Codex CLI, GitHub Copilot, and other AI coding tools from a single repository. This specification covers host detection, stdin normalization for hook scripts, per-host hook event mapping, per-host manifest structure, and the cross-host CLI launcher that resolves the Archcore CLI binary on demand.

MCP server registration is **partially in scope**: Claude Code and Codex CLI receive a plugin-shipped `.mcp.json` wired to the bundled launcher (see the Bundled CLI Launcher ADR). Cursor and other hosts still rely on user-registered MCP — the launcher itself is host-agnostic, only the plugin-level MCP wiring is host-specific.

## Scope

The compatibility layer — specifically: `bin/lib/normalize-stdin.sh`, `bin/archcore` / `bin/archcore.cmd` / `bin/archcore.ps1` / `bin/CLI_VERSION` (the launcher and its version pin), per-host `hooks/*.hooks.json` files, per-host plugin manifests, plugin-shipped MCP configs (`.mcp.json` for Claude Code, `.codex.mcp.json` for Codex CLI), and the Codex-specific subagent TOML files (`agents/archcore-*.toml`). Does NOT cover the shared hook script logic (skills, agents, hook scripts themselves), which are host-agnostic by design, nor the CLI binary's own behavior.

## Authority

This specification is authoritative for cross-host behavior. The Multi-Host Plugin Architecture ADR provides the architectural rationale for the shared-core / per-host-adapter split. The Bundled CLI Launcher ADR is authoritative for launcher resolution order, auto-install policy, and plugin-owned MCP wiring. The Hooks and Validation System Specification remains authoritative for hook semantics (what each hook does); this spec defines how hooks adapt to different host runtimes.

## Subject

### System Overview

The plugin splits into a **shared core** (skills, agents, bin scripts, CLI launcher) and a **host adapter layer** (manifests, hooks configs, stdin normalization, plugin-shipped MCP wiring for Claude Code and Codex CLI, Codex-specific subagent TOML files). The adapter layer is pure configuration plus one small shell library for stdin format detection.

```
┌─────────────────────────────────────────────────────────┐
│                    Shared Core                           │
│                                                         │
│  skills/ (17)  agents/ (2 MD + 2 TOML)                  │
│  bin/ — 6 hook scripts + 3 launcher scripts + pin file  │
│                                                         │
│  100% host-agnostic — uses only MCP tools + Read/Grep   │
├─────────────────────────────────────────────────────────┤
│              Host Adapter Layer                          │
│                                                         │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌────┐ │
│  │ Claude Code │ │ Cursor      │ │ Codex CLI   │ │ Co │ │
│  │             │ │             │ │             │ │ pi │ │
│  │ .claude-    │ │ .cursor-    │ │ .codex-     │ │ lo │ │
│  │  plugin/    │ │  plugin/    │ │  plugin/    │ │ t  │ │
│  │ hooks.json  │ │ cursor.hk   │ │ codex.hk    │ │ TBD│ │
│  │ .mcp.json   │ │ (user MCP)  │ │ .codex.mcp  │ │    │ │
│  │ MD agents   │ │ MD agents   │ │ TOML agents │ │    │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └────┘ │
│                                                         │
│  bin/lib/normalize-stdin.sh — detects host, normalizes  │
├─────────────────────────────────────────────────────────┤
│     CLI Launcher (shared, invoked by host MCP config)   │
│                                                         │
│  bin/archcore{,.cmd,.ps1} + bin/CLI_VERSION             │
│  Resolves: $ARCHCORE_BIN → PATH → cache → download      │
│  Cache: $CODEX_PLUGIN_DATA → $CLAUDE_PLUGIN_DATA → XDG  │
└─────────────────────────────────────────────────────────┘
```

### Supported Hosts

| Host           | Priority | Plugin Manifest              | Hooks Config              | MCP Wiring                                     | Status      |
| -------------- | -------- | ---------------------------- | ------------------------- | ---------------------------------------------- | ----------- |
| Claude Code    | P0       | `.claude-plugin/plugin.json` | `hooks/hooks.json`        | Plugin-shipped `.mcp.json`                     | Production  |
| Cursor         | P1       | `.cursor-plugin/plugin.json` | `hooks/cursor.hooks.json` | User-registered externally                     | Implemented |
| Codex CLI      | P1       | `.codex-plugin/plugin.json`  | `hooks/codex.hooks.json`  | Plugin-shipped `.codex.mcp.json` (via manifest pointer) | Implemented |
| GitHub Copilot | P2       | TBD                          | TBD                       | TBD                                            | Future      |

## Contract Surface

### 1. Stdin Normalization (`bin/lib/normalize-stdin.sh`)

A POSIX shell library sourced by all bin scripts. Reads raw stdin JSON, detects the host, and exports normalized variables.

#### Canonical normalized variables

| Variable             | Description          | Source: Claude Code       | Source: Cursor                             | Source: Codex CLI                  |
| -------------------- | -------------------- | ------------------------- | ------------------------------------------ | ---------------------------------- |
| `ARCHCORE_HOST`      | Host identifier      | `"claude-code"` (default) | `"cursor"` (from `conversation_id`)        | `"codex"` (from `turn_id`)         |
| `ARCHCORE_RAW_STDIN` | Unmodified stdin     | Full stdin                | Full stdin                                 | Full stdin                         |
| `ARCHCORE_TOOL_NAME` | Normalized tool name | `tool_name` as-is         | Prefixed `mcp__archcore__` for MCP events  | `tool_name` as-is (snake_case)     |
| `ARCHCORE_FILE_PATH` | Target file path     | `tool_input.file_path`    | `file_path`                                | `tool_input.file_path`             |
| `ARCHCORE_DOC_PATH`  | Document path (MCP)  | `tool_input.path`         | Extracted from escaped `tool_input` string | `tool_input.path`                  |

#### Host detection heuristic

Priority: `$ARCHCORE_HOST` env var (if set) > stdin detection > default.

```
if stdin contains "conversation_id"  → cursor
if stdin contains "hookEventName"    → copilot
if stdin contains "turn_id"          → codex
else                                 → claude-code (default/fallback)
```

Cursor includes `conversation_id` in all hook events. GitHub Copilot uses camelCase `hookEventName` (distinct from `hook_event_name` snake_case used by Cursor and Codex). Codex CLI sends `turn_id` in turn-scoped events (`PreToolUse`, `PostToolUse`, `PermissionRequest`, `UserPromptSubmit`, `Stop`); SessionStart has no `turn_id`, but Codex shares Claude Code's snake_case schema, so the claude-code fallback handles SessionStart correctly (the explicit `codex` branch is a clarity-and-future-proofing measure, not a functional requirement for SessionStart).

#### Tool name normalization

Cursor's `afterMCPExecution` event sends bare MCP tool names (`create_document`, `update_document`). The normalizer prefixes them with `mcp__archcore__` so bin scripts work unchanged. Claude Code and Codex CLI send the fully-prefixed name as-is — no normalization needed.

```
Cursor afterMCPExecution: tool_name="create_document"
  → ARCHCORE_TOOL_NAME="mcp__archcore__create_document"

Cursor preToolUse: tool_name="Write"
  → ARCHCORE_TOOL_NAME="Write" (no change)

Claude Code: tool_name="mcp__archcore__create_document"
  → ARCHCORE_TOOL_NAME="mcp__archcore__create_document" (no change)

Codex CLI: tool_name="mcp__archcore__create_document"
  → ARCHCORE_TOOL_NAME="mcp__archcore__create_document" (no change)
```

#### Escaped JSON extraction

Cursor's `afterMCPExecution` sends `tool_input` as a JSON string (double-escaped). The normalizer provides `_archcore_json_val_unescaped()` to extract fields from escaped strings. Used as fallback when direct extraction fails for `ARCHCORE_DOC_PATH`. Claude Code and Codex CLI send `tool_input` as a structured object — direct extraction works.

#### Usage in bin scripts

```sh
#!/bin/sh
SCRIPT_DIR=$(dirname "$0")
. "$SCRIPT_DIR/lib/normalize-stdin.sh"

# Now use normalized variables:
# $ARCHCORE_HOST, $ARCHCORE_FILE_PATH, $ARCHCORE_TOOL_NAME, etc.
```

#### Output helpers

**`archcore_hook_block "reason"`** — Block the operation and exit. Uses `exit 2` with stderr message for all hosts. Exit code 2 is the universal blocking signal recognized by Claude Code, Cursor, and Codex CLI.

**`archcore_hook_info "message"`** — Emit informational message to the agent from a **PostToolUse** hook. Format varies by host:

| Host                          | Output format                                                                      |
| ----------------------------- | ---------------------------------------------------------------------------------- |
| Claude Code / Codex / Copilot | `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"..."}}` |
| Cursor                        | `{"additional_context":"..."}`                                                     |

**`archcore_hook_pretool_info "message"`** — Emit context injection from a **PreToolUse** hook (additive, non-blocking). Preserves multi-line output by encoding newlines as JSON `\n`. Callers exit 0 after invoking.

| Host                          | Output format                                                                      |
| ----------------------------- | ---------------------------------------------------------------------------------- |
| Claude Code / Codex / Copilot | `{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"..."}}`  |
| Cursor                        | `{"additional_context":"..."}` (support is host-version-dependent; graceful degradation) |

**`archcore_hook_allow`** — Allow the operation silently. `exit 0` for all hosts.

### 2. CLI Launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION`)

A host-agnostic launcher that resolves the Archcore CLI binary on demand. Invoked by host MCP configs (Claude Code's and Codex CLI's plugin-shipped `.mcp.json`) and by hook scripts that need the CLI (`bin/validate-archcore`, `bin/session-start`).

#### Resolution order (all platforms)

1. `$ARCHCORE_BIN` — explicit path to a binary. Enterprise pin / local dev escape hatch.
2. `archcore` on `PATH` — respects an existing global install. Loop guard: skipped if `command -v archcore` resolves back to the launcher itself.
3. Plugin-managed cache: `<cache>/archcore-v${VERSION}` where `<cache>` is (first-match):
   - POSIX: `$CODEX_PLUGIN_DATA/archcore/cli` → `$CLAUDE_PLUGIN_DATA/archcore/cli` → `$XDG_DATA_HOME/archcore-plugin/cli` → `$HOME/.local/share/archcore-plugin/cli`
   - Windows: `$env:CODEX_PLUGIN_DATA\archcore\cli` → `$env:CLAUDE_PLUGIN_DATA\archcore\cli` → `$env:LOCALAPPDATA\archcore-plugin\cli`
4. Download from `github.com/archcore-ai/cli/releases/download/v${VERSION}/archcore_<os>_<arch>.{tar.gz,zip}`, verify against `checksums.txt` (SHA-256), atomically stage into cache, then `exec`.

`$VERSION` is read from `bin/CLI_VERSION` (single-line semver). Bumping the plugin's CLI pin is a one-file change. The cache directory order favors the host's own data dir first (Codex prefers `CODEX_PLUGIN_DATA`, Claude Code prefers `CLAUDE_PLUGIN_DATA`); both fall through to XDG/LOCALAPPDATA.

#### Environment contract

| Variable                 | Effect                                                                                                |
| ------------------------ | ----------------------------------------------------------------------------------------------------- |
| `ARCHCORE_BIN`           | If set and executable, used unconditionally. Skips all other resolution steps.                       |
| `ARCHCORE_SKIP_DOWNLOAD` | If `"1"`, step 4 (download) is skipped and the launcher exits 1 when the cache miss. Used by `bin/session-start` to keep SessionStart non-blocking. |
| `ARCHCORE_HIDE_EMPTY_NUDGE` | If `"1"`, `bin/session-start` suppresses the empty-state advisory that points at `/archcore:bootstrap`. Does **not** suppress the "no .archcore/ directory" init prompt — that message is always required so agents know to call `init_project`. Use when Archcore is installed but you do not want users nudged about `/archcore:bootstrap`. |

Stdin, stdout, stderr pass through unchanged. Exit code is the CLI's exit code verbatim.

#### Checksum verification

SHA-256 via `sha256sum` / `shasum -a 256` (POSIX) or `Get-FileHash -Algorithm SHA256` (Windows). No fallback — if neither hashing tool is available, the launcher refuses to proceed. Checksum mismatch aborts install; the staged file is discarded.

#### Windows-specific handling

- `bin/archcore.cmd` is a one-line shim invoking PowerShell with `-NoProfile -NonInteractive -ExecutionPolicy Bypass`.
- `bin/archcore.ps1` strips Mark-of-the-Web via `Unblock-File` after staging, preventing SmartScreen prompts on first execution.
- Architecture detection uses `RuntimeInformation.OSArchitecture` (OS architecture, not process) so x64 PowerShell under ARM64 Prism emulation still installs the ARM64 binary.

### 3. Hook Event Mapping

| Plugin Hook                      | Claude Code Event                 | Cursor Event         | Codex CLI Event                   | Notes                                                                 |
| -------------------------------- | --------------------------------- | -------------------- | --------------------------------- | --------------------------------------------------------------------- |
| Session context load             | `SessionStart`                    | `sessionStart`       | `SessionStart`                    | All three hosts support this event                                    |
| Block .archcore/ writes          | `PreToolUse` (Write\|Edit)        | `preToolUse` (Write) | `PreToolUse` (Write\|Edit\|apply_patch) | Cursor has no Edit tool; Codex's native edit primitive is `apply_patch` |
| Inject context for source edits  | `PreToolUse` (Write\|Edit)        | `preToolUse` (Write) | `PreToolUse` (Write\|Edit\|apply_patch) | Second entry on same matcher; disjoint path set from the block hook    |
| Validate after MCP ops           | `PostToolUse` (mcp**archcore**\*) | `afterMCPExecution`  | `PostToolUse` (mcp**archcore**\*) | Cursor has dedicated MCP event; Codex mirrors Claude Code             |
| Cascade detection                | `PostToolUse` (update_document)   | `afterMCPExecution`  | `PostToolUse` (update_document)   | Script filters for update internally                                  |
| Precision check                  | `PostToolUse` (create/update)     | (not registered)     | `PostToolUse` (create/update)     | Cursor's afterMCPExecution doesn't currently include precision check  |

Key differences:

- **Event naming**: Claude Code and Codex CLI use PascalCase (`PreToolUse`); Cursor uses camelCase (`preToolUse`).
- **MCP hooks**: Claude Code and Codex CLI use `PostToolUse` with MCP tool matcher; Cursor has `afterMCPExecution` — a dedicated event for all MCP operations.
- **Cascade filtering**: Claude Code and Codex matchers filter for `update_document` only; Cursor's `afterMCPExecution` fires for all MCP tools — `check-cascade` script exits early when `ARCHCORE_DOC_PATH` is empty.
- **Two `PreToolUse` hooks on the same matcher**: all three hosts register `check-archcore-write` AND `check-code-alignment` on `Write|Edit` / `Write` / `Write|Edit|apply_patch`. They do not conflict — the block hook acts only on `.archcore/*.md`, the injection hook acts only outside `.archcore/`.
- **No Write/Edit/apply_patch PostToolUse**: no host runs `validate-archcore` after `Write`/`Edit`/`apply_patch`. `PreToolUse` already blocks writes to `.archcore/*.md` (PostToolUse only fires on success), so a Write/Edit PostToolUse entry would fork a shell on every write anywhere in the repo for no benefit. Validation runs only on the MCP path.

### 4. Per-Host Hooks Configuration

#### Claude Code (`hooks/hooks.json`)

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "", "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/session-start" }] }
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
        "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/validate-archcore", "timeout": 3 }]
      },
      {
        "matcher": "mcp__archcore__update_document",
        "hooks": [{ "type": "command", "command": "${CLAUDE_PLUGIN_ROOT}/bin/check-cascade", "timeout": 3 }]
      }
    ]
  }
}
```

#### Cursor (`hooks/cursor.hooks.json`)

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [
      { "matcher": "", "hooks": [{ "type": "command", "command": "${CURSOR_PLUGIN_ROOT}/bin/session-start" }] }
    ],
    "preToolUse": [
      {
        "matcher": "Write",
        "hooks": [
          { "type": "command", "command": "${CURSOR_PLUGIN_ROOT}/bin/check-archcore-write", "timeout": 1 },
          { "type": "command", "command": "${CURSOR_PLUGIN_ROOT}/bin/check-code-alignment", "timeout": 1 }
        ]
      }
    ],
    "afterMCPExecution": [
      {
        "matcher": "",
        "hooks": [
          { "type": "command", "command": "${CURSOR_PLUGIN_ROOT}/bin/validate-archcore", "timeout": 3 },
          { "type": "command", "command": "${CURSOR_PLUGIN_ROOT}/bin/check-cascade", "timeout": 3 }
        ]
      }
    ]
  }
}
```

#### Codex CLI (`hooks/codex.hooks.json`)

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "", "hooks": [{ "type": "command", "command": "./bin/session-start" }] }
    ],
    "PreToolUse": [
      {
        "matcher": "Write|Edit|apply_patch",
        "hooks": [
          { "type": "command", "command": "./bin/check-archcore-write", "timeout": 1 },
          { "type": "command", "command": "./bin/check-code-alignment", "timeout": 1 }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "mcp__archcore__create_document|mcp__archcore__update_document|mcp__archcore__remove_document|mcp__archcore__add_relation|mcp__archcore__remove_relation",
        "hooks": [{ "type": "command", "command": "./bin/validate-archcore", "timeout": 3 }]
      },
      {
        "matcher": "mcp__archcore__update_document",
        "hooks": [{ "type": "command", "command": "./bin/check-cascade", "timeout": 3 }]
      },
      {
        "matcher": "mcp__archcore__create_document|mcp__archcore__update_document",
        "hooks": [{ "type": "command", "command": "./bin/check-precision", "timeout": 3 }]
      }
    ]
  }
}
```

The Codex config mirrors Claude Code's structure with the `apply_patch` matcher addition for Codex's native edit primitive and the `check-precision` PostToolUse entry that has been part of the Claude Code config since the precision-over-coverage initiative.

Codex hook commands MUST be plugin-relative (`./bin/...`). Current Codex plugin examples use relative command paths and the installed Codex binary does not expose a documented `${CODEX_PLUGIN_ROOT}` substitution variable.

Codex hooks are also gated by Codex runtime support: the official hooks documentation requires `[features].codex_hooks = true`, and current upstream code/issues show plugin-local hook discovery has been in flux. The plugin MUST still ship `hooks/codex.hooks.json` and the manifest pointer so it is ready for the documented plugin surface, but end-to-end hook execution MUST be treated as a runtime smoke test rather than assumed from static packaging alone.

### 5. Plugin Manifests

#### Claude Code (`.claude-plugin/plugin.json`)

```json
{
  "name": "archcore",
  "description": "Make your AI agent code with your project's architecture, rules, and decisions.",
  "version": "0.3.7",
  "author": { "name": "Archcore" },
  "license": "Apache-2.0",
  "repository": "https://github.com/archcore-ai/plugin"
}
```

Claude Code discovers skills, agents, and hooks by convention (fixed directory names). Manifest contains only metadata. MCP registration is separate — see section 6.

#### Cursor (`.cursor-plugin/plugin.json`)

```json
{
  "name": "archcore",
  "description": "...",
  "version": "0.3.7",
  "author": { "name": "Archcore" },
  "license": "Apache-2.0",
  "repository": "https://github.com/archcore-ai/plugin",
  "skills": "skills/",
  "agents": "agents/",
  "hooks": "hooks/cursor.hooks.json",
  "rules": "rules/"
}
```

Cursor requires explicit paths to components. Only `name` is required; all other fields are optional but recommended. No `mcpServers` field — Cursor users register MCP externally.

#### Codex CLI (`.codex-plugin/plugin.json`)

```json
{
  "name": "archcore",
  "description": "...",
  "version": "0.3.7",
  "author": { "name": "Archcore" },
  "license": "Apache-2.0",
  "repository": "https://github.com/archcore-ai/plugin",
  "skills": "./skills/",
  "hooks": "./hooks/codex.hooks.json",
  "mcpServers": "./.codex.mcp.json",
  "interface": {
    "displayName": "Archcore",
    "category": "Coding",
    "capabilities": ["Interactive", "Read", "Write"]
  }
}
```

Codex CLI requires component paths to be explicit and prefixed with `./` (relative-to-plugin-root convention). Manifest declares `mcpServers` pointing at the Codex-specific plugin-root `.codex.mcp.json` — Codex resolves the pointer and registers the MCP server automatically. UI metadata belongs under `interface{}`; legacy top-level `displayName`, `category`, and `tags` fields MUST NOT be used in the Codex manifest. `.codex-plugin/` contains only `plugin.json`; marketplace metadata lives in `.agents/plugins/marketplace.json`.

Plugin manifests MUST use identical `name`, `description`, and `version` across all hosts. This is enforced by `test/structure/codex-plugin.bats` and the existing claude/cursor parity tests.

### 6. MCP Server Wiring

#### Claude Code — plugin-shipped `.mcp.json`

The plugin root contains `.mcp.json`:

```json
{
  "mcpServers": {
    "archcore": {
      "command": "${CLAUDE_PLUGIN_ROOT}/bin/archcore",
      "args": ["mcp"]
    }
  }
}
```

Claude Code reads this on plugin load and registers `archcore` as a plugin-provided MCP server. The `command` points at the bundled launcher (section 2), which resolves the actual CLI binary at invocation time. No manual `claude mcp add` or project-level `.mcp.json` is required.

**Duplicate suppression**: if a user has registered `archcore` globally (`claude mcp add ...`) or the repo has its own `.mcp.json` with a matching command, Claude Code dedupes. Because the launcher defers to `PATH` when a global `archcore` exists, users with prior installs see no behavior change — both registrations effectively resolve the same binary.

#### Cursor — user-registered

Cursor users register MCP externally via Cursor's MCP settings UI or a project `mcp.json`. The launcher still works (same binary, same resolution order) — just isn't wired in via a plugin-shipped MCP config. Users point Cursor's MCP config at the launcher by absolute path, or install the CLI globally and point at `archcore`.

This divergence is deliberate: Cursor's plugin runtime does not expose `${CLAUDE_PLUGIN_ROOT}`-equivalent path substitution for plugin-provided MCP, so portable MCP wiring across hosts is blocked until that gap closes.

#### Codex CLI — plugin-shipped via manifest pointer

Codex CLI does not auto-discover the Claude-oriented `.mcp.json` the way Claude Code does — instead, the plugin manifest's `mcpServers` field points at a Codex-specific MCP file:

```json
// .codex-plugin/plugin.json
{ "mcpServers": "./.codex.mcp.json", ... }
```

The file uses the same wrapper shape as Claude Code but a plugin-relative command path:

```json
{
  "mcpServers": {
    "archcore": {
      "command": "./bin/archcore",
      "args": ["mcp"]
    }
  }
}
```

Codex resolves the pointer at plugin load time and registers `archcore` as a plugin-provided MCP server. The Codex MCP config MUST NOT reference `${CLAUDE_PLUGIN_ROOT}` or `${CODEX_PLUGIN_ROOT}`.

### 7. Codex Subagent TOML Files

Codex CLI requires subagents in TOML format with `developer_instructions`, `sandbox_mode`, optional `disabled_tools[]`, and optional `model_reasoning_effort`. Two TOML files ship alongside the MD originals:

- `agents/archcore-auditor.toml` — `sandbox_mode = "read-only"` plus `disabled_tools = [...]` listing the five mutating MCP tools (`create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`). Read-only enforcement comes from sandbox mode; the `disabled_tools[]` list is a defense-in-depth layer that prevents the auditor from invoking mutating MCP tools even within its read-only sandbox.

- `agents/archcore-assistant.toml` — `sandbox_mode = "workspace-write"` (full read/write). No `disabled_tools[]` — assistant has full MCP access by design.

Both TOML files contain identical `developer_instructions` content to their MD counterparts (knowledge-tree bootstrap preamble, core principle, working guidelines). Tests enforce this structural parity (`test/structure/agents.bats`).

### 8. Cursor Rules

Rules in `rules/` provide context injection. Two files:

**`rules/archcore-context.mdc`** — `alwaysApply: true`. Injected into every session. Contains: document type reference, MCP tool names, MCP-only principle.

**`rules/archcore-files.mdc`** — `globs: ".archcore/**"`. Injected when `.archcore/` files are in context. Reminds about MCP-only operations.

## Normative Behavior

- All bin scripts MUST source `bin/lib/normalize-stdin.sh` before processing stdin.
- The normalizer MUST detect the host and export `ARCHCORE_HOST` correctly (claude-code, cursor, copilot, codex).
- The normalizer MUST normalize MCP tool names to `mcp__archcore__` prefix for Cursor's `afterMCPExecution` events.
- The normalizer MUST handle escaped JSON strings in Cursor's `tool_input` field.
- `archcore_hook_block` MUST use exit code 2 for all hosts (universally recognized).
- `archcore_hook_info` MUST emit the correct PostToolUse JSON format per host (`hookSpecificOutput` for Claude Code/Codex/Copilot, `additional_context` for Cursor).
- `archcore_hook_pretool_info` MUST emit the correct PreToolUse JSON format per host, preserving multi-line messages via JSON `\n` escapes.
- Per-host hooks config files MUST map the active hook functions for that host's event model. Claude Code and Codex CLI register: session-start, check-archcore-write + check-code-alignment on PreToolUse, validate-archcore on the MCP-mutation matcher, check-cascade on update_document, check-precision on create/update. Cursor registers session-start, check-archcore-write + check-code-alignment, and validate-archcore + check-cascade on `afterMCPExecution`. No host MUST register `validate-archcore` on the Write/Edit/apply_patch PostToolUse path.
- Both PreToolUse hooks on the `Write|Edit` / `Write` / `Write|Edit|apply_patch` matcher MUST coexist and act on disjoint path sets — `check-archcore-write` on `.archcore/*.md`, `check-code-alignment` on source paths outside `.archcore/`.
- Plugin manifests MUST use identical `name`, `description`, and `version` across all hosts.
- Plugin manifests for Claude Code (`.claude-plugin/plugin.json`) MUST NOT declare `mcpServers`; MCP wiring lives in the plugin-root `.mcp.json`. Plugin manifests for Codex CLI (`.codex-plugin/plugin.json`) MUST declare `mcpServers` as a pointer to `.codex.mcp.json` (Codex should not consume Claude's `${CLAUDE_PLUGIN_ROOT}`-based `.mcp.json`).
- The CLI launcher MUST resolve in order: `$ARCHCORE_BIN` → `PATH` (with loop guard) → cache → download. The cache directory list MUST favor host-prefixed dirs (`$CODEX_PLUGIN_DATA` and `$CLAUDE_PLUGIN_DATA`) before XDG/LOCALAPPDATA fallbacks. Downloads MUST be checksum-verified.
- `bin/session-start` MUST pass `ARCHCORE_SKIP_DOWNLOAD=1` when invoking the launcher so SessionStart never blocks on network.
- `bin/session-start` MUST respect `ARCHCORE_HIDE_EMPTY_NUDGE=1` by suppressing the bootstrap advisory line while still emitting the `init_project` prompt for missing `.archcore/`.
- For Codex CLI, both subagents MUST ship as TOML files (`agents/archcore-{assistant,auditor}.toml`) alongside the MD originals. The auditor TOML MUST have `sandbox_mode = "read-only"` and `disabled_tools[]` containing all five mutating MCP tools.
- Adding a new host MUST NOT require changes to skills, agents (MD or TOML), core bin script logic, or the launcher.

## Constraints

- `bin/lib/normalize-stdin.sh` MUST be POSIX shell compatible (no bashisms).
- `bin/archcore` (POSIX launcher) MUST be POSIX shell compatible.
- Host detection MUST work without external dependencies (no `jq`, only `grep`/`sed`).
- Stdin normalization MUST complete within 100ms (included in hook timeout budget).
- Launcher resolution steps 1–3 MUST complete in under 100ms on a warm filesystem.
- Launcher download (step 4) MAY take seconds — it runs only on first use, not inside hook-timeout-bounded contexts.
- Plugin root variable handling varies by host. Claude Code hook configs use `${CLAUDE_PLUGIN_ROOT}`; Cursor hook configs use `${CURSOR_PLUGIN_ROOT}` (and Cursor also recognizes `${CLAUDE_PLUGIN_ROOT}` as an alias); Codex hook configs MUST use plugin-relative commands (`./bin/...`).

## Invariants

- Shared core components (skills, agents, hook scripts, launcher) are identical across all hosts.
- A change to a skill, agent, or launcher benefits all hosts simultaneously.
- Per-host adapter files contain no business logic — only configuration and format mapping.
- The normalizer always falls back to Claude Code format if host detection fails (backward compatible).
- Hook semantics (what gets blocked, what gets validated, what gets injected) are identical across hosts — only the wire format differs.
- Exit code 2 blocks operations universally across all supported hosts.
- The launcher always prefers an existing global `archcore` on `PATH` over the plugin-managed cache (avoids double-binary situations on systems where the user manages their own install).
- Codex TOML subagents and MD subagents share the same `developer_instructions` content; structural drift is detected by `test/structure/agents.bats`.

## Error Handling

- **Unknown host detected**: Fall back to Claude Code format. Log warning to stderr.
- **Stdin JSON missing expected fields**: Export empty variables. Bin script logic handles missing fields gracefully.
- **Escaped JSON extraction fails**: `ARCHCORE_DOC_PATH` remains empty. `check-cascade` exits early (no cascade possible).
- **Plugin root variable not set**: Bin scripts use `$(dirname "$0")` for relative paths.
- **Launcher cannot resolve CLI and `ARCHCORE_SKIP_DOWNLOAD=1`**: exits 1 with a stderr message. Calling hook scripts (`validate-archcore`, `check-cascade`) treat this as a silent skip and exit 0 (don't break the session).
- **Launcher download fails (network, checksum mismatch, unsupported OS/arch)**: exits 1 with a diagnostic on stderr. MCP calls fail until resolved; the agent surfaces the error to the user. `bin/session-start` never hits this path because it always passes `ARCHCORE_SKIP_DOWNLOAD=1`.
- **`.archcore/` exists but is functionally empty (no `.md` file ≥ 200 bytes)**: `bin/session-start` emits a non-blocking advisory pointing at `/archcore:bootstrap` unless `ARCHCORE_HIDE_EMPTY_NUDGE=1`. Empty-state check uses `bin/lib/empty-state.sh` (POSIX shell, no jq, no MCP calls).
- **Cursor `preToolUse` does not honor `additional_context`**: the injection hook's output is silently ignored by the host. Graceful degradation — the SessionStart index and the `/archcore:context` pull skill still cover JTBD #1 on Cursor until Cursor exposes an equivalent.

## Conformance

The multi-host compatibility layer conforms to this specification if:

1. All bin scripts source `bin/lib/normalize-stdin.sh` and use normalized variables
2. Host detection correctly identifies Claude Code, Cursor, Codex CLI, and Copilot (and any additional hosts)
3. MCP tool names are normalized to `mcp__archcore__` prefix regardless of host
4. Output helpers emit correct format per detected host — `archcore_hook_info` for PostToolUse, `archcore_hook_pretool_info` for PreToolUse
5. The CLI launcher implements the full resolution order, with checksum verification on downloads, and the cache directory chain favors host-prefixed dirs (`CODEX_PLUGIN_DATA`, `CLAUDE_PLUGIN_DATA`) before XDG/LOCALAPPDATA
6. Each supported host has a complete hooks config mapping the active hook functions for its event model
7. Each supported host has a valid plugin manifest with consistent metadata. Claude Code and Cursor manifests do NOT contain a `mcpServers` field; Codex CLI manifest declares `mcpServers` as a pointer to `.codex.mcp.json`
8. The plugin root ships `.mcp.json` for Claude Code and `.codex.mcp.json` for Codex CLI; both point at the launcher with `args: ["mcp"]`
9. Codex CLI ships TOML subagent files (`agents/archcore-{assistant,auditor}.toml`) with `developer_instructions` content matching the MD originals; auditor TOML has `sandbox_mode = "read-only"` and `disabled_tools[]` for all five mutating MCP tools
10. Shared components (skills, agents, hook scripts, launcher) contain zero host-specific references
11. Adding a new host requires only new config files, not changes to shared components
12. `bin/session-start` passes `ARCHCORE_SKIP_DOWNLOAD=1` when invoking the launcher
13. `bin/session-start` honors `ARCHCORE_HIDE_EMPTY_NUDGE=1` by suppressing the bootstrap advisory line (and only that line)
