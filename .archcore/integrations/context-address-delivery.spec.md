---
title: "Context Address Delivery — One Sentence First in Every Agent Channel"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "integrations"
  - "mcp"
  - "multi-host"
---

## Purpose & Scope

This spec defines the context address — the text that tells an agent where to fetch project context — and the channels that carry it to the model. Normative for the managed-block writer (@cli/internal/agents/instructions.go), the MCP server instructions (@cli/internal/mcp/server.go), the session-start recap (`buildSessionContext` in @cli/cmd/hooks_common.go), and the plugin's session-start launcher (@plugin/plugins/archcore/bin/session-start). The pre-edit hint carries no address: it fires on every edit, and its own header line points at its rows.

Out of scope: the rows of the pre-edit hint and of `for_path`; the recap budget and its blocks; the content of the tool descriptions.

## Surface

- The address, one text constant in the CLI, with two forms: the MCP form (`search_documents` with `for_path` set to the file, then a topic search) and the file form (the `.archcore/**/*.rule.md` files first, then a text search of `.archcore/` for the file name).
- The managed block: `<!-- archcore:start -->` … `<!-- archcore:end -->` in `CLAUDE.md`, `AGENTS.md`, and `GEMINI.md`.
- The channels: managed block, MCP server instructions, session-start recap, CLI-missing message of the plugin launcher.

## Normative Behavior

1. The address MUST NOT exceed 500 characters.
2. The address MUST name the `search_documents` call with `for_path`, and the file form for a session without the Archcore MCP tools.
3. Each channel MUST emit the address before any other Archcore text it emits.
4. The MCP server instructions MUST NOT exceed 2 048 characters, the language and global-source notices included.
5. WHEN the writer adds a managed block to a file without one, the writer MUST insert the block before the existing content.
6. WHEN the file already holds a managed block, the writer MUST replace that block in place.
7. The managed block MUST tell the agent to read the project's rules before its first code edit in a session.
8. The managed block MUST tell the agent to report a conflict with an accepted document before it edits.
9. WHEN the `archcore` CLI is absent, the plugin launcher MUST tell the agent to inform the user once and to read `.archcore/` as files.
10. The session-start recap MUST NOT end with a pointer to the MCP server instructions as its only usage guidance.
11. The writer MUST put the block-format version in the block header as `block vN`.
12. WHEN a committed block carries an older version or edited text, the CLI MUST name `archcore instructions install --agent <id>` as the fix, with the agent whose writer refreshes that file.
13. WHEN a committed block carries a newer version than the CLI writes, the CLI MUST name `archcore update` as the fix.
14. WHEN the block in `AGENTS.md` ends past 32 KiB, the CLI MUST tell the reader to move it to the top.
15. The session-start recap and `archcore doctor` MUST NOT rewrite a stale block.

## Constraints & Invariants

- Constraint: Claude Code cuts MCP server instructions at 2 048 characters, and Copilot CLI 1.0.93 drops them for a non-allowlisted server (probe, 2026-10-09).
- Constraint: Codex reads every `AGENTS.md` under one 32 KiB budget and drops the tail without a notice (`project_doc_max_bytes`).
- Constraint: the managed block body MUST NOT exceed 40 lines. Claude Code advises at most 200 lines for a whole `CLAUDE.md`.
- Constraint: the session-start recap MUST report stale blocks at most once per project in 24 hours.
- Invariant: every CLI channel takes the address from one constant. The plugin launcher copies it, and a test compares the copy with the constant.
- Invariant: the address names tools and paths only. It names no plugin command, because a CLI-only host has no plugin.
- Invariant: a header without `block vN` reads as version 1, the form CLIs before block v2 wrote.
- Invariant: a test pins the block body to its version, so a body change without a version bump fails the build.

## Failure Behavior

1. IF the managed-block write fails, THEN the writer MUST leave the file unchanged.
2. IF an instruction file holds an orphaned start marker, THEN the writer MUST treat it as user content and insert a new block before it.
3. IF the server instructions would exceed 2 048 characters, THEN the build MUST fail the instruction-size test.

## Conformance

An implementation is conformant when it satisfies behaviors 1–15, holds every constraint and invariant, and degrades per the failure rules. Checks: @cli/internal/agents/instructions_fixtures_test.go, @plugin/test/unit/session-start-goldens.bats, and the server-instruction size test beside @cli/internal/mcp/tool_description_spec_test.go.
