---
title: "When to Use the spec Document Type"
status: accepted
tags:
  - "document-types"
---

## Rule

1. The author MUST use the `spec` type for the canonical normative behavior contract of one concrete subject that others rely on: a boundary (API, interface, schema, protocol) or a feature or subsystem (behavior, constraints, invariants, conformance).
2. The author MUST cover one subject in one spec document.
3. The author MUST write the normative behavior first and MUST NOT use a spec document as a general reference dump.
4. The author MUST follow the `spec` versus `doc`, `spec` versus `rule`, and `spec` versus `adr` distinctions, the one-question test, and the type examples as defined in the `archcore` global source `concepts/document-types-reference`. This rule does not restate them.

## Rationale

A spec is the document another team, another service, or a test suite is checked against. Mixing reference material into it hides which sentences are binding. Keeping the conceptual definition in the global source and the obligations here prevents a second copy of the type definition from drifting.

## Enforcement (CLI)

- The type-selection rules in `@internal/mcp/server.go` and `@internal/mcp/tools/create_document.go` carry the `spec` versus `doc`, `spec` versus `rule`, and `spec` versus `adr` disambiguation.
- `@templates/templates.go` registers the `spec` type with its six-section template: Purpose & Scope, Surface, Normative Behavior (EARS clauses + BCP 14 keywords), Constraints & Invariants, Failure Behavior, Conformance.

## References

- `@templates/templates.go` — type constant, category mapping, template function
- `@internal/mcp/server.go` — MCP server instructions with disambiguation rules
- `@internal/mcp/tools/create_document.go` — MCP tool description with the type entry
