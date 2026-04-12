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
- Understanding of the Skill File Structure Standard (rule) — especially Type skill (Layer 3) requirements
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
name: <type-name>
argument-hint: "[topic]"
description: >
  <Describe specific triggers and context signals. Be precise —
  this determines when Claude activates the skill automatically.
  Include key phrases users might say and situations where this
  type is appropriate.>
---
```

Frontmatter rules for Type skills (Layer 3):

- `name` MUST be the Archcore document type identifier (e.g., `adr`, `prd`, `rule`)
- Non-high-frequency type skills MUST prefix their `description` with "Expert —"
- Type skills do NOT include `disable-model-invocation: true` (that is for Intent and Track skills only)

### 5. Write the 3 required sections

Type skills (Layer 3) follow a compact 3-section structure. Follow this order exactly:

1. **When to Use**: Specific scenarios and conversation cues. Include contrast with similar types (e.g., ADR vs RFC, PRD vs MRD, guide vs doc). Use bullet lists for clarity.

2. **Quick Create**: A concrete example showing the `create_document` MCP call with appropriate parameters, followed by an `add_relation` call connecting to an existing document. Brief explanation of why these parameters and relations were chosen.

3. **Relations**: Which relation types and target document types are commonly used. Include both incoming and outgoing relations. Show typical flows this type participates in.

Do NOT include Overview, Required Sections, Best Practices, Common Mistakes, or Example Workflow as separate sections — those belong to the old flat structure and would exceed the line limit.

### 6. Validate against the standard

Check your SKILL.md against the Skill File Structure Standard:

- [ ] Frontmatter has `name` (Archcore type identifier) and `description`
- [ ] Non-high-frequency types have "Expert —" prefix in description
- [ ] All 3 sections present in correct order: When to Use, Quick Create, Relations
- [ ] Quick Create uses `create_document`, not Write/Edit
- [ ] No embedded template content (reference template system instead)
- [ ] Under 100 lines total

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

- Check the Quick Create section — it must show MCP tool calls.
- Ensure the skill doesn't contain language like "create a file" or "write to .archcore/".

### Skill content drifts from CLI templates

- Don't embed template content. Reference the template system: "The template includes sections for Context, Decision, Alternatives Considered, and Consequences."
- When the CLI updates templates, skills remain accurate because they describe sections rather than reproducing them.

### Skill exceeds the 100-line limit

- Type skills must be compact. Move detailed guidance to the relevant guide or spec documents instead of embedding it in the skill.
- Focus Quick Create on a single realistic example, not multiple scenarios.
