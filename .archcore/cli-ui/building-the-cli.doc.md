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

This document holds the procedures. For the structure of the system — the surfaces, the package
tiers, the state stores — read [Archcore CLI System Overview](../architecture/system-overview.doc.md)
first.

See also: [Supported AI Agents Registry](../integrations/supported-ai-agents.doc.md),
[CLI Hooks Reference](../integrations/cli-hooks-reference.doc.md),
[Hook Runtime Contract](../integrations/hook-runtime.spec.md),
[Hook Wire Protocol Per Host](../integrations/hook-wire-protocol.spec.md),
[Hook Payload Reading](../integrations/hook-payload-reading.spec.md),
[Agent Integration Reference](../integrations/agent-hooks-integration.doc.md).

## Prerequisites

- Go 1.25+, Git
- Familiarity with Go modules, packages, and testing

## Build & Test

```bash
go build -o archcore .     # Build binary
go test ./...              # Run all tests
go test ./cmd/ -run TestX  # Run a specific test
golangci-lint run ./...    # The analyzers CI runs; config in .golangci.yml
```

## Getting Started

```bash
./archcore init           # Interactive setup wizard (directories, agent detection)
./archcore doctor         # Health check: structure + settings + host wiring
./archcore status         # Structural checks only (naming, frontmatter, categories)
./archcore update         # Self-update to the latest release
./archcore plugin status  # Report the Archcore plugin per host
```

`init` creates `.archcore/` with a free-form directory structure — documents are organized by
domain/feature/team, and category is derived from the filename suffix (`slug.type.md`). Settings go in
`.archcore/settings.json`. It also auto-detects AI agents, installs hooks and MCP config for all found
agents, reports whether each host can actually run its hooks, then offers (opt-in) to write a
usage-nudge instruction file per agent. A host the user checks in the agent picker also gets the
Archcore plugin installed; a host detected without a picker does not, because a detection is not a
consent.

## Command Surface

The public surface is ten commands: `init`, `config`, `doctor`, `status`, `hooks`, `mcp`,
`instructions`, `plugin`, `sync`, `update` — registered in one `root.AddCommand` call at
`@cmd/root.go`. Hook event handlers are hidden leaves under `hooks`, not new commands — they are a
protocol surface, not a user surface.

`plugin` installs, updates, removes, and reports the Archcore plugin per host. It shares one planner
and one executor with the plugin step of `archcore update` and the delivery step of `archcore init`;
the three entry points differ only in which actions they select and how they word their output. See
[Plugin Delivery](../integrations/plugin-delivery.spec.md).

Two commands carry update behavior beyond their own surface. `update` runs the plugin step after its
binary phase. `mcp` starts one unattended update attempt on a background goroutine 60 s after it
begins serving; that attempt never writes to stdout, which the JSON-RPC stream owns. See
[How archcore update Works](../update/self-update-command.doc.md).

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
source code by the `PreToolUse` injection; the CLI never writes this key itself. See
[The Advisory Subsystem](../architecture/advisory-subsystem.doc.md) for what that injection does with
it.

See [Sync Mode Field Validation](../sync/sync-mode-field-validation.rule.md) for per-sync-mode field
rules, [Optional Settings Omit Defaults](../cli/optional-settings-omit-defaults.rule.md) for the
convention on optional fields, and
[Forward-Compatible Settings Parsing](../cli/forward-compatible-settings-parsing.rule.md) for how an
older binary handles a field a newer release added.

A hook reads this file on the pre-write path, so its size is on a latency budget — see the runtime
contract's constraint on reading project state once per invocation.

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
6. If the command reads or writes `.archcore/`, resolve its root with
   `resolveProjectRoot(projectFlag, os.Getenv("ARCHCORE_PROJECT_ROOT"))` and register a `--project`
   flag whose help text names the environment variable. Never call `os.Getwd()` directly:
   `resolveProjectRoot` also refuses a root inside a host's plugin install cache, which hosts have
   been observed spawning agents into (host-cwd-misrouting.adr).
   `TestCommands_OfferProjectFlag` walks the tree and fails on a command that does neither.
