---
title: "Always Use MCP Tools for Document Operations"
status: accepted
tags:
  - "architecture"
  - "plugin"
---

## Context

The plugin ships skills, commands, and agents that help users create and manage `.archcore/` documents, and each of them could either instruct the agent to write those files directly with Write and Edit, or delegate every operation to the Archcore MCP tools. Direct file writes are simpler to implement but bypass the whole Archcore toolchain, which enforces frontmatter and slug validation, generates templates for each document type, maintains `.sync-state.json`, and auto-discovers relations between documents that share a directory. A document written directly can therefore carry invalid frontmatter, miss required sections, hold no manifest entry, and drift from the manifest state that `archcore doctor` checks.

## Decision

Every `.archcore/` document operation MUST go through an MCP tool — `create_document`, `update_document`, `remove_document`, `add_relation`, `remove_relation`, `list_documents`, `get_document` — and no plugin component may instruct the agent to write a `.archcore/` file through Write, Edit, or Bash, with a PreToolUse hook intercepting such a call and blocking it with a redirect message naming the MCP tool to use instead.

Enforcement is not advisory. The blocking hook makes the rule structural rather than a matter of prompt discipline.

## Alternatives Considered

1. **Direct file writes, with skills and commands instructing Write or Edit on `.archcore/` files** — rejected because it bypasses frontmatter validation for title, status, and tag format; bypasses slug-format validation; skips template generation per document type; leaves `.sync-state.json` unupdated; does not trigger nearby-document relation hints; and creates drift between file content and manifest state.
2. **A hybrid that allows direct writes for updates and requires MCP for creation** — rejected because it gives users and the agent an inconsistent mental model, because an update can still corrupt frontmatter or break slug conventions, and because partial enforcement is harder to reason about than total enforcement.

## Consequences

- Validation always applies, so a malformed document cannot be created.
- Templates always apply, so the required sections are present by default.
- The sync manifest stays consistent, so relations and file hashes stay accurate, and nearby relations are discovered at creation.
- The agent's decision space narrows to one path for every operation.
- Tradeoff: document operations depend on MCP server availability. When the server fails, no document operation is possible, which is why `bin/session-start` surfaces a missing CLI at session boot.
- Tradeoff: tool calls are more verbose than a direct Write, and a quick fix to a `.archcore/` file must go through `update_document`.

## Superseded when

- A host ships a file-write mechanism that applies the same validation, template, and manifest guarantees, which would remove the reason the redirect exists.
- The sync manifest is eliminated in favor of state derived entirely from the files on disk, which would remove the drift class this decision prevents.
