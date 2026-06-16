---
title: "Globals Are Read-Only Everywhere Outside the MCP Read Tools"
status: accepted
tags:
  - "globals"
  - "mcp"
---

## Rule

A mounted global source is a **read-only context surface**. It is visible **only** through the MCP read tools — `list_documents`, `get_document`, `search_documents` — where every global document is tagged `source_kind: "global"`, `read_only: true`, and a `source_id`. Everywhere else it is invisible or refused:

- **Not writable.** `create_document`, `update_document`, and `remove_document` reject any path that is a declared global source **or** inside the reserved `.archcore/global/` tree (declared or not).
- **Not a relation endpoint.** `add_relation` refuses an edge whose source **or** target is a global document, in **either** direction — including an in-tree global under `.archcore/global/`. Relations connect local documents only. (`remove_relation` stays open so a pre-existing edge can be cleaned up.)
- **Not in CLI surfaces.** `archcore status` (tag hygiene, counts) and the SessionStart context operate on local documents only — a consumer can neither fix nor is responsible for an upstream global's tags.

"Global" is one predicate, asked the same way everywhere — `isReadOnlyGlobalPath` in @internal/mcp/tools/common.go: a declared source from `settings.json` **or** anything under the reserved `.archcore/global/` directory.

## Rationale

- One predicate, one mental model: "is this global?" has a single answer used by every write and relation guard, so behavior is predictable and there are no in-tree/external corner cases.
- Read-only is a property of the *mount*, not the repo. The same repo opened directly is fully writable; mounted as another project's global it is read-only — and never accumulates relation edges from its consumers, preserving the one-directional `local → global` reference invariant.
- Keeping globals out of `status` / session context avoids false "fix this tag" signals a consumer cannot act on, and keeps injected context focused on what the project owns.

## Examples

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

- `isReadOnlyGlobalPath` (@internal/mcp/tools/common.go) backs the write guards (@internal/mcp/tools/create_document.go, @internal/mcp/tools/update_document.go, @internal/mcp/tools/remove_document.go) and the relation guard (@internal/mcp/tools/add_relation.go).
- `archcore status` and the SessionStart context filter to local documents (@cmd/status.go, @cmd/hooks_common.go).
- Tests: @internal/mcp/tools/globals_test.go, @internal/mcp/integration/globals_test.go, @cmd/status_test.go, @cmd/hooks_common_test.go.