---
title: "Hook Runtime Contract"
status: accepted
tags:
  - "cli"
  - "component:cli"
  - "integrations"
---

## Purpose & Scope

This spec defines what `archcore hooks <host> <event>` does with a decoded payload: which guard may
block, in what order the work runs, and how the command degrades. Normative for
`@cli/cmd/hook_command.go` and `@cli/cmd/hook_write_guard.go`.

Out of scope, each owned by its own spec: what the process writes and how it exits per host; how the
payload is read into a tool identity and its targets; the text of the session-start recap.

## Surface

`archcore hooks <host> <event>`, hidden leaves. The payload arrives as one JSON object on stdin. The verdict leaves as stdout content and an exit code.

## Normative Behavior

1. The command MUST read the payload from stdin only.
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
11. The post-write precision advisory MUST validate its document path through `docs.ValidateReadPath` before it opens the file.
12. WHEN `ARCHCORE_DISABLE_INJECTION` is `1`, the command MUST emit no code-alignment context.
13. Each dedup scope MUST claim its stamp in a directory of its own.
14. WHEN a target path is absolute, the write guard MUST resolve the project root as the nearest ancestor directory of that path that holds `.archcore/`, per @.archcore/mcp/call-root-resolution.spec.md.
15. WHEN a target path is relative, the write guard MUST read the path against the payload working directory before it resolves the project root per behavior 14.
16. WHEN the resolved root of a target differs from the session root, the pre-tool-use command MUST emit one advisory per session, root, and host naming the `project` argument, claimed through a dedup stamp keyed by session id, root, and host.
17. WHEN the event is `post-tool-use`, the command MUST validate the document under the first root below that is an absolute directory holding `.archcore/`:
    - the `project` field of the tool result;
    - the `project` argument in `tool_input`;
    - the nearest ancestor of the payload working directory that holds `.archcore/`.
18. WHEN a `pre-tool-use` payload names no file path and carries a patch, the command MUST feed the first patch target to the code-alignment advisory.

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
- Constraint: behavior 17 reads the payload working directory last. That directory is the shell's,
  and a `cd` moves it away from the root the server wrote to. The tool result names that root
  (@.archcore/mcp/call-root-resolution.spec.md behavior 10). Read first, the working directory
  reported `.archcore/ directory not found` after a write from a subdirectory, and checked a nested
  project instead of the written one.

## Failure Behavior

1. IF a handler panics, THEN the command MUST recover, report the recovery on stderr, and allow.
2. IF the host or event name is unrecognized, THEN the command MUST write an empty stdout and exit 0.
3. IF a positional argument is present, THEN the command MUST ignore it.
4. IF the project root cannot be resolved from the working directory and no target path has an ancestor holding `.archcore/`, THEN the command MUST allow.
5. IF `settings.json` is unreadable or invalid, THEN the command MUST NOT deny for that reason.
6. IF the payload names no target, THEN the write guard MUST allow.
7. IF no root that behavior 17 lists is an absolute directory holding `.archcore/`, THEN the `post-tool-use` command MUST allow.

## Conformance

An implementation is conformant when it satisfies behaviors 1–18, holds every invariant, and degrades
per the failure rules. Failure rule 2 needs a zero exit, so failure rule 3 is what keeps an argument parser from refusing before the handler runs: Copilot reads any non-zero exit as a deny.
