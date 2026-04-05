---
name: status
description: Shows Archcore documentation dashboard with document counts, relation stats, and potential issues.
disable-model-invocation: true
---

# Archcore Documentation Status

## Verify MCP

Call `mcp__archcore__list_documents` first. If the tool is unavailable, stop and tell the user:
- Install CLI: `curl -fsSL https://archcore.ai/install.sh | bash`
- Initialize: `archcore init`
- Restart the session

## Dashboard

Call `mcp__archcore__list_documents` and `mcp__archcore__list_relations`, then present:

### Documents by Category

| Category   | Count |
| ---------- | ----- |
| Vision     | _n_   |
| Knowledge  | _n_   |
| Experience | _n_   |
| **Total**  | _n_   |

### Documents by Status

| Status   | Count |
| -------- | ----- |
| draft    | _n_   |
| accepted | _n_   |
| rejected | _n_   |

### Documents by Type

List each type with count.

### Relations

| Type       | Count |
| ---------- | ----- |
| related    | _n_   |
| implements | _n_   |
| extends    | _n_   |
| depends_on | _n_   |

### Issues

- Orphaned documents (no relations)
- Draft document count

Keep output compact. Data only, no explanations.
