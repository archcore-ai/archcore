---
title: "Source Documents (MRD/BRD/URD) Must Not Be Used as Specifications"
status: accepted
tags:
  - "document-types"
---

## Rule

The two-layer model (Sources vs Specifications), the rule that sources stay informal and never substitute for ISO specs, the formalization direction (sources → specifications, never the reverse), and the PRD hybrid exception live in the `archcore` global source `concepts/requirements-layers`. Not restated here.

## Enforcement (CLI)

- MCP server instructions in `@internal/mcp/server.go` include the REQUIREMENTS LAYERS block with cross-layer disambiguation (brs vs brd, strs vs urd).
- `create_document` in `@internal/mcp/tools/create_document.go` enforces structural section requirements that differ between source and specification types.
- Templates in `@templates/templates.go` are structurally distinct: sources have discovery sections (TAM/SAM/SOM, Personas, ROI); specifications have ISO sections (Mission, ConOps, Verification).

## References

- `@internal/mcp/server.go` — REQUIREMENTS LAYERS block, disambiguation rules
- `@internal/mcp/tools/create_document.go` — type entries with distinct required sections
- `@templates/templates.go` — structurally distinct template generators
