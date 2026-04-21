---
name: standard-track
argument-hint: "[topic]"
description: "Advanced — Create ADR, rule, and guide to codify a standard."
disable-model-invocation: true
---

# Standard Track: ADR → rule → guide

Establishes a team standard from decision to enforceable rule to practical how-to. Best for codifying technical decisions into mandatory practices.

## Step 1: Check existing

`mcp__archcore__list_documents(types=["adr", "rule", "guide"])` — see what exists. If `$ARGUMENTS` provided, check for duplicates on this topic.

## Step 2: Determine scope

If related documents already exist (e.g., an ADR without a rule), pick up where the chain left off — don't recreate.

## Step 3: ADR

Use the `AskUserQuestion` tool to ask: "What decision was made? Why this approach over alternatives?"

Compose content covering Context, Decision, Alternatives Considered, Consequences. Create via `mcp__archcore__create_document(type="adr")`.

## Step 4: Rule

Use the `AskUserQuestion` tool to ask: "What are the mandatory behaviors? How should this be enforced?"

Compose content covering Rule (imperative statements), Rationale, Examples (Good/Bad), Enforcement. Create via `mcp__archcore__create_document(type="rule")`.

Add relation: `mcp__archcore__add_relation` — rule `implements` adr.

## Step 5: Guide

Use the `AskUserQuestion` tool to ask: "What steps should developers follow? What are common pitfalls?"

Compose content covering Prerequisites, Steps (numbered), Verification, Common Issues. Create via `mcp__archcore__create_document(type="guide")`.

Add relation: `mcp__archcore__add_relation` — guide `related` rule.

## Step 6: Relate to existing

Check for specs, plans, or other documents that should be linked. Suggest additional `add_relation` calls.

## Result

Three linked documents: ADR → rule → guide (rule `implements` adr, guide `related` rule).
