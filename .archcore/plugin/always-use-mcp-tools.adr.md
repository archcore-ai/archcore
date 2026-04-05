---
title: "Always Use MCP Tools for Document Operations"
status: accepted
tags:
  - "architecture"
  - "plugin"
---

## Context

The Archcore Claude Plugin provides skills, commands, and an agent that help users create and manage `.archcore/` documents. These components could either:

1. Instruct Claude to write `.archcore/` files directly using Write/Edit tools
2. Delegate all operations to Archcore MCP tools (`create_document`, `update_document`, `add_relation`, etc.)

Direct file writes are simpler to implement but bypass the entire Archcore toolchain. MCP tools enforce validation, generate templates, manage the sync manifest, auto-discover nearby relations, and ensure consistent frontmatter.

## Decision

All `.archcore/` document operations MUST go through MCP tools. No plugin component — skill, command, or agent — should ever instruct the agent to write `.archcore/` files directly via Write, Edit, or Bash tools.

This applies to:

- Document creation → `create_document`
- Document updates → `update_document`
- Document deletion → `remove_document`
- Relation management → `add_relation`, `remove_relation`
- Document reading → `list_documents`, `get_document`

Enforcement is not advisory-only. A PreToolUse hook will intercept Write/Edit calls targeting `.archcore/**/*.md` and block them with a redirect message pointing to the appropriate MCP tool.

## Alternatives Considered

### Direct file writes

Skills and commands instruct Claude to use Write/Edit on `.archcore/` files. Rejected because:

- Bypasses frontmatter validation (title, status, tags format)
- Bypasses slug format validation
- Skips template generation for document types
- Does not update the sync manifest (`.sync-state.json`)
- Does not trigger nearby-document relation hints
- Creates drift between file content and manifest state

### Hybrid approach

Allow direct writes for updates, require MCP for creation. Rejected because:

- Inconsistent mental model for users and the agent
- Updates could still corrupt frontmatter or break slug conventions
- Partial enforcement is harder to reason about than total enforcement

## Consequences

### Positive

- Validation always applied — malformed documents cannot be created
- Templates always used — required sections present by default
- Sync manifest always consistent — relations and file hashes stay accurate
- Nearby relations auto-discovered on creation
- Single source of truth for document operations
- Simplifies the agent's decision space — one path for all operations

### Negative

- Dependency on MCP server availability (if the server fails, no document operations are possible)
- Slightly more verbose tool calls compared to direct Write
- Cannot use Write/Edit for quick fixes to `.archcore/` files (must go through `update_document`)
