---
name: product-track
argument-hint: "[topic]"
description: "Advanced — Lightweight product requirements flow: idea → PRD → plan. Best for individual features, small teams, or rapid prototyping. For engineer-led feature delivery use /archcore:feature-track; for ISO requirements cascade use /archcore:iso-track."
disable-model-invocation: true
---

# Product Track: idea → PRD → plan

Lightweight requirements flow. Best for individual features, small teams, rapid prototyping.

## Step 0: Verify MCP

Check if `mcp__archcore__list_documents` exists in your available tools. If the tool does not exist or returns an error, **stop immediately** and tell the user:

**Archcore CLI is not installed.** The plugin provides skills and hooks, but document operations need the CLI (it runs the MCP server).

To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

Do not proceed without MCP tools. Do not write to `.archcore/` directly.

## Step 1: Check existing

`mcp__archcore__list_documents(types=["idea", "prd", "plan"])` — see what exists. If `$ARGUMENTS` provided, check for duplicates on this topic.

## Step 2: Determine scope

If related documents already exist (e.g., an idea without a PRD), pick up where the chain left off — don't recreate.

## Step 3: Idea

Use the `AskUserQuestion` tool to ask: "What's the core concept? Who would benefit?"

Compose content covering Idea, Value, Possible Implementation, Risks and Constraints. Create via `mcp__archcore__create_document(type="idea")`.

## Step 4: PRD

Use the `AskUserQuestion` tool to ask: "What problem does this solve? What are the success metrics?"

Compose content covering Vision, Problem Statement, Goals and Success Metrics, Requirements. Create via `mcp__archcore__create_document(type="prd")`.

Add relation: `mcp__archcore__add_relation` — prd `implements` idea.

## Step 5: Plan

Use the `AskUserQuestion` tool to ask: "What are the key phases? What are the dependencies?"

Compose content covering Goal, Tasks (phased), Acceptance Criteria, Dependencies. Create via `mcp__archcore__create_document(type="plan")`.

Add relation: `mcp__archcore__add_relation` — plan `implements` prd.

## Step 6: Relate to existing

Check for ADRs, specs, or other documents that should be linked. Suggest additional `add_relation` calls.

## Result

Three linked documents: idea → prd → plan (each `implements` previous).
