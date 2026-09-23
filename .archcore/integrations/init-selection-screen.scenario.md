---
title: "Selecting Hosts on the init Screen"
status: draft
tags:
  - "actor:developer"
  - "component:cli"
  - "integrations"
---

## Subject

The agent selection screen of `archcore init` (`resolveAgents` and `defaultPickAgents` in `@cli/cmd/init.go`); illustrates clauses 3, 4, 7, 29, 30, 31, and 32 of the plugin-delivery spec. The init implementers and the closeout reviewer depend on it.

## Actors

| Actor | Who they are | What they want |
|---|---|---|
| Dana | a developer running `archcore init` in a repository that already carries `.claude/` | wire the hosts she uses and get the plugin without a second command |

## Flows

### Dana

Anchors: @cli/cmd/init.go, @cli/cmd/init_test.go, @cli/cmd/init_plugin_test.go

1. Dana runs `archcore init` in a terminal; the CLI creates `.archcore/` and opens the selection screen.
2. Dana sees Claude Code pre-checked and every other host unchecked; the screen shows the plugin disclosure.
3. Dana checks Cursor beside Claude Code; the screen keeps both checked.
4. Dana confirms the selection; the CLI wires both hosts and installs the plugin for Claude Code.
5. Dana reads the closing lines; the CLI prints the Cursor UI instruction and the Ready line.

Extensions:

- 2a. Dana runs init with no terminal; the CLI wires Claude Code unscreened and prints the plugin hint.
- 3a. Dana unchecks Claude Code and checks Skip; the CLI wires nothing and prints the Skipped line.
- 4a. Dana passed `--yes`; the CLI wires the checked hosts and prints the install commands instead.

## Examples

Background: Dana's repository carries `.claude/` and no other host marker; `claude` is on `PATH`.

### Confirming the pre-checked host

Illustrates: clauses 29, 30, and 3.
Given Dana ran `archcore init` in a terminal.
When she confirms the screen with Claude Code still checked.
Then she sees the hooks and MCP config written and the plugin installed at user scope.

### Adding a second host beside the detected one

Illustrates: clauses 30 and 5.
Given Dana ran `archcore init` in a terminal.
When she checks Cursor and confirms.
Then she sees both hosts wired and the Cursor UI instruction, with no host command run for Cursor.

### Unchecking the detected host

Illustrates: clauses 31 and 4.
Given Dana ran `archcore init` in a terminal.
When she unchecks Claude Code, checks Skip, and confirms.
Then she sees no host wired and the Skipped line naming a later install command.

### Running without a terminal

Illustrates: clauses 7 and 32.
Given Dana ran `archcore init` from a script with no `/dev/tty`.
When the run reaches host selection.
Then she sees Claude Code wired and one hint naming `archcore plugin install`.

## Open Questions

- Whether a rerun on a fully wired project skips the screen when nothing changed. [assumption] No: the screen is the consent surface on every interactive run, and the pre-checked default costs one keypress.
