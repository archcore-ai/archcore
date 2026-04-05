---
name: review
argument-hint: "[category or tag]"
description: Reviews Archcore documentation for gaps, staleness, orphaned documents, and missing relations.
disable-model-invocation: true
---

# Review Archcore Documentation

## Step 0: Verify MCP

Call `mcp__archcore__list_documents` first. If the tool is unavailable, stop and tell the user:
- Install CLI: `curl -fsSL https://archcore.ai/install.sh | bash`
- Initialize: `archcore init`
- Restart the session

## Step 1: Gather data

Call `mcp__archcore__list_documents` and `mcp__archcore__list_relations` to get the full inventory.

## Step 2: Analyze

If `$ARGUMENTS` is provided, filter to that category, type, or tag.

Check for:

**Coverage gaps:** ADRs without rules/guides, PRDs without plans, rules without guides, empty categories.

**Staleness:** Documents stuck in `draft` that may need `accepted` or `rejected`.

**Relation health:** Orphaned documents (no relations), plans without `implements` to a PRD, specs without `implements` to requirements.

**Tag hygiene:** Tags used only once, inconsistent naming.

## Step 3: Report

Present a concise summary:

1. **Overview** — totals by category and status
2. **Gaps** — missing documents or relations with recommendations
3. **Staleness** — documents needing attention
4. **Orphans** — documents with no relations
5. **Actions** — prioritized list of fixes
