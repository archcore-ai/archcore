---
title: "Background Update Trigger in the MCP Server"
status: draft
tags:
  - "cli"
  - "mcp"
  - "update"
---

## Purpose & Scope

This spec defines the **background update trigger inside `archcore mcp`** — the only caller of the unattended update policy in phase 1. It answers "when"; every "whether" belongs to the policy.

Normative for `@cmd/mcp.go` and the server run loop. Dependents: the MCP server run loop, and any host that reads the server's stdout as a JSON-RPC stream. Out of scope: which conditions permit a replacement, and the behavior of `archcore update`.

## Surface

- Trigger site: the `mcp` command, after `RunStdio` begins serving — `@cmd/mcp.go:50`.
- Delay before the attempt: 60 s. [assumption] session startup and the first tool calls complete inside that window.
- Session context: `cmd.Context()`, cancelled by the host closing stdio or by `signal.NotifyContext` in `@main.go`.
- Streams: stdout carries JSON-RPC framing owned by `mcpserver.RunStdio`; stderr carries the banner and warnings the command already writes.
- Policy: the unattended update policy, invoked in-process.

## Normative Behavior

1. WHEN the MCP server begins serving, the server MUST start the update attempt on a background goroutine.
2. The server MUST delay the attempt by 60 s before invoking the policy.
3. The server MUST become ready without waiting for the attempt.
4. WHILE the attempt runs, the server MUST continue answering JSON-RPC requests.
5. The trigger MUST NOT write to stdout.
6. WHEN a replacement completes, the trigger MUST write exactly one line to stderr naming the new version.
7. WHEN a replacement does not complete, the trigger MUST write nothing to stderr.
8. IF the session context is cancelled before the delay elapses, THEN the goroutine MUST exit without invoking the policy.
9. WHEN the session ends, the server MUST NOT wait for an attempt in flight.
10. The trigger MUST run at most once per server process.
11. The server MUST NOT terminate itself after a replacement.
12. The server MUST NOT restart or re-exec itself after a replacement.

## Constraints & Invariants

- Constraint: the same trigger MUST NOT be added to `archcore hooks`. Rationale: that leaf is short-lived, runs on a budget of seconds, and its stdout is the host's protocol channel.
- Constraint: the delay MUST NOT be lowered to zero. Rationale: at zero the attempt competes for network and disk with the host's `initialize` round trip, the one phase of a session where latency is visible.
- Constraint: a session shorter than the delay produces no attempt. One-shot agent runs, host capability probes, and a server the user stops immediately all cancel the context first, and those machines stay on their current version until a session outlives the delay.
- Constraint: the stderr line of requirement 6 lands in the host's server log, not in front of a user. It is an operational signal, and it MUST NOT be treated as the user-facing disclosure that the update-telemetry contract assigns to the published privacy page.
- Constraint: the goroutine MUST NOT hold a reference to any request-scoped state of the server.
- Invariant: the server serves the image it was launched with for its entire life. A replacement changes the file on disk and nothing about the running process.
- Invariant: the new version takes effect at the next launch of `archcore` — the next session, the next hook invocation, or the next command a user types.
- Invariant: concurrent MCP servers on one machine produce at most one replacement, because the policy's claim is keyed by the binary path.

## Failure Behavior

1. IF the policy refuses, THEN the trigger MUST produce nothing observable on either stream.
2. IF the update fails at any stage, THEN the trigger MUST produce nothing observable on either stream. The policy's own telemetry rule applies unchanged.
3. IF the process is killed while an attempt is in flight, THEN the running binary MUST stay intact. A leftover temporary file falls to the policy's own pre-write sweep.
4. IF stderr is closed or unwritable, THEN the trigger MUST discard the line of requirement 6.
5. IF `checkGlobals` returns an error, THEN the trigger MUST NOT run. Rationale: `RunE` returns before `RunStdio` is reached, so a project with a broken global mount never starts the server.

## Conformance

An implementation is conformant when the attempt runs on a background goroutine that never touches stdout, the server reaches ready state and keeps serving regardless of the attempt's progress, the process is neither terminated nor re-executed after a replacement, and cancellation before the delay elapses skips the attempt entirely.

Given a host that opens an MCP session and closes it after ten seconds, when the delay is 60 s, then the goroutine exits on cancellation, no network request is made, and no file is written.