7. If the command has verbs, bind the shared flags once and apply that binding to the group and to
   every verb, so `archcore x --flag verb` and `archcore x verb --flag` are one invocation. Give the
   group and every verb `cobra.NoArgs`: a group without it answers a misspelled verb with usage text
   on stdout and exit 0.
8. Choose the stdout handle by what the output is — a self-contained report goes through
   `cmd.OutOrStdout()`, an interleaved run goes through `os.Stdout`. The go-code-quality rule states
   the criterion.

A command that prints a report separates the report from the printer: `collectStatus` builds a
`statusReport` as data, and `writeTo` renders it. The hook path needs the data form, because on a hook
stdout carries the host protocol and nothing may print to it. See `@cmd/status_report.go`.

## How to Add a New Document Type

1. Add a `TypeXxx` constant in `@templates/templates.go`
2. Add it to `categoryMap` with the correct virtual category
3. Add it to `ValidTypes()` in the same file. It is a hand-written list, not derived —
   `TestValidTypes_Completeness` holds its length against `categoryMap`, so a type added to one and
   not the other fails there.
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
| `@internal/docs/scan.go` | `Scan`, `ScanFull`, `ScanTypes`, `ScanLocal`, `ScanLocalTypes`, `ScanCount` |
| `@internal/docs/relpath.go` | `RelativeToBase` — a host-supplied path to a baseDir-relative slash path |
| `@internal/docs/cache.go` | mtime-and-size-keyed per-file cache, `InvalidateCache`, `ResetCache` |
| `@internal/docs/guard.go` | `GuardWritablePath`, `ValidateReadPath`, `ValidateArchcorePath`, and the package-private `checkSymlinkContainment` they layer |
| `@internal/docs/globals.go` | `IsGlobalPath`, `IsReservedGlobalDir`, `IsExternalGlobalDocument`, `AnnotateSource` |
| `@internal/docs/inspect.go` | `InspectGlobals`, `GlobalState`, `GlobalInspection` |
| `@internal/mcp/tools/docs_bridge.go` | The seam. Aliases `LocalDocument` to `docs.Document` and re-exports the helpers under the short names the tool handlers call. |

`LocalDocument` stays the name on the MCP wire; `docs.Document` is the domain name. The alias keeps
both true with no conversion layer. The bridge's re-exports are package-private (`scanDocuments`,
`scanDocumentsFull`, `readDocumentContent`, `guardWritablePath`, `validateReadPath`,
`annotateSource`); a caller outside `internal/mcp/tools` uses the exported `docs.*` name instead.

The guards return classified sentinels rather than messages, so the MCP tools and the hook render the
same verdict in their own words — see
[A Shared Guard Returns Classified Sentinels](../code-quality/shared-guards-return-classified-sentinels.rule.md).

## How to Add a New MCP Tool

The MCP server (`archcore mcp`) exposes eleven tools: `init_project`, `list_documents`,
`get_document`, `search_documents`, `create_document`, `update_document`, `remove_document`,
`add_relation`, `remove_relation`, `list_relations`, and `install_host_config` (conditionally
registered — see below). The server declares no prompts capability; see the ADR on removing the MCP
track prompts.

The file shape is normative and is stated once, in
[The Shape of an MCP Tool File](../code-quality/the-shape-of-an-mcp-tool-file.rule.md). Read it before
step 1; the steps below are the surrounding wiring, not the shape.

1. Create `internal/mcp/tools/<name>.go` exporting the pair `New<Name>Tool() mcp.Tool` and
   `Handle<Name>(root RootProvider) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)`
2. Resolve the project root inside the returned closure with `root.Root(ctx)`. Never capture a
   `baseDir` at construction: a session that moves into a git worktree moves the server's root with
   it (project-root-resolution.spec §3), and a captured root silently serves the old project.
3. Create `internal/mcp/tools/<name>_test.go`, using `StaticRoot(base)` as the provider
4. Register in `@internal/mcp/server.go` via `s.AddTool()`
5. Use the helpers re-exported by `@internal/mcp/tools/docs_bridge.go` (`scanDocuments`,
   `scanDocumentsFull`, `readDocumentContent`, `guardWritablePath`, `validateReadPath`)
