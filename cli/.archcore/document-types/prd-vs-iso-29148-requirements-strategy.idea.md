---
title: "Three Requirements Tracks: PRD, Sources, and ISO 29148"
status: accepted
tags:
  - "document-types"
---

## Idea

Archcore supports three requirements engineering approaches side by side: a **product track** (PRD), a **sources track** (MRD → BRD → URD), and a **specification track** (ISO 29148 cascade: BRS → StRS → SyRS → SRS). Users choose based on project complexity, team size, and regulatory needs. Archcore actively guides this choice through MCP instructions and disambiguation rules.

### Problem / Opportunity

- The PRD type is a pragmatic hybrid that covers business context, stakeholder needs, and functional requirements in a single document. This works well for most projects.
- MRD/BRD/URD capture where requirements come from — market analysis, business justification, user needs — enabling structured discovery before engineering begins.
- ISO/IEC/IEEE 29148:2018 defines four specialized requirements specification types (BRS, StRS, SyRS, SRS) that decompose requirements into progressively detailed levels. This is essential for complex, regulated, or multi-team systems.
- The three tracks address different axes: PRD (all-in-one), Sources (where requirements originate), ISO (how requirements refine).

## Value

### For Users

- **Simple projects** stay simple: one PRD covers everything needed. No pressure to create 4+ separate documents.
- **Discovery-oriented projects** get structured capture: MRD for market, BRD for business, URD for users.
- **Complex projects** get proper structure: the ISO cascade ensures traceability from business objectives through to testable software specs.
- Clear guidance on which track to use and when to combine tracks.

### For AI Agents

- Agents can read a PRD and immediately start implementing for simple features.
- For discovery, agents follow the sources track: MRD provides market context, BRD provides business justification, URD provides user scenarios.
- For complex features, agents follow the ISO cascade: BRS provides alignment context, StRS provides operational scenarios, SyRS provides system boundaries, SRS provides per-endpoint specifications they can directly translate to code and tests.
- The `implements` relation naturally connects sources to specifications and cascades through the ISO levels.

### For Business

- Positions archcore as a tool that scales from indie developer to enterprise — the same platform supports simple, discovery, and ISO-compliant requirements workflows.

## Possible Implementation

### Three Tracks, One Platform

**Product track (PRD)**:

- Single document covering vision, problem, requirements, and solution overview
- Best for: individual features, small teams, rapid prototyping, internal tools
- The PRD template remains unchanged — it already works well

**Sources track (MRD → BRD → URD)**:

- Three documents capturing market, business, and user perspectives
- Best for: product teams doing discovery, business analysts, stakeholder alignment
- MRD (market landscape) → BRD (business justification) → URD (user needs)

**Specification track (ISO 29148 cascade)**:

- Four documents with formal traceability between levels
- Best for: regulated systems, multi-team projects, hardware+software integration, external contracts, complex distributed systems
- Each level adds precision: BRS (why) → StRS (who needs what) → SyRS (system behavior) → SRS (software behavior)

### How Tracks Relate

Sources feed into the ISO cascade via `implements` relations:

| Source Type | Feeds Into (ISO) | Relationship |
|-------------|-------------------|--------------|
| MRD (market needs) | BRS (business formalization) | Market requirements get formalized as business requirements |
| BRD (business objectives) | BRS / StRS (business + stakeholder formalization) | Business goals decompose into formal business and stakeholder requirements |
| URD (user needs) | StRS (stakeholder requirements) | User needs become formal stakeholder requirements with ConOps |
| PRD (condensed) | ≈ all four ISO types | PRD is a pragmatic hybrid covering all levels |

### How PRD Relates to ISO Types

The PRD is essentially a condensed version that merges all four ISO levels:

| PRD Section                                   | Equivalent ISO Type | ISO Section                                            |
| --------------------------------------------- | ------------------- | ------------------------------------------------------ |
| Vision, Strategic Alignment                   | BRS                 | §9.3.7 Mission, Goals and Objectives                   |
| Problem Statement, Target Users, User Stories | StRS                | §9.4.15 User Requirements, §9.4.16 Operational Concept |
| Non-Functional Requirements, Constraints      | SyRS                | §9.5.5-§9.5.13 System Requirements                     |
| Functional Requirements (P0/P1/P2)            | SRS                 | §9.6.10-§9.6.12 Specified Requirements                 |
| Solution Overview                             | SyRS/SRS            | §9.5.4 System Overview, §9.6.4 Product Perspective     |

Users can start with a PRD and later decompose it into ISO documents when complexity demands it.

### User Control

Users can:

- Mix approaches freely (some features use PRD, others use full cascade)
- Start with PRD and gradually decompose into ISO types as complexity grows
- Use partial cascade (e.g., only BRS + SRS, skipping StRS and SyRS)
- Combine sources + ISO (write MRD/BRD/URD first, then formalize into BRS/StRS)

## Risks and Constraints

### Potential Risks

- **Type proliferation**: 10 vision types total (prd + idea + plan + mrd + brd + urd + brs + strs + syrs + srs). Mitigated by organizing as three clear tracks with disambiguation rules.
- **Over-engineering**: Teams might feel pressured to use full cascade when PRD would suffice. Mitigated by clearly positioning PRD as the default/recommended approach.
- **Agent selection errors**: Agents might create source documents when ISO types would be more appropriate (or vice versa). Mitigated by structural disambiguation cues and cross-layer rules in MCP instructions.

### Known Constraints

- ISO 29148:2018 is a copyrighted standard — templates are inspired by the structure but do not reproduce copyrighted text verbatim.
- MRD/BRD/URD are industry conventions, not formal standards — templates reflect common practice from product management and business analysis (BABOK/IIBA).
- Traceability between documents relies on the existing `implements` relation type — no new infrastructure needed.

## Next Steps

- [x] Implement MRD, BRD, URD document types (templates, MCP descriptions, disambiguation rules)
- [x] Implement BRS, StRS, SyRS, SRS document types (templates, MCP descriptions, disambiguation rules)
- [x] Add disambiguation rules to MCP server instructions and create_document tool
- [x] Update categories-and-document-types.doc.md with all three tracks documented
- [ ] Future: Add track guidance in the CLI suggesting simple vs. sources vs. ISO approach. Track guidance MUST NOT return as an MCP prompts capability — that was implemented and removed; see the ADR on removing the MCP track prompts.
- [ ] Future: Add `requirements_approach` setting to settings.json
- [ ] Future: Add `doctor` check suggesting ISO decomposition for large PRDs

## Related Materials

- ISO/IEC/IEEE 29148:2018 — Systems and software engineering — Life cycle processes — Requirements engineering
- BABOK Guide (Business Analysis Body of Knowledge) — IIBA standard for business analysis practices
- Existing PRD template in @templates/templates.go
- MCP server instructions in @internal/mcp/server.go
- Companion idea: @.archcore/document-types/mrd-brd-urd-requirement-sources.idea.md
