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

The neighboring documents carry the detail this reference points at: the hook runtime contract states
the normative behavior, the SessionStart context spec states what the recap holds, the ADR on running
guardrails in the CLI records why they moved out of the plugin, and the rule on reporting effective
hook state governs what `hooks install` prints.

## Commands

### `archcore hooks install`

Installs hooks for every detected agent, then installs the MCP config for the same agents.

```
archcore hooks install                    # auto-detect agents
archcore hooks install --agent cursor     # one agent only
archcore hooks install --project /path    # project root holding .archcore/
```

After a successful install for a host, the command prints the notes that say whether that host can
actually run what was written. Silence means the wiring works.

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

The MCP document-tool matcher covers `create_document`, `update_document`, `remove_document`,
`add_relation`, and `remove_relation` under all four name spellings a host may deliver.

Timeouts: 1 second on the pre-write event, 3 seconds on the session and post-write events. Gemini CLI
takes the same values in milliseconds. Claude Code and Codex CLI entries carry no timeout field.

Re-installing updates an archcore entry in place and keeps every field archcore does not write — a
`timeout` you added to our hook, or a key written by a newer archcore, survives. An entry that
differs only by such a field is left alone entirely; the file keeps its bytes and its mtime.

Source: `@internal/wiring/hooks_agents.go`.

## Host dialects

A host that does not recognize the envelope reads no context at all, and one that misreads an exit
code blocks the user's edit. Each row is the host's documented behavior.

| Host | SessionStart output | Context envelope | Deny style | Pre-write context |
|---|---|---|---|---|
| Claude Code | `hookSpecificOutput` + `systemMessage` | `hookSpecificOutput` wrapper | stderr, exit 2 | yes |
| Cursor | `hookSpecificOutput` + `systemMessage` | `additional_context` | stderr, exit 2 | yes |
| Gemini CLI | `hookSpecificOutput` + `systemMessage` | `hookSpecificOutput` wrapper | stderr, exit 2 | yes |
| GitHub Copilot | `additionalContext` | `additionalContext` | `permissionDecision` JSON, exit 0 | **no** |
| Codex CLI | `hookSpecificOutput` + `systemMessage` | `hookSpecificOutput` wrapper | stderr, exit 2 | yes |
| OpenCode | `hookSpecificOutput` + `systemMessage` | plain text | stderr, exit 2 | yes |

Copilot's `preToolUse` carries only a permission decision, so code-alignment context is discarded
there rather than emitted. Copilot also reads any non-zero exit as a deny and drops the reason with
it, which is why its deny writes JSON and exits 0.

[assumption] The OpenCode SessionStart envelope is unverified. Its tool-event envelope is plain text
per the bridge documentation; the session shape is set to the Claude schema and has not been probed.

Source: `@cmd/hook_dialect.go`, `@cmd/hook_session_start.go`.

## Hook input

The payload arrives as one JSON object on stdin and is read by explicit key paths. A field that is
itself a JSON document in string form is decoded on the way through, which is how Copilot's `toolArgs`
and Cursor's `tool_input` are read.

| Purpose | Keys read, in order |
|---|---|
| Session identity | `session_id`, `conversation_id` |
| Session source | `source` |
| Project root | `cwd` |
| Tool name | `tool_name`, `toolName` |
| Edited file | `tool_input.file_path`, `toolArgs.file_path`, `toolArgs.filePath`, `file_path`, `filePath`, then `tool_input.path` and `toolArgs.path` when the caller is not an archcore MCP tool |
| Document path | `tool_input.path`, `toolArgs.path`, `path` |

An empty, truncated, or non-JSON stdin yields an empty payload, and every guard then allows. Bytes
after the leading JSON object are ignored rather than rejected — rejecting them would produce that
same empty payload, and turn a padded payload into a way past the write guard.

MCP tool names arrive in four spellings and fold to the canonical form: `mcp__archcore__*`,
`mcp__plugin_archcore_archcore__*`, `mcp_archcore_*` (Gemini), and `archcore-*` (Copilot). A tool from
another MCP server is left untouched.

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
