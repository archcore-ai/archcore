---
title: "CLI Hooks Reference"
status: accepted
tags:
  - "cli"
  - "integrations"
---

## Overview

Archcore hooks intercept AI host lifecycle events. Three events are active:

- `SessionStart` — injects the project recap into a new session.
- `PreToolUse` — blocks a direct write to an `.archcore/` document, and injects the documents that
  constrain the file about to be edited.
- `PostToolUse` — reports structure problems, relation cascades, and precision findings after a
  document mutation.

Six host dialects are implemented. Five of them also receive install wiring; `opencode` has command
leaves but no declarative config, because its hooks are JavaScript plugins.

This document is the reference: what is wired where, and what each host is sent. The binding behavior
lives in three specs, split by what each owns — the hook runtime contract states which guard blocks
and how the command degrades, the hook wire protocol states what the process writes and how it exits
per host, and hook payload reading states how a payload becomes a tool identity and a set of targets.
The SessionStart context spec states what the recap holds, the ADR on running guardrails in the CLI
records why they moved out of the plugin, and the rule on reporting effective hook state governs what
`hooks install` prints.

## Commands

### `archcore hooks install`

Installs hooks for every detected agent, then installs the MCP config for the same agents.

```
archcore hooks install                    # auto-detect agents
archcore hooks install --agent cursor     # one agent only
archcore hooks install --project /path    # project root holding .archcore/
```

After a successful install for a host, the command prints the notes that say whether that host can
actually run what was written. Silence means the wiring works. A host whose hooks load as plugin code
rather than from a config, such as OpenCode, is reported as such instead of as hookless.

`archcore doctor` prints the same notes, but only for a host whose config actually holds an archcore
hook command. A detected host that was never wired is not reported on, and with no host wired the
command says nothing about hooks rather than claiming they are healthy.

### `archcore hooks <host> <event>`

Handles one hook event for one host. The host invokes these; a user does not run them directly. Every
leaf is hidden, and an unrecognized host or event writes an empty stdout and exits 0.

```
archcore hooks claude-code session-start
archcore hooks cursor pre-tool-use
archcore hooks copilot post-tool-use
```

Hosts: `claude-code`, `cursor`, `gemini-cli`, `copilot`, `codex-cli`, `opencode`.
Events: `session-start`, `pre-tool-use`, `post-tool-use`.

## Guards by event

The command leaves live in `cmd/`; the advisory logic they call lives in `internal/advisory/`.

| Event | Guard | Can deny | Source |
|---|---|---|---|
| `session-start` | project recap | no | `@cmd/hooks_common.go`, `@cmd/hook_session_start.go` |
| `session-start` | staleness advisory | no | `@internal/advisory/staleness.go` |
| `pre-tool-use` | write guard | **yes** | `@cmd/hook_write_guard.go` |
| `pre-tool-use` | code-alignment injection | no | `@internal/advisory/code_alignment.go` |
| `post-tool-use` | structure validation | no | `@cmd/hook_post_tool_use.go` |
| `post-tool-use` | relation cascade notice | no | `@cmd/hook_post_tool_use.go` |
| `post-tool-use` | precision findings | no | `@internal/advisory/precision.go` |

The write guard is the only guard that blocks. It runs first and alone, so a failure in the advisory
path cannot change the verdict on a write. It refuses what the MCP write tools refuse — including a
document inside a global source mounted from outside the store, which the tools cannot address at all
and which the guard therefore checks separately.

The guard runs over two kinds of target. The ordinary one is the edited file path. The second is an
apply-patch document, which names its files only inside the patch body: every `*** Add File:`,
`*** Update File:`, `*** Delete File:`, and `*** Move to:` line is read, and a guarded target among
them blocks the whole call, because a patch applies as a unit.

`*** Move to:` is the rename destination, and it is the only line that names it. A rename is written
as an update of the source plus that line, so reading the first three directives alone sees an
ordinary source file and lets the patch land a document in the store.

The patch scan stops after 20000 lines. The payload cap bounds the bytes; this bounds the lines,
because the pre-write guard blocks the user while it runs. A target past that line is not guarded.

One invocation builds one guard, so the project state a verdict reads — whether the store exists, and
which globals are declared — is read once however many targets the call carries, and only when a
verdict needs it. Read per target, `settings.json` alone put a maximal patch within reach of the
one-second host budget.

Code-alignment behavior: source roots default to `src`, `lib`, `app`, `pkg`, `cmd`, `internal`,
`apps`, `packages`, `modules`, `components`, and `codeAlignment.sourceRoots` in `settings.json`
replaces that list. A declared root is normalized on load (`./src`, `src/`, and `src` are one root),
so a root that validates also matches. Only `rule`, `cpat`, `adr`, `spec`, and `guide` documents are
injected, ranked in that order; at most 3 documents and 2048 runes reach the host. A matching global
source is included and marked `[global]`. `ARCHCORE_DISABLE_INJECTION=1` turns the injection off.

