---
title: "When to Use the spec Document Type"
status: accepted
tags:
  - "document-types"
---

## Description

The `spec` document type captures the **canonical normative contract** for a concrete system, component, interface, schema, or protocol. It belongs to the **knowledge** category.

A spec defines externally observable behavior, constraints, invariants, and conformance requirements. It is the authoritative source of truth — if implementation differs from the spec, the spec takes precedence until amended.

## Rule

**Use `spec` when the document:**

1. Fixes the canonical contract for a concrete technical boundary.
2. Describes externally observable behavior with normative requirements (MUST/SHOULD/MAY).
3. Is needed for verifying implementation or compliance.
4. Contains invariants, constraints, and conformance criteria.

**Do NOT use `spec` when the document primarily:**

5. Explains, teaches, or educates — use `guide`.
6. Discusses alternatives, trade-offs, or historical decisions — use `adr`.
7. Sets a cross-cutting standard that applies to many subsystems — use `rule`.
8. Is a catalog, registry, lookup table, or general reference — use `doc`.

**Guardrails to keep `spec` narrow:**

9. **One subject per spec.** One document = one contract object. Do not bundle multiple components.
10. **Normative first.** If a section cannot be used to verify implementation, it should be short or absent.
11. **No general reference dumping.** Registries, inventories, glossaries of everything, historical notes — not here.

## Rationale

`spec` was introduced to separate normative contracts from non-behavioral reference material (`doc`). Without it, `doc` absorbed both behavioral contracts and catalogs, weakening agent type selection.

The core definition:

> Spec documents the canonical normative contract of a concrete system, component, interface, schema, or protocol.

The one-question test: "Does this document define a normative contract for a specific technical boundary, with precision sufficient for implementation or compliance checking?" If yes → `spec`.

## Examples

### Good

- `payment-api.spec.md` — defines endpoints, request/response schemas, error codes, rate limits, conformance criteria
- `sync-protocol.spec.md` — defines message format, state machine, delivery guarantees, failure semantics
- `document-frontmatter.spec.md` — defines required YAML fields, valid values, validation rules, conformance
- `webhook-delivery.spec.md` — defines delivery guarantees, retry policy, payload format, signature verification

### Bad

- Agent registry listing supported agents and their config paths → use `doc` (non-behavioral reference)
- "Always validate user input at API boundaries" → use `rule` (cross-cutting team standard)
- "We chose PostgreSQL because..." → use `adr` (decision record with rationale)
- "Proposal: switch from REST to gRPC" → use `rfc` (proposal under review)
- Step-by-step instructions for deploying a service → use `guide`
- Glossary of all project terminology → use `doc` (general reference)

## Exceptions

None. If content defines a normative contract for a specific technical boundary, it is a `spec`. If it does not, it belongs in another type.

## Enforcement

- Agent type selection rules in @internal/mcp/server.go and @internal/mcp/tools/create_document.go include `spec vs doc`, `spec vs rule`, and `spec vs adr` disambiguation.
- The `spec` type is registered in @templates/templates.go with a template structured around: Purpose, Scope, Authority, Subject, Contract Surface, Normative Behavior, Constraints, Invariants, Error Handling, Conformance.

## References

- @templates/templates.go — type constant, category mapping, template function
- @internal/mcp/server.go — MCP server instructions with disambiguation rules
- @internal/mcp/tools/create_document.go — MCP tool description with type entry
