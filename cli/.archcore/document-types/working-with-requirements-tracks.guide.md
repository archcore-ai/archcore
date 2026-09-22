---
title: "Working with Requirements Tracks in Archcore"
status: accepted
tags:
  - "document-types"
---

## Overview

Choose and use one of Archcore's three requirements tracks: Product (`prd`), Sources (`mrd` → `brd` → `urd`), and ISO (`brs` → `strs` → `syrs` → `srs`). Each track serves a different level of project complexity and formality.

Intended readers:

- A product manager or engineer creating requirements documents.
- An AI agent deciding which document types a project needs.

## Prerequisites

- Familiarity with the Archcore document types. The related reference document on categories and document types covers them.
- Knowledge of the project's complexity, team size, and regulatory needs.

## Inputs

- The list of existing requirements documents: run `list_documents` before you start.
- A decision on whether the project needs discovery, specification, or both.

## Procedure

### 1. Choose the track

| Signal | Recommended track |
|--------|-------------------|
| Single team, internal tool, quick feature | Product (`prd`) |
| Stakeholder alignment needed, product discovery under way | Sources (`mrd` → `brd` → `urd`) |
| Regulated system, formal traceability required | ISO (`brs` → `strs` → `syrs` → `srs`) |
| Complex project that needs discovery and formal specifications | Sources and ISO combined |

Default to the Product track. Move to Sources or ISO only when a signal above demands it.

### 2. Create the documents for the chosen track

Product track — one document:

```
create_document type=prd filename=my-feature
```

Sources track — in discovery order:

```
create_document type=mrd filename=market-analysis          # 1. Market landscape
create_document type=brd filename=business-case            # 2. Business justification
create_document type=urd filename=user-needs               # 3. User requirements
```

ISO track — in cascade order:

```
create_document type=brs filename=business-requirements     # 1. Business mission and goals
create_document type=strs filename=stakeholder-requirements # 2. Stakeholder needs
create_document type=syrs filename=system-requirements      # 3. System boundary and interfaces
create_document type=srs filename=software-requirements     # 4. Per-function specifications
```

Combined — create the sources first, then formalize:

```
# Discovery phase
create_document type=mrd filename=market-analysis
create_document type=brd filename=business-case
create_document type=urd filename=user-needs

# Formalization phase
create_document type=brs filename=business-requirements
create_document type=strs filename=stakeholder-requirements
```

### 3. Link the documents with relations

Same-track peers use `related`:

```
add_relation source=mrd-file target=brd-file type=related
add_relation source=brd-file target=urd-file type=related
```

A specification links to its source with `implements`; the specification is the relation source:

```
add_relation source=brs-file target=mrd-file type=implements   # BRS formalizes MRD
add_relation source=brs-file target=brd-file type=implements   # BRS formalizes BRD
add_relation source=strs-file target=urd-file type=implements  # StRS formalizes URD
```

The ISO cascade uses `implements`:

```
add_relation source=strs-file target=brs-file type=implements  # StRS decomposes BRS
add_relation source=syrs-file target=strs-file type=implements # SyRS decomposes StRS
add_relation source=srs-file target=syrs-file type=implements  # SRS decomposes SyRS
```

A PRD links to an ISO type with `related`, because the PRD is an alternative path:

```
add_relation source=prd-file target=brs-file type=related
```

### 4. Evolve the requirements over time

A track is not static. These patterns recur:

- Start simple, grow later. Begin with a PRD. WHEN complexity grows, decompose into ISO types and link the PRD to the new specifications with `related`.
- Add sources retroactively. WHEN ISO specifications already exist, create the MRD, BRD, and URD that record where the requirements came from, and link them with `implements`.
- Use a partial cascade. Not every project needs all four ISO levels. BRS plus SRS, skipping StRS and SyRS, is valid when the intermediate levels add nothing; use `srs implements brs` directly.
- Mix tracks per feature. Some features use a PRD while others use the full Sources and ISO chain. Tracks coexist within one project.

### 5. Verify traceability

Run `list_relations` and check the graph.

Expected result:

- Every specification traces back to at least one source through `implements`.
- The ISO cascade flows `brs` → `strs` → `syrs` → `srs`.
- Same-layer documents are connected through `related`.
- No requirements document is left without a relation.

## Verification

- Each requirements document carries the correct type for its purpose.
- Relations connect sources to specifications through `implements`.
- The ISO cascade relations point in the documented direction.
- No source document (`mrd`, `brd`, `urd`) carries formal ISO-structured content.
- A PRD is used for a simple project, or as an alternative to the full cascade.

## Troubleshooting

### Over-engineering a simple project

Symptom: MRD, BRD, URD, BRS, StRS, SyRS, and SRS all created for a small internal feature.
Cause: defaulting to the most comprehensive track instead of assessing the actual need.
Solution: start with one PRD. Add Sources or ISO only when the project signals demand it — multiple stakeholders, regulation, or formal traceability.

### Mixing sources and specifications

Symptom: a BRD carries formal ISO-structured requirements such as mission statements, operational concepts, or ConOps.
Cause: specification content written into a source document.
Solution: keep sources informal and discovery-oriented. Create a BRS or StRS when formalization is needed. The related rule on sources not being specifications states the constraint.

### Missing relations between layers

Symptom: a BRS exists alongside an MRD with no `implements` relation between them.
Cause: a specification created without a link to its source.
Solution: add the `implements` relation from the specification to the source immediately after creating the specification. The related rule on source-to-specification relations states the convention.

### Wrong relation direction

Symptom: `mrd implements brs` instead of `brs implements mrd`.
Cause: confusion about which document is the relation source.
Solution: the more specific document, the specification, is always the relation source. The edge reads "BRS implements what MRD describes".
