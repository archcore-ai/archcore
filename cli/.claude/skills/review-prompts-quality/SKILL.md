---
name: review-prompts-quality
description: Review quality of changed or staged prompts in the archcore CLI project. Use when the user runs `/review-prompts-quality`. Inspects git diff (staged + unstaged) for changes in MCP tool descriptions (internal/mcp/tools/*.go), system instructions (internal/mcp/server.go), and document templates (templates/templates.go), then delegates quality analysis to the prompt-engineer agent across 4 axes: effectiveness, best practices, token cost, and reliability.
---

# Review Prompts Quality

## Prompt Locations in This Project

| File pattern | What it contains |
|---|---|
| `internal/mcp/tools/*.go` | MCP tool descriptions (`mcp.WithDescription(...)`) and parameter descriptions |
| `internal/mcp/server.go` | `mcpServerInstructions` — system context served to all AI agents |
| `templates/templates.go` | Document templates with section guidance for human authors |

## Workflow

### Step 1: Find changed prompt files

Run both commands to capture staged and unstaged changes:
```bash
git diff --cached --name-only
git diff --name-only
```

Filter the results to files matching these patterns:
- `internal/mcp/tools/*.go`
- `internal/mcp/server.go`
- `templates/templates.go`

If no matching files are found, report: "No prompt-related files changed." and stop.

### Step 2: Extract changed prompt content

For each matched file, read the current content of that file. Focus on:
- **`internal/mcp/tools/*.go`**: All `mcp.WithDescription(...)` call arguments and parameter description strings
- **`internal/mcp/server.go`**: The `mcpServerInstructions` variable and `buildInstructions()` function
- **`templates/templates.go`**: The `generate*Template()` functions

Also run `git diff HEAD -- <file>` (or `git diff --cached -- <file>`) to see exactly what changed.

### Step 3: Delegate to prompt-engineer agent

Launch the `ai-kit:prompt-engineer` agent with all extracted prompt content and diffs. Ask it to review each changed prompt on these four axes:

1. **Effectiveness** — Does the prompt clearly communicate intent? Are instructions unambiguous and actionable? Would an AI agent follow them correctly without guessing?

2. **Best practices** — For MCP tool descriptions: does it specify when to call the tool, what inputs are expected, and what the output looks like? For system instructions: is there a logical hierarchy, no contradictions, clear scope? For templates: do sections guide the author toward complete, useful content?

3. **Token cost** — Is the prompt unnecessarily verbose? Identify: repeated information, overly long examples, content that could be condensed without loss. Estimate rough token impact for significant issues.

4. **Reliability** — Are edge cases handled? Is error guidance present where needed? Are instructions consistent with other tools/context in the same file?

### Step 4: Report findings

Present a structured report grouped by file, then by dimension. Use severity labels:
- `[critical]` — Would cause incorrect AI behavior
- `[warning]` — Suboptimal but not breaking
- `[suggestion]` — Improvement opportunity

End with a summary table: file → dimensions affected → severity.
