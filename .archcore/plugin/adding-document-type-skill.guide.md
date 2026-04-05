---
title: "Adding a New Document Type Skill"
status: draft
tags:
  - "development"
  - "plugin"
  - "skills"
---

## Prerequisites

- Familiarity with the target Archcore document type (purpose, sections, use cases)
- Understanding of the Skill File Structure Standard (rule)
- Plugin development environment set up (see Plugin Development Guide)
- Access to the Archcore CLI to test templates: `archcore mcp` starts the MCP server

## Steps

### 1. Study the document type template

Call `create_document` with only `type` and `filename` (no `content`) to see the generated template. This shows the required sections and placeholder text:

```
create_document(type="adr", filename="template-preview")
```

Read the created file to see the full template, then delete it:

```
remove_document(path="template-preview.adr.md")
```

### 2. Study real-world examples

Search the Archcore documentation and reference materials for examples of well-written documents of this type. Note what makes them effective: clarity, completeness, specificity.

### 3. Create the skill directory

```bash
mkdir -p skills/<type-name>
```

Use the exact Archcore type identifier: `adr`, `rfc`, `rule`, `guide`, `doc`, `spec`, `prd`, `idea`, `plan`, `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs`, `task-type`, `cpat`.

### 4. Write the SKILL.md frontmatter

```yaml
---
name: archcore-<type-name>
description: >
  <Describe specific triggers and context signals. Be precise —
  this determines when Claude activates the skill automatically.
  Include key phrases users might say and situations where this
  type is appropriate.>
---
```

The `description` is critical — it's the activation trigger. Be specific about when this type should be used AND when it should NOT be used (to help Claude disambiguate similar types).

### 5. Write the 7 required sections

Follow this order exactly:

1. **Overview**: What the type is, its purpose, its virtual category (vision/knowledge/experience). Keep to 3-5 sentences.

2. **When to Use**: Specific scenarios and conversation cues. Include contrast with similar types (e.g., ADR vs RFC, PRD vs MRD, guide vs doc). Use bullet lists for clarity.

3. **Required Sections**: List each section from the template with a 1-2 sentence description of what belongs there. Do NOT copy the template verbatim — summarize and add guidance.

4. **Best Practices**: Domain-specific tips for writing a high-quality document of this type. What separates a good document from a mediocre one? Include 4-6 actionable points.

5. **Common Mistakes**: Pitfalls to avoid. Include type confusion (using the wrong type), structural issues (missing sections, wrong level of detail), and content issues (too vague, too verbose). Include 3-5 specific mistakes.

6. **Relation Guidance**: Which relation types and target document types are commonly used. Include both incoming and outgoing relations. Show typical flows this type participates in.

7. **Example Workflow**: A concrete, realistic example showing:
   - The `create_document` MCP call with appropriate parameters
   - A follow-up `add_relation` call connecting to an existing document
   - Brief explanation of why these parameters and relations were chosen

### 6. Validate against the standard

Check your SKILL.md against the Skill File Structure Standard:

- [ ] Frontmatter has `name` (prefixed `archcore-`) and `description`
- [ ] All 7 sections present in correct order
- [ ] Example Workflow uses `create_document`, not Write/Edit
- [ ] No embedded template content (reference template system instead)
- [ ] Under 500 lines

### 7. Test the skill

```bash
claude --plugin-dir .
```

Then `/reload-plugins` to load the new skill. Test by:

- Discussing a topic that should trigger the skill
- Verifying Claude creates the document via `create_document` MCP tool
- Checking that Claude suggests appropriate relations

## Verification

- Skill appears in available skills after `/reload-plugins`
- Claude activates the skill in relevant conversation contexts
- Created documents pass `archcore validate`
- Claude uses `create_document` MCP tool (not Write/Edit)
- Claude suggests relations after document creation

## Common Issues

### Skill doesn't activate

- The `description` field may be too vague. Add specific trigger phrases.
- Another skill may be competing. Check the "When to Use" contrast sections.

### Claude uses Write instead of create_document

- Check the Example Workflow section — it must show MCP tool calls.
- Ensure the skill doesn't contain language like "create a file" or "write to .archcore/".

### Skill content drifts from CLI templates

- Don't embed template content. Reference the template system: "The template includes sections for Context, Decision, Alternatives Considered, and Consequences."
- When the CLI updates templates, skills remain accurate because they describe sections rather than reproducing them.
