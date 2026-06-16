---
title: "Relation Conventions Between Requirement Sources and Specifications"
status: accepted
tags:
  - "document-types"
---

## Description

Conventions for linking requirement source documents (mrd, brd, urd, prd) to requirement specification documents (brs, strs, syrs, srs) and within the ISO cascade. These ensure consistent traceability so agents can follow the refinement path from informal discovery through formal specification.

## Rule

### Cross-Layer Relations (Sources → Specifications)

1. When a BRS formalizes market requirements from an MRD, add: `brs implements mrd`
2. When a BRS formalizes business objectives from a BRD, add: `brs implements brd`
3. When a StRS formalizes user needs from a URD, add: `strs implements urd`
4. When a StRS formalizes business stakeholder needs from a BRD, add: `strs implements brd`
5. The specification document is always the **source** of the `implements` relation; the source document is the **target**. ("BRS implements BRD" reads as: "the BRS is the document that implements what the BRD describes.")

### ISO Cascade Relations (within Specifications)

6. Within the ISO cascade, each level implements the previous: `strs implements brs`, `syrs implements strs`, `srs implements syrs`
7. Partial cascades are valid — `srs implements brs` is acceptable when intermediate levels are skipped.

### Same-Layer Relations

8. Documents at the same layer use `related`, not `implements`: `mrd related brd`, `brd related urd`
9. PRD links to ISO types via `related` (PRD is an alternative path, not a source for ISO specs).

### Relation Direction

10. Always add `implements` with the MORE SPECIFIC document as source and the MORE GENERAL document as target. This reads naturally: "SRS implements SyRS" (the software spec fulfills the system spec).

## Rationale

The two-layer architecture (Sources vs Specifications) requires consistent relation conventions so agents can trace requirements from informal discovery through formal specification. The `implements` relation creates a directed acyclic graph that mirrors the refinement cascade: market/business/user needs → formalized specifications → decomposed specifications.

Without this convention, agents would create ad-hoc relation patterns, breaking traceability.

## Examples

### Good

- `brs implements mrd` — BRS formalizes market requirements from MRD
- `strs implements brs` — StRS decomposes requirements from BRS in the ISO cascade
- `strs implements urd` — StRS formalizes user needs from URD
- `mrd related brd` — MRD and BRD are both sources, peer relationship
- `prd related brs` — PRD is an alternative path, not a formalization source

### Bad

- `mrd implements brs` — wrong direction: source doesn't implement spec
- `brs extends mrd` — wrong relation type: use `implements` for source→spec
- `mrd implements brd` — sources don't implement each other: use `related`
- `prd implements brs` — PRD is a peer alternative, not a formalization source: use `related`

## Exceptions

- PRD can be linked to any ISO type via `related` since PRD is a pragmatic hybrid that roughly covers all four ISO levels. Using `implements` for PRD→ISO would imply a formalization direction that doesn't apply.
- When a project uses only partial ISO cascade (e.g., BRS + SRS without StRS/SyRS), direct `implements` relations across skipped levels are valid (e.g., `srs implements brs`).

## Enforcement

- MCP `add_relation` tool in @internal/mcp/tools/add_relation.go includes REQUIREMENTS LAYER HINTS as soft guidance for correct relation usage.
- MCP server instructions in @internal/mcp/server.go document the full layer mapping in the REQUIREMENTS LAYERS block.
- Template Traceability sections in all 4 ISO templates (@templates/templates.go) name upstream sources.

## References

- @internal/mcp/tools/add_relation.go — layer hints in tool description
- @internal/mcp/server.go — REQUIREMENTS LAYERS instructions
- @templates/templates.go — ISO template traceability sections
- .archcore/dir/categories-and-document-types.doc.md — layer definitions and disambiguation