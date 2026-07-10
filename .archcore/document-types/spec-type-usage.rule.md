---
title: "When to Use the spec Document Type"
status: accepted
tags:
  - "document-types"
---

## Rule

The conceptual definition of the `spec` type — when to use it, the spec-vs-doc/rule/adr distinctions, the one-question test, and good/bad examples — lives in the `archcore` global source `concepts/document-types-reference`. It is not restated here.

In short: use `spec` for the canonical normative behavior contract of one concrete subject others rely on — a boundary (API, interface, schema, protocol) or a feature/subsystem (behavior, constraints, invariants, conformance). One subject per spec; normative-first; no general-reference dumping.

## Enforcement (CLI)

- Type-selection rules in `@internal/mcp/server.go` and `@internal/mcp/tools/create_document.go` include the `spec vs doc`, `spec vs rule`, and `spec vs adr` disambiguation.
- The `spec` type is registered in `@templates/templates.go` with the six-section template: Purpose & Scope, Surface, Normative Behavior (EARS clauses + BCP 14 keywords), Constraints & Invariants, Failure Behavior, Conformance.

## References

- `@templates/templates.go` — type constant, category mapping, template function
- `@internal/mcp/server.go` — MCP server instructions with disambiguation rules
- `@internal/mcp/tools/create_document.go` — MCP tool description with type entry