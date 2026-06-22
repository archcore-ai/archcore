---
title: "Relation Conventions Between Requirement Sources and Specifications"
status: accepted
tags:
  - "document-types"
---

## Rule

The relation conventions between requirement sources and specifications — the `implements` direction (the spec is the source, the source-doc is the target), the ISO cascade, partial cascades, same-layer `related`, the more-specific-implements-more-general direction rule, and the PRD exception — live in the `archcore` global source `concepts/requirements-layers`. Not restated here.

## Enforcement (CLI)

- The MCP `add_relation` tool in `@internal/mcp/tools/add_relation.go` includes REQUIREMENTS LAYER HINTS as soft guidance for correct relation usage.
- MCP server instructions in `@internal/mcp/server.go` document the full layer mapping in the REQUIREMENTS LAYERS block.
- Traceability sections in the four ISO templates (`@templates/templates.go`) name their upstream sources.

## References

- `@internal/mcp/tools/add_relation.go` — layer hints in tool description
- `@internal/mcp/server.go` — REQUIREMENTS LAYERS instructions
- `@templates/templates.go` — ISO template traceability sections
