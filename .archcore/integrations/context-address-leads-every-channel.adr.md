---
title: "The Context Address Leads Every Agent Channel, and Server Instructions Fit 2,048 Characters"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "integrations"
  - "mcp"
  - "multi-host"
---

## Context

On 2026-10-09 a marker probe showed Claude Code 2.1.295 passing 5 of 40 instruction markers (a cut at 2 048 characters) and Copilot CLI 1.0.93 passing none from a non-allowlisted server, so the 19 539-character server instructions in @cli/internal/mcp/server.go reach the model whole only on Codex and on Copilot in VS Code. Codex reads every `AGENTS.md` under one 32 KiB budget and drops the tail without a notice (`project_doc_max_bytes`), while `upsertFencedBlock` in @cli/internal/agents/instructions.go appends a new managed block after the user's content. The managed block names no tool call and no route for a session without the MCP tools, and the session-start recap in @cli/cmd/hooks_common.go ends by pointing at the server instructions that two hosts never show.

## Decision

Adopted one context address of at most 500 characters — the `search_documents` call that returns the context for a file, plus the plain-file route over `.archcore/` rules — placed first in the managed block, the MCP server instructions, the session-start recap, and the plugin's CLI-missing message; server instructions shrink to at most 2 048 characters including the language and global-source notices, and a newly written managed block goes to the top of `CLAUDE.md`, `AGENTS.md`, and `GEMINI.md` while an existing block keeps its position (user choice, 2026-10-09).

## Alternatives Considered

1. Keep the 19 539-character server instructions as the teaching channel — rejected because the probe delivered 5 of 40 markers on Claude Code and 0 of 40 on Copilot CLI.
2. Ask Copilot users to pass `--allow-all-mcp-server-instructions` — ruled out because it is a launch flag of the user's session that neither the plugin nor `archcore init` controls.
3. Move an existing managed block to the top on every upsert — rejected by the user on 2026-10-09 because it reorders user-curated files on every `archcore init`.
4. Keep appending a new block at the end of the file — rejected because a combined `AGENTS.md` over 32 KiB loses the block on Codex without a warning.
5. Deliver the address only through hooks — rejected because Codex caps hook context at 2 500 tokens per handler and Bash edits fire no edit hook.

## Consequences

Positive:

- The address reaches the model on every probed host (probe, 2026-10-09): through the server instructions on Claude Code and Copilot CLI, and through the session-start recap on Codex, which shows the server instructions only after tool search loads the namespace.
- The language and global-source notices move inside the 2 048 characters that Claude Code shows; today both sit after character 19 500.
- [expected] The session-start recap and the CLI-missing message tell the agent what to call instead of pointing to text it cannot see.

Negative:

- The document-type catalog, the relation conventions, and the type-selection rules leave the server instructions. They must reach the agent through tool descriptions and the plugin skills; `create_document` already exceeds the cap at 3 696 characters.
- A project whose existing block sits at the end of a large `AGENTS.md` keeps the Codex risk until the user moves the block.
- [expected] The same 500-character address appears in up to four channels, about 2 000 characters per session at most. The pre-edit hint stays without it, because it fires on every edit.

## Superseded when

- Claude Code delivers MCP server instructions longer than 2 048 characters by default.
- Copilot CLI includes the instructions of a non-allowlisted MCP server by default.
- A supported host is measured to drop or cut the managed block of an instruction file under 32 KiB.
