---
title: "Working with Requirements Tracks in Archcore"
status: accepted
tags:
  - "document-types"
---

## Overview

How to choose and use Archcore's three requirements engineering tracks: **Product** (PRD), **Sources** (MRD → BRD → URD), and **ISO** (BRS → StRS → SyRS → SRS). Each track serves a different level of project complexity and formality.

### Target Audience

- Product managers and engineers creating requirements documents
- AI agents deciding which document types to create for a project

### Time Estimate

- 5 minutes to choose a track
- 10-30 minutes to set up initial documents per track

## Prerequisites

### Required Knowledge

- Familiarity with archcore document types (see .archcore/dir/categories-and-document-types.doc.md)
- Understanding of your project's complexity, team size, and regulatory needs

### Before You Start

- Run `list_documents` to see existing requirements documents in the project
- Assess whether you need discovery, specification, or both

## Steps

### Step 1: Choose Your Track

Use these signals to pick the right track:

| Signal | Recommended Track |
|--------|-------------------|
| Single team, internal tool, quick feature | **Product** (prd) |
| Need stakeholder alignment, doing product discovery | **Sources** (mrd → brd → urd) |
| Regulated system, formal traceability required | **ISO** (brs → strs → syrs → srs) |
| Complex project needing both discovery AND formal specs | **Sources + ISO** combined |

**Default to Product track (PRD)** — upgrade to Sources or ISO only when the project signals demand it.

### Step 2: Create Documents for Your Track

**Product track** — create one document:

```
create_document type=prd filename=my-feature
```

**Sources track** — create in discovery order:

```
create_document type=mrd filename=market-analysis       # 1. Market landscape
create_document type=brd filename=business-case          # 2. Business justification
create_document type=urd filename=user-needs              # 3. User requirements
```

**ISO track** — create in cascade order:

```
create_document type=brs filename=business-requirements   # 1. Business mission/goals
create_document type=strs filename=stakeholder-requirements # 2. Stakeholder needs
create_document type=syrs filename=system-requirements     # 3. System boundary/interfaces
create_document type=srs filename=software-requirements    # 4. Per-function specs
```

**Combined (Sources + ISO)** — create sources first, then formalize:

```
# Discovery phase
create_document type=mrd filename=market-analysis
create_document type=brd filename=business-case
create_document type=urd filename=user-needs

# Formalization phase
create_document type=brs filename=business-requirements
create_document type=strs filename=stakeholder-requirements
```

### Step 3: Link Documents with Relations

**Same-track peers** — use `related`:

```
add_relation source=mrd-file target=brd-file type=related
add_relation source=brd-file target=urd-file type=related
```

**Sources → Specifications** — use `implements` (spec is source, source doc is target):

```
add_relation source=brs-file target=mrd-file type=implements   # BRS formalizes MRD
add_relation source=brs-file target=brd-file type=implements   # BRS formalizes BRD
add_relation source=strs-file target=urd-file type=implements  # StRS formalizes URD
```

**ISO cascade** — use `implements`:

```
add_relation source=strs-file target=brs-file type=implements  # StRS decomposes BRS
add_relation source=syrs-file target=strs-file type=implements # SyRS decomposes StRS
add_relation source=srs-file target=syrs-file type=implements  # SRS decomposes SyRS
```

**PRD to ISO types** — use `related` (PRD is an alternative path):

```
add_relation source=prd-file target=brs-file type=related
```

### Step 4: Evolve Requirements Over Time

Requirements tracks are not static. Common evolution patterns:

- **Start simple, grow later**: Begin with a PRD. When complexity grows, decompose into ISO types. Link the PRD via `related` to the new specs.
- **Add sources retroactively**: Already have ISO specs? Create MRD/BRD/URD to document where requirements came from and link with `implements`.
- **Use partial cascades**: Not every project needs all four ISO levels. BRS + SRS (skipping StRS/SyRS) is valid when intermediate levels add no value. Use `srs implements brs` directly.
- **Mix tracks per feature**: Some features use PRD, others use full Sources + ISO. This is fine — tracks coexist within a project.

### Step 5: Verify Traceability

Run `list_relations` and check:

- Every specification traces back to at least one source (via `implements`)
- ISO cascade relations flow correctly (brs → strs → syrs → srs)
- Same-layer documents are connected (via `related`)
- No orphaned requirements documents without any relations

## Verification

- [ ] Each requirements document has the correct type for its purpose
- [ ] Relations connect sources to specifications via `implements`
- [ ] ISO cascade relations flow in the right direction
- [ ] No source document (mrd/brd/urd) contains formal ISO-structured content
- [ ] PRD is used only for simple projects or as an alternative to the full cascade

## Common Issues

### Over-engineering simple projects

**Symptom**: Creating MRD + BRD + URD + BRS + StRS + SyRS + SRS for a small internal feature.
**Cause**: Defaulting to the most comprehensive track instead of assessing actual needs.
**Solution**: Start with a single PRD. Add Sources or ISO only when the project signals demand it (multiple stakeholders, regulation, formal traceability).

### Mixing sources and specifications

**Symptom**: BRD contains formal ISO-structured requirements (mission statements, operational concepts, ConOps).
**Cause**: Writing specification content in a source document.
**Solution**: Keep sources informal and discovery-oriented. Create BRS/StRS when you need to formalize. See rule: sources-are-not-specifications.

### Missing relations between layers

**Symptom**: BRS exists alongside MRD but no `implements` relation connects them.
**Cause**: Creating specification documents without linking them to their sources.
**Solution**: Always add `implements` relation from spec to source immediately after creating. See rule: source-to-specification-relations.

### Wrong relation direction

**Symptom**: `mrd implements brs` instead of `brs implements mrd`.
**Cause**: Confusion about which document is the source of the relation.
**Solution**: The MORE SPECIFIC document (specification) is always the source of `implements`. It reads: "BRS implements what MRD describes."

## Related Resources

- .archcore/dir/categories-and-document-types.doc.md — full type reference and disambiguation rules
- .archcore/document-types/source-to-specification-relations.rule.md — relation conventions
- .archcore/document-types/sources-are-not-specifications.rule.md — layer separation rule
- .archcore/document-types/writing-spec-documents.guide.md — guide for the `spec` type
- .archcore/document-types/spec-type-usage.rule.md — when to use `spec` vs other types