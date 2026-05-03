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

Define the contract for the multi-host compatibility layer that enables the Archcore plugin to run in Claude Code, Cursor, GitHub Copilot, and other AI coding tools from a single repository. This specification covers host detection, stdin normalization for hook scripts, per-host hook event mapping, per-host manifest structure, and the cross-host CLI launcher that resolves the Archcore CLI binary on demand.

MCP server registration is now **partially in scope**: Claude Code receives a plugin-shipped `.mcp.json` wired to the bundled launcher (see the Bundled CLI Launcher ADR). Cursor and other hosts still rely on user-registered MCP — the launcher itself is host-agnostic, only the plugin-level MCP wiring is host-specific.

## Scope

The compatibility layer — specifically: `bin/lib/normalize-stdin.sh`, `bin/archcore` / `bin/archcore.cmd` / `bin/archcore.ps1` / `bin/CLI_VERSION` (the launcher and its version pin), per-host `hooks/*.hooks.json` files, per-host plugin manifests, and the Claude Code `.mcp.json`. Does NOT cover the shared hook script logic (skills, agents, hook scripts themselves), which are host-agnostic by design, nor the CLI binary's own behavior.

## Authority

This specification is authoritative for cross-host behavior. The Multi-Host Plugin Architecture ADR provides the architectural rationale for the shared-core / per-host-adapter split. The Bundled CLI Launcher ADR is authoritative for launcher resolution order, auto-install policy, and plugin-owned MCP wiring. The Hooks and Validation System Specification remains authoritative for hook semantics (what each hook does); this spec defines how hooks adapt to different host runtimes.

## Subject

### System Overview

The plugin splits into a **shared core** (skills, agents, bin scripts, CLI launcher) and a **host adapter layer** (manifests, hooks configs, stdin normalization, Claude Code MCP wiring). The adapter layer is pure configuration plus one small shell library for stdin format detection.

```
┌─────────────────────────────────────────────────────────┐
│                    Shared Core                           │
│                                                         │
│  skills/ (16)  agents/ (2)                              │
│  bin/ — 6 hook scripts + 3 launcher scripts + pin file  │
│                                                         │
│  100% host-agnostic — uses only MCP tools + Read/Grep   │
├─────────────────────────────────────────────────────────┤
│              Host Adapter Layer                          │
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │ Claude Code  │  │ Cursor       │  │ Copilot      │   │
│  │              │  │              │  │   (future)   │   │
│  │ .claude-     │  │ .cursor-     │  │ .github/     │   │
│  │  plugin/     │  │  plugin/     │  │              │   │
│  │ hooks.json   │  │ cursor.hooks │  │              │   │
│  │ .mcp.json    │  │ (user MCP)   │  │              │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                         │
│  bin/lib/normalize-stdin.sh — detects host, normalizes  │
├─────────────────────────────────────────────────────────┤
│     CLI Launcher (shared, invoked by host MCP config)   │
│                                                         │
│  bin/archcore{,.cmd,.ps1} + bin/CLI_VERSION             │
│  Resolves: $ARCHCORE_BIN → PATH → cache → download      │
└─────────────────────────────────────────────────────────┘
```

### Supported Hosts

| Host           | Priority | Plugin Manifest              | Hooks Config              | MCP Wiring                 | Status      |
| -------------- | -------- | ---------------------------- | ------------------------- | -------------------------- | ----------- |
| Claude Code    | P0       | `.claude-plugin/plugin.json` | `hooks/hooks.json`        | Plugin-shipped `.mcp.json` | Production  |
| Cursor         | P1       | `.cursor-plugin/plugin.json` | `hooks/cursor.hooks.json` | User-registered externally | Implemented |
| GitHub Copilot | P2       | TBD                          | TBD                       | TBD                        | Future      |
| Codex CLI      | P2       | TBD                          | TBD                       | TBD                        | Future      |

## Contract Surface

