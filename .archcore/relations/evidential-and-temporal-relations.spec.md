---
title: "Evidential and Temporal Relation Values"
status: accepted
tags:
  - "component:cli"
  - "mcp"
  - "relations"
---

## Purpose & Scope

This contract extends the local relation vocabulary consumed by MCP clients, manifest readers, and advisory hooks. The three values follow `concepts/research-and-evidence-types` [global · archcore · read-only]. Remote server acceptance is outside scope.

## Surface

The relation enum and persisted triples live in @cli/internal/sync/manifest.go; the MCP entry points are the relation tools in @cli/internal/mcp/tools/.

| Value | Axis | Stored direction |
|---|---|---|
| `supports` | Evidential | Material to the statement it backs |
| `contradicts` | Evidential | Challenger to the statement it disputes |
| `supersedes` | Temporal | Newer document to the older document it replaces |

Endpoints remain documents. The word “statement” describes the target document's content, not a new node type.

## Normative Behavior

1. The `add_relation` tool MUST preserve the caller's direction for each of the three values.
2. The server instructions MUST describe the three axes and each value's direction.
3. The cascade selector MUST exclude the three values.
4. The restatement selector MUST exclude the three values.

## Constraints & Invariants

Manifest shape, version 1, limits, duplicate handling, and path containment remain governed by the existing sync contract and @cli/internal/sync/manifest.go.

1. The relation tools MUST NOT restrict the new values by document type or category.
2. The relation tools MUST NOT infer an inverse edge.
3. The relation tools MUST NOT change document status when adding `supersedes`.
4. The relation tools MUST NOT resolve a contradiction automatically.
5. The relation tools MUST reject global documents as either endpoint.

Older binaries reject unknown relation values (@cli/internal/sync/manifest.go). Adding values therefore does not give backward readability for a manifest that contains them. No migration or downgrade rewrite exists.

## Failure Behavior

1. WHEN the requested type is unknown, `add_relation` MUST return the existing invalid-relation-type error.
2. WHEN either endpoint is missing, `add_relation` MUST reject the relation without persisting it.
3. WHEN source equals target, `add_relation` MUST reject the relation.
4. WHEN an endpoint escapes the project boundary, the relation tools MUST reject the operation.
5. WHEN manifest validation fails, the relation tools MUST preserve the existing manifest.

## Conformance

Conformance requires every numbered obligation above. The relation tool tests, @cli/internal/sync/manifest_test.go, @cli/cmd/hook_post_tool_use_test.go, and @cli/internal/advisory/restatement_test.go cover them.
