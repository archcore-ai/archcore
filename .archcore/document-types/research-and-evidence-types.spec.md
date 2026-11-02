---
title: "Research and Evidence Document Types"
status: accepted
tags:
  - "component:cli"
  - "document-types"
  - "mcp"
---

## Purpose & Scope

This contract defines the `research` and `evidence` document types for MCP clients and document readers. The vocabulary follows `concepts/research-and-evidence-types` [global · archcore · read-only]. Execution of research methods is outside scope. The registry, templates, and precision data live in @cli/templates/templates.go and @cli/templates/precision.go; the generated sections are pinned by tests there.

## Surface

`research` is a `vision` type with sections Goal, Scope, Coverage, Sources, Findings, Synthesis, Open Gaps. `evidence` is a `knowledge` type with sections Locator, Extract, Notes. The generated Locator starts with `Address:`, `Access date:`, `Publication date:`, and `Publisher:`, in that order. Unknown publication dates and publishers use visible placeholders.

## Normative Behavior

1. The Archcore MCP server MUST expose both types through creation and type-filtered discovery.
2. The server instructions MUST distinguish coverage-based `research` from recommendation-based `rnd`.
3. The server instructions MUST describe `evidence` as one material.
4. The `create_document` `tags` parameter description MUST describe source classes through the five source tags below.
5. The CodeAlignment selector MUST exclude both types.

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

Conformance requires every numbered obligation above. Lifecycle scenarios in @cli/internal/mcp/integration/research_spec_test.go cover creation, listing, retrieval, search, and rejected-document visibility.