6. Validate inputs and check path safety (no `..`, must resolve inside `.archcore/`)
7. Return a user-caused refusal as `errorResult(...)`; return a Go error only for the process's own
   failure, such as marshalling the result
8. Never expose absolute filesystem paths in error messages — see
   [No Absolute Paths in MCP Errors](../mcp/no-absolute-paths-in-mcp-errors.rule.md)
9. Add the tool name to `archcoreMCPTools` in `@cmd/hook_payload.go`.
   `TestArchcoreMCPTools_MatchesTheServer` fails otherwise. Three hosts flatten an MCP tool name by
   joining the server name to the tool name with one separator, and that separator also occurs inside
   names, so the fold accepts only a tool this list knows. A tool missing from it is read by the write
   guard as a direct edit and denied on those hosts.
10. If the tool mutates the knowledge base, add it to `mutatingMCPTools` in the same file and to
    `mcpDocumentTools` in `@internal/wiring/hooks_agents.go`, which is the host-side matcher that
    decides whether the post-write process starts at all
11. If the tool crosses another tool's state — one handler writes, another reads — add the scenario to
    `internal/mcp/integration/` rather than to the per-tool unit file
    (in-process-mcp-integration-tests.adr)

### Conditionally Registered Tools (executor injection)

A tool whose implementation lives in the cmd layer (because it reuses CLI installers) is registered via
a `ServerOption` instead of unconditionally: the handler takes an executor function type defined in
`internal/mcp/tools`, and `NewServer` adds the tool only when the option supplies one.
`install_host_config` is the pattern's example — `@cmd/mcp.go` passes
`mcpserver.WithHostWiring(hostWiringExecutor())`. The executor takes no `baseDir`: the root arrives
per call from the server's root provider, so the wiring lands under the project root the session is on
now (project-root-resolution.spec §3). It adapts `internal/wiring.Apply` for the MCP boundary —
project-relative paths, sanitized errors. This avoids a cmd→internal/mcp import cycle and guarantees
headless/test servers built without the option do not expose the tool. See
[install_host_config Tool Contract](../mcp/install-host-config-tool-contract.adr.md).

The same option seam carries the background update task: `WithBackgroundTask(func(ctx))` takes an
opaque function that `RunStdio` starts between the stdout shield and `Listen`. The function is opaque
so `internal/mcp` never links the update stack; `TestPackage_DoesNotLinkTheUpdateStack` walks the
import graph and fails if it ever does. Both seams are obligations, not conveniences — see
[Package Dependency Direction](../architecture/package-dependency-direction.rule.md).

## How to Modify Hooks

Hooks intercept host lifecycle events. Three events are active — `SessionStart`, `PreToolUse`, and
`PostToolUse` — in each host's own spelling. The `Stop` and `UserPromptSubmit` families stay
unsupported; see the [ADR on removing them](../integrations/disable-stop-and-prompt-hooks.adr.md).

Three specs are normative here, split by what they own, and the one covering the part you are
changing comes before the code:

- [Hook Runtime Contract](../integrations/hook-runtime.spec.md) — which guard blocks, in what order
  the work runs, and how the command degrades (`@cmd/hook_command.go`, `@cmd/hook_write_guard.go`).
- [Hook Wire Protocol Per Host](../integrations/hook-wire-protocol.spec.md) — what the process writes
  and how it exits, per dialect (`@cmd/hook_dialect.go`).
- [Hook Payload Reading](../integrations/hook-payload-reading.spec.md) — how the payload becomes a
  tool identity and a set of targets (`@cmd/hook_payload.go`).

### File Structure

Install-time logic (writing host configs) lives in `internal/wiring`; runtime logic (handling a fired
event) stays in `cmd/`. There is no per-agent runtime file — a host is one row in a dialect table.
The advisory work the hooks call lives in `internal/advisory/`, not in `cmd/`.

