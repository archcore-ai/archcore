---
title: "Hook Runtime Contract"
status: accepted
tags:
  - "cli"
  - "integrations"
---

## Purpose & Scope

This spec defines how `archcore hooks <host> <event>` behaves at run time: what it writes, when it
blocks, and how it fails. Normative for `@cmd/hook_command.go`, `@cmd/hook_dialect.go`, and
`@cmd/hook_payload.go`. Out of scope: the session-start payload text, owned by the session-start
context spec, and the install-time wiring in `@internal/wiring/`.

## Surface

`archcore hooks <host> <event>`. `<host>` is `claude-code`, `cursor`, `gemini-cli`, `copilot`,
`codex-cli`, or `opencode`. `<event>` is `session-start`, `pre-tool-use`, or `post-tool-use`. Every
leaf is hidden. The payload arrives as one JSON object on stdin. The verdict leaves as stdout content
and an exit code.

## Normative Behavior

1. The command MUST read the payload from stdin.
2. The command MUST NOT read a positional argument.
3. WHEN a guard blocks a tool call on a host whose deny style is exit-code based, the command MUST
   write the reason to stderr, write an empty stdout, and exit 2.
4. WHEN a guard blocks a tool call on Copilot, the command MUST write one `permissionDecision`
   document to stdout and exit 0.
5. The command MUST write at most one JSON document to stdout.
6. The command MUST write every diagnostic to stderr.
7. WHEN the event is `post-tool-use`, the command MUST NOT deny.
8. WHEN the host cannot carry context on its pre-write event, the command MUST discard the advisory
   context instead of emitting it.
9. The command MUST fold every known archcore MCP tool-name spelling to the canonical
   `mcp__archcore__` form.
10. The command MUST leave a tool name from another MCP server unchanged.
11. The command MUST read the edited file path from explicit payload keys, never from the raw
    payload text.
12. WHILE the caller is an archcore MCP tool, the command MUST NOT read the bare `path` key as an
    edited file path.
13. The write guard MUST decide through `docs.GuardWritablePath`, the predicate the MCP write tools
    use.
14. The write guard MUST refuse a write to a document inside a global source mounted from outside
    the store, whose path `docs.GuardWritablePath` never classifies.
15. The write guard MUST be the only guard that denies.
16. The command MUST compute the deny decision before it starts any advisory work.
17. The post-write precision advisory MUST validate its document path through
    `docs.ValidateReadPath` before it opens the file.
18. WHEN `ARCHCORE_DISABLE_INJECTION` is `1`, the command MUST emit no code-alignment context.
19. Each dedup scope MUST claim its stamp in a directory of its own.

## Constraints & Invariants

- Invariant: the zero-value decision allows and writes nothing. Every failure path produces it.
- Invariant: on Copilot stdout holds exactly one JSON document. The host strips single-line progress
  objects, concatenates the rest, and runs one `JSON.parse`. A failed parse discards the whole
  payload, so a stray line costs the payload rather than the line.
- Constraint: Copilot reads any non-zero exit as a deny and discards the reason with it. The zero
  exit is what carries the explanation to the user.
- Constraint: the pre-write deny decision runs no full document scan.
- Constraint: a sweep deletes every stamp older than its own window, which is why scopes with
  different windows may not share a directory.
- Constraint: the payload decoder reads the leading JSON object and ignores what follows. Rejecting
  a padded payload would make it empty, and an empty payload allows — the strict reading is weaker.

## Failure Behavior

1. IF stdin is empty, unparsable, or not a JSON object, THEN the command MUST allow.
2. IF a handler panics, THEN the command MUST recover, report the recovery on stderr, and allow.
3. IF the host name is unrecognized, THEN the command MUST write an empty stdout and exit 0.
4. IF the event name is unrecognized, THEN the command MUST write an empty stdout and exit 0.
5. IF a positional argument is present, THEN the command MUST ignore it. Behaviors 3 and 4 need a
   zero exit for an unrecognized host or event, an argument parser that refuses first exits before
   the handler runs, and Copilot reads that non-zero exit as a deny.
6. IF the project root cannot be resolved, THEN the command MUST allow.
7. IF a JSON document cannot be marshalled, THEN the command MUST write nothing, not a fragment.
8. IF `settings.json` is unreadable or invalid, THEN the command MUST NOT deny for that reason.

## Conformance

An implementation is conformant when it satisfies behaviors 1–19, holds every invariant, and degrades
per the failure rules. The single-document invariant is verified by driving a diagnostic-producing
path and parsing stdout as one object.
