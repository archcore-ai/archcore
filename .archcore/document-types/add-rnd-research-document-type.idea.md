---
title: "Add rnd Research Document Type"
status: draft
tags:
  - "document-types"
---

## Idea

Add `rnd` as a new vision-layer document type for focused research that produces a recommendation and a concrete next action.

`rnd` fills the gap between early exploration and execution. It can be used before an `idea`, after an `idea`, or before a `plan`, depending on what needs to be validated.

### Problem / Opportunity

- Archcore currently has `idea` and `plan`, plus discovery/specification tracks, but no dedicated document for recommendation-oriented research.
- Users often need to investigate a question before committing to an idea or before writing a plan.
- Today that work is awkwardly forced into `idea`, mixed into `plan`, or left outside `.archcore/` entirely.
- Research work is not universal. Product/business research and technical research need different templates and prompts, even if they belong to the same document family.

## Value

### For Users

- Gives a clear home for exploratory work that is more structured than a note and less committed than a plan.
- Supports flexible workflows: research before idea, research after idea, or research before plan.
- Makes it easier to capture why a team should proceed, refine, defer, or stop.

### For Business

- Preserves important discovery work in the repo instead of losing it in chat threads or ad hoc docs.
- Improves decision quality by requiring a recommendation and next action, not just scattered findings.
- Helps teams separate evidence-gathering from implementation planning.

### For Team

- Gives AI agents a clearer type to create when the user is asking for investigation rather than requirements or execution.
- Reduces misuse of `idea` for evidence collection and misuse of `plan` for unresolved research.
- Creates a reusable pattern for both product/business and technical R&D without adding two separate top-level types.

## Possible Implementation

### Technical Approach

- Add `rnd` as a new `vision` document type.
- Require a `variant` field for `rnd` with values `product` or `technical`.
- Generate different templates based on variant instead of using one universal template.
- Make every `rnd` document end with a recommendation and a next action.

Suggested product/business `rnd` sections:

- Research Goal
- Context and Trigger
- Research Questions / Hypotheses
- Inputs and Evidence
- Findings
- Product / Business Implications
- Recommendation
- Next Action
- Risks and Unknowns
- Related Materials

Suggested technical `rnd` sections:

- Research Goal
- Context and Trigger
- Research Questions / Hypotheses
- Current Technical Context
- Options / Experiments
- Findings
- Technical Implications
- Recommendation
- Next Action
- Risks and Unknowns
- Related Materials

### Integrations

- `create_document` should support `type=rnd` with `variant=product|technical`.
- MCP instructions should explain `rnd` vs `idea`, `rnd` vs `plan`, and `rnd` vs source-track documents.
- Relations should remain flexible: `rnd related idea`, `plan depends_on rnd`, and `plan implements rnd` when research already defines the path forward.

## Risks and Constraints

### Potential Risks

- Type proliferation: adding another vision type can increase selection complexity.
- Template confusion: users may not understand when to use `rnd` instead of `idea`.
- Variant drift: product and technical variants may become too broad or overlap.

### Known Constraints

- `rnd` should stay in the vision layer, not knowledge.
- This should remain one type with two variants, not two separate top-level types.
- `variant` must be preserved in frontmatter across create, read, update, validate, and sync flows.
- The template must not become universal; the split between product and technical research is intentional.

## Next Steps

- [ ] Add `rnd` to the valid document types and map it to `vision`
- [ ] Add `variant` support for `rnd` in document tooling
- [ ] Implement separate `product` and `technical` templates
- [ ] Update MCP instructions and type-selection rules
- [ ] Add tests for template generation and frontmatter round-tripping

## Related Materials

- `.archcore/dir/categories-and-document-types.doc.md`
- `.archcore/document-types/working-with-requirements-tracks.guide.md`
- Proposed implementation plan for adding `rnd`
