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

Check if `mcp__archcore__list_documents` exists in your available tools. If the tool does not exist or returns an error, **stop immediately** and tell the user:

**Archcore CLI is not installed.** The plugin provides skills and hooks, but document operations need the CLI (it runs the MCP server).

To set up:
1. Install: `curl -fsSL https://archcore.ai/install.sh | bash`
2. Initialize project: `archcore init`
3. Restart the session, then rerun this command.

Do not proceed without MCP tools. Do not write to `.archcore/` directly.

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
