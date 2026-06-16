---
title: "Implementation Plan: Add ISO 29148 Document Types (BRS, StRS, SyRS, SRS)"
status: accepted
tags:
  - "document-types"
---

## Goal

Add all four ISO/IEC/IEEE 29148:2018 requirements specification types — BRS, StRS, SyRS, SRS — to the vision category, enabling a detailed requirements engineering cascade alongside the existing simple PRD approach.

### Context

- Archcore currently has 18 document types (10 vision, 6 knowledge, 2 experience)
- ISO 29148 defines a cascade: BRS (business mission) → StRS (stakeholder needs) → SyRS (system behavior) → SRS (software specs)
- PRD remains the simple/default approach; ISO types are for detailed elaboration
- MRD/BRD/URD (Requirement Sources layer) were implemented prior to this
- See companion idea: @.archcore/document-types/prd-vs-iso-29148-requirements-strategy.idea.md

## Tasks

### Phase 1: Core Type System

- [x] Add constants to @templates/templates.go: `TypeBRS = "brs"`, `TypeStRS = "strs"`, `TypeSyRS = "syrs"`, `TypeSRS = "srs"`
- [x] Add all 4 to `categoryMap` as `CategoryVision`
- [x] Extend `ValidTypes()` return slice with all 4 slugs
- [x] Add 4 `case` arms in `GenerateTemplate()` switch
- [x] Implement `generateBRSTemplate()` — sections per ISO §9.3: Business Purpose/Scope, Business Overview (Stakeholders, Environment), Mission/Goals/Objectives, Business Model (Processes, Policies/Rules), Business Constraints, High-Level Operational Concept (Scenarios, Modes), Success Criteria, Assumptions/Dependencies, Traceability
- [x] Implement `generateStRSTemplate()` — sections per ISO §9.4: Purpose/Scope, Stakeholder Classes (Priorities), Operational Concept/ConOps (Current Ops, Proposed Ops, Scenarios), Stakeholder Requirements (User, Usability, Quality), Operational Policies/Rules, Operational Constraints (Modes/States), Compliance/Regulatory, Project Constraints, Traceability
- [x] Implement `generateSyRSTemplate()` — sections per ISO §9.5: System Purpose/Scope (Boundary), System Overview, System Requirements (Functional, Usability, Performance, Security, Reliability), System Interfaces (User, System-to-System, Hardware), System Operations (Modes/States, Physical/Environmental), Policy/Regulation, Life Cycle Sustainment, Verification Approach, Traceability
- [x] Implement `generateSRSTemplate()` — sections per ISO §9.6: Purpose/Scope (Component, Boundaries), Product Perspective (Functions, User Characteristics, Limitations), Software Requirements (Functional, Behavioral, Error Handling), External Interfaces (API Endpoints, Internal), Data Requirements (Logical Database, Data Flows), Performance, Design Constraints (Standards Compliance), Software Quality Attributes, Verification Matrix, Traceability

### Phase 2: MCP Integration

- [x] Update @internal/mcp/server.go `mcpServerInstructions`: add types to vision listing, add 4 WHEN TO CREATE lines, add 8 TYPE SELECTION RULES, rewrite REQUIREMENTS TRACKS with two-layer separation (Sources vs Specifications)
- [x] Update @internal/mcp/tools/create_document.go tool description: add 4 type entries with required sections, add 8 disambiguation rules (including cross-layer: brs vs brd, strs vs urd)
- [x] Update @internal/mcp/tools/list_documents.go: add new types to `types` parameter description
- [x] Update @internal/mcp/tools/add_relation.go: add REQUIREMENTS LAYER HINTS (Sources → Specs: implements, ISO cascade: implements, same layer: related)

### Phase 3: Tests

- [x] Add 4 entries to template table-driven tests in @templates/templates_test.go
- [x] Add 4 entries to AllTypes test in @internal/mcp/tools/create_document_test.go
- [x] Update TestTemplateStructure with minLength assertions (BRS=2000, StRS=2000, SyRS=2500, SRS=2500)
- [x] Update TestTemplateMarkdownFormatting types slice (14 → 18)

### Phase 4: Documentation

- [x] Update @.archcore/dir/categories-and-document-types.doc.md: add ISO Track table with ISO references, add Requirements Layers section, add 8 disambiguation entries

## Acceptance Criteria

- [x] `go build -o archcore .` succeeds
- [x] `go test ./...` passes (all existing + new tests)
- [x] Creating a document of each new type via MCP renders the correct ISO-aligned template
- [x] Disambiguation rules clearly distinguish brs/strs/syrs/srs from existing prd/adr types
- [x] Cross-layer disambiguation rules distinguish sources (brd/urd) from specifications (brs/strs)
- [x] Traceability sections in all 4 ISO templates enable the requirement cascade
- [x] add_relation tool includes layer hints for correct relation usage

## Dependencies

| Dependency | Type | Status |
|------------|------|--------|
| ISO/IEC/IEEE 29148:2018 standard (reference) | External | Available (PDF in repo) |
| Existing template system in templates.go | Internal | Ready |
| MCP server instructions in server.go | Internal | Ready |
| MRD/BRD/URD implementation (Phase 1) | Internal | Complete |

## Notes

### Two-Layer Architecture

The implementation enforces a clear separation between two requirement layers:

- **Layer A (Sources):** mrd, brd, urd, prd — informal, discovery-oriented documents
- **Layer B (Specifications):** brs, strs, syrs, srs — formal, ISO-structured specifications

This separation is communicated through:
1. MCP server instructions (REQUIREMENTS LAYERS block with explicit "Do NOT confuse" directive)
2. create_document disambiguation rules (brs vs brd, strs vs urd cross-layer rules)
3. add_relation layer hints (sources → specs use "implements", same layer use "related")
4. Template traceability sections (each ISO template names its upstream sources)

### Disambiguation Strategy (8 Rules Added)

1. **brs vs prd**: brs has ONLY business objectives with ISO structure, no user stories. prd has user stories + solution.
2. **strs vs prd**: strs groups per STAKEHOLDER CLASS with ConOps. prd lists by priority.
3. **syrs vs adr**: syrs = WHOLE SYSTEM BOUNDARY. adr = single decision.
4. **srs vs prd**: srs = PER-ENDPOINT/PER-FUNCTION with verification matrix. prd = product-level.
5. **brs vs strs**: brs = WHY (business outcomes). strs = WHAT stakeholders need.
6. **syrs vs srs**: syrs = WHOLE SYSTEM. srs = SINGLE COMPONENT.
7. **brs vs brd**: brs = FORMAL ISO SPEC. brd = INFORMAL SOURCE.
8. **strs vs urd**: strs = FORMAL ISO SPEC. urd = INFORMAL SOURCE.

### ISO 29148 Section References

- BRS content: ISO §9.3 (§9.3.2–§9.3.19)
- StRS content: ISO §9.4 (§9.4.1–§9.4.19)
- SyRS content: ISO §9.5 (§9.5.1–§9.5.19)
- SRS content: ISO §9.6 (§9.6.1–§9.6.20)
