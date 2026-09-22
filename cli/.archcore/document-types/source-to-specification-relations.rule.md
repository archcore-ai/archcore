---
title: "Relation Conventions Between Requirement Sources and Specifications"
status: accepted
tags:
  - "document-types"
---

## Rule

1. WHEN the author links a specification to the requirement source it formalizes, the author MUST create an `implements` relation with the specification as the relation source and the requirement source document as the relation target.
2. The author MUST follow the ISO cascade, partial cascades, same-layer `related` links, the more-specific-implements-more-general direction, and the PRD exception as defined in the `archcore` global source `concepts/requirements-layers`. This rule does not restate them.

## Rationale

Relation direction carries meaning in the graph: `implements` reads as "this document fulfills that one". Pointing it the other way inverts traceability, so a reader walking from a source document cannot tell which specification answers it.

Keeping the cascade conventions in the global source and the direction obligation here prevents two copies of the same convention from drifting apart.

## Enforcement (CLI)

- The MCP tool `add_relation` in `@internal/mcp/tools/add_relation.go` carries REQUIREMENTS LAYER HINTS as soft guidance for relation direction.
- MCP server instructions in `@internal/mcp/server.go` document the full layer mapping in the REQUIREMENTS LAYERS block.
- The Traceability sections of the four ISO templates (`@templates/templates.go`) name their upstream sources.

## References

- `@internal/mcp/tools/add_relation.go` — layer hints in the tool description
- `@internal/mcp/server.go` — REQUIREMENTS LAYERS instructions
- `@templates/templates.go` — ISO template traceability sections
