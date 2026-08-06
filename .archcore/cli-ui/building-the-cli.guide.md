---
title: "Building the Archcore CLI"
status: accepted
tags:
  - "cli-ui"
---

## Overview

Guide for building, testing, and extending the Archcore CLI — a Go tool that manages a local
`.archcore/` directory of structured documents and integrates with AI coding agents (Claude Code,
Cursor, Gemini CLI, Codex CLI, GitHub Copilot, and others) via MCP and hooks.

See also: [Supported AI Agents Registry](../integrations/supported-ai-agents.doc.md),
[CLI Hooks Reference](../integrations/cli-hooks-reference.doc.md),
[Hook Runtime Contract](../integrations/hook-runtime.spec.md),
[Agent Integration Guide](../integrations/agent-hooks-integration.guide.md).

## Prerequisites

- Go 1.25+, Git
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
./archcore doctor     # Health check: structure + settings + host wiring
./archcore status     # Structural checks only (naming, frontmatter, categories)
./archcore update     # Self-update to the latest release
```

`init` creates `.archcore/` with a free-form directory structure — documents are organized by
domain/feature/team, and category is derived from the filename suffix (`slug.type.md`). Settings go in
`.archcore/settings.json`. It also auto-detects AI agents, installs hooks and MCP config for all found
agents, reports whether each host can actually run its hooks, then offers (opt-in) to write a
usage-nudge instruction file per agent.

## Command Surface

The public surface is nine commands: `init`, `config`, `doctor`, `status`, `hooks`, `mcp`,
`instructions`, `sync`, `update`. Hook event handlers are hidden leaves under `hooks`, not new
commands — they are a protocol surface, not a user surface.

## Settings and Configuration

Settings live in `.archcore/settings.json`. Use the `config` command to read and modify them:

```bash
./archcore config get sync         # Read a setting
./archcore config set language ru  # Set a setting
```

### Settings Fields

| Field           | Type   | Description                                                          |
|-----------------|--------|----------------------------------------------------------------------|
| `sync`          | string | Sync mode: `none`, `cloud`, or `on-prem`                            |
| `project_id`    | int    | Project ID for cloud/on-prem sync (optional)                         |
| `archcore_url`  | string | Server URL for on-prem sync (required for on-prem)                   |
| `language`      | string | Language for MCP-generated document content (default: `"en"`)        |
| `globals`       | array  | Declared global sources, read-only, allowed in every sync mode       |
| `codeAlignment` | object | Pre-write context injection tuning; holds `sourceRoots` (array)      |

`language` controls the language the MCP server uses when generating document content. It is
sync-independent. `codeAlignment.sourceRoots` replaces the built-in list of directories treated as
source code by the `PreToolUse` injection; the CLI never writes this key itself.

See [Sync Mode Field Validation](../sync/sync-mode-field-validation.rule.md) for per-sync-mode field
rules, [Optional Settings Omit Defaults](../cli/optional-settings-omit-defaults.rule.md) for the
convention on optional fields, and
[Forward-Compatible Settings Parsing](../cli/forward-compatible-settings-parsing.rule.md) for how an
older binary handles a field a newer release added.

## How to Add a New Setting

1. Add the field to the `Settings` struct in `@internal/config/config.go` with a
   `json:"...,omitempty"` tag
2. Add the field name to `allowedFields` for the relevant sync types (all types if sync-independent)
3. Add to `requiredFields` only if it is mandatory for a sync type
4. Update `Validate()` with any constraints
5. Update `MarshalJSON()` — add the field to **all three** per-sync-mode struct literals. A field
   added to one is silently dropped when the project switches sync mode.
6. Update `UnmarshalJSON()` — add decoding and type validation
7. Add `"fieldname"` cases to `getSettingsValue()` and `setSettingsValue()` in `@cmd/config.go`
8. Add tests in both `internal/config/config_test.go` and `cmd/config_cmd_test.go`
9. Ship the release no earlier than the release that made the parser tolerant of unknown fields — see
   the release guide's sequencing section

## How to Add a New Command

1. Create `cmd/<name>.go` with a `newXxxCmd() *cobra.Command` function
2. Create `cmd/<name>_test.go` with table-driven tests
3. Register in `cmd/root.go` via `root.AddCommand(newXxxCmd())`
4. If the command needs the CLI version (like `update`), pass it from `NewRootCmd`: `newXxxCmd(cleaned)`
5. Keep cobra wiring minimal — extract logic into testable functions that accept a base directory

A command that prints a report separates the report from the printer: `collectStatus` builds a
`statusReport` as data, and `writeTo` renders it. The hook path needs the data form, because on a hook
stdout carries the host protocol and nothing may print to it. See `@cmd/status_report.go`.

## How to Add a New Document Type

1. Add a `TypeXxx` constant in `@templates/templates.go`
2. Add it to `categoryMap` with the correct virtual category
3. Ensure it is returned by `ValidTypes()` (derived from `categoryMap`)
4. Create a `generateXxxTemplate()` function
5. Add the case to `GenerateTemplate()` switch
6. Update the MCP server instructions in `@internal/mcp/server.go` (`mcpServerInstructions` — document
   types list and the "WHEN TO CREATE" block)
7. If the type has required sections, add them to `RequiredSections` in `@templates/precision.go`, so
   the post-write precision check measures against the same contract

Document types map to virtual categories:

| Category | Types |
|----------|-------|
| `knowledge` | adr, rfc, rule, guide, doc, spec |
| `vision` | prd, idea, plan, rnd, mrd, brd, urd, brs, strs, syrs, srs |
| `experience` | task-type, cpat |

Files follow the naming convention: `<slug>.<type>.md` (e.g., `use-postgres.adr.md`). The directory
structure under `.archcore/` is free-form — see
[Free-Form Directory Structure ADR](../dir/free-form-directory-structure.adr.md).

## The Document Model — `internal/docs`

`internal/docs` owns the document model, the scan, the per-file cache, and the path guards. The MCP
tools, the hook handlers, and the status report all read documents through it. See the ADR on
[internal/docs owning the document model](../cli/docs-package-owns-the-document-model.adr.md).

| File | Contents |
|------|----------|
| `@internal/docs/document.go` | `Document`, `EnrichedDocument`, `DocumentRelation`, `ReadDocumentContent`, `NormalizeRelPath`, `WriteFileAtomic` |
| `@internal/docs/scan.go` | `Scan`, `ScanFull`, `ScanLocal`, `BuildDoc` |
| `@internal/docs/cache.go` | mtime-and-size-keyed per-file cache, `InvalidateCache` |
| `@internal/docs/guard.go` | `GuardWritablePath`, `ValidateReadPath`, `ValidateArchcorePath`, `CheckSymlinkContainment` |
| `@internal/docs/globals.go` | `IsGlobalPath`, `IsReservedGlobalDir`, `IsReadOnlyGlobalPath`, `AnnotateSource` |
| `@internal/docs/inspect.go` | `InspectGlobals`, `GlobalState`, `GlobalInspection` |
| `@internal/mcp/tools/docs_bridge.go` | The seam. Aliases `LocalDocument` to `docs.Document` and re-exports the helpers under the short names the tool handlers call. |

`LocalDocument` stays the name on the MCP wire; `docs.Document` is the domain name. The alias keeps
both true with no conversion layer.

## How to Add a New MCP Tool

The MCP server (`archcore mcp`) exposes eleven tools: `init_project`, `list_documents`,
`get_document`, `search_documents`, `create_document`, `update_document`, `remove_document`,
`add_relation`, `remove_relation`, `list_relations`, and `install_host_config` (conditionally
registered — see below). The server declares no prompts capability; see the ADR on removing the MCP
track prompts.

1. Create `internal/mcp/tools/<name>.go` returning `(mcp.Tool, server.ToolHandlerFunc)`
2. Create `internal/mcp/tools/<name>_test.go`
3. Register in `@internal/mcp/server.go` via `s.AddTool()`
4. Use the helpers re-exported by `@internal/mcp/tools/docs_bridge.go` (`ScanDocuments`,
   `ReadDocumentContent`, `guardWritablePath`, `validateReadPath`)
5. Validate inputs and check path safety (no `..`, must resolve inside `.archcore/`)
6. Never expose absolute filesystem paths in error messages — see
   [No Absolute Paths in MCP Errors](../mcp/no-absolute-paths-in-mcp-errors.rule.md)

### Conditionally Registered Tools (executor injection)

A tool whose implementation lives in the cmd layer (because it reuses CLI installers) is registered via
a `ServerOption` instead of unconditionally: the handler takes an executor function type defined in
`internal/mcp/tools`, and `NewServer` adds the tool only when the option supplies one.
`install_host_config` is the pattern's example — `@cmd/mcp.go` passes
`mcpserver.WithHostWiring(hostWiringExecutor(baseDir))`, where the executor adapts
`internal/wiring.Apply` for the MCP boundary (project-relative paths, sanitized errors). This avoids a
cmd→internal/mcp import cycle and guarantees headless/test servers built without the option do not
expose the tool. See [install_host_config Tool Contract](../mcp/install-host-config-tool-contract.adr.md).

## How to Modify Hooks

Hooks intercept host lifecycle events. Three events are active — `SessionStart`, `PreToolUse`, and
`PostToolUse` — in each host's own spelling. The `Stop` and `UserPromptSubmit` families stay
unsupported; see the [ADR on removing them](../integrations/disable-stop-and-prompt-hooks.adr.md).

The normative behavior of the runtime side is the
[Hook Runtime Contract](../integrations/hook-runtime.spec.md). Change it before changing the code.

### File Structure

Install-time logic (writing host configs) lives in `internal/wiring`; runtime logic (handling a fired
event) stays in `cmd/`. There is no per-agent runtime file — a host is one row in a dialect table.

| File | Purpose |
|------|---------|
| `@internal/wiring/hooks_install.go` | Generic hook-config surgery: `installHookEvents`, the `archcore hooks ` ownership marker, entry classification (current / stale-archcore / foreign) |
| `@internal/wiring/hooks_agents.go` | Per-host installers and the event tables (names, matchers, timeouts, entry shapes) + the `hooksInstallers` router |
| `@internal/wiring/hooks_effective.go` | `EffectiveHookNotes` — whether a host will read what was written |
| `@internal/wiring/wiring.go` | `Apply()` / `EnsureProjectInitialized()` — shared host-wiring entry points |
| `@cmd/hooks.go` | `hooks install` command wiring |
| `@cmd/hook_dialect.go` | `hostDialect` table: session shape, context envelope, deny style, pre-write context support; `emitDecision` |
| `@cmd/hook_command.go` | Command tree, `hookHandler` type, `safeHandle` panic recovery, per-event safety rules |
| `@cmd/hook_payload.go` | Payload decoding by explicit key paths, MCP tool-name folding |
| `@cmd/hook_session_start.go` | SessionStart response shapes and the dedup wrapper |
| `@cmd/hooks_common.go` | `buildSessionContext()` — the injected session-start text |
| `@cmd/hook_write_guard.go` | The one blocking guard |
| `@cmd/hook_code_alignment.go` | Pre-write context injection |
| `@cmd/hook_post_tool_use.go` | Post-write dispatcher: validation, cascade |
| `@cmd/hook_precision.go` | Post-write precision findings; canon in `@templates/precision.go` |
| `@cmd/hook_staleness.go` | Drift advisory, rate-limited to 24 hours |
| `@cmd/hook_stamp.go` | Dedup stamp claim, one directory per scope |

Installed commands are recognized across CLI versions by the `archcore hooks ` prefix — see
[Hook Command Marker ADR](../integrations/hook-command-marker-prefix.adr.md).

### Safety Rules

These are the rules a change to the hook path must not break. The runtime contract states them
normatively.

- A handler returns a decision; the command layer owns output and exit codes.
- The zero-value decision allows and writes nothing. Every failure path produces it.
- `safeHandle` converts a panic into an allow. Without it a defect would exit non-zero, which several
  hosts read as an explicit deny.
- The write guard runs first and alone. Advisory work happens after the verdict.
- On Copilot, stdout carries exactly one JSON document. Every diagnostic goes to stderr.
- An unknown host or event writes an empty stdout and exits 0.

### Modifying the Injected Context

`buildSessionContext()` in `@cmd/hooks_common.go` builds the session-start text. Its output contract —
section order, the 24-line document budget, the `rejected` exclusion, the tag cap — is the
[SessionStart Context Output Contract](../integrations/session-start-context.spec.md). Changes here
apply uniformly to every host.

## How to Add a New Agent

1. **Define the agent ID** — Add a new `AgentID` constant in `@internal/agents/agents.go`

2. **Create the agent file** — Add `internal/agents/<name>.go` returning an `*Agent` struct:
   - `ID`, `DisplayName` — identity
   - `DetectFn` — check for the agent's marker directory/file
   - `MCPConfigPath` — return the path to the MCP config file
   - `WriteMCPConfig` — write the archcore MCP entry (use `WriteStandardMCPJSON` if the agent uses the
     standard `mcpServers` format)
   - `WriteHooksConfig` — set to `nil` if hooks are not supported
   - `ManualMCPInstallHint` — set if MCP must be installed manually (e.g., Cline)
   - `InstructionsPath`, `WriteInstructions`, `RemoveInstructions` — the usage-nudge target. Point them
     at a shared helper in `@internal/agents/instructions.go`. `TestAllAgents_RequiredFields` fails if
     any are nil.

3. **Register the agent** — Add the constructor call to the `all` slice in `@internal/agents/agents.go`

4. **Add tests** — Create `internal/agents/<name>_test.go` covering detection, MCP config writing,
   idempotency, and the instruction target

5. **If hooks are supported:**
   - Add a `hostDialect` row in `@cmd/hook_dialect.go`. The three hidden event leaves are generated
     from it; no per-agent runtime file is needed.
   - Add an `InstallXxxHooks()` function in `@internal/wiring/hooks_agents.go` covering all three
     events — the installed command MUST start with `archcore hooks ` (marker contract)
   - Add the agent to `hooksInstallers` in the same file
   - If the host can accept the config and still not run it, add its case to `EffectiveHookNotes` in
     `@internal/wiring/hooks_effective.go`

6. **Update documentation:**
   - Add the agent to [Supported AI Agents Registry](../integrations/supported-ai-agents.doc.md)
   - Add the event rows to [CLI Hooks Reference](../integrations/cli-hooks-reference.doc.md)
   - Add config examples to [Agent Integration Guide](../integrations/agent-hooks-integration.guide.md)

## Key Design Patterns

- **Sync modes drive validation** — `none`/`cloud`/`on-prem` each require or forbid different settings
  fields. Custom JSON marshaling in `internal/config/` enforces this, and unknown fields are carried
  through rather than rejected.
- **Constructor functions** — each command is `newXxxCmd() *cobra.Command` with logic extracted into
  testable functions.
- **Report as data, printer as a thin layer** — `collectStatus` returns a `statusReport`; `writeTo`
  renders it. The hook path consumes the data form because it cannot print.
- **One document model** — `internal/docs` owns the scan, the cache, and the path guards. The write
  guard and the MCP write tools call the same predicate, so a path MCP refuses cannot be reached by
  editing the file directly.
- **One hook entry per (host, event)** — the process dispatches by tool name internally, so three
  post-write checks cost one process start.
- **Dialect table, not per-host code** — a host is a row describing its protocol; the handlers are
  shared.
- **Fail open everywhere except the write guard** — an internal defect in a hook must cost a missing
  hint, never a blocked edit.
- **Report effective state** — after writing a host config, say whether that host will read it. See
  [Report Effective Hook State](../integrations/report-effective-hook-state.rule.md).
- **Host-wiring domain in `internal/wiring`** — install-time logic is shared by `init --agent`,
  `hooks install`, `doctor --fix`, and the `install_host_config` MCP tool; cobra commands and MCP
  sanitization stay in `cmd/`.
- **Usage-nudge instruction files** — `archcore init` (opt-in) and `archcore instructions
  install`/`remove` write a discovery hint per agent into `CLAUDE.md` / `AGENTS.md` / `GEMINI.md`. All
  targets use an idempotent fenced upsert that preserves user content.
- **Co-located tests** — every command and package has adjacent `_test.go` files using `t.TempDir()`
  and table-driven subtests.
- **Invalid config backup** — corrupted config files are backed up as `.bak` before being overwritten.
  See [Backup Invalid Configs](../integrations/backup-invalid-configs.adr.md).