Staleness behavior: the advisory compares the last commit that touched `.archcore/` against the
commits since, names documents that mention the changed directories, and is rate-limited to once per
24 hours per project.

Precision behavior: the advisory validates the written document's path through the containment-safe
read guard before opening it, so a symlink out of `.archcore/` produces no findings at all. Body
length is measured in characters, not bytes, so a short non-ASCII document is flagged as a
placeholder just as an English one is.

## Installed wiring per host

One archcore-owned hook entry per (host, event) pair. The process dispatches by tool name internally.

| Host | Config file | Session event | Pre-write event and matcher | Post-write event and matcher |
|---|---|---|---|---|
| Claude Code | `.claude/settings.json` | `SessionStart` | `PreToolUse`, `Write\|Edit` | `PostToolUse`, MCP document tools |
| Codex CLI | `.codex/hooks.json` | `SessionStart` | `PreToolUse`, `Write\|Edit\|apply_patch` | `PostToolUse`, MCP document tools |
| Cursor | `.cursor/hooks.json` | `sessionStart` | `preToolUse`, `Write` | `afterMCPExecution`, no matcher |
| Gemini CLI | `.gemini/settings.json` | `SessionStart` | `BeforeTool`, `write_file` | `AfterTool`, MCP document tools |
| GitHub Copilot | `.github/hooks/archcore.json` | `sessionStart` | `preToolUse`, `create\|edit\|str_replace_editor\|apply_patch` | `postToolUse`, MCP document tools |
| OpenCode | not wired | — | — | — |

Two of these matchers name `apply_patch`, so the patch targets described above are read on Codex CLI
and GitHub Copilot as well as on OpenCode. The patch route is not an OpenCode-only concern.

The MCP document-tool matcher covers `create_document`, `update_document`, `remove_document`,
`add_relation`, and `remove_relation`. It carries four of the five spellings the in-process fold
knows: OpenCode's `archcore_*` is absent because OpenCode is the one host this matcher is never
written for, so including it would only rewrite five other hosts' configs to match a name none of
them sends.

Timeouts: 1 second on the pre-write event, 3 seconds on the session and post-write events. Gemini CLI
takes the same values in milliseconds. Claude Code and Codex CLI entries carry no timeout field.

Re-installing updates an archcore entry in place and keeps every field archcore does not write — a
`timeout` you added to our hook, or a key written by a newer archcore, survives. An entry that
differs only by such a field is left alone entirely; the file keeps its bytes and its mtime.

Source: `@internal/wiring/hooks_agents.go`.

## Host dialects

A host that does not recognize the envelope reads no context at all, and one that misreads an exit
code blocks the user's edit. Each row is the host's documented behavior; the wire protocol spec
states the same rows normatively.

| Host | SessionStart output | Context envelope | Deny style | Pre-write context |
|---|---|---|---|---|
| Claude Code | `hookSpecificOutput` + `systemMessage` | `hookSpecificOutput` wrapper | stderr, exit 2 | yes |
| Cursor | `hookSpecificOutput` + `systemMessage` | `additional_context` | stderr, exit 2 | yes |
| Gemini CLI | `hookSpecificOutput` + `systemMessage` | `hookSpecificOutput` wrapper | stderr, exit 2 | yes |
| GitHub Copilot | `additionalContext` | `additionalContext` | `permissionDecision` JSON, exit 0 | **no** |
| Codex CLI | `hookSpecificOutput` + `systemMessage` | `hookSpecificOutput` wrapper | stderr, exit 2 | yes |
| OpenCode | plain text, no banner | plain text | stderr, exit 2 | yes |

Copilot's `preToolUse` carries only a permission decision, so code-alignment context is discarded
there rather than emitted. Copilot also reads any non-zero exit as a deny and drops the reason with
it, which is why its deny writes JSON and exits 0.

OpenCode is plain text on the session event as well as the tool events. Its route is a TypeScript
bridge that carries no decision logic: the plugin's `bin/session-start` streams this command's stdout
through unchanged, and the bridge appends those bytes to the session verbatim. Nothing on that path
parses JSON, so an envelope emitted here does not frame the recap — it reaches the model as literal
JSON with the recap escaped inside it. The same route leaves no slot for the SessionStart banner,
which the plain-text envelope therefore drops, on the reasoning Copilot's envelope already uses: the
banner is a line for the user, and the context channel is input for the model. A deny stays exit 2
with the reason on stderr, which the bridge rethrows as an `Error` whose message the model receives.

Source: `@cmd/hook_dialect.go`, `@cmd/hook_session_start.go`, and the bridge contract in the plugin
repository (`plugins/archcore/bin/lib/normalize-stdin.sh`, `host-adapter-contract.spec`).

## Hook input

The payload arrives as one JSON object on stdin and is read by explicit key paths. A field that is
itself a JSON document in string form is decoded on the way through, which is how Copilot's `toolArgs`
and Cursor's `tool_input` are read. The payload-reading spec states the binding rules; the table below
is the current key list.

