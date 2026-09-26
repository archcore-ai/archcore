---
title: "update_document Accepts Exact-Match edits Instead of Requiring a Full Body"
status: draft
tags:
  - "component:cli"
  - "mcp"
  - "performance"
---

## Context

In 410 `update_document` calls from the maintainer's Claude Code transcripts, the median new body kept 91% of the old body unchanged, and model generation cost about 4 s per 1000 characters while the tool itself ran in about 0.1 s (measured 2026-09-26). A change of a few sentences therefore cost as much as rewriting the whole document. The accepted `always-use-mcp-tools` decision keeps every `.archcore/` write on the MCP tools, so the cheaper update has to live in `update_document` itself (@cli/internal/mcp/tools/update_document.go).

## Decision

`update_document` accepts an `edits` array of exact-match `{old_string, new_string}` replacements that the handler applies in order to the body, writes all-or-nothing, and refuses together with `content`, as implemented in @cli/internal/mcp/tools/update_document.go.

## Alternatives Considered

1. Native `Write` and `Edit` on `.archcore/` with validation after the write — rejected because `always-use-mcp-tools` forbids direct writes, and `Write` generates at the same rate as `create_document` (4.2 against 4.4 s per 1000 characters), so only the edit path would gain.
2. A cheaper model or parallel writer agents for document packages — ruled out because in a writer fan-out bench on 2026-09-26 cheaper writers were slower (Claude Code 348 s against 170 s; Codex 263 s against 206 s), and the fastest parallel variant saved 15% at 2.2 times the cost on Claude Code.

## Consequences

Positive:

- On Claude Code with `claude-opus-5-5`, one review round of three comments took a median 15 s per document instead of 32 s, with 717 instead of 3221 output tokens and 0.24 instead of 0.64 USD (update bench, 12 tasks per variant, 2026-09-26).
- Agents chose `edits` in 24 of 24 bench tasks without an instruction to use it, and all 48 bench tasks passed their checks.
- The update payload fell from a median 7835 to 818 characters on Claude Code and from 7837 to 1297 on Codex.

Negative:

- On Codex with `gpt-6-astra`, the same round took a median 40 s instead of 44 s: about 28 s of each task is fixed overhead before the read and after the write, and the full 13k-character body took about 9 s to generate.
- [expected] A CLI released before `edits` ignores the argument, so a call that combines `edits` with `status` on that CLI writes the status and drops the edits without an error.
- The tool now has two ways to change a body, and the handler refuses a call that uses both.

## Superseded when

- A supported host lets agents write `.archcore/` files with the same validation, which is the trigger that supersedes `always-use-mcp-tools`.
- The median `edits` payload in the update bench rises above 50% of the full body (10% on Claude Code on 2026-09-26).