| File | Purpose |
|------|---------|
| `@internal/wiring/hooks_install.go` | Generic hook-config surgery: `installHookEvents`, the `archcore hooks ` ownership marker, entry classification (current / stale-archcore / foreign) |
| `@internal/wiring/hooks_agents.go` | Per-host installers and the event tables (names, matchers, timeouts, entry shapes) + the `hooksInstallers` router |
| `@internal/wiring/hooks_effective.go` | `EffectiveHookNotes` — whether a host will read what was written |
| `@internal/wiring/wiring.go` | `Apply()` / `EnsureProjectInitialized()` — shared host-wiring entry points |
| `@cmd/hooks.go` | `hooks install` command wiring; `servesHookEvents` and `noHookWiringNote`, which separate a hookless agent from one whose hooks load as plugin code |
| `@cmd/hook_dialect.go` | `hostDialect` table: session shape, context envelope, deny style, pre-write context support; `emitDecision` |
| `@cmd/hook_command.go` | Command tree, `hookHandler` type, `safeHandle` panic recovery, per-event safety rules |
| `@cmd/hook_payload.go` | Payload decoding by explicit key paths, MCP tool-name folding, apply-patch target extraction |
| `@cmd/hook_session_start.go` | SessionStart response shapes and the dedup wrapper |
| `@cmd/hooks_common.go` | `buildSessionContext()` — the injected session-start text |
| `@cmd/hook_write_guard.go` | The one blocking guard, and the per-invocation `writeGuard` that caches what its verdicts read |
| `@cmd/hook_post_tool_use.go` | Post-write dispatcher: validation, cascade |
| `@internal/advisory/` | The four advisory engines: `code_alignment.go`, `precision.go`, `restatement.go`, `staleness.go`. See [The Advisory Subsystem](../architecture/advisory-subsystem.doc.md) for their triggers, settings, and bounds. |
| `@internal/stamp/` | Dedup stamp claim, one directory per scope |

Installed commands are recognized across CLI versions by the `archcore hooks ` prefix — see
[Hook Command Marker ADR](../integrations/hook-command-marker-prefix.adr.md).

The dialect table answers which hosts this binary serves; `hooksInstallers` answers which hosts get a
config written. They are not the same set, and the difference is what `hooks install` reports: a host
in the first and not the second loads its hooks as plugin code.

### Safety Rules

These are the rules a change to the hook path must not break. The specs above state them normatively.

- A handler returns a decision; the command layer owns output and exit codes.
- The zero-value decision allows and writes nothing. Every failure path produces it.
- `safeHandle` converts a panic into an allow. Without it a defect would exit non-zero, which several
  hosts read as an explicit deny.
- The write guard runs first and alone. Advisory work happens after the verdict.
- The write guard reads the globals list fail-closed for a path outside the project, and fail-open for
  a path inside `.archcore/` — otherwise an unreadable `settings.json` would block the edit that
  repairs it. See
  [Fail-Open or Fail-Closed Reads](../code-quality/fail-open-or-fail-closed-reads.rule.md).
- On Copilot, stdout carries exactly one JSON document. Every diagnostic goes to stderr.
- An unknown host or event writes an empty stdout and exits 0.
- A file-mutation tool that names no path still has a target. `apply_patch` names its files inside the
  patch body, and the guard reads them; a host whose registry replaces `write` and `edit` with it
  would otherwise run unguarded and look no different from a guarded session.
- The pre-write path runs inside a one-second host budget. State a verdict reads is read once per
  invocation and only when a verdict needs it.
- The background update trigger is not added here. A hook leaf is short-lived, runs on a budget of
  seconds, and its stdout is the host's protocol channel.

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
   - Add the agent to `hooksInstallers` in the same file. Skip this step only when the host loads
     hooks as plugin code and there is no declarative file to write; the dialect row alone then makes
     `hooks install` describe it correctly instead of calling it hookless.
   - If the host flattens MCP tool names into a spelling not already folded, add its prefix in
     `@cmd/hook_payload.go`. An unfolded spelling makes the write guard deny the host's own document
     tools.
   - If the host names a file-mutation tool that carries no path argument, add the key its patch or
     path arrives under in the same file. A missing key fails silently: no target, allow, and an
     unprotected session that looks clean.
   - If the host can accept the config and still not run it, add its case to `EffectiveHookNotes` in
     `@internal/wiring/hooks_effective.go`

