---
title: "Evidential and Temporal Relation Values"
status: draft
tags:
  - "mcp"
  - "relations"
---

## Purpose & Scope

This contract extends the local relation vocabulary consumed by MCP clients, manifest readers, and advisory hooks. The three values follow `concepts/research-and-evidence-types` [global · archcore · read-only]. Remote server acceptance is outside scope.

## Surface

The relation enum and persisted triples live in @internal/sync/manifest.go. MCP entry points live in @internal/mcp/tools/add_relation.go, @internal/mcp/tools/remove_relation.go, @internal/mcp/tools/list_relations.go, and @internal/mcp/tools/get_document.go.

| Value | Axis | Stored direction |
|---|---|---|
| `supports` | Evidential | Material to the statement it backs |
| `contradicts` | Evidential | Challenger to the statement it disputes |
| `supersedes` | Temporal | Newer document to the older document it replaces |

Endpoints remain documents. The word “statement” describes the target document's content, not a new node type.

## Normative Behavior

1. The relation registry MUST accept the three Surface values alongside `related`, `implements`, `extends`, and `depends_on`.
2. The manifest validator MUST accept all seven relation values.
3. The `add_relation` tool MUST accept each new value between distinct existing local documents.
4. The `add_relation` tool MUST preserve the caller's direction.
5. The `remove_relation` tool MUST accept each new value.
6. The `list_relations` tool MUST return stored values unchanged.
7. The `get_document` tool MUST return the new values in incoming and outgoing relations.
8. The server instructions MUST describe the three axes and each new value's direction.
9. The cascade selector MUST exclude the three new values.
10. The restatement selector MUST exclude the three new values.
11. The `archcore doctor` command MUST accept the new relation values during manifest validation.

## Constraints & Invariants

Manifest shape, version 1, limits, duplicate handling, and path containment remain governed by the existing sync contract and @internal/sync/manifest.go.

1. The relation tools MUST NOT restrict the new values by document type or category.
2. The relation tools MUST NOT infer an inverse edge.
3. The relation tools MUST NOT change document status when adding `supersedes`.
4. The relation tools MUST NOT resolve a contradiction automatically.
5. The relation tools MUST reject global documents as either endpoint.

Older binaries currently reject unknown relation values — @internal/sync/manifest.go. Adding values therefore does not provide backward readability for a manifest containing them. This change provides no manifest migration or downgrade rewrite.

## Failure Behavior

1. WHEN the requested type is unknown, `add_relation` MUST return the existing invalid-relation-type error.
2. WHEN either endpoint is missing, `add_relation` MUST reject the relation without persisting it.
3. WHEN source equals target, `add_relation` MUST reject the relation.
4. WHEN an endpoint escapes the project boundary, the relation tools MUST reject the operation.
5. WHEN manifest validation fails, the relation tools MUST preserve the existing manifest.

## Conformance

Conformance requires every numbered obligation above. Regression tests cover @internal/sync/manifest_test.go, @internal/mcp/tools/add_relation_test.go, @internal/mcp/tools/remove_relation_test.go, @internal/mcp/tools/get_document_test.go, and @internal/mcp/tools/list_relations_test.go. Hook and restatement checks target @cmd/hook_post_tool_use_test.go and @internal/advisory/restatement_test.go. An isolated doctor fixture in @cmd/doctor_test.go verifies enum acceptance without requiring unrelated health checks to pass.
