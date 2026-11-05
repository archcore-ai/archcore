---
title: "Hook Payload Reading"
status: accepted
tags:
  - "cli"
  - "component:cli"
  - "integrations"
---

## Purpose & Scope

This spec defines how one hook payload is read into the facts a guard decides on: which tool called, which files that call would write, and which project a tool result names. Normative for `@cli/cmd/hook_payload.go`.

Out of scope, each owned by its own spec: which guard blocks and how the command degrades; what the process writes and how it exits per host.

## Surface

One JSON object on stdin, bounded at 4 MiB, read through explicit key paths. Six hosts send six shapes for the same event, and two nest a JSON document inside a string field. The reader yields a tool name in canonical form, an edited file path, a document path, the targets of an apply-patch document, and the `project` field of an echoed tool result.

## Normative Behavior

1. WHILE the caller is an archcore MCP tool, the reader MUST NOT read the bare `path` key as an edited file path.
2. The reader MUST fold every known archcore MCP tool-name spelling to the canonical `mcp__archcore__` form and leave a tool name from another MCP server unchanged.
3. WHILE a spelling joins the server name to the tool name with a separator that also occurs inside either name, the reader MUST fold it only onto a tool the MCP server registers.
4. WHEN a tool call carries an apply-patch document instead of a file path, the reader MUST yield every file the patch names, including a rename destination.
5. The reader MUST stop scanning a patch document after a fixed number of lines.
6. WHEN a payload echoes a tool result, the reader MUST yield the `project` field of that result from `tool_response`, `toolResult.textResultForLlm`, or `result_json`.

## Constraints & Invariants

- Constraint: behavior 1 is what keeps a sanctioned MCP write from being read as a direct one. The same key means an edited file or an acted-on document depending on who sent it.
- Constraint: behavior 3 bounds three of the five spellings behavior 2 covers. Two delimit the server name with a double underscore; the three host flattenings cannot, because `archcore_docs_create_document` names a server called `archcore_docs` and the string does not say so. Unbounded, behavior 1 would exempt a foreign server's write.
- Constraint: behaviors 2 and 3 pull against each other and behavior 3 wins. A tool the server gains and the fold does not know is denied under a flattened spelling rather than allowed — a denied edit is visible, an unguarded write is not.
- Constraint: behavior 4 is not an edge case. `opencode` enables `apply_patch` and disables `write` and `edit` for `gpt-` models, and `codex-cli` and `copilot` both name `apply_patch` in the matchers `@cli/internal/wiring/hooks_agents.go` installs. A rename is an update of the source plus a destination line, so only the destination line names where the bytes go.
- Constraint: the bound of behavior 5 is part of the contract, because a target past that line is not guarded. It exists because the pre-write guard blocks the user while it runs.
- Constraint: behavior 6 reads three shapes of one result. Claude Code sends the content array in `tool_response`, with `project` in the text, captured on 2026-11-05 as @plugin/test/fixtures/stdin/claude-code/posttooluse-mcp-list-documents.json. Codex sends the result object in `tool_response` (@plugin/test/fixtures/stdin/codex/mcp-list-documents.json) and Copilot the text in `toolResult.textResultForLlm` (@plugin/test/fixtures/stdin/copilot/posttooluse-mcp-update.json); both captures predate the `project` field, so they show the shape and not the field. Cursor sends `result_json`, composed from its hooks reference and not captured. No Gemini CLI payload with a result was observed. The OpenCode bridge @plugin/plugins/opencode/src/bridge.ts does not forward the result it holds. On these two hosts the hook resolves the root from the working directory.
- Constraint: directive matching is looser than the patch format in case and in spacing. This reader cannot see the parser that will apply the patch, and matching one host's exact strictness would make coverage a bet on someone else's code.

## Failure Behavior

1. IF stdin is empty, truncated, or not a JSON object, THEN the reader MUST yield an empty payload.
2. IF bytes follow the leading JSON object, THEN the reader MUST ignore them rather than reject the payload.
3. IF a patch document is empty or unparsable, THEN the reader MUST yield no targets.
4. IF a patch line quotes a directive rather than stating one, THEN the reader MAY yield it as a target.
5. IF a tool result holds more than one content block, or its text is not a JSON object, THEN the reader MUST yield no project.

## Conformance

An implementation is conformant when it satisfies behaviors 1–6, holds every constraint, and degrades per the failure rules. Behavior 3 is verified against the MCP server's own tool registration rather than a copied list. Failure rule 2 is what keeps a padded payload from becoming an empty one, and an empty payload lets every guard allow.
