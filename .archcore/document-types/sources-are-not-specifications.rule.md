---
title: "Source Documents (MRD/BRD/URD) Must Not Be Used as Specifications"
status: accepted
tags:
  - "component:cli"
  - "document-types"
---

## Rule

1. The author MUST NOT use an `mrd`, `brd`, or `urd` document as a specification.
2. WHEN the author formalizes a requirement that a source document raised, the author MUST write the formal requirement into an ISO specification type (`brs`, `strs`, `syrs`, `srs`).
3. The author MUST keep the formalization direction one-way: source to specification, never specification to source.
4. The author MUST follow the two-layer model, the informality of sources, and the PRD hybrid exception as defined in the `archcore` global source `concepts/requirements-layers`. This rule does not restate them.

## Rationale

Sources and specifications answer different questions. A source records what the market, the business, or a user asked for, and it stays informal on purpose. A specification states verifiable behavior that an implementation and a test can be checked against. Treating a source as a specification pushes unverifiable wording into the layer that engineering builds from.

The conceptual model lives in one place — the global source — so the CLI repository carries the obligations and the enforcement points, not a second copy of the model that can drift.

## Enforcement (CLI)

- The `add_relation` tool description in `@cli/internal/mcp/tools/add_relation.go` carries the requirements-layer conventions. The MCP server instructions hold no REQUIREMENTS LAYERS block since they were cut to 2 048 characters (2026-10-09).
- `create_document` in `@cli/internal/mcp/tools/create_document.go` enforces the section requirements that differ between source types and specification types.
- Templates in `@cli/templates/templates.go` are structurally distinct: sources carry discovery sections (TAM/SAM/SOM, Personas, ROI); specifications carry ISO sections (Mission, ConOps, Verification).

## References

- `@cli/internal/mcp/server.go` — REQUIREMENTS LAYERS block, disambiguation rules
- `@cli/internal/mcp/tools/create_document.go` — type entries with distinct required sections
- `@cli/templates/templates.go` — structurally distinct template generators
