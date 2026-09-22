---
title: "Compatibility Contract Between the Archcore CLI and the Archcore Plugin"
status: accepted
tags:
  - "cli"
  - "integrations"
  - "mcp"
---

## Rule

The Archcore plugin lives in the separate `archcore-ai/plugin` repository and ships on its own schedule.
Any CLI release therefore meets an unknown plugin version, and any plugin release meets an unknown
CLI version. These obligations bind the CLI side of that pair.

1. The CLI MUST keep the verbs the plugin invokes: `mcp`, `hooks`, `doctor`, and `--version`.
2. WHEN the CLI adds a field to an MCP tool response, the field MUST be an addition. The CLI MUST NOT
   remove or repurpose `source_id`, `source_kind`, or `read_only`.
3. Outside the plugin surface, the CLI MUST NOT change what it does based on whether a plugin is
   installed. The plugin surface — the `archcore plugin` command and the plugin steps of
   `archcore update` and `archcore init` — exists to act on the plugin: it MAY read the host's own
   plugin state, and it MUST NOT change any behavior outside that surface.
4. WHEN the CLI detects an installed plugin during `hooks install` or `doctor`, the CLI MUST report
   that hooks may run twice until the plugin is updated.
5. IF plugin detection fails, THEN the CLI MUST omit the notice and MUST NOT change any other
   behavior.
6. WHEN the plugin and the project config both route an event to this binary, the CLI MUST
   deduplicate the run through the dedup stamp.
7. WHEN `archcore hooks` receives an unrecognized host or an unrecognized event, the CLI MUST write
   an empty stdout and MUST exit 0.
8. A version mismatch MUST produce a message, and MUST NOT block the session.
9. WHEN the CLI changes the wire protocol of an existing `archcore hooks <host> <event>` leaf, the
   change MUST ship in a new release, and the release notes MUST name the host and the leaf. A leaf
   whose protocol changed inside a version the plugin already gates on cannot be detected by that
   gate.
10. WHEN the CLI narrows what a leaf accepts on a host that already ships, the release notes MUST name
    the narrowing. Requirement 9 covers a leaf that answers differently; this covers one that answers
    less, which an existence gate cannot see either.
11. The CLI MUST NOT change `archcore-ai/plugin`, `archcore-plugins`, or `archcore@archcore-plugins`
    except in step with the plugin repository. These are the plugin's public identifiers, in the same
    spirit as requirement 9: a released CLI carrying a renamed identifier addresses a plugin that no
    longer answers to it.

## Rationale

Requirement 7 is the one that closes a real defect. Cobra answers an unknown subcommand by printing
usage to stdout and exiting 0. On a hook, stdout is the protocol channel, so a plugin calling a host
or event name that a given CLI does not have would deliver several hundred bytes of help text
straight into the model's context. Answering with silence makes an unknown name harmless, which is
what lets a newer plugin call a leaf an older CLI lacks.

Requirements 3 and 5 keep the CLI independent of another repository's install layout. Detection reads
a cache directory the plugin owns; treating a detection miss as a behavior switch would make the CLI
break when that layout changes. The plugin surface carve-out in requirement 3 exists because a
command whose job is installing, updating, or reporting the plugin cannot do that job blind: it reads
the host's own answer — a host CLI's exit and output, or the host's plugin registry — and acts only
within the plugin surface. Skipping a host that answers "not installed" is that surface doing its
job, not a behavior switch.

Requirement 6 bounds the overlap cost. While an old plugin is installed, its own hooks and the CLI
entries both fire, and the result is duplicated advisory output and duplicated denies — extra tokens
and repeated messages, not a wrong verdict or a corrupted file. Copilot parses each hook entry
independently, so the payload survives the duplication. Once the plugin delegates to
`archcore hooks <host> <event>`, both entries reach the same binary and the stamp collapses them.

Requirement 9 comes from the OpenCode leaves. They existed in v0.7.0 and answered in the wrong
protocol: the session event emitted a JSON envelope into a channel the bridge appends verbatim, and
the host's `archcore_*` MCP spelling was unfolded, so the write guard denied the document tools it
was meant to protect. A plugin gating on `cli-gte 0.7.0` passes on a CLI with both defects, because
the gate can only ask whether the leaf exists, not whether it answers correctly. A bridge that needs
the corrected protocol must gate on the release that carries it.

Requirement 10 comes from the fix to requirement 9's second defect. Folding OpenCode's `archcore_*`
spelling meant bounding the loose spellings to this server's own tool set, and Gemini CLI and GitHub
Copilot use loose spellings too. Their leaves now decline to fold a tool the server does not
register, which is a narrowing on hosts that were already shipping. Nothing observable from the
plugin distinguishes the narrowed leaf from the old one until a tool falls outside the set, so the
release notes are the only place it can be seen.

An older CLI meeting a newer plugin cannot be fixed from this repository; already released binaries
are fixed. The plugin closes that cell with its own minimum-version gate.

## Expectations on the plugin

Descriptive, not normative for this repository. Recorded here so the CLI side knows what it may rely
on. The plugin repository guards these with a property-based check.

- The plugin never writes `globals` into `settings.json`.
- Executable code under `bin/` does not branch on `source_id`, `source_kind`, or `read_only`, because
  an older CLI does not send them.
- A skill may read those fields, and every clause is data-gated: when the fields are absent, behavior
  matches the path without global sources.
- The plugin raises its minimum-CLI gate and turns its `bin/check-*` scripts into delegators once the
  CLI ships the equivalent guardrails.
- The OpenCode bridge gates above v0.7.0. Its two protocol fixes — the plain-text session envelope
  and the `archcore_*` fold — landed after that tag, and v0.7.0 already answers the leaves, so the
  existence gate cannot tell the two apart.

## Examples

Non-normative examples.

### Good

```
$ archcore hooks unknown-host session-start
$ echo $?
0
```

```
Warning: An Archcore plugin is installed (.claude/plugins/archcore). Until it is updated, its own
hooks and these may both fire and you will see duplicated context. Updating the plugin resolves it.
```

### Bad

```
$ archcore hooks unknown-host session-start
Usage:
  archcore hooks [command]
...
```

Usage text on stdout reaches the model as session context.

## Enforcement

- `@cmd/hook_command.go` — `Args: cobra.ArbitraryArgs` with a silent `RunE` on the host group, so an
  unrecognized event never reaches cobra's help printer.
- `@cmd/hooks.go` — the same guard on the `hooks` group itself.
- `@internal/wiring/hooks_effective.go` — `detectInstalledPlugin`, `EffectiveHookNotes`.
- `@cmd/hook_session_start.go` and `@internal/stamp/` — the dedup claim shared by every hook scope.
- `@cmd/hook_dialect.go` and `@cmd/hook_payload.go` — the per-host wire protocol requirements 9 and
  10 govern.