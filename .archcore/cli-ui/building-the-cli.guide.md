---
title: "Building the Archcore CLI"
status: accepted
tags:
  - "cli-ui"
---

## Overview

Guide for building, testing, and extending the Archcore CLI — a Go tool that manages a local `.archcore/` directory of structured documents and integrates with AI coding agents (Claude Code, Cursor, Gemini CLI, GitHub Copilot, and others) via MCP and hooks.

See also: [Supported AI Agents Registry](../integrations/supported-ai-agents.doc.md), [CLI Hooks Reference](../integrations/cli-hooks-reference.doc.md), [Agent Integration Guide](../integrations/agent-hooks-integration.guide.md).

## Prerequisites

- Go 1.24+, Git
- Familiarity with Go modules, packages, and testing

## Build & Test

```bash
go build -o archcore .     # Build binary
go test ./...              # Run all tests
go test ./cmd/ -run TestX  # Run a specific test
```

## Getting Started

```bash
./archcore init       # Interactive setup wizard (directories, agent detection)
./archcore doctor     # Health check: structure + settings + server connectivity
./archcore status     # Structural checks only (naming, frontmatter, categories)
./archcore update     # Self-update to the latest release
```

`init` creates `.archcore/` with a free-form directory structure — documents are organized by domain/feature/team, and category is derived from the filename suffix (`slug.type.md`). Settings go in `.archcore/settings.json`. It also auto-detects AI agents and installs hooks + MCP config for all found agents, then offers (opt-in) to write a usage-nudge instruction file per agent.

## Settings and Configuration

Settings live in `.archcore/settings.json`. Use the `config` command to read and modify them:

```bash
./archcore config get sync        # Read a setting
./archcore config set language ru  # Set a setting
```

### Settings Fields

| Field          | Type   | Description                                                    |
|----------------|--------|----------------------------------------------------------------|
| `sync`         | string | Sync mode: `none`, `cloud`, or `on-prem`                      |
| `project_id`   | int    | Project ID for cloud/on-prem sync (optional)                   |
| `archcore_url` | string | Server URL for on-prem sync (required for on-prem)             |
| `language`     | string | Language for MCP-generated document content (default: `"en"`)  |

The `language` field controls the language the MCP server uses when generating document content (section headers, placeholders, descriptions). It is sync-independent — available in all modes. When not set, defaults to `"en"`. Uses `omitempty` so it only appears in settings.json when explicitly configured.

See [Sync Mode Field Validation](../sync/sync-mode-field-validation.rule.md) for per-sync-mode field rules, and [Optional Settings Omit Defaults](../cli/optional-settings-omit-defaults.rule.md) for the convention on optional fields.

## How to Add a New Setting

1. Add the field to `Settings` struct in `internal/config/config.go` with appropriate `json:"...,omitempty"` tag
2. Add the field name to `allowedFields` for the relevant sync types (all types if sync-independent)
3. Add to `requiredFields` only if it's mandatory for a sync type
4. Update `Validate()` with any constraints
5. Update `MarshalJSON()` — add the field to the per-sync-type struct literals
6. Update `UnmarshalJSON()` — add decoding and type validation
7. Add `"fieldname"` cases to `getSettingsValue()` and `setSettingsValue()` in `cmd/config.go`
8. Add tests in both `internal/config/config_test.go` and `cmd/config_cmd_test.go`

See [Optional Settings Omit Defaults](../cli/optional-settings-omit-defaults.rule.md) for conventions on optional fields.

## How to Add a New Command

1. Create `cmd/<name>.go` with a `newXxxCmd() *cobra.Command` function
2. Create `cmd/<name>_test.go` with table-driven tests
3. Register in `cmd/root.go` via `root.AddCommand(newXxxCmd())`
4. If the command needs the CLI version (like `update`), pass it from `NewRootCmd`: `newXxxCmd(cleaned)`
5. Keep cobra wiring minimal — extract logic into testable functions that accept a base directory

## How to Add a New Document Type

1. Add a `TypeXxx` constant in `templates/templates.go`
2. Add it to `categoryMap` with the correct virtual category
3. Ensure it is returned by `ValidTypes()` (derived from `categoryMap`)
4. Create a `generateXxxTemplate()` function
5. Add the case to `GenerateTemplate()` switch
6. Update the MCP server instructions in `internal/mcp/server.go` (`mcpServerInstructions` — document types list and the "WHEN TO CREATE" block)

