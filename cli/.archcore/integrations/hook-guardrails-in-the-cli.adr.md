---
title: "Run the Hook Guardrails Inside the CLI Binary, Not as Plugin Shell Scripts"
status: accepted
tags:
  - "cli"
  - "integrations"
  - "mcp"
---

## Context

The guardrails that surround a document write — the direct-write block, the pre-write context
injection, the post-write validation, the relation cascade notice, the precision check, and the
staleness advisory — were shell scripts in the `archcore/plugin` repository, wired onto four hosts.

Two consequences followed.

A user who installs only the CLI receives none of them. That user gets no write protection, no
code-alignment context before an edit, and no validation after a document mutation. The CLI wired
one event, `SessionStart`, and nothing else.

The same rules had two owners. The precision canon existed as `skills/_shared/precision-rules.md` in
the plugin and had no counterpart in the CLI, so a change to the document contract in this repository
did not reach the check that measures against it. Two copies of one contract drift.

The shell implementation also carried defects that a text-matching approach makes easy to write: the
write guard searched the raw payload for the first occurrence of a key rather than reading a known
path, and it treated any directory whose name ends in `.archcore` as the document store.

## Decision

Port the guardrails into the Go binary as handlers behind `archcore hooks <host> <event>`.

- Three events per host: `SessionStart`, `PreToolUse`, `PostToolUse`, in each host's own spelling.
- One archcore-owned hook entry per (host, event) pair. The process dispatches by tool name
  internally, so three post-write checks cost one process start.
- The write guard calls `docs.GuardWritablePath`, the same predicate the MCP write tools consult. An
  editor write and an MCP mutation are judged by one implementation, so a path the MCP tools refuse
  cannot be reached by going around them.
- The precision canon lives in `@templates/precision.go`, beside the templates it measures against.
- The blocking guard and the advisory guards stay separate. Only the write guard can deny.

The plugin keeps the interview and track logic, which is prompt work with no CLI equivalent.

## Alternatives

**Keep the guardrails in the plugin.** Rejected: it leaves every CLI-only user unprotected, and it
keeps the document contract and the check that enforces it in two repositories.

**Ship the shell scripts with the CLI.** Rejected: the scripts need a POSIX shell and `grep`, so
Windows loses them, and the per-host wiring would still be written twice.

**Enforce only inside the MCP write tools.** Rejected: the MCP layer never sees an editor writing a
file directly, which is the case the write guard exists for.

## Consequences

- A CLI-only installation now carries the same protection as a plugin installation.
- The write guard's segment-aware path matching removes the false positive on a directory whose name
  merely ends in `.archcore`.
- While an old plugin stays installed, its hooks and the CLI's both fire. The result is duplicated
  advisory output and duplicated denies — extra tokens, not a wrong verdict. The compatibility rule
  covers the overlap.
- The plugin must reduce its precision canon to a pointer, or the two copies reproduce the problem
  this decision ends.
- Copilot's `preToolUse` carries only a permission decision, so code-alignment context cannot be
  delivered on that host. This is a host limitation, reported rather than worked around.
- Every hook handler now runs inside a process that must never block the user by accident. The hook
  runtime contract states the fail-open obligations that follow.
