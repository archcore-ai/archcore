---
title: "Hook Payload Reading"
status: accepted
tags:
  - "cli"
  - "integrations"
---

## Purpose & Scope

This spec defines how one hook payload is read into the facts a guard decides on: which tool called,
and which files that call would write. Normative for `@cmd/hook_payload.go`.

Out of scope, each owned by its own spec: which guard blocks and how the command degrades; what the
process writes and how it exits per host.

## Surface

One JSON object on stdin, bounded at 4 MiB, held as a generic tree and read through explicit key
paths. Six hosts send six shapes for the same event, and two nest a JSON document inside a string
field. The reader yields a tool name in canonical form, an edited file path, a document path, and
the targets of an apply-patch document.

## Normative Behavior

1. The reader MUST read the edited file path from explicit payload keys, never from the raw payload
   text.
2. WHILE the caller is an archcore MCP tool, the reader MUST NOT read the bare `path` key as an
   edited file path.
3. The reader MUST fold every known archcore MCP tool-name spelling to the canonical
   `mcp__archcore__` form.
4. The reader MUST leave a tool name from another MCP server unchanged.
5. WHILE a spelling joins the server name to the tool name with a separator that also occurs inside
   either name, the reader MUST fold it only onto a tool the MCP server registers.
6. WHEN a tool call carries an apply-patch document instead of a file path, the reader MUST yield
   every file the patch names.
7. The reader MUST yield a rename destination as a file the patch names, not only the source it
   renames.
8. The reader MUST stop scanning a patch document after a fixed number of lines.

## Constraints & Invariants

- Constraint: behavior 2 is what keeps a sanctioned MCP write from being read as a direct one. The
  same key means an edited file or an acted-on document depending on who sent it.
- Constraint: behavior 5 bounds three of the five spellings behavior 3 covers. Two delimit the server
  name with a double underscore; the three host flattenings cannot, because
  `archcore_docs_create_document` names a server called `archcore_docs` and the string does not say
  so. Unbounded, behavior 2 would exempt a foreign server's write.
- Constraint: behaviors 3 and 5 pull against each other and behavior 5 wins. A tool the server gains
  and the fold does not know is denied under a flattened spelling rather than allowed — a denied edit
  is visible, an unguarded write is not.
- Constraint: behavior 6 is not an edge case. `opencode` enables `apply_patch` and disables `write`
  and `edit` for `gpt-` models, and `codex-cli` and `copilot` both name `apply_patch` in the matchers
  `@internal/wiring/hooks_agents.go` installs.
- Constraint: behavior 7 is what makes behavior 6 complete. A rename is an update of the source plus
  a destination line, so the other directives name only where the bytes came from.
- Constraint: behavior 8's bound is part of the contract rather than an implementation detail,
  because a target past that line is not guarded. It exists because the pre-write guard blocks the
  user while it runs.
- Constraint: directive matching is looser than the patch format in case and in spacing. This reader
  cannot see the parser that will apply the patch, and matching one host's exact strictness would
  make coverage a bet on someone else's code.

## Failure Behavior

1. IF stdin is empty, truncated, or not a JSON object, THEN the reader MUST yield an empty payload.
2. IF bytes follow the leading JSON object, THEN the reader MUST ignore them rather than reject the
   payload.
3. IF a patch document is empty or unparsable, THEN the reader MUST yield no targets.
4. IF a patch line quotes a directive rather than stating one, THEN the reader MAY yield it as a
   target.

## Conformance

An implementation is conformant when it satisfies behaviors 1–8, holds every constraint, and degrades
per the failure rules. Behavior 5 is verified against the MCP server's own tool registration rather
than a copied list. Failure rule 2 is what keeps a padded payload from becoming an empty one, and an
empty payload lets every guard allow.