### 1. Stdin Normalization (`bin/lib/normalize-stdin.sh`)

A POSIX shell library sourced by all bin scripts. Reads raw stdin JSON, detects the host, and exports normalized variables.

#### Canonical normalized variables

| Variable             | Description          | Source: Claude Code       | Source: Cursor                             |
| -------------------- | -------------------- | ------------------------- | ------------------------------------------ |
| `ARCHCORE_HOST`      | Host identifier      | `"claude-code"` (default) | `"cursor"` (from `conversation_id`)        |
| `ARCHCORE_RAW_STDIN` | Unmodified stdin     | Full stdin                | Full stdin                                 |
| `ARCHCORE_TOOL_NAME` | Normalized tool name | `tool_name` as-is         | Prefixed `mcp__archcore__` for MCP events  |
| `ARCHCORE_FILE_PATH` | Target file path     | `tool_input.file_path`    | `file_path`                                |
| `ARCHCORE_DOC_PATH`  | Document path (MCP)  | `tool_input.path`         | Extracted from escaped `tool_input` string |

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

**`archcore_hook_info "message"`** — Emit informational message to the agent from a **PostToolUse** hook. Format varies by host:

| Host        | Output format                                                                      |
| ----------- | ---------------------------------------------------------------------------------- |
| Claude Code | `{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"..."}}` |
| Cursor      | `{"additional_context":"..."}`                                                     |

**`archcore_hook_pretool_info "message"`** — Emit context injection from a **PreToolUse** hook (additive, non-blocking). Preserves multi-line output by encoding newlines as JSON `\n`. Callers exit 0 after invoking.

| Host        | Output format                                                                      |
| ----------- | ---------------------------------------------------------------------------------- |
| Claude Code | `{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"..."}}`  |
| Cursor      | `{"additional_context":"..."}` (support is host-version-dependent; graceful degradation) |

**`archcore_hook_allow`** — Allow the operation silently. `exit 0` for all hosts.

### 2. CLI Launcher (`bin/archcore`, `bin/archcore.cmd`, `bin/archcore.ps1`, `bin/CLI_VERSION`)

