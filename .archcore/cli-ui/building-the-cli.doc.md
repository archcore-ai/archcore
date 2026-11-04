---
title: "Building the Archcore CLI"
status: accepted
tags:
  - "cli-ui"
  - "component:cli"
---

## Overview

Run the CLI commands in this document from `cli/`. Source references beginning with `@cli/` resolve from the monorepo root; project documents live in its single `.archcore/`.

This document holds the procedures for extending the Archcore CLI. For the structure of the system — the surfaces, the package
tiers, the state stores — read [Archcore CLI System Overview](../architecture/system-overview.doc.md)
first. Build and test commands are in the `Makefile` and `CLAUDE.md`.

See also: [Supported AI Agents Registry](../integrations/supported-ai-agents.doc.md),
[Hook Runtime Contract](../integrations/hook-runtime.spec.md),
[Hook Wire Protocol Per Host](../integrations/hook-wire-protocol.spec.md),
[Hook Payload Reading](../integrations/hook-payload-reading.spec.md),
[Agent Integration Reference](../integrations/agent-hooks-integration.doc.md).

## Command Surface

The public surface is the set of commands registered in one `root.AddCommand` call at
`@cli/cmd/root.go`. Hook event handlers are hidden leaves under `hooks`, not new commands — they are a
protocol surface, not a user surface.

`plugin` installs, updates, removes, and reports the Archcore plugin per host. It shares one planner
and one executor with the plugin step of `archcore update` and the delivery step of `archcore init`;
the three entry points differ only in which actions they select and how they word their output. See
[Plugin Delivery](../integrations/plugin-delivery.spec.md).

Two commands carry update behavior beyond their own surface. `update` runs the plugin step after its
binary phase. `mcp` starts one unattended update attempt on a background goroutine 60 s after it
begins serving; that attempt never writes to stdout, which the JSON-RPC stream owns. See
[How archcore update Works](../update/self-update-command.doc.md).

## Settings

Settings live in `.archcore/settings.json`; read and modify them with `archcore config get|set`. `codeAlignment.sourceRoots` replaces the built-in list of directories treated as
source code by the `PreToolUse` injection; the CLI never writes this key itself. See
[The Advisory Subsystem](../architecture/advisory-subsystem.doc.md).

See `sync/sync-engine.spec` for per-sync-mode field rules and
[Forward-Compatible Settings Parsing](../cli/forward-compatible-settings-parsing.rule.md) for how an
older binary handles a field a newer release added.

A hook reads this file on the pre-write path, so its size is on a latency budget — see the runtime
contract's constraint on reading project state once per invocation.

## How to Add a New Setting

1. Add the field to the `Settings` struct in `@cli/internal/config/config.go` with a
   `json:"...,omitempty"` tag. A default is resolved at read time and never written to the file.
2. Add the field name to `allowedFields` for the relevant sync types (all types if sync-independent)
3. Add to `requiredFields` only if it is mandatory for a sync type
4. Update `Validate()` with any constraints
5. Update `MarshalJSON()` — add the field to **all three** per-sync-mode struct literals. A field
   added to one is silently dropped when the project switches sync mode.
6. Update `UnmarshalJSON()` — add decoding and type validation
7. Add `"fieldname"` cases to `getSettingsValue()` and `setSettingsValue()` in `@cli/cmd/config.go`
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
stdout carries the host protocol and nothing may print to it. See `@cli/cmd/status_report.go`.

## How to Add a New Document Type

Follow [Add a Document Type to the Engine](../document-types/add-a-document-type.task-type.md).

## The Document Model — `internal/docs`

`internal/docs` owns the document model, the scan, the per-file cache, and the path guards. The MCP
tools, the hook handlers, and the status report all read documents through it. See the ADR on
[internal/docs owning the document model](../cli/docs-package-owns-the-document-model.adr.md).

`LocalDocument` stays the name on the MCP wire; `docs.Document` is the domain name. The alias in `@cli/internal/mcp/tools/docs_bridge.go` keeps
both true with no conversion layer. The bridge's re-exports are package-private; a caller outside `internal/mcp/tools` uses the exported `docs.*` name instead.

