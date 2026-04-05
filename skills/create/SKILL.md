---
name: create
argument-hint: "[topic]"
description: Interactive Archcore document creation wizard with type selection guidance.
disable-model-invocation: true
---

# Create Archcore Document

## Step 0: Verify MCP

Call `mcp__archcore__list_documents` first. If the tool is unavailable, stop and tell the user:

- Install CLI: `curl -fsSL https://archcore.ai/install.sh | bash`
- Initialize: `archcore init`
- Restart the session

## Step 1: Check existing

`mcp__archcore__list_documents` — see what exists, prevent duplicates.

## Step 2: Determine type

If `$ARGUMENTS` provided, infer type and topic from context.

Otherwise, use the `AskUserQuestion` tool to determine the document type. Present the most fitting options based on key contrasts:

- Decision made → adr. Still discussing → rfc.
- Product scope → prd. Concept exploration → idea.
- Mandatory standard → rule. How-to → guide.
- Reference lookup → doc. Normative contract → spec.
- Recurring task → task-type. Code pattern shift → cpat.

## Step 3: Gather and compose

Use the `AskUserQuestion` tool to ask 2-3 focused questions for the chosen type. Compose rich content covering all required sections using the user's answers for depth.

## Step 4: Create

Pass the composed content as `content` parameter to `mcp__archcore__create_document`.

## Step 5: Relate

Suggest `mcp__archcore__add_relation` calls based on existing documents.
