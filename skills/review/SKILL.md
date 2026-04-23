---
name: review
argument-hint: "[category or tag]"
description: "Audit documentation health — finds coverage gaps, stale statuses, orphaned documents, and missing relations. Use when you want a full health report with recommendations. For quick counts use /archcore:status; for code-drift detection use /archcore:actualize."
---

# /archcore:review

Review Archcore documentation health. Finds gaps, stale documents, orphaned relations, and produces actionable recommendations.

## When to use

- "Review the docs"
- "Check documentation health"
- "Are there any documentation gaps?"
- "Audit the knowledge base"

**Not review:**
- Quick counts and stats → `/archcore:status`
- Creating new documentation → `/archcore:capture`, `/archcore:plan`, `/archcore:decide`
- Reading applicable rules/ADRs/specs before coding → `/archcore:context`
- Picking up where work left off → `/archcore:context`

## Routing table

| Signal | Route | Scope |
|---|---|---|
| No arguments | → full review | All documents |
| Category name ("knowledge", "vision") | → category review | Filter to category |
| Tag name ("auth", "payments") | → tag review | Filter to tag |
| Specific type ("adr", "spec") | → type review | Filter to type |

Default: full review of all documents.

## Execution

### Step 1: Gather data

Call `mcp__archcore__list_documents` and `mcp__archcore__list_relations` to get the full inventory. If `$ARGUMENTS` provided, apply as filter.

### Step 2: Analyze

Check for:

**Coverage gaps:**
- ADRs without rules/guides (decisions not codified)
- PRDs without plans (requirements without implementation path)
- Rules without guides (standards without instructions)
- Empty categories or types with zero documents

**Staleness:**
- Documents stuck in `draft` that may need `accepted` or `rejected`
- Documents with stale content indicators

**Relation health:**
- Orphaned documents (no incoming or outgoing relations)
- Plans without `implements` to a PRD
- Specs without `implements` to requirements
- Broken chains (ISO cascade with gaps)

**Tag hygiene:**
- Tags used only once (potential inconsistency)
- Related documents with different tags

### Step 3: Report

Present a concise summary:

1. **Overview** — totals by category and status
2. **Gaps** — missing documents or relations with specific recommendations
3. **Staleness** — documents needing attention
4. **Orphans** — documents with no relations
5. **Actions** — prioritized list of fixes, most impactful first

## Result

Actionable review report with prioritized fixes. No verbose analysis — findings and recommendations only.