The guards return classified sentinels rather than messages, so the MCP tools and the hook render the
same verdict in their own words — see
[A Shared Guard Returns Classified Sentinels](../code-quality/shared-guards-return-classified-sentinels.rule.md).

## How to Add a New MCP Tool

The file shape is normative and is stated once, in
[The Shape of an MCP Tool File](../code-quality/the-shape-of-an-mcp-tool-file.rule.md). Read it before
step 1; the steps below are the surrounding wiring, not the shape. The server declares no prompts capability.

1. Create `internal/mcp/tools/<name>.go` exporting the pair `New<Name>Tool() mcp.Tool` and
   `Handle<Name>(root RootProvider) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)`
2. Resolve the project root inside the returned closure with `root.Root(ctx)`. Never capture a
   `baseDir` at construction: a session that moves into a git worktree moves the server's root with
   it (project-root-resolution.spec §3), and a captured root silently serves the old project.
3. Create `internal/mcp/tools/<name>_test.go`, using `StaticRoot(base)` as the provider
4. Register in `@cli/internal/mcp/server.go` via `s.AddTool()`
5. Use the helpers re-exported by `@cli/internal/mcp/tools/docs_bridge.go`
6. Validate inputs and check path safety (no `..`, must resolve inside `.archcore/`)
7. Return a user-caused refusal as `errorResult(...)`; return a Go error only for the process's own
   failure, such as marshalling the result
8. Never expose absolute filesystem paths in error messages — see
   [No Absolute Paths in MCP Errors](../mcp/no-absolute-paths-in-mcp-errors.rule.md)
9. Add the tool name to `archcoreMCPTools` in `@cli/cmd/hook_payload.go`.
   `TestArchcoreMCPTools_MatchesTheServer` fails otherwise. Three hosts flatten an MCP tool name by
   joining the server name to the tool name with one separator, and that separator also occurs inside
   names, so the fold accepts only a tool this list knows. A tool missing from it is read by the write
   guard as a direct edit and denied on those hosts.
10. If the tool mutates the knowledge base, add it to `mutatingMCPTools` in the same file and to
    `mcpDocumentTools` in `@cli/internal/wiring/hooks_agents.go`, which is the host-side matcher that
    decides whether the post-write process starts at all
11. If the tool crosses another tool's state — one handler writes, another reads — add the scenario to
    `internal/mcp/integration/` rather than to the per-tool unit file
    (in-process-mcp-integration-tests.adr)

### Conditionally Registered Tools (executor injection)

A tool whose implementation lives in the cmd layer (because it reuses CLI installers) is registered via
a `ServerOption` instead of unconditionally: the handler takes an executor function type defined in
`internal/mcp/tools`, and `NewServer` adds the tool only when the option supplies one.
`install_host_config` is the pattern's example — `@cli/cmd/mcp.go` passes
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
  the work runs, and how the command degrades (`@cli/cmd/hook_command.go`, `@cli/cmd/hook_write_guard.go`).
- [Hook Wire Protocol Per Host](../integrations/hook-wire-protocol.spec.md) — what the process writes
  and how it exits, per dialect (`@cli/cmd/hook_dialect.go`).
- [Hook Payload Reading](../integrations/hook-payload-reading.spec.md) — how the payload becomes a
  tool identity and a set of targets (`@cli/cmd/hook_payload.go`).

Install-time logic (writing host configs) lives in `internal/wiring`; runtime logic (handling a fired
event) stays in `cmd/`. There is no per-agent runtime file — a host is one row in a dialect table.
The advisory work the hooks call lives in `internal/advisory/`, not in `cmd/` — see
[The Advisory Subsystem](../architecture/advisory-subsystem.doc.md).

Installed commands are recognized across CLI versions by the `archcore hooks ` prefix — see
[Hook Command Marker ADR](../integrations/hook-command-marker-prefix.adr.md).

The dialect table answers which hosts this binary serves; `hooksInstallers` answers which hosts get a
config written. They are not the same set, and the difference is what `hooks install` reports: a host
in the first and not the second loads its hooks as plugin code.

