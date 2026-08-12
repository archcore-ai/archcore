---
title: "Supported AI Agents Registry"
status: accepted
tags:
  - "integrations"
---

## Summary

Archcore integrates with 8 AI coding agents. Each agent has its own combination of hooks support
(lifecycle event interception), MCP support (document tool access), and an instruction-nudge file
(discovery hint).

The canonical, tool-agnostic host roster lives in the `archcore` global source
(`architecture/supported-ai-hosts`). This document is the CLI's per-host integration reference: config
paths, hook formats, detection, and the add-an-agent recipe. The CLI hooks reference carries the event
matrix and the per-host protocol dialects.

## Agent registry

| Agent | ID | Hooks | MCP | Detection Marker | Link |
|-------|----|-------|-----|------------------|------|
| Claude Code | `claude-code` | Yes | Yes | `.claude/` dir | [docs.anthropic.com](https://docs.anthropic.com/en/docs/claude-code) |
| Cursor | `cursor` | Yes | Yes | `.cursor/` dir | [cursor.com](https://www.cursor.com/) |
| Gemini CLI | `gemini-cli` | Yes | Yes | `.gemini/` dir | [github.com/google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli) |
| Codex CLI | `codex-cli` | Yes | Yes | `.codex/` dir | [github.com/openai/codex](https://github.com/openai/codex) |
| GitHub Copilot | `copilot` | Yes | Yes | `.github/copilot-instructions.md` file | [github.com/features/copilot](https://github.com/features/copilot) |
| OpenCode | `opencode` | Via plugin | Yes | `opencode.json` file or `.opencode/` dir | [opencode.ai](https://opencode.ai/) |
| Roo Code | `roo-code` | No | Yes | `.roo/` dir | [roocode.com](https://roocode.com/) |
| Cline | `cline` | No | Manual | `.clinerules/` dir | [cline.bot](https://cline.bot/) |

## Integration levels

### Full integration (hooks and MCP)

Agents: Claude Code, Cursor, Gemini CLI, Codex CLI, GitHub Copilot.

These agents support the three archcore lifecycle events and MCP tool access. `archcore init`
auto-detects them, installs the hooks, and writes the MCP config.

Two of them carry a host limitation that `hooks install` reports at write time:

- Codex CLI keeps hooks behind an experimental feature flag that is off by default, does not support
  hooks on Windows, and loads project-local hooks only when the `.codex/` layer is trusted.
- GitHub Copilot cannot carry context on its pre-write event, so the write guard runs there but the
  code-alignment injection does not.

### Hooks through a plugin

Agent: OpenCode.

OpenCode loads hooks as plugin code, so the CLI writes no hook config and cannot: there is no
declarative file to write. It ships the `archcore hooks opencode <event>` leaves instead, and the
Archcore OpenCode plugin registers the host's events and calls them. The three events are the same
ones every other host runs; only the wiring route differs. `hooks install` says so rather than
reporting the host as hookless.

### MCP only

Agent: Roo Code.

Roo Code supports `onSave` hooks only, which do not serve lifecycle events.

### Manual

Agent: Cline.

Cline stores its MCP config in the VS Code `globalStorage`, not in a project-level file. The user adds
the Archcore MCP server through Cline's MCP settings interface. Archcore prints a hint when it detects
Cline.

## Instruction nudge files

Beyond hooks and MCP, `archcore init` (opt-in) and `archcore instructions install` write a short,
always-on "use Archcore" hint into each agent's instruction file, so that a CLI-only user without the
Archcore plugin discovers and invokes the MCP tools. Under Tool Search, MCP tools are deferred and only
names load at startup, so this nudge is the discovery trigger. The related ADR on the per-agent usage
nudge records the decision.

| Agent | Instruction file(s) | Write mode |
|-------|---------------------|------------|
| Claude Code | `CLAUDE.md` **and** `AGENTS.md` | fenced upsert (both) |
| Gemini CLI | `GEMINI.md` | fenced upsert |
| Cursor, OpenCode, Codex CLI, Roo Code, Cline, GitHub Copilot | `AGENTS.md` | fenced upsert |

- Claude Code gets `CLAUDE.md` and `AGENTS.md`. Per Anthropic's documentation
  (code.claude.com/docs/en/memory), Claude Code reads `CLAUDE.md` natively and does not auto-read
  `AGENTS.md`, so `CLAUDE.md` delivers the nudge. The `AGENTS.md` block is written as well, so the
  repository carries the standard that the plugin and the six other hosts converge on. `CLAUDE.md`
  holds the full nudge body directly, with no `@import`, so the two files stay uncoupled. Claude Code
  loads only `CLAUDE.md`, so the second file adds no duplicated context cost.
- Fenced upsert: for every target (`CLAUDE.md`, `AGENTS.md`, `GEMINI.md`), Archcore touches only the
  span between `<!-- archcore:start -->` and `<!-- archcore:end -->`. User content outside the markers
  stays unmodified, and a second write is idempotent, so Claude Code and a co-installed `AGENTS.md`
  agent both writing the `AGENTS.md` block collapse into one block.
- Dedup: the six `AGENTS.md`-only agents share one file, written once. Claude Code dedupes on its own
  `CLAUDE.md` path, which is distinct from `AGENTS.md`, so it always runs its own write — which also
  refreshes the `AGENTS.md` block — regardless of the agent order.
- Reverse: `archcore instructions remove [--agent <id>]` strips the fenced block and keeps user
  content. Removing Claude Code alone strips only its `CLAUDE.md` block and deletes the legacy
  `.claude/rules/archcore.md`, leaving the shared `AGENTS.md` block to the `AGENTS.md` agents' own
  remove. Both run in the remove-all path.
- Deprecated: earlier CLI versions wrote an owned `.claude/rules/archcore.md`. Re-wiring now removes
  that file.
- Source: `@internal/agents/instructions.go`, `@cmd/instructions.go`.

## Per-agent details

Three lifecycle events are active: `SessionStart`, `PreToolUse`, and `PostToolUse`, in each host's own
spelling. The `Stop` and `UserPromptSubmit` families remain unsupported; the related ADR records the
rationale. The runtime side of every hook lives in the shared `cmd/hook_*.go` files, not in a
per-agent file — the ADR on running the guardrails in the CLI records that consolidation.

Every command that reads or writes `.archcore/` resolves its project root through
`resolveProjectRoot`, which honors `--project` and `ARCHCORE_PROJECT_ROOT` and refuses a root inside a
host's plugin install cache. The hook leaves are the exception: there the host names the project in
the payload's `cwd` key, and the process's own working directory is the host's.

### Claude Code

- Config paths: `.claude/settings.json` (hooks), `.mcp.json` (MCP)
- Hook events: `SessionStart`; `PreToolUse` with matcher `Write|Edit`; `PostToolUse` with the MCP
  document-tool matcher
- Hook commands: `archcore hooks claude-code session-start|pre-tool-use|post-tool-use`
- MCP format: standard `mcpServers` JSON (`{"command": "archcore", "args": ["mcp"]}`)
- Instruction file: `CLAUDE.md` (fenced upsert, read natively by Claude Code) and `AGENTS.md` (fenced
  upsert, for the plugin and other hosts)
- Source: `@internal/agents/claude_code.go`, `@internal/wiring/hooks_agents.go`

### Cursor

- Config paths: `.cursor/hooks.json` (hooks), `.cursor/mcp.json` (MCP)
- Hook events: `sessionStart`; `preToolUse` with matcher `Write`; `afterMCPExecution` without a
  matcher
- Hook commands: `archcore hooks cursor session-start|pre-tool-use|post-tool-use`
- Hook format: flat entries with a `version` field; MCP results arrive on their own event, which takes
  no matcher
- MCP format: standard `mcpServers` JSON
- Instruction file: `AGENTS.md` (fenced upsert)
- Source: `@internal/agents/cursor.go`, `@internal/wiring/hooks_agents.go`

### Gemini CLI

- Config paths: `.gemini/settings.json` (hooks and MCP in one shared file)
- Hook events: `SessionStart`; `BeforeTool` with matcher `write_file`; `AfterTool` with the MCP
  document-tool matcher
- Hook commands: `archcore hooks gemini-cli session-start|pre-tool-use|post-tool-use`
- Hook format: matcher-wrapped entries with timeouts in **milliseconds**
- MCP format: standard `mcpServers` JSON inside the same `settings.json`
- Instruction file: `GEMINI.md` (fenced upsert; Gemini CLI's default `contextFileName`)
- Source: `@internal/agents/gemini_cli.go`, `@internal/wiring/hooks_agents.go`
- [assumption] The tool events are wired from the published reference and are not confirmed against a
  running host. Their event names, tool names, and timeout unit all differ from every other host.

### Codex CLI

- Config paths: `.codex/hooks.json` (hooks), `.codex/config.toml` (MCP)
- Hook events: `SessionStart`; `PreToolUse` with matcher `Write|Edit|apply_patch`; `PostToolUse` with
  the MCP document-tool matcher
- Hook commands: `archcore hooks codex-cli session-start|pre-tool-use|post-tool-use`
- Hook format: Claude Code's schema in a separate file. Codex also accepts an inline `[hooks]` table in
  `config.toml`; the separate file keeps one writer per file, so the hook wiring never rewrites the
  TOML the MCP wiring owns.
- MCP format: the TOML block `[mcp_servers.archcore]` with `command` and `args`
- Note: the only agent that uses a TOML config format for MCP
- Instruction file: `AGENTS.md` (fenced upsert)
- Source: `@internal/agents/codex_cli.go`, `@internal/wiring/hooks_agents.go`
- Hooks are experimental and off by default (`[features]` with `hooks = true`, spelled
  `codex_hooks = true` before Codex 0.129.0), unavailable on Windows, and project-local hooks load only
  when the `.codex/` layer is trusted. `hooks install` and `doctor` report these conditions.
- `apply_patch` names its files only inside the patch body, so the write guard reads the body rather
  than a file-path argument. [assumption] The key that carries the patch is unverified from this
  repository; `patchText`, `input`, and `patch` are all read.

### GitHub Copilot

- Config paths: `.github/hooks/archcore.json` (hooks), `.mcp.json` (MCP)
- Hook events: `sessionStart`; `preToolUse` with matcher `create|edit|str_replace_editor|apply_patch`;
  `postToolUse` with the MCP document-tool matcher
- Hook commands: `archcore hooks copilot session-start|pre-tool-use|post-tool-use`
- Hook format: uses the `bash` field instead of `command` (`{"type": "command", "bash": "..."}`), with
  `timeoutSec`
- MCP format: standard `mcpServers` JSON — the same file and the same shape Claude Code gets. Both
  hosts key on `mcpServers` and accept a bare `{command, args}` stdio entry, so one file serves both
  and the write merges idempotently.
- Not `.vscode/mcp.json`: Copilot CLI dropped that source in v1.0.37 (github/copilot-cli#3019), so it
  is dead config for the CLI and belongs to VS Code alone. Not `.github/mcp.json` either — the
  config-dir documentation lists it, but it has never been read as a workspace source
  (github/copilot-cli#1886). Copilot CLI discovers `.mcp.json` from the working directory up to the
  git root, so a repository-root file covers monorepo layouts too.
- Detection: the `.github/copilot-instructions.md` file
- Instruction file: `AGENTS.md` (fenced upsert, read natively alongside
  `.github/copilot-instructions.md`)
- Source: `@internal/agents/copilot.go`, `@internal/wiring/hooks_agents.go`
- Copilot's `preToolUse` carries only a permission decision, so the write guard runs and the
  code-alignment injection does not. Copilot also reads `.claude/settings.json`; in a repository wired
  for both hosts, `hooks install` reports the possible duplicate run.
- Its matcher names `apply_patch` too, so the patch-body scan applies here on the same terms as on
  Codex CLI.

### OpenCode

- Config path: `opencode.json` (MCP)
- MCP format: `{"mcp": {"archcore": {"type": "local", "command": ["archcore", "mcp"]}}}`
- Note: OpenCode uses a different MCP JSON structure, with `type` and with `command` as an array. Its
  optional per-server keys are `enabled`, `cwd`, `timeout`, and `environment` — spelled that way, not
  `env`. The published schema sets `additionalProperties: false`, so a wrong key is rejected rather
  than ignored. OpenCode's own documentation gives the `timeout` default as 5000 ms while its source
  uses 30000 ms; archcore sets no timeout, so neither value applies to what the CLI writes.
- Hooks: not wired declaratively, and never will be — OpenCode loads hooks as plugin code. The
  `archcore hooks opencode <event>` leaves exist for the Archcore OpenCode plugin to call.
- Protocol: plain text on every event, including `session-start`, because the plugin's launcher
  streams this binary's stdout to a bridge that appends it verbatim. Deny stays exit 2 with the reason
  on stderr, which the bridge rethrows as an `Error` whose message the model receives. The CLI hooks
  reference carries the detail.
- MCP tool names: OpenCode flattens an MCP tool to `<server>_<tool>` joined by a single underscore,
  with the server name passed through verbatim and no truncation, so archcore's arrive as
  `archcore_create_document`. `cmd.foldToolName` folds that spelling; without it the write guard reads
  a sanctioned MCP write as a direct edit and blocks the document tools.
- The same separator occurs inside server names, so `archcore_docs_create_document` belongs to a
  server called `archcore_docs` and the spelling alone cannot say which. The fold therefore accepts
  only a tool this MCP server registers. Unbounded, it would exempt a foreign server's write from the
  guard entirely.
- File-writing tools: OpenCode's `write` and `edit` name their argument `filePath`, not `file_path`,
  and its registry replaces both with `apply_patch` for `gpt-` models. The guard reads the camelCase
  key and the patch body for that reason.
- Instruction file: `AGENTS.md` (fenced upsert)
- Source: `@internal/agents/opencode.go`, `@cmd/hook_dialect.go`, `@cmd/hook_payload.go`

### Roo Code

- Config path: `.roo/mcp.json` (MCP)
- MCP format: standard `mcpServers` JSON
- Note: Roo Code supports `onSave` hooks only, which do not serve lifecycle events
- Instruction file: `AGENTS.md` (fenced upsert)
- Source: `@internal/agents/roo_code.go`

### Cline

- Config path: VS Code `globalStorage`, not project-level
- MCP format: manual installation through the Cline MCP settings interface
- Hint shown: "MCP config is stored in VS Code globalStorage — add manually via Cline MCP settings"
- Instruction file: `AGENTS.md` (fenced upsert)
- Source: `@internal/agents/cline.go`

## Adding a new agent

1. Add a new `AgentID` constant in `@internal/agents/agents.go`.
2. Add `internal/agents/<name>.go` implementing the `Agent` struct with `DetectFn`, `MCPConfigPath`,
   `WriteMCPConfig`, and, when the agent supports hooks, `WriteHooksConfig`.
3. Set `InstructionsPath`, `WriteInstructions`, and `RemoveInstructions` with the helpers in
   `@internal/agents/instructions.go`: `agentsMDInstructions*` for `AGENTS.md`, `geminiInstructions*`
   for `GEMINI.md`, or `claudeInstructions*` for the `CLAUDE.md` and `AGENTS.md` dual write.
   `TestAllAgents_RequiredFields` fails while any of the three is nil.
4. Add the agent constructor to the `all` slice in `@internal/agents/agents.go`.
5. Add `internal/agents/<name>_test.go`.
6. IF the agent supports hooks, THEN add a `hostDialect` row in `@cmd/hook_dialect.go` describing its
   session shape, context envelope, deny style, and whether its pre-write event can carry context. The
   three hidden event leaves are generated from that row; no per-agent runtime file is needed.
7. IF the agent supports hooks, THEN add an `InstallXxxHooks` function in
   `@internal/wiring/hooks_agents.go` covering all three events, and add the agent to
   `hooksInstallers` in the same file. The installed command MUST start with `archcore hooks `.
8. IF the agent's host flattens MCP tool names into a spelling not already folded, THEN add its prefix
   in `@cmd/hook_payload.go`. An unfolded spelling does not merely skip the post-write checks — it
   makes the write guard deny the host's own document tools. A flattening that joins the server name
   to the tool name with a single separator is matched against `archcoreMCPTools`, so it claims only
   this server's tools and not a foreign server whose name starts the same way.
9. IF the agent's host names a file-mutation tool that carries no path argument, THEN add the key its
   patch or path arrives under in `@cmd/hook_payload.go`. A missing key fails silently: the guard
   finds no target, allows, and an unprotected session looks exactly like a clean one.
10. IF the host can accept a written config and still not run it, THEN add its case to
    `EffectiveHookNotes` in `@internal/wiring/hooks_effective.go`.
11. Add the agent to the registry table and to the instruction-nudge table in this document.
12. Update the CLI hooks reference, the agent-hooks integration guide, and the building-the-CLI guide.

## Adding a new MCP tool

Add its name to `archcoreMCPTools` in `@cmd/hook_payload.go` in the same change.
`TestArchcoreMCPTools_MatchesTheServer` fails otherwise. A tool the fold does not know is not folded
under a flattened spelling, so on Gemini CLI, GitHub Copilot, and OpenCode the write guard reads that
sanctioned MCP write as a direct edit and denies it.

## Adding a new command

A command that reads or writes `.archcore/` MUST resolve its root through `resolveProjectRoot` and
MUST register a `--project` flag naming `ARCHCORE_PROJECT_ROOT` in its help text.
`TestCommands_OfferProjectFlag` walks the command tree and fails on a command that does neither.
