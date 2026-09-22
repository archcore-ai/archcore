---
title: "Scenario and Journey Document Types"
status: draft
tags:
  - "component:cli"
  - "document-types"
  - "mcp"
---

## Purpose & Scope

This contract defines the `scenario` and `journey` document types for MCP clients, the plugin runtime, and document readers. The vocabulary follows `concepts/scenario-and-journey-types` [global · archcore · read-only]. The precision checks and the advisory engines' treatment of both types are a separate contract. Execution of a scenario is outside scope.

## Surface

The registry and template dispatcher live in @cli/templates/templates.go; the source-extension list lives in @cli/templates/source_extensions.go. MCP creation and discovery use @cli/internal/mcp/tools/create_document.go, @cli/internal/mcp/tools/list_documents.go, and @cli/internal/mcp/tools/search_documents.go; the server instructions live in @cli/internal/mcp/server.go.

| Type | Category | Generated sections, in order |
|---|---|---|
| `journey` | `vision` | Intent, Actors, Journeys, Open Questions |
| `scenario` | `knowledge` | Subject, Actors, Flows, Examples, Open Questions |

Generated shapes: Intent opens with the header `In order to [goal] / As a [actor] / I want [outcome]`; Actors is a table with the columns `Actor`, `Who they are`, `What they want`; Journeys and Flows hold one `###` subsection per actor with numbered steps and an `Extensions` list; each Flows subsection opens with an `Anchors:` line of `@path` references; Examples holds a `Background` block and one titled example with an `Illustrates:` line and unfenced Given/When/Then lines.

## Normative Behavior

1. The template registry MUST recognize both types in the Surface table.
2. The template registry MUST assign each type its category from the Surface table.
3. WHEN generating either type, the template generator MUST emit the section sequence from the Surface table.
4. WHEN generating either type, the template generator MUST emit the Actors table with the three named columns.
5. WHEN generating `journey`, the template generator MUST emit the Intent header in the named form.
6. WHEN generating `scenario`, the template generator MUST emit an `Anchors:` placeholder line in every Flows subsection.
7. WHEN generating `scenario`, the template generator MUST emit one example with a title, an `Illustrates:` line, and unfenced Given/When/Then lines.
8. The template generator MUST NOT emit a BCP 14 modal in a step placeholder of either type.
9. The Archcore MCP server MUST expose both types through creation and type-filtered discovery.
10. The server instructions MUST state the routing test between the pair: a covering `spec` exists, `scenario`; none exists, `journey`.
11. The server instructions MUST state the boundary against `spec` as the subject of the line: component with a modal, `spec`; actor without one, `scenario`.
12. The server instructions MUST name the relation conventions `scenario depends_on spec`, `scenario implements journey`, and `journey related prd`.
13. The server instructions MUST describe the status meanings in Constraints & Invariants below.
14. The source-extension list MUST include `.feature`, so a bare `features/login.feature` mention passes the bare-mention filter of `search_documents`.

## Constraints & Invariants

The status meanings are authoring conventions: an accepted `journey` means the team agreed on the wanted interaction; an accepted `scenario` means a reader confirmed the examples against the running system; a rejected `scenario` means the examples no longer hold. The engine verifies none of them.

The tags `actor:<type>`, `component:<name>`, and `nfr:<concern>` are conventions, not a closed tag enum.

1. The registry MUST count 23 types after the change: 13 vision, 8 knowledge, 2 experience.
2. The relation vocabulary MUST stay at its seven values.
3. The Archcore MCP server MUST NOT create relations implicitly.
4. The Archcore MCP server MUST retain the existing three status values.
5. The template generator MUST NOT prescribe a test runner, a discovery technique, or a feature-file layout.

## Failure Behavior

1. WHEN an unknown type is requested, the Archcore MCP server MUST return its existing invalid-type error shape.
2. WHEN a write targets a global source, the Archcore MCP server MUST reject the write.
3. IF a binary that predates this release scans a `.scenario.md` or `.journey.md` file, THEN that binary skips the file in the scan and reports an invalid type in `status`; this is the current behavior of @cli/internal/docs/scan.go and @cli/cmd/status.go, and no migration changes it.

## Conformance

Conformance requires every numbered obligation above. Regression coverage includes @cli/templates/templates_test.go, @cli/templates/source_extensions_test.go, @cli/internal/mcp/tools/create_document_test.go, and @cli/internal/mcp/tools/list_documents_test.go. Lifecycle scenarios in @cli/internal/mcp/integration/ cover creation, listing, retrieval, and search for both types.
