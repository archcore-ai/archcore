---
title: "Local Documents Take Precedence Over Source Documents, and a Write to a Source Names Its project"
status: accepted
tags:
  - "component:cli"
  - "component:plugin"
  - "globals"
  - "mcp"
  - "skills"
---

## Rule

1. WHEN a local document and a source document cover one topic, the agent MUST apply the local document to the session root's project. Applies to every skill under `plugin/plugins/archcore/skills/` and every agent under `plugin/plugins/archcore/agents/`.
2. The agent MUST read `source_kind` on each row to tell a local document from a source document. Applies to the same paths.
3. The agent MUST NOT infer a document's authority from its path or its title.
4. WHEN a local document and a source document share a slug, the agent MUST present the source document as background. Applies to the same paths.
5. WHEN the agent writes to a source document, the agent MUST pass the row's `source_root` as `project`. Applies to every `create_document`, `update_document`, and `remove_document` call a skill makes.
6. WHEN a write response carries a `source` value other than `"local"`, the agent MUST name that source and its `project` in the task result.
7. WHEN the user asks for a change to a topic a source document covers, the agent MUST ask whether the change is local or goes to the source, unless the request names the target.
8. The MCP read tools MUST return both same-slug documents. Applies to @cli/internal/mcp/tools/list_documents.go and @cli/internal/mcp/tools/search_documents.go.

## Rationale

A source holds the organization default; a project tightens or relaxes it in its own `.archcore/`. Returning both same-slug documents keeps that lineage visible. A write to a source reaches every project that declares it after a commit in the source repository, and nothing in the CLI stops that write (globals/global-sources-are-a-read-policy.adr), so the agent's report and the `project` it passed are the only record of where a change went.

## Examples

### Good

```text
list_documents → both appear:
  source_kind=local   source_id=local                    error-handling.rule.md   (this service's variant)
  source_kind=global  source_id=org  source_root=/w/g    error-handling.rule.md   (organization default)

Agent: "This repo overrides the organization error-handling rule locally; the
organization rule is the default it refines." → applies the local rule.

User: "fix the typo in the org logging rule"
Agent: update_document(project="/w/g", path=".archcore/concepts/logging.rule.md", …)
Result: "Changed logging.rule.md in source org (/w/g); commit it in that repository."
```

### Bad

```text
Agent reads only the source error-handling rule, ignores source_kind, and applies
the organization default although a local rule overrides it.

User: "tighten the logging rule for us"
Agent: update_document(project="/w/g", …)   ← edits the organization default
instead of creating the local override, and reports nothing about source org.
```

## Enforcement

- Rules 1–4 and 8: `@cli/internal/mcp/tools/globals_test.go` (`TestScanDocuments_WithGlobals`) and the fixture `@cli/examples/07-local-overrides-global/` pin that both same-slug documents return with their source fields.
- Rules 5–7: manual review of skill text; `@plugin/test/structure/cli-compat-invariant.bats` pins that a skill branches on `source_kind` only as an optional CLI output.