| Purpose | Keys read, in order |
|---|---|
| Session identity | `session_id`, `conversation_id` |
| Session source | `source` |
| Project root | `cwd` |
| Tool name | `tool_name`, `toolName` |
| Edited file | `tool_input.file_path`, `tool_input.filePath`, `toolArgs.file_path`, `toolArgs.filePath`, `file_path`, `filePath`, then `tool_input.path` and `toolArgs.path` when the caller is not an archcore MCP tool |
| Patch document | `tool_input.patchText`, `toolArgs.patchText`, `tool_args.patchText`, `patchText`, then the same three containers for `input`, then for `patch` |
| Patch targets | inside that text: every `*** Add File:`, `*** Update File:`, `*** Delete File:`, `*** Move to:` line, up to 20000 lines |
| Document path | `tool_input.path`, `toolArgs.path`, `path` |

An empty, truncated, or non-JSON stdin yields an empty payload, and every guard then allows. Bytes
after the leading JSON object are ignored rather than rejected — rejecting them would produce that
same empty payload, and turn a padded payload into a way past the write guard.

Both spellings of the edited-file key are read because a host names it either way and a bridge cannot
be forced to rename it. OpenCode's `write` and `edit` tools take `filePath`; the snake_case key stays
first, so a payload carrying both resolves to the normalized one.

The patch rows exist because a patch tool names its targets nowhere else. OpenCode's `apply_patch`
takes only `patchText`, and on that host the tool is not an alternative to `write` and `edit` but a
replacement: its registry enables `apply_patch` and disables the other two for `gpt-` models. Without
these rows, those sessions would run with no write protection and look no different from protected
ones. Codex CLI and Copilot name `apply_patch` in their installed matchers as well, so the same
path-less call arrives on hosts the CLI does wire. [assumption] Their argument keys are unverified
from this repository, which is why `input` and `patch` are read alongside `patchText`; a key that
carries something other than a patch costs a scan that finds no directives.

Directive matching accepts any case and any spacing inside the directive, and each line is trimmed
before it is matched. Both are looser than the format, on purpose: this guard cannot see the parser
that will apply the patch, so matching one host's exact strictness would make the guard's coverage a
bet on someone else's code. The cost is that a patch quoting a directive in a hunk is denied —
visible, unlike a write that slipped past a parser more forgiving than this one.

MCP tool names arrive in five spellings and fold to the canonical form: `mcp__archcore__*`,
`mcp__plugin_archcore_archcore__*`, `mcp_archcore_*` (Gemini), `archcore-*` (Copilot), and
`archcore_*` (OpenCode). A tool from another MCP server is left untouched.

The fold is what keeps an archcore MCP call from being read as a direct file edit. An unfolded
spelling does not merely skip the post-write checks: the edited-file lookup then falls through to the
bare `path` key, the write guard reads a sanctioned MCP write as a direct one, and Archcore blocks
its own document tools on that host. OpenCode's spelling was missing and did exactly that.

The first two spellings delimit the server name with a double underscore, so they fold whatever
follows. The other three are host flattenings that join the server name to the tool name with a
single separator, and that separator also appears inside both names: nothing in
`archcore_docs_create_document` says whether the server is `archcore` or `archcore_docs`. Those three
therefore fold only onto a tool this server actually registers. Unbounded, they would claim a foreign
server whose name merely starts with ours, and the cost is not a mislabeled name — the edited-file
lookup would stop reading the bare `path` key and wave that server's write into `.archcore/` through.

The bounded set is every tool the MCP server registers, read back off a built server so the two
cannot drift. A tool the server gains and the fold does not know is not folded under a flattened
spelling, so the write guard denies it as a direct edit. That failure is visible, which is why the
bound is set that way.

Source: `@cmd/hook_payload.go`.

## Hook output examples

Non-normative examples.

### SessionStart, Claude-compatible hosts

```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "<recap text>"
  },
  "systemMessage": "<connected banner>"
}
```

### SessionStart, Copilot

```json
{ "additionalContext": "<recap text>" }
```

### SessionStart, OpenCode

The recap goes to stdout as text, with no envelope and no banner. The plugin launcher streams it
through and the bridge appends it to the session verbatim.

```
[Archcore — Git-native context for AI coding agents]
CORPUS: 86 documents — knowledge 69, vision 16, experience 1 · draft 10, accepted 76
…
```

### Deny, Copilot

```json
{
  "permissionDecision": "deny",
  "permissionDecisionReason": "Direct writes to .archcore/ documents are not allowed. …"
}
```

### Deny, every other host

The reason goes to stderr, stdout stays empty, and the process exits 2.

## Removed hooks

The `Stop` and `UserPromptSubmit` families were removed because keyword matching produced too many
false positives. The related ADR records the rationale. They have not returned: `PreToolUse` and
`PostToolUse` match on tool names and file paths, not on message text.

Previously supported and unsupported now:

- `Stop` — scanned assistant messages for keywords such as "decided to", then blocked the agent.
- `UserPromptSubmit`, `BeforeSubmitPrompt`, `BeforeAgent` — scanned user prompts for keywords such as
  "should we use", then injected task instructions.