A host-agnostic launcher that resolves the Archcore CLI binary on demand. Invoked by host MCP configs (Claude Code's `.mcp.json` today) and by hook scripts that need the CLI (`bin/validate-archcore`, `bin/session-start`).

#### Resolution order (all platforms)

1. `$ARCHCORE_BIN` — explicit path to a binary. Enterprise pin / local dev escape hatch.
2. `archcore` on `PATH` — respects an existing global install. Loop guard: skipped if `command -v archcore` resolves back to the launcher itself.
3. Plugin-managed cache: `<cache>/archcore-v${VERSION}` where `<cache>` is (first-match):
   - POSIX: `$CLAUDE_PLUGIN_DATA/archcore/cli` → `$XDG_DATA_HOME/archcore-plugin/cli` → `$HOME/.local/share/archcore-plugin/cli`
   - Windows: `$env:CLAUDE_PLUGIN_DATA\archcore\cli` → `$env:LOCALAPPDATA\archcore-plugin\cli`
4. Download from `github.com/archcore-ai/cli/releases/download/v${VERSION}/archcore_<os>_<arch>.{tar.gz,zip}`, verify against `checksums.txt` (SHA-256), atomically stage into cache, then `exec`.

`$VERSION` is read from `bin/CLI_VERSION` (single-line semver). Bumping the plugin's CLI pin is a one-file change.

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

| Plugin Hook                      | Claude Code Event                 | Cursor Event         | Notes                                                                 |
| -------------------------------- | --------------------------------- | -------------------- | --------------------------------------------------------------------- |
| Session context load             | `SessionStart`                    | `sessionStart`       | Both hosts support this event                                         |
| Block .archcore/ writes          | `PreToolUse` (Write\|Edit)        | `preToolUse` (Write) | Cursor has no Edit tool                                               |
| Inject context for source edits  | `PreToolUse` (Write\|Edit)        | `preToolUse` (Write) | Second entry on same matcher; disjoint path set from the block hook    |
| Validate after MCP ops           | `PostToolUse` (mcp**archcore**\*) | `afterMCPExecution`  | Cursor has dedicated MCP event                                        |
| Cascade detection                | `PostToolUse` (update_document)   | `afterMCPExecution`  | Script filters for update internally                                  |

Key differences:

- **Event naming**: Claude Code uses PascalCase (`PreToolUse`), Cursor uses camelCase (`preToolUse`)
- **MCP hooks**: Claude Code uses `PostToolUse` with MCP tool matcher; Cursor has `afterMCPExecution` — a dedicated event for all MCP operations
- **Cascade filtering**: Claude Code matcher filters for `update_document` only; Cursor's `afterMCPExecution` fires for all MCP tools — `check-cascade` script exits early when `ARCHCORE_DOC_PATH` is empty
- **Two `PreToolUse` hooks on the same matcher**: both hosts register `check-archcore-write` AND `check-code-alignment` on `Write|Edit` / `Write`. They do not conflict — the block hook acts only on `.archcore/*.md`, the injection hook acts only outside `.archcore/`.
- **No Write/Edit PostToolUse**: neither host runs `validate-archcore` after `Write`/`Edit`. `PreToolUse` already blocks writes to `.archcore/*.md` (PostToolUse only fires on success), so a Write/Edit PostToolUse entry would fork a shell on every write anywhere in the repo for no benefit. Validation runs only on the MCP path.

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

### 5. Plugin Manifests

#### Claude Code (`.claude-plugin/plugin.json`)

```json
{
  "name": "archcore",
  "description": "Git-native context for AI coding agents",
  "version": "0.1.0",
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
  "description": "Git-native context for AI coding agents",
  "version": "0.1.0",
  "author": { "name": "Archcore" },
  "license": "Apache-2.0",
  "repository": "https://github.com/archcore-ai/plugin",
  "keywords": ["documentation", "architecture", "knowledge-base", "mcp"],
  "skills": "skills/",
  "agents": "agents/",
  "hooks": "hooks/cursor.hooks.json",
  "rules": "rules/"
}
```

Cursor requires explicit paths to components. Only `name` is required; all other fields are optional but recommended. No `mcpServers` field — Cursor users register MCP externally.

Plugin manifests MUST use identical `name`, `description`, and `version` across all hosts.

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

### 7. Cursor Rules

Rules in `rules/` provide context injection. Two files:

**`rules/archcore-context.mdc`** — `alwaysApply: true`. Injected into every session. Contains: document type reference, MCP tool names, MCP-only principle.

**`rules/archcore-files.mdc`** — `globs: ".archcore/**"`. Injected when `.archcore/` files are in context. Reminds about MCP-only operations.

## Normative Behavior

- All bin scripts MUST source `bin/lib/normalize-stdin.sh` before processing stdin.
- The normalizer MUST detect the host and export `ARCHCORE_HOST` correctly.
- The normalizer MUST normalize MCP tool names to `mcp__archcore__` prefix for Cursor's `afterMCPExecution` events.
- The normalizer MUST handle escaped JSON strings in Cursor's `tool_input` field.
- `archcore_hook_block` MUST use exit code 2 for all hosts (universally recognized).
- `archcore_hook_info` MUST emit the correct PostToolUse JSON format per host.
- `archcore_hook_pretool_info` MUST emit the correct PreToolUse JSON format per host, preserving multi-line messages via JSON `\n` escapes.
- Per-host hooks config files MUST map the five active hook functions (session-start, check-archcore-write, check-code-alignment, validate-archcore on the MCP path, check-cascade on update_document). No host MUST register `validate-archcore` on the Write/Edit PostToolUse path.
- Both PreToolUse hooks on the `Write|Edit` / `Write` matcher MUST coexist and act on disjoint path sets — `check-archcore-write` on `.archcore/*.md`, `check-code-alignment` on source paths outside `.archcore/`.
- Plugin manifests MUST use identical `name`, `description`, and `version` across all hosts.
- Plugin manifests (i.e., `plugin.json`) MUST NOT declare `mcpServers`. Claude Code MCP wiring lives in the plugin-root `.mcp.json`, not in the manifest.
- The CLI launcher MUST resolve in order: `$ARCHCORE_BIN` → `PATH` (with loop guard) → cache → download. Downloads MUST be checksum-verified.
- `bin/session-start` MUST pass `ARCHCORE_SKIP_DOWNLOAD=1` when invoking the launcher so SessionStart never blocks on network.
- `bin/session-start` MUST respect `ARCHCORE_HIDE_EMPTY_NUDGE=1` by suppressing the bootstrap advisory line while still emitting the `init_project` prompt for missing `.archcore/`.
- Adding a new host MUST NOT require changes to skills, agents, core bin script logic, or the launcher.

## Constraints

- `bin/lib/normalize-stdin.sh` MUST be POSIX shell compatible (no bashisms).
- `bin/archcore` (POSIX launcher) MUST be POSIX shell compatible.
- Host detection MUST work without external dependencies (no `jq`, only `grep`/`sed`).
- Stdin normalization MUST complete within 100ms (included in hook timeout budget).
- Launcher resolution steps 1–3 MUST complete in under 100ms on a warm filesystem.
- Launcher download (step 4) MAY take seconds — it runs only on first use, not inside hook-timeout-bounded contexts.
- Plugin root variable name varies by host (`${CLAUDE_PLUGIN_ROOT}`, `${CURSOR_PLUGIN_ROOT}`) — hooks configs MUST use the host-specific variable name. Cursor also recognizes `${CLAUDE_PLUGIN_ROOT}` as an alias.

## Invariants

- Shared core components (skills, agents, hook scripts, launcher) are identical across all hosts.
- A change to a skill, agent, or launcher benefits all hosts simultaneously.
- Per-host adapter files contain no business logic — only configuration and format mapping.
- The normalizer always falls back to Claude Code format if host detection fails (backward compatible).
- Hook semantics (what gets blocked, what gets validated, what gets injected) are identical across hosts — only the wire format differs.
- Exit code 2 blocks operations universally across all supported hosts.
- The launcher always prefers an existing global `archcore` on `PATH` over the plugin-managed cache (avoids double-binary situations on systems where the user manages their own install).

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
2. Host detection correctly identifies Claude Code and Cursor (and any additional hosts)
3. MCP tool names are normalized to `mcp__archcore__` prefix regardless of host
4. Output helpers emit correct format per detected host — `archcore_hook_info` for PostToolUse, `archcore_hook_pretool_info` for PreToolUse
5. The CLI launcher implements the full resolution order, with checksum verification on downloads
6. Each supported host has a complete hooks config mapping the five active hook functions (session-start, check-archcore-write, check-code-alignment, MCP-path validate-archcore, update_document check-cascade) and does NOT register validate-archcore on Write/Edit PostToolUse
7. Each supported host has a valid plugin manifest with consistent metadata and no `mcpServers` field in the manifest itself
8. Claude Code's plugin root ships `.mcp.json` pointing at `${CLAUDE_PLUGIN_ROOT}/bin/archcore mcp`
9. Shared components (skills, agents, hook scripts, launcher) contain zero host-specific references
10. Adding a new host requires only new config files, not changes to shared components
11. `bin/session-start` passes `ARCHCORE_SKIP_DOWNLOAD=1` when invoking the launcher
12. `bin/session-start` honors `ARCHCORE_HIDE_EMPTY_NUDGE=1` by suppressing the bootstrap advisory line (and only that line)
