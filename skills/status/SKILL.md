---
name: status
description: Show Archcore documentation dashboard — document counts, relation stats, and potential issues.
disable-model-invocation: true
---

# /archcore:status

Quick dashboard of your Archcore knowledge base. Compact numbers, no analysis.

## When to use

- "Show status"
- "How many docs do we have?"
- "Dashboard"

**Not status:**
- Detailed health review with recommendations → `/archcore:review`

## Routing table

No routing needed. Single behavior: gather data, present dashboard.

## Execution

### Step 0: Verify MCP

Call `mcp__archcore__list_documents` first. If unavailable, stop and tell the user:
- Install CLI: `curl -fsSL https://archcore.ai/install.sh | bash`
- Initialize: `archcore init`
- Restart the session

### Step 1: Gather

Call `mcp__archcore__list_documents` and `mcp__archcore__list_relations`.

### Step 2: Present

**Documents by Category**

| Category | Count |
|---|---|
| Vision | _n_ |
| Knowledge | _n_ |
| Experience | _n_ |
| **Total** | _n_ |

**Documents by Status**

| Status | Count |
|---|---|
| draft | _n_ |
| accepted | _n_ |
| rejected | _n_ |

**Documents by Type** — list each type with count, skip types with 0.

**Relations**

| Type | Count |
|---|---|
| related | _n_ |
| implements | _n_ |
| extends | _n_ |
| depends_on | _n_ |

**Issues** — orphaned documents (no relations), high draft count. One line each, no explanations.

## Result

Compact dashboard. Data only, no analysis. For deeper review use `/archcore:review`.
