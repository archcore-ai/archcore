---
title: "Hook Runtime Contract"
status: accepted
tags:
  - "cli"
  - "integrations"
---

## Purpose & Scope

This spec defines what `archcore hooks <host> <event>` does with a decoded payload: which guard may
block, in what order the work runs, and how the command degrades. Normative for
`@cmd/hook_command.go` and `@cmd/hook_write_guard.go`.

Out of scope, each owned by its own spec: what the process writes and how it exits per host; how the
payload is read into a tool identity and its targets; the text of the session-start recap.

## Surface

`archcore hooks <host> <event>`. `<host>` is `claude-code`, `cursor`, `gemini-cli`, `copilot`,
`codex-cli`, or `opencode`. `<event>` is `session-start`, `pre-tool-use`, or `post-tool-use`. Every
leaf is hidden. The payload arrives as one JSON object on stdin. The verdict leaves as stdout content
and an exit code.

## Normative Behavior

1. The command MUST read the payload from stdin.
2. The command MUST NOT read a positional argument.
3. WHEN the event is `post-tool-use`, the command MUST NOT deny.
4. The write guard MUST be the only guard that denies.
5. The command MUST compute the deny decision before it starts any advisory work.
6. The write guard MUST decide through `docs.GuardWritablePath`, the predicate the MCP write tools
   use.
7. The write guard MUST refuse a write to a document inside a global source mounted from outside the
   store, whose path `docs.GuardWritablePath` never classifies.
8. WHEN a tool call names several targets, the write guard MUST deny the call if any one of them is
   guarded.
9. The write guard MUST read each piece of project state at most once per invocation, however many
   targets that invocation carries.
10. The write guard MUST read that state on demand, not before a verdict needs it.
11. The post-write precision advisory MUST validate its document path through `docs.ValidateReadPath`
    before it opens the file.
12. WHEN `ARCHCORE_DISABLE_INJECTION` is `1`, the command MUST emit no code-alignment context.
13. Each dedup scope MUST claim its stamp in a directory of its own.

## Constraints & Invariants

- Invariant: the zero-value decision allows and writes nothing. Every failure path produces it.
- Constraint: the pre-write deny decision runs no full document scan. It blocks the user while it
  runs.
- Constraint: behavior 4 exists because the allow cases are enumerated and everything else denies.
  `docs.GuardWritablePath` reports four classes of refusal and only two carry a comparable sentinel,
  so a default-allow silently permits the rest.
- Constraint: behavior 8 holds because a patch applies as a unit. Allowing a call whose other targets
  are ordinary would write the guarded one anyway.
- Constraint: behaviors 9 and 10 pull opposite ways and both bind. Read per target, `settings.json`
  cost a maximal patch hundreds of milliseconds against a one-second host budget, and seconds on a
  large config. Read before a verdict needs it, an ordinary source edit — most of what this hook sees
  — pays for the rare case on every write.
- Constraint: a sweep deletes every stamp older than its own window, which is why scopes with
  different windows may not share a directory.

## Failure Behavior

1. IF a handler panics, THEN the command MUST recover, report the recovery on stderr, and allow.
2. IF the host name is unrecognized, THEN the command MUST write an empty stdout and exit 0.
3. IF the event name is unrecognized, THEN the command MUST write an empty stdout and exit 0.
4. IF a positional argument is present, THEN the command MUST ignore it.
5. IF the project root cannot be resolved, THEN the command MUST allow.
6. IF `settings.json` is unreadable or invalid, THEN the command MUST NOT deny for that reason.
7. IF the payload names no target, THEN the write guard MUST allow.

## Conformance

An implementation is conformant when it satisfies behaviors 1–13, holds every invariant, and degrades
per the failure rules. Failure rules 2 and 3 need a zero exit, so failure rule 4 is what keeps an
argument parser from refusing before the handler runs: Copilot reads any non-zero exit as a deny.
