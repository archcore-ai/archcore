---
title: "MRD/BRD/URD as Requirement Sources Track Alongside PRD and ISO 29148"
status: accepted
tags:
  - "document-types"
---

## Idea

Add MRD (Market Requirements Document), BRD (Business Requirements Document), and URD (User Requirements Document) as a **requirement sources** track alongside PRD (simple) and ISO 29148 (formal decomposition). These represent the horizontal axis of requirements engineering — where requirements come from — complementing ISO 29148's vertical axis of how requirements refine.

### Problem / Opportunity

- Two orthogonal axes of requirements engineering exist:
  - **Source axis** (MRD/BRD/URD): captures *where* requirements originate — market analysis, business objectives, user needs
  - **Decomposition axis** (ISO 29148: BRS→StRS→SyRS→SRS): captures *how* requirements refine from business goals to testable software specs
- Archcore supports the decomposition axis (ISO cascade) and a simple path (PRD), but lacks the source axis used in product management (MRD/PRD) and business analysis (BABOK/IIBA frameworks: BRD/URD)
- These are complementary, not competing — an MRD can feed into a BRS, a URD can feed into a StRS, and all can feed into a PRD for simpler projects

## Value

### For Users

- **Separate perspectives**: market analysis (MRD), business justification (BRD), and user needs (URD) get their own focused documents instead of being crammed into a single PRD
- **Product management workflow**: MRD→BRD→URD matches how product teams actually discover and refine requirements before engineering begins
- **Business analysis workflow**: BRD and URD align with BABOK knowledge areas, supporting formal BA processes

### For AI Agents

- **Structured inputs**: agents can extract market constraints from MRD, business rules from BRD, and user scenarios from URD — each with purpose-built templates
- **Cross-axis linking**: agents follow `implements` relations to trace from requirement sources into the ISO cascade or directly into PRD
- **Better disambiguation**: template sections are structurally distinct — MRD has TAM/SAM/SOM, BRD has ROI/Business Rules, URD has Personas/Journeys

### For Business

- Covers product management and business analysis workflows alongside engineering requirements
- Positions archcore as a tool that spans the full requirements lifecycle: discovery (sources) → specification (ISO/PRD) → implementation (plan)

## Possible Implementation

### Three Tracks, One Platform

**Product track (PRD)** — simple, all-in-one:
- Single document covering vision, problem, requirements, and solution overview
- Best for: individual features, small teams, rapid prototyping, internal tools
- The PRD template remains unchanged

**Sources track (MRD → BRD → URD)** — where requirements come from:
- Three documents capturing market, business, and user perspectives
- Best for: product teams doing discovery, business analysts, stakeholder alignment
- MRD (market landscape) → BRD (business justification) → URD (user needs)

**ISO track (BRS → StRS → SyRS → SRS)** — how requirements decompose:
- Four documents with formal traceability between levels
- Best for: regulated systems, multi-team projects, complex distributed systems
- BRS (why) → StRS (who needs what) → SyRS (system behavior) → SRS (software behavior)

### How Sources Relate to ISO Types

Sources feed into the ISO cascade. The cross-axis link uses the `implements` relation:

| Source Type | Feeds Into (ISO) | Relationship |
|-------------|-------------------|--------------|
| MRD (market needs) | BRS (business formalization) | Market requirements get formalized as business requirements |
| BRD (business objectives) | BRS / StRS (business + stakeholder formalization) | Business goals decompose into formal business and stakeholder requirements |
| URD (user needs) | StRS (stakeholder requirements) | User needs become formal stakeholder requirements with ConOps |
| PRD (condensed) | ≈ all four ISO types | PRD is a pragmatic hybrid covering all levels |

### Template Sections

**MRD (Market Requirements Document)**:
- Market Landscape (industry trends, market size, dynamics)
- TAM / SAM / SOM (addressable market analysis)
- Competitive Analysis (competitors, positioning, differentiation)
- Market Needs (pain points, unmet needs, opportunities)
- Opportunity & Timing (window, urgency, market readiness)

**BRD (Business Requirements Document)**:
- Business Objectives (goals, strategic alignment)
- Stakeholders (sponsors, decision-makers, influence map)
- Business Rules & Constraints (policies, regulations, budget)
- Success Metrics & ROI (KPIs, expected returns, payback period)
- Dependencies (organizational, technical, external)

**URD (User Requirements Document)**:
- User Personas (profiles, goals, pain points, context)
- User Journeys (current state, desired state, touchpoints)
- User Requirements (functional needs per persona)
- Usability Requirements (accessibility, learnability, efficiency)
- Acceptance Criteria (user-facing validation conditions)

### Disambiguation Rules

These structural cues help agents choose the right type across all three tracks:

1. **mrd vs prd**: MRD analyzes the MARKET (TAM/SAM/SOM, competitors, timing) without proposing a solution. PRD proposes a PRODUCT with requirements and solution overview.
2. **brd vs brs**: BRD justifies a BUSINESS INITIATIVE (ROI, stakeholders, budget). BRS formalizes business REQUIREMENTS per ISO structure (mission, goals, operational concept).
3. **urd vs strs**: URD captures user needs via PERSONAS and JOURNEYS (discovery-oriented). StRS formalizes stakeholder requirements per ISO structure with ConOps (specification-oriented).
4. **mrd vs brs**: MRD is MARKET ANALYSIS (external-facing, pre-decision). BRS is BUSINESS REQUIREMENTS (internal-facing, post-decision, ISO-structured).
5. **brd vs prd**: BRD focuses on BUSINESS JUSTIFICATION (ROI, budget, organizational impact). PRD focuses on PRODUCT DEFINITION (features, user stories, solution).

## Risks and Constraints

### Potential Risks

- **Type proliferation**: 10 vision types total (prd + idea + plan + brs + strs + syrs + srs + mrd + brd + urd). Mitigated by organizing as three clear tracks with disambiguation rules — users pick a track, not from a flat list of 10.
- **Track confusion**: Users might not know which track to use. Mitigated by clear guidance: PRD for simple projects, Sources for discovery/PM workflow, ISO for formal decomposition.
- **Agent selection errors**: Agents might create source documents when ISO types (or vice versa) would be more appropriate. Mitigated by structural disambiguation cues (section-based, not abstract definitions) and cross-track rules.

### Known Constraints

- MRD/BRD/URD are industry conventions, not formal standards — templates should reflect common practice from product management and business analysis (BABOK/IIBA) without claiming standardization.
- Cross-axis relations (e.g., MRD→BRS) use the existing `implements` relation type — no new infrastructure needed.
- The three-track guidance system (suggesting which track to use) is a future enhancement, not part of the initial type implementation.

## Next Steps

- [ ] Implement MRD, BRD, URD document types (templates, MCP descriptions, disambiguation rules)
- [ ] Add cross-track disambiguation rules to MCP server instructions and create_document tool
- [ ] Update categories-and-document-types.doc.md with all three tracks documented
- [ ] Update prd-vs-iso-29148-requirements-strategy.idea.md to reference three tracks instead of two
- [ ] Future: Add track guidance prompts in CLI suggesting simple vs. sources vs. ISO approach

## Related Materials

- BABOK Guide (Business Analysis Body of Knowledge) — IIBA standard for business analysis practices
- Existing PRD template in @templates/templates.go
- MCP server instructions in @internal/mcp/server.go
- Companion idea: @.archcore/document-types/prd-vs-iso-29148-requirements-strategy.idea.md
- ISO implementation plan: @.archcore/document-types/iso-29148-document-types-implementation.plan.md