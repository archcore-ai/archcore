---
title: "Implementation Plan: Add MRD, BRD, URD Document Types as Requirement Sources Track"
status: accepted
tags:
  - "document-types"
---

## Goal

Add MRD (Market Requirements Document), BRD (Business Requirements Document), and URD (User Requirements Document) to the vision category as a requirement sources track, enabling market/business/user perspective capture alongside the existing PRD (simple) and ISO 29148 (formal decomposition) approaches.

### Context

- Archcore currently has 10 document types (3 vision, 5 knowledge, 2 experience) — will be 14 after ISO types, 17 after this plan
- MRD/BRD/URD represent the "source axis" (where requirements come from) vs ISO's "decomposition axis" (how requirements refine)
- PRD remains the simple/default approach; source types are for structured discovery and stakeholder alignment
- See companion idea: @.archcore/document-types/mrd-brd-urd-requirement-sources.idea.md
- Follows the same implementation pattern as: @.archcore/document-types/iso-29148-document-types-implementation.plan.md

## Tasks

### Phase 1: Core Type System

- [x] Add constants to @templates/templates.go: `TypeMRD = "mrd"`, `TypeBRD = "brd"`, `TypeURD = "urd"`
- [x] Add all 3 to `categoryMap` as `CategoryVision`
- [x] Extend `ValidTypes()` return slice with all 3 slugs
- [x] Add 3 `case` arms in `GenerateTemplate()` switch
- [x] Implement `generateMRDTemplate()` — sections: Market Landscape (Industry Trends, Market Size, Dynamics), TAM/SAM/SOM (Addressable Market), Competitive Analysis (Competitors, Positioning, Differentiation), Market Needs (Pain Points, Unmet Needs, Opportunities), Opportunity & Timing (Window, Urgency, Market Readiness)
- [x] Implement `generateBRDTemplate()` — sections: Business Objectives (Goals, Strategic Alignment), Stakeholders (Sponsors, Decision-Makers, Influence Map), Business Rules & Constraints (Policies, Regulations, Budget), Success Metrics & ROI (KPIs, Expected Returns, Payback Period), Dependencies (Organizational, Technical, External)
- [x] Implement `generateURDTemplate()` — sections: User Personas (Profiles, Goals, Pain Points, Context), User Journeys (Current State, Desired State, Touchpoints), User Requirements (Functional Needs Per Persona), Usability Requirements (Accessibility, Learnability, Efficiency), Acceptance Criteria (User-Facing Validation Conditions)

### Phase 2: MCP Integration

- [x] Update @internal/mcp/server.go `mcpServerInstructions`: add types to vision listing, add 3 WHEN TO CREATE entries, add 5 cross-track disambiguation rules, update REQUIREMENTS APPROACH section to describe three tracks
- [x] Update @internal/mcp/tools/create_document.go tool description: add 3 type entries with required sections, add 5 disambiguation rules
- [x] Update @internal/mcp/tools/list_documents.go: add new types to `types` parameter description

### Phase 3: Tests

- [x] Add 3 entries to template table-driven tests in @templates/templates_test.go
- [x] Add 3 entries to AllTypes test in @internal/mcp/tools/create_document_test.go
- [x] Update ValidTypes count assertions and TypesByCategory vision count

### Phase 4: Documentation

- [x] Update @.archcore/dir/categories-and-document-types.doc.md: add 3 types to vision table, add cross-track disambiguation entries, document three-track model
- [x] Update @.archcore/document-types/prd-vs-iso-29148-requirements-strategy.idea.md: reference three tracks (Product, Sources, ISO) instead of two tracks (Simple, Detailed)

## Acceptance Criteria

- [x] `go build -o archcore .` succeeds
- [x] `go test ./...` passes (all existing + new tests)
- [x] `./archcore mcp` shows all 3 new types in MCP instructions
- [x] Creating a document of each new type via MCP renders the correct template
- [x] Disambiguation rules clearly distinguish mrd/brd/urd from existing prd and ISO types (brs/strs)
- [x] Cross-track relations (e.g., MRD→BRS via `implements`) work with existing relation infrastructure

## Dependencies

| Dependency | Type | Status |
|------------|------|--------|
| Existing template system in templates.go | Internal | Ready |
| MCP server instructions in server.go | Internal | Ready |
| ISO 29148 types (BRS, StRS, SyRS, SRS) | Internal | Complete |
| Existing `implements` relation type | Internal | Ready |

## Notes

### Disambiguation Strategy (5 Cross-Track Rules)

These structural cues help agents choose the right type across all three tracks:

1. **mrd vs prd**: MRD analyzes the MARKET (TAM/SAM/SOM, competitors, timing) without proposing a solution. PRD proposes a PRODUCT with requirements and solution overview.
2. **mrd vs brs**: MRD is MARKET ANALYSIS (external-facing, pre-decision). BRS is BUSINESS REQUIREMENTS (internal-facing, post-decision, ISO-structured).
3. **brd vs brs**: BRD justifies a BUSINESS INITIATIVE (ROI, stakeholders, budget). BRS formalizes business REQUIREMENTS per ISO structure (mission, goals, operational concept).
4. **brd vs prd**: BRD focuses on BUSINESS JUSTIFICATION (ROI, budget, organizational impact). PRD focuses on PRODUCT DEFINITION (features, user stories, solution).
5. **urd vs strs**: URD captures user needs via PERSONAS and JOURNEYS (discovery-oriented). StRS formalizes stakeholder requirements per ISO structure with ConOps (specification-oriented).

### Source-to-ISO Mapping

| Source | ISO Target | Rationale |
|--------|-----------|-----------|
| MRD | BRS | Market requirements get formalized as business requirements |
| BRD | BRS / StRS | Business goals decompose into formal business and stakeholder requirements |
| URD | StRS | User needs become formal stakeholder requirements |