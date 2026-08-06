---
title: "Globals Are Read-Only Everywhere Outside the MCP Read Tools"
status: accepted
tags:
  - "globals"
  - "mcp"
---

## Rule

A path is global when `settings.json` declares it as a global source, or when it sits under the reserved `.archcore/global/` directory. `docs.IsGlobalPath` and `docs.IsReservedGlobalDir` in `@internal/docs/globals.go` answer those two halves for every guard.

1. The MCP read tools `list_documents`, `get_document`, and `search_documents` MUST expose global documents and MUST tag each one with `source_kind: "global"`, `read_only: true`, and a `source_id`.
2. `create_document`, `update_document`, and `remove_document` MUST reject any path that is global.
3. `add_relation` MUST reject an edge whose source or target is a global document, in either direction.
4. `remove_relation` MUST stay open for global endpoints, so that a pre-existing edge can be removed.
5. `archcore status` MUST report tag hygiene and counts for local documents only.
6. The SessionStart context MUST inject local documents only.
7. The pre-write code-alignment injection MAY name a global document, and MUST mark it `[global]`.
8. The pre-write hook guard MUST refuse a write to a global source mounted from outside the store, which the MCP write tools cannot address at all.

## Rationale

One predicate answers "is this global?" for every write guard and relation guard, so the behavior holds for a declared external source and an in-tree `.archcore/global/` document alike, with no corner cases.

Requirement 8 is that same idea reached by a different route. `docs.GuardWritablePath` takes a path under `.archcore/`, so a source mounted outside the store never reaches it: the MCP tools refuse those paths as unaddressable, and the hook, which is handed a host-supplied absolute path, saw only "outside the project" and allowed the write. An external global was therefore editable straight from the editor while every in-tree global was protected. `docs.IsExternalGlobalDocument` closes that gap so both surfaces reach one verdict.

Read-only is a property of the mount, not of the repository. The same repository opened directly is fully writable; mounted as another project's global source it is read-only, and it never accumulates relation edges from its consumers. That keeps the reference direction one-way: local to global.

Excluding globals from `status` and from the session context removes "fix this tag" signals that a consumer cannot act on, and keeps the injected context focused on what the project owns.

The code-alignment injection is the one exception, and it is a read surface: an organization rule that constrains the file being edited is exactly what the agent needs before the edit. Marking it `[global]` keeps the reader from trying to change it.

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

- `docs.IsGlobalPath`, `docs.IsReservedGlobalDir`, and their case-folding siblings (`@internal/docs/globals.go`) back `docs.GuardWritablePath`, which serves the MCP write guards (`@internal/mcp/tools/create_document.go`, `@internal/mcp/tools/update_document.go`, `@internal/mcp/tools/remove_document.go`), the relation guard (`@internal/mcp/tools/add_relation.go`), and the pre-write hook guard (`@cmd/hook_write_guard.go`).
- `docs.IsExternalGlobalDocument` (`@internal/docs/globals.go`) covers requirement 8 — the paths `GuardWritablePath` cannot classify.
- `archcore status` and the SessionStart context filter to local documents (`@cmd/status.go`, `@cmd/hooks_common.go`).
- The code-alignment injection includes globals and marks them (`@internal/advisory/code_alignment.go`).
- Tests: `@internal/docs/guard_test.go`, `@internal/mcp/tools/globals_test.go`, `@internal/mcp/integration/globals_test.go`, `@cmd/status_test.go`, `@cmd/hooks_common_test.go`, `@cmd/hook_write_guard_test.go`.