6. **If the host ships an Archcore plugin:** add its row to the host table in `@internal/plugin/hosts.go`
   — the CLI name, the read-only listing command, the on-disk registry path, and the install, update,
   and remove commands. A host with no CLI mechanism carries a UI note instead. Nothing else changes:
   the planner reads the row, and the three entry points share it.

7. **Add the host CLI to the test trap** — `hostCLIs` in `@internal/testsupport/isolate.go`, so a test
   that reaches the real binary is caught instead of changing the developer's machine. See
   [Isolating the Developer's Machine](../code-quality/isolating-the-machine-from-the-test-suite.guide.md).

8. **Update documentation:**
   - Add the agent to [Supported AI Agents Registry](../integrations/supported-ai-agents.doc.md)
   - Add the event rows to [CLI Hooks Reference](../integrations/cli-hooks-reference.doc.md)
   - Add config examples to [Agent Integration Reference](../integrations/agent-hooks-integration.doc.md)

A host is therefore described by five registries in four packages. That split is deliberate — see the
accepted trade-offs in [Archcore CLI System Overview](../architecture/system-overview.doc.md).

## Key Design Patterns

- **Sync modes drive validation** — `none`/`cloud`/`on-prem` each require or forbid different settings
  fields. Custom JSON marshaling in `internal/config/` enforces this, and unknown fields are carried
  through rather than rejected.
- **Constructor functions** — each command is `newXxxCmd() *cobra.Command` with logic extracted into
  testable functions.
- **One root resolution** — every command reading the store goes through `resolveProjectRoot`. See
  the rule in "How to Add a New Command"; the plugin-cache refusal is the part that matters.
- **Report as data, printer as a thin layer** — `collectStatus` returns a `statusReport`; `writeTo`
  renders it. The hook path consumes the data form because it cannot print.
- **One document model** — `internal/docs` owns the scan, the cache, and the path guards. The write
  guard and the MCP write tools call the same predicate, so a path MCP refuses cannot be reached by
  editing the file directly.
- **Layered imports** — a package imports its own tier or a lower one, never a surface. See
  [Package Dependency Direction](../architecture/package-dependency-direction.rule.md).
- **Rendering at the boundary** — a domain package returns absolute paths and raw errors; the terminal
  and MCP boundaries each transform them. See
  [Rendering Happens at the Boundary](../code-quality/rendering-happens-at-the-boundary.rule.md).
- **Pure planner, one executor** — `internal/plugin` decides from host evidence and acts in one place.
  Silence for a host without the plugin is structural: it falls out of the evidence, and no code path
  reads a host's error text.
- **Guard or advisory, never both** — every read of external state declares which it serves, and the
  two fail in opposite directions. See
  [Fail-Open or Fail-Closed Reads](../code-quality/fail-open-or-fail-closed-reads.rule.md).
- **Bounded, ordered output** — anything leaving the process is capped by a named constant and sorted
  before it is cut. See
  [Bounded and Deterministic Output](../code-quality/bounded-and-deterministic-output.rule.md).
- **Comments are the exception** — the code carries the meaning; a comment records a workaround or a
  reason the code cannot hold. See
  [Code Carries No Comment Unless the Code Cannot Speak](../code-quality/comments-are-the-exception.rule.md).
- **Platform variance is a file split** — a `//go:build` variant per platform, never a `runtime.GOOS`
  branch. See
  [Platform Splits Are Files](../code-quality/platform-splits-are-files.rule.md).
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
  and table-driven subtests. A package whose code reaches `$HOME`, an XDG state directory, a host CLI,
  or git arms the isolation in `TestMain`; without it the suite mutates the developer's own machine
  and stays green while doing it.
- **Invalid config backup** — corrupted config files are backed up as `.bak` before being overwritten.
  See [Backup Invalid Configs](../integrations/backup-invalid-configs.adr.md).
