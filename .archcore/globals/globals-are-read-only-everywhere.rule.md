---
title: "Globals Are Read-Only Everywhere Outside the MCP Read Tools"
status: accepted
tags:
  - "globals"
  - "mcp"
---

## Rule

A path is global when `settings.json` declares it as a global source, or when it sits under the reserved `.archcore/global/` directory. `isReadOnlyGlobalPath` in `@internal/mcp/tools/common.go` answers that question for every guard.

1. The MCP read tools `list_documents`, `get_document`, and `search_documents` MUST expose global documents and MUST tag each one with `source_kind: "global"`, `read_only: true`, and a `source_id`.
2. `create_document`, `update_document`, and `remove_document` MUST reject any path that is global.
3. `add_relation` MUST reject an edge whose source or target is a global document, in either direction.
4. `remove_relation` MUST stay open for global endpoints, so that a pre-existing edge can be removed.
5. `archcore status` MUST report tag hygiene and counts for local documents only.
6. The SessionStart context MUST inject local documents only.

## Rationale

One predicate answers "is this global?" for every write guard and relation guard, so the behavior holds for a declared external source and an in-tree `.archcore/global/` document alike, with no corner cases.

Read-only is a property of the mount, not of the repository. The same repository opened directly is fully writable; mounted as another project's global source it is read-only, and it never accumulates relation edges from its consumers. That keeps the reference direction one-way: local to global.

Excluding globals from `status` and from the session context removes "fix this tag" signals that a consumer cannot act on, and keeps the injected context focused on what the project owns.

## Examples

Non-normative examples.

### Good

```text
An agent reads a company logging rule via get_document (read_only: true) and applies it.
To record that a local plan follows it, the agent links local → local documents,
never local → the global.
```

### Bad

```text
add_relation(source: local-plan, target: .archcore/global/company/logging.rule.md)
  → rejected: "cannot add a relation involving a read-only global source document
     — relations connect local documents only"

create_document(directory: "global/foo")
  → rejected: reserved global mount space
```

## Enforcement

- `isReadOnlyGlobalPath` (`@internal/mcp/tools/common.go`) backs the write guards (`@internal/mcp/tools/create_document.go`, `@internal/mcp/tools/update_document.go`, `@internal/mcp/tools/remove_document.go`) and the relation guard (`@internal/mcp/tools/add_relation.go`).
- `archcore status` and the SessionStart context filter to local documents (`@cmd/status.go`, `@cmd/hooks_common.go`).
- Tests: `@internal/mcp/tools/globals_test.go`, `@internal/mcp/integration/globals_test.go`, `@cmd/status_test.go`, `@cmd/hooks_common_test.go`.