Document types map to virtual categories:

| Category | Types |
|----------|-------|
| `knowledge` | adr, rfc, rule, guide, doc, spec |
| `vision` | prd, idea, plan, rnd, mrd, brd, urd, brs, strs, syrs, srs |
| `experience` | task-type, cpat |

Files follow the naming convention: `<slug>.<type>.md` (e.g., `use-postgres.adr.md`). The directory structure under `.archcore/` is free-form — see [Free-Form Directory Structure ADR](../dir/free-form-directory-structure.adr.md).

## How to Add a New MCP Tool

The MCP server (`archcore mcp`) exposes eleven tools: `init_project`, `list_documents`, `get_document`, `search_documents`, `create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`, `list_relations`, and `install_host_config` (conditionally registered — see below).

1. Create `internal/mcp/tools/<name>.go` returning `(mcp.Tool, server.ToolHandlerFunc)`
2. Create `internal/mcp/tools/<name>_test.go`
3. Register in `internal/mcp/server.go` via `s.AddTool()`
4. Use helpers from `common.go` (`ScanDocuments`, `ReadDocumentContent`, `ExtractDocType`, `splitDocument`)
5. Validate inputs and check path safety (no `..`, must resolve inside `.archcore/`)
6. Never expose absolute filesystem paths in error messages — see [No Absolute Paths in MCP Errors](../mcp/no-absolute-paths-in-mcp-errors.rule.md)

### Conditionally Registered Tools (executor injection)

A tool whose implementation lives in the cmd layer (because it reuses CLI installers) is registered via a `ServerOption` instead of unconditionally: the handler takes an executor function type defined in `internal/mcp/tools`, and `NewServer` adds the tool only when the option supplies one. `install_host_config` is the pattern's example — `cmd/mcp.go` passes `mcpserver.WithHostWiring(hostWiringExecutor(baseDir))`, where the executor adapts `internal/wiring.Apply` for the MCP boundary (project-relative paths, sanitized errors). This avoids a cmd→internal/mcp import cycle and guarantees headless/test servers built without the option do not expose the tool. See [install_host_config Tool Contract](../mcp/install-host-config-tool-contract.adr.md).

## How to Modify Hooks

Hooks intercept agent lifecycle events to inject documentation context at session start. Only the `SessionStart` event is active — `Stop` and `UserPromptSubmit`-family events were removed; see [Disable Stop and Prompt Hooks ADR](../integrations/disable-stop-and-prompt-hooks.adr.md).

### File Structure

| File | Purpose |
|------|---------|
| `internal/wiring/hooks_install.go` | Generic hook-config surgery: `installHookEvents`, the `archcore hooks ` ownership marker, entry classification (current / stale-archcore / foreign) |
| `internal/wiring/hooks_agents.go` | Per-agent installers (`InstallClaudeCodeHooks`, `InstallCursorHooks`, `InstallGeminiCLIHooks`, `InstallCopilotHooks`) + `InstallHooksForAgent()` router |
| `internal/wiring/wiring.go` | `Apply()` / `EnsureProjectInitialized()` — the shared host-wiring entry points |
| `cmd/hooks.go` | `hooks install` command wiring |
| `cmd/hooks_claude_code.go` | Claude Code subcommand, `hookInput`/`hookOutput` structs, `newSessionStartHookCmd` factory, `handleSessionStart`, SessionStart dedup stamps |
| `cmd/hooks_cursor.go` / `cmd/hooks_gemini_cli.go` / `cmd/hooks_copilot.go` | Per-agent hook subcommands (runtime side) |
| `cmd/hooks_common.go` | `buildSessionContext()` — the injected session-start text |

Install-time logic (writing host configs) lives in `internal/wiring`; runtime logic (handling a fired hook event) stays in `cmd/`. Installed commands are recognized across CLI versions by the `archcore hooks ` prefix — see [Hook Command Marker ADR](../integrations/hook-command-marker-prefix.adr.md).

### Shared Handler Pattern

All hook-supporting agents share `newSessionStartHookCmd(use, short, version)` from `cmd/hooks_claude_code.go`, which invokes `handleSessionStart()` that delegates to `buildSessionContext()`. Each agent registers its subcommand using this factory with its own event name and config format.

### Event Coverage by Agent

