---
title: "Research and Evidence Document Types"
status: draft
tags:
  - "document-types"
  - "mcp"
---

## Purpose & Scope

This contract defines the `research` and `evidence` document types for MCP clients and document readers. The vocabulary follows `concepts/research-and-evidence-types` [global · archcore · read-only]. Execution of research methods is outside scope.

## Surface

The registry and template dispatcher live in @templates/templates.go; precision data lives in @templates/precision.go. MCP creation and discovery use @internal/mcp/tools/create_document.go, @internal/mcp/tools/list_documents.go, and @internal/mcp/tools/search_documents.go.

| Type | Category | Generated sections, in order |
|---|---|---|
| `research` | `vision` | Goal, Scope, Coverage, Sources, Findings, Synthesis, Open Gaps |
| `evidence` | `knowledge` | Locator, Extract, Notes |

The generated Locator starts with `Address:`, `Access date:`, `Publication date:`, and `Publisher:`, in that order. Unknown publication dates and publishers use visible placeholders.

## Normative Behavior

1. The template registry MUST recognize both types in the Surface table.
2. The template registry MUST assign each type its category from the Surface table.
3. WHEN generating either type, the template generator MUST emit the corresponding section sequence.
4. WHEN generating `research`, the template generator MUST include a table placeholder in Coverage.
5. WHEN generating `research`, the template generator MUST include a table placeholder in Sources.
6. WHEN generating `evidence`, the template generator MUST emit the four Locator lines.
7. The precision checker MUST require every section assigned to each type.
8. The precision checker MUST apply the ISO profile to both types.
9. The Archcore MCP server MUST expose both types through creation and type-filtered discovery.
10. The server instructions MUST distinguish coverage-based `research` from recommendation-based `rnd`.
11. The server instructions MUST describe `evidence` as one material.
12. The server instructions MUST describe source classes through the five source tags below.
13. The search ranker MUST use its default type priority for both types.
14. The CodeAlignment selector MUST exclude both types.

## Constraints & Invariants

The source tags are `source:primary`, `source:secondary`, `source:measurement`, `source:interview`, and `source:dataset`. They remain conventions, not a closed tag enum.

1. The Archcore MCP server MUST retain the existing `rnd` category and status behavior.
2. The template generator MUST NOT prescribe a tool, search strategy, or agent role.
3. The Archcore MCP server MUST NOT create relations implicitly.
4. The Archcore MCP server MUST retain the existing three status values.
5. The Archcore MCP server MUST NOT add a provenance field to its frontmatter schema.

The accepted-status meanings are authoring conventions: research synthesis is current at revision time; a second reader confirms evidence existence and extract. The engine does not verify either convention.

## Failure Behavior

1. WHEN a required section is absent, the precision checker MUST emit an advisory naming that section.
2. WHEN a required section is absent, the Archcore MCP server MUST allow the document write.
3. WHEN an unknown type is requested, the Archcore MCP server MUST return its existing invalid-type error shape.
4. WHEN a write targets a global source, the Archcore MCP server MUST reject the write.

## Conformance

Conformance requires every numbered obligation above. Regression coverage includes @templates/templates_test.go, @internal/advisory/precision_findings_test.go, @internal/mcp/tools/search_documents_test.go, and @internal/advisory/code_alignment_test.go. Registered-schema and lifecycle scenarios in @internal/mcp/integration/research_spec_test.go cover creation, listing, retrieval, search, and rejected-document visibility.
