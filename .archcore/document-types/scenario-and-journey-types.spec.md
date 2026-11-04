---
title: "Scenario and Journey Document Types"
status: accepted
tags:
  - "component:cli"
  - "document-types"
  - "mcp"
---

## Purpose & Scope

This contract defines the `scenario` and `journey` document types for MCP clients, the plugin runtime, and document readers. The vocabulary follows `concepts/scenario-and-journey-types` [global · archcore · read-only]. The precision checks and the advisory engines' treatment of both types are a separate contract. Execution of a scenario is outside scope.

## Surface

`journey` is a `vision` type; `scenario` is a `knowledge` type. The registry and generated sections live in @cli/templates/templates.go and are pinned by tests; the source-extension list lives in @cli/templates/source_extensions.go.

## Normative Behavior

1. The Archcore MCP server MUST expose both types through creation and type-filtered discovery.
2. The server instructions MUST state the routing test between the pair: a covering `spec` exists, `scenario`; none exists, `journey`.
3. The server instructions MUST state the boundary against `spec` as the subject of the line: component with a modal, `spec`; actor without one, `scenario`.
4. The server instructions MUST name the relation conventions `scenario depends_on spec`, `scenario implements journey`, and `journey related prd`.
5. The `create_document` `status` parameter description MUST describe the status meanings in Constraints & Invariants below.
6. The source-extension list MUST include `.feature`, so a bare `features/login.feature` mention passes the bare-mention filter of `search_documents`.
7. The template generator MUST NOT emit a BCP 14 modal in a step placeholder of either type.

## Constraints & Invariants

The status meanings are authoring conventions: an accepted `journey` means the team agreed on the wanted interaction; an accepted `scenario` means a reader confirmed the examples against the running system; a rejected `scenario` means the examples no longer hold. The engine verifies none of them.

The tags `actor:<type>`, `component:<name>`, and `nfr:<concern>` are conventions, not a closed tag enum.

1. The relation vocabulary MUST stay at its seven values.
2. The Archcore MCP server MUST NOT create relations implicitly.
3. The Archcore MCP server MUST retain the existing three status values.
4. The template generator MUST NOT prescribe a test runner, a discovery technique, or a feature-file layout.

## Failure Behavior

1. WHEN an unknown type is requested, the Archcore MCP server MUST return its existing invalid-type error shape.
2. WHEN a write targets a global source, the Archcore MCP server MUST reject the write.
3. IF a binary that predates these types scans a `.scenario.md` or `.journey.md` file, THEN that binary MUST skip the file in the scan and MUST report an invalid type in `status` (@cli/internal/docs/scan.go, @cli/cmd/status.go). No migration changes this.

## Conformance

Conformance requires every numbered obligation above. Lifecycle scenarios in @cli/internal/mcp/integration/ cover creation, listing, retrieval, and search for both types.
