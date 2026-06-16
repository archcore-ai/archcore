---
title: "Source Documents (MRD/BRD/URD) Must Not Be Used as Specifications"
status: accepted
tags:
  - "document-types"
---

## Description

Source documents (mrd, brd, urd) capture where requirements come from — market analysis, business justification, user needs. They are informal, discovery-oriented documents. They must never be treated as formal specifications or used in place of ISO 29148 specification types (brs, strs, syrs, srs).

## Rule

1. MRD, BRD, and URD documents MUST remain informal and discovery-oriented.
2. Do NOT write formal ISO-structured requirements (mission statements, operational concepts, ConOps, verification matrices) in source documents.
3. When source requirements need formalization, create the corresponding specification document and link via `implements` relation:
   - Market requirements (MRD) → formalize in BRS
   - Business objectives (BRD) → formalize in BRS and/or StRS
   - User needs (URD) → formalize in StRS
4. Source documents MAY contain acceptance criteria and success metrics (they are useful for discovery), but these are NOT normative specifications.
5. Do NOT create a BRS/StRS/SyRS/SRS and then copy its content into a BRD/URD/MRD — the formalization direction is sources → specifications, not reverse.

## Rationale

The two-layer architecture separates requirements discovery (Sources) from requirements formalization (Specifications). Conflating the two layers defeats the purpose of having separate tracks:

- Sources become bloated with formal structure they weren't designed for
- Specifications lose their ISO rigor by drawing from already-formalized sources
- Agents cannot reliably disambiguate between types when content overlaps
- Traceability breaks because the refinement direction is unclear

The MCP instructions explicitly state: "Do NOT confuse source documents (mrd/brd/urd) with specification documents (brs/strs/syrs/srs). Sources are informal, discovery-oriented. Specifications are formal, ISO-structured."

## Examples

### Good

- MRD with TAM/SAM/SOM analysis, competitive landscape, market needs — informal, discovery-oriented
- BRD with business objectives, ROI analysis, stakeholder list — informal, justification-oriented
- URD with user personas, journey maps, usability needs — informal, user-centered
- BRS that formalizes MRD + BRD into ISO structure with `brs implements mrd`, `brs implements brd`

### Bad

- MRD with ISO §9.3 section structure (Mission, Goals, Operational Concept) — that content belongs in BRS
- BRD with per-stakeholder-class requirements and ConOps — that content belongs in StRS
- URD with verification matrix and formal requirement IDs — that content belongs in StRS or SRS

## Exceptions

- PRD is intentionally a hybrid that covers aspects of both layers. It is the sole exception to strict layer separation, by design. It belongs to Layer A (Sources) but can serve as a pragmatic substitute for the full ISO cascade.

## Enforcement

- MCP server instructions in @internal/mcp/server.go include TYPE SELECTION RULES with cross-layer disambiguation (brs vs brd, strs vs urd).
- `create_document` tool in @internal/mcp/tools/create_document.go includes structural section requirements that differ between source and specification types.
- Template sections in @templates/templates.go are structurally distinct: sources have discovery-oriented sections (TAM/SAM/SOM, Personas, ROI), specifications have ISO-oriented sections (Mission, ConOps, Verification).

## References

- @internal/mcp/server.go — REQUIREMENTS LAYERS block, disambiguation rules
- @internal/mcp/tools/create_document.go — type entries with distinct required sections
- @templates/templates.go — structurally distinct template generators
- .archcore/dir/categories-and-document-types.doc.md — layer definitions and disambiguation