| Agent          | Config File                   | Event          |
|----------------|-------------------------------|----------------|
| Claude Code    | `.claude/settings.json`       | `SessionStart` |
| Cursor         | `.cursor/hooks.json`          | `sessionStart` |
| Gemini CLI     | `.gemini/settings.json`       | `SessionStart` |
| GitHub Copilot | `.github/hooks/archcore.json` | `sessionStart` |

### Modifying the Injected Context

`buildSessionContext()` in `cmd/hooks_common.go` builds the session-start text: header, existing documents grouped by virtual category, top tag frequencies, document-relation summary, and a pointer to the MCP server instructions. Changes here apply uniformly to every hook-supporting agent.

## How to Add a New Agent

To add support for a new AI coding agent:

1. **Define the agent ID** — Add a new `AgentID` constant in `internal/agents/agents.go`

2. **Create the agent file** — Add `internal/agents/<name>.go` returning an `*Agent` struct:
   - `ID`, `DisplayName` — identity
   - `DetectFn` — check for the agent's marker directory/file
   - `MCPConfigPath` — return the path to the MCP config file
   - `WriteMCPConfig` — write the archcore MCP entry (use `WriteStandardMCPJSON` if the agent uses standard `mcpServers` format)
   - `WriteHooksConfig` — set to `nil` if hooks not supported
   - `ManualMCPInstallHint` — set if MCP must be installed manually (e.g., Cline)
   - `InstructionsPath`, `WriteInstructions`, `RemoveInstructions` — the usage-nudge target. Point them at a shared helper in `internal/agents/instructions.go` (`agentsMDInstructions*` for `AGENTS.md`, `geminiInstructions*` for `GEMINI.md`, or an owned target like Claude's). `TestAllAgents_RequiredFields` fails if any are nil.

3. **Register the agent** — Add the constructor call to the `all` slice in `internal/agents/agents.go`

4. **Add tests** — Create `internal/agents/<name>_test.go` covering detection, MCP config writing, idempotency, and the instruction target

5. **If hooks are supported:**
   - Create `cmd/hooks_<name>.go` with a `newHooksXxxCmd()` subcommand using `newSessionStartHookCmd`
   - Create `cmd/hooks_<name>_test.go`
   - Add an `InstallXxxHooks()` function in `internal/wiring/hooks_agents.go` — the installed command MUST start with `archcore hooks ` (marker contract)
   - Register the subcommand in `cmd/hooks.go:newHooksCmd()`
   - Add the agent case to `hooksInstallers` in `internal/wiring/hooks_agents.go`

6. **Update documentation:**
   - Add the agent to [Supported AI Agents Registry](../integrations/supported-ai-agents.doc.md) (registry table + Instruction Nudge Files table)
   - Add config examples to [Agent Integration Guide](../integrations/agent-hooks-integration.guide.md)

## Key Design Patterns

- **Sync modes drive validation** — `none`/`cloud`/`on-prem` each require/forbid different settings fields. Custom JSON marshaling in `internal/config/` enforces this.
- **Constructor functions** — each command is `newXxxCmd() *cobra.Command` with logic extracted into testable functions.
- **Version-aware commands** — commands needing the CLI version (like `update`) receive it as a parameter from `NewRootCmd`.
- **Interactive forms** — `charmbracelet/huh` for interactive input, with flag-based fallbacks.
- **Co-located tests** — every command and package has adjacent `_test.go` files using `t.TempDir()` and table-driven subtests.
- **Shared session-start handler** — all hook-supporting agents use the same `handleSessionStart` and `buildSessionContext` via the `newSessionStartHookCmd` factory, differing only in event name and config format.
- **Host-wiring domain in `internal/wiring`** — install-time logic (hooks config surgery, per-agent installers, `Apply`/`EnsureProjectInitialized`, path helpers) is shared by `init --agent`, `hooks install`, `doctor --fix`, and the `install_host_config` MCP tool; cobra commands and MCP sanitization stay in `cmd/`.
- **Usage-nudge instruction files** — `archcore init` (opt-in) and `archcore instructions install`/`remove` write a discovery hint per agent into `AGENTS.md` / `GEMINI.md` / `.claude/rules/archcore.md`. Shared files use an idempotent fenced upsert that preserves user content; helpers live in `internal/agents/instructions.go`, the command in `cmd/instructions.go`. See [Usage-Nudge Instruction File per Agent](../integrations/instruction-nudge-on-init.adr.md).
- **Invalid config backup** — corrupted config files are backed up as `.bak` before being overwritten. See [Backup Invalid Configs](../integrations/backup-invalid-configs.adr.md).
