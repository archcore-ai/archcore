---
title: "When to Use the spec Document Type"
status: accepted
tags:
  - "document-types"
---

## Rule

The conceptual definition of the `spec` type — when to use it, the spec-vs-doc/rule/adr distinctions, the one-question test, and good/bad examples — lives in the `archcore` global source `concepts/document-types-reference`. It is not restated here.

In short: use `spec` for the canonical normative contract of one concrete technical boundary (behavior, constraints, invariants, conformance). One subject per spec; normative-first; no general-reference dumping.

## Enforcement (CLI)

- Type-selection rules in `@internal/mcp/server.go` and `@internal/mcp/tools/create_document.go` include the `spec vs doc`, `spec vs rule`, and `spec vs adr` disambiguation.
- The `spec` type is registered in `@templates/templates.go` with a template structured around: Purpose, Scope, Authority, Subject, Contract Surface, Normative Behavior, Constraints, Invariants, Error Handling, Conformance.

## References

- `@templates/templates.go` — type constant, category mapping, template function
- `@internal/mcp/server.go` — MCP server instructions with disambiguation rules
- `@internal/mcp/tools/create_document.go` — MCP tool description with type entry