The background update trigger is not added to a hook. A hook leaf is short-lived, runs on a budget of
seconds, and its stdout is the host's protocol channel.

### Modifying the Injected Context

`buildSessionContext()` in `@cli/cmd/hooks_common.go` builds the session-start text. Its output contract —
section order, the 24-line document budget, the `rejected` exclusion, the tag cap — is the
[SessionStart Context Output Contract](../integrations/session-start-context.spec.md). Changes here
apply uniformly to every host.

## How to Add a New Agent

1. **Define the agent ID** — Add a new `AgentID` constant in `@cli/internal/agents/agents.go`

2. **Create the agent file** — Add `internal/agents/<name>.go` returning an `*Agent` struct:
   - `ID`, `DisplayName` — identity
   - `DetectFn` — check for the agent's marker directory/file
   - `MCPConfigPath` — return the path to the MCP config file
   - `WriteMCPConfig` — write the archcore MCP entry (use `WriteStandardMCPJSON` if the agent uses the
     standard `mcpServers` format)
   - `WriteHooksConfig` — set to `nil` if hooks are not supported
   - `ManualMCPInstallHint` — set if MCP must be installed manually (e.g., Cline)
   - `InstructionsPath`, `WriteInstructions`, `RemoveInstructions` — the usage-nudge target. Point them
     at a shared helper in `@cli/internal/agents/instructions.go`. `TestAllAgents_RequiredFields` fails if
     any are nil.

3. **Register the agent** — Add the constructor call to the `all` slice in `@cli/internal/agents/agents.go`

4. **Add tests** — Create `internal/agents/<name>_test.go` covering detection, MCP config writing,
   idempotency, and the instruction target

5. **If hooks are supported:**
   - Add a `hostDialect` row in `@cli/cmd/hook_dialect.go`. The three hidden event leaves are generated
     from it; no per-agent runtime file is needed.
   - Add an `InstallXxxHooks()` function in `@cli/internal/wiring/hooks_agents.go` covering all three
     events — the installed command must start with `archcore hooks ` (marker contract)
   - Add the agent to `hooksInstallers` in the same file. Skip this step only when the host loads
     hooks as plugin code and there is no declarative file to write; the dialect row alone then makes
     `hooks install` describe it correctly instead of calling it hookless.
   - If the host flattens MCP tool names into a spelling not already folded, add its prefix in
     `@cli/cmd/hook_payload.go`. An unfolded spelling makes the write guard deny the host's own document
     tools.
   - If the host names a file-mutation tool that carries no path argument, add the key its patch or
     path arrives under in the same file. A missing key fails silently: no target, allow, and an
     unprotected session that looks clean.
   - If the host can accept the config and still not run it, add its case to `EffectiveHookNotes` in
     `@cli/internal/wiring/hooks_effective.go`

6. **If the host ships an Archcore plugin:** add its row to the host table in `@cli/internal/plugin/hosts.go`
   — the CLI name, the read-only listing command, the on-disk registry path, and the install, update,
   and remove commands. A host with no CLI mechanism carries a UI note instead. Nothing else changes:
   the planner reads the row, and the three entry points share it.

7. **Add the host CLI to the test trap** — `hostCLIs` in `@cli/internal/testsupport/isolate.go`, so a test
   that reaches the real binary is caught instead of changing the developer's machine. See
   [Isolating the Developer's Machine](../code-quality/isolating-the-machine-from-the-test-suite.guide.md).

8. **Update documentation:**
   - Add the agent to [Supported AI Agents Registry](../integrations/supported-ai-agents.doc.md)
   - Add the host's event rows to [Hook Wire Protocol Per Host](../integrations/hook-wire-protocol.spec.md)
   - Add config examples to [Agent Integration Reference](../integrations/agent-hooks-integration.doc.md)

A host is therefore described by five registries in four packages. That split is deliberate — see the
accepted trade-offs in [Archcore CLI System Overview](../architecture/system-overview.doc.md).
