---
title: "Categories and Document Types"
status: accepted
tags:
  - "directory-structure"
---

## Overview

This file is the CLI repository's entry point to the Archcore document-type model. The full reference lives in the `archcore` global source, and this document does not restate it.

- `concepts/document-types-reference` — per-type tables, the type-selection matrix, and how to choose a requirements track
- `concepts/requirements-layers` — the Sources versus Specifications two-layer model and the relation conventions
- `concepts/core-concepts` and `concepts/document-tracks` — the high-level model and the document flows

## How the CLI applies the model

- The MCP server type-selection instructions carry the disambiguation rules that decide which type a new document gets.
- `@templates/templates.go` registers every type with its template and its category mapping.
- The category is derived from the `.type` suffix in the filename, never from the directory.


## Research vocabulary in the CLI

The current implementation registers 21 document types, comprising 12 vision, seven knowledge, and two experience types — @templates/templates.go. The registry implements the accepted category revision. Release publication and plugin routing remain separate handoff steps.

| Type | Category | Stored subject |
|---|---|---|
| `research` | `vision` | Territory investigation closed by coverage |
| `evidence` | `knowledge` | One external material with its locator and extract |
| `rnd` | `vision` | Decision-bound investigation closed by a recommendation |

The required sections and ISO profile for the two new types live in @templates/precision.go. The Archcore MCP server explains their status and source-tag conventions in @internal/mcp/server.go.
