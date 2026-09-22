---
title: "Hook Wire Protocol Per Host"
status: accepted
tags:
  - "cli"
  - "integrations"
---

## Purpose & Scope

This spec defines what one hook invocation writes and how it exits, per host. Normative for
`@cmd/hook_dialect.go`.

Six hosts disagree on how a hook answers: which envelope carries context, whether a deny is an exit
code or a document, and whether stdout is parsed at all. A host that does not recognize the envelope
reads no context, and one that misreads an exit code blocks the user's edit.

Out of scope, each owned by its own spec: which guard blocks and how the command degrades; how the
payload is read; the text of the session-start recap.

## Surface

One `hostDialect` row per host, holding its session-start envelope, its context envelope, its deny
style, and whether its pre-write event can carry context. Stdout carries the protocol and stderr
carries everything else. `claude-code`, `cursor`, `gemini-cli`, `codex-cli`, and `copilot` parse
stdout as JSON; `opencode` appends it to the session verbatim.

## Normative Behavior

1. WHEN a guard blocks a tool call on a host whose deny style is exit-code based, the command MUST
   write the reason to stderr, write an empty stdout, and exit 2.
2. WHEN a guard blocks a tool call on Copilot, the command MUST write one `permissionDecision`
   document to stdout and exit 0.
3. The command MUST write at most one JSON document to stdout.
4. The command MUST write every diagnostic to stderr.
5. WHEN the host cannot carry context on its pre-write event, the command MUST discard the advisory
   context instead of emitting it.
6. WHEN the host consumes stdout verbatim, the command MUST write context as plain text on every
   event, including `session-start`.
7. WHEN the host consumes stdout verbatim, the command MUST NOT write a JSON envelope.
8. WHEN the host's envelope has no slot for the SessionStart banner, the command MUST drop the
   banner rather than merge it into the context.

## Constraints & Invariants

- Invariant: on Copilot stdout holds exactly one JSON document. The host strips single-line progress
  objects, concatenates the rest, and runs one `JSON.parse`; a failed parse discards the whole
  payload, so a stray line costs the payload rather than the line.
- Constraint: Copilot reads any non-zero exit as a deny and discards the reason with it. The zero
  exit of behavior 2 is what carries the explanation to the user.
- Constraint: behaviors 6 and 7 hold for `opencode`, whose bridge appends stdout to the session with
  no JSON parse anywhere on the path. An envelope there does not frame the recap — it reaches the
  model as literal JSON with the recap escaped inside it.
- Constraint: behavior 8 follows from that same channel. Every byte written becomes model input, and
  the banner is a line for the user, so the channel has no slot for it.
- Constraint: behavior 5 costs Copilot its code-alignment context, because its pre-write event
  carries a permission decision and nothing else.

## Failure Behavior

1. IF a JSON document cannot be marshalled, THEN the command MUST write nothing, not a fragment.
2. IF the context is empty on a host that consumes stdout verbatim, THEN the command MUST write
   nothing rather than a bare newline.

## Conformance

An implementation is conformant when it satisfies behaviors 1–8, holds every invariant, and degrades
per the failure rules. The single-document invariant is verified by parsing stdout as one object;
behaviors 6 and 7 are verified in the opposite direction, by requiring that parse to fail.
