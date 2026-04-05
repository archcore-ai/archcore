---
title: "Creating a Batch of Related Skills"
status: draft
tags:
  - "experience"
  - "plugin"
  - "skills"
---

## What

A pattern for efficiently creating multiple SKILL.md files when building out coverage for a category of Archcore document types (e.g., all 6 knowledge types, all 10 vision types, or all 2 experience types).

## When to Use

- Adding skill coverage for an entire Archcore category at once
- Building the initial 18 skills during Phase 2 of plugin development
- Updating all skills in a category after a structural change to the Skill File Structure Standard

## Steps

### 1. Pick the batch scope

Choose a category to batch: knowledge (6 types), vision (10 types), or experience (2 types). Start with the category you understand best.

### 2. Create one exemplar skill

Pick the most representative type in the category (e.g., ADR for knowledge, PRD for vision, task-type for experience). Write its SKILL.md thoroughly — this becomes the quality bar and structural template for the batch.

Get the exemplar reviewed and validated:

- Follows all 7 sections in order
- Example Workflow uses MCP tools
- Description field has precise trigger conditions
- Contrast with similar types is clear
- Under 500 lines

### 3. Extract the pattern

Identify which parts of the exemplar are:

- **Fixed**: section headings, frontmatter structure, MCP tool references — these stay the same
- **Variable**: type name, description triggers, section descriptions, best practices, relation flows, example parameters — these change per type

### 4. Create remaining skills in the batch

For each remaining type in the category:

1. Copy the exemplar's structure (not content)
2. Study that type's template via `create_document` (no content) + `remove_document`
3. Fill in the variable sections with type-specific content
4. Pay special attention to "When to Use" — the contrast with other types must be accurate
5. Verify the Example Workflow uses realistic parameters for this type

### 5. Cross-validate the batch

After all skills in the batch are created:

- Read all "When to Use" sections together — ensure no ambiguity between types
- Check all "Relation Guidance" sections — ensure flows are consistent across the batch
- Verify all Example Workflows use correct MCP tool names
- Run `/reload-plugins` and test each skill's activation trigger

## Example

Creating knowledge category skills (6 types):

1. **Exemplar**: `skills/adr/SKILL.md` — ADR is well-understood, has clear contrasts (vs RFC, vs rule)
2. **Extract pattern**: sections are fixed, but triggers differ (ADR: "decision made"; RFC: "proposal to discuss"; rule: "team standard")
3. **Batch create**: rfc, rule, guide, doc, spec — each with unique triggers, sections, and relation guidance
4. **Cross-validate**: ensure "When to Use" for ADR clearly separates from RFC, rule from guide, doc from spec

## Things to Watch Out For

- **Copy-paste drift**: After copying the exemplar structure, ensure all content is customized. Leftover ADR-specific content in an RFC skill is confusing.
- **Trigger overlap**: Two skills with similar descriptions will compete for activation. The "When to Use" section must clearly differentiate.
- **Relation guidance inconsistency**: If the ADR skill says "rules implement ADRs" but the rule skill doesn't mention ADRs in its incoming relations, agents get confused.
- **Template embedding**: Don't copy template sections verbatim into skills. The templates change with CLI versions. Reference them instead.
- **Batch fatigue**: Quality drops on the last few skills in a batch. Review the final skills in the batch with the same rigor as the first.
