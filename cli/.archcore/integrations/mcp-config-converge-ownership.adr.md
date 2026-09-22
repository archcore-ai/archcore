---
title: "Converge Merges Only Archcore-Owned MCP Entry Fields"
status: accepted
tags:
  - "config"
  - "integrations"
  - "mcp"
---

## Context

Agent MCP config writers (@internal/agents/mcp_helpers.go) run in two modes: `mcpKeepExisting` (plain install — an existing `archcore` entry is left alone) and `mcpConverge` (`doctor --fix` — an existing entry is brought back to the desired shape). Users legitimately extend the archcore server entry with their own fields — `env` with secrets, timeouts, host-specific keys. A converge that replaced the whole entry object would silently destroy those customizations.

## Decision

Converge **merges, never replaces**. Archcore owns exactly the fields it writes — `command`, `args`, `type` — and overlays only those onto the existing entry (`mergeEntryFields` in @internal/agents/mcp_helpers.go), preserving key order and every field it does not own. Ownership boundaries:

- **Owned fields** (`command`, `args`, `type`): converge rewrites them to the desired values.
- **User fields on the archcore entry** (`env`, timeouts, anything else): survive converge byte-for-byte.
- **Foreign server entries** (any key other than `archcore` under `mcpServers`/`servers`): never touched in any mode.
- **Non-object archcore entry** (corrupt, e.g. a string): replaced wholesale — there is nothing meaningful to preserve.
- A converge that changes nothing is a no-op: the file is not rewritten.

Adding a new archcore-owned field in a future version is allowed (it starts being converged); **removing** a field from the owned set is not enough to delete it from user configs — old values simply stop being managed.

## Alternatives

- **Replace the whole entry object**: simplest, but deletes `env` and other user fields on every `doctor --fix`. Rejected — this was the reviewed defect.
- **Never converge (keep-existing everywhere)**: leaves genuinely broken entries (wrong binary path after relocation) unfixable by `doctor --fix`. Rejected.
- **Prompt per field**: interactive noise for a repair command; `doctor --fix` must be scriptable. Rejected.

## Consequences

- `doctor --fix` is safe to run unattended: it can only correct the three owned fields (pinned by `TestConvergeCursorMCPJSON_PreservesUserFieldsOnEntry` in @internal/agents/cursor_workspacefolder_spec_test.go).
- The `install_host_config` MCP tool inherits the same guarantee, which is part of why it can honestly declare itself non-destructive (see install-host-config-tool-contract.adr).
- Anyone touching the config writers must keep the owned-field list in `mergeEntryFields` in sync with what the entry builders emit.
