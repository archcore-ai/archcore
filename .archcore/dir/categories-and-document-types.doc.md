---
title: "Categories and Document Types"
status: accepted
tags:
  - "component:cli"
  - "directory-structure"
---

## Overview

This file is the CLI repository's entry point to the Archcore document-type model. The full reference lives in the `archcore` global source, and this document does not restate it.

- `concepts/document-types-reference` — per-type tables, the type-selection matrix, and how to choose a requirements track
- `concepts/requirements-layers` — the Sources versus Specifications two-layer model and the relation conventions
- `concepts/core-concepts` and `concepts/document-tracks` — the high-level model and the document flows

## How the CLI applies the model

- The MCP server type-selection instructions carry the disambiguation rules that decide which type a new document gets.
- `@cli/templates/templates.go` registers every type with its template and its category mapping.
- The category is derived from the `.type` suffix in the filename, never from the directory.


## Research vocabulary in the CLI

The current implementation registers 23 document types, comprising 13 vision, eight knowledge, and two experience types — @cli/templates/templates.go. Release v0.8.3 shipped 21; the two actor-subject types below are in the working tree and unreleased. The registry implements the accepted category revision. Release publication and plugin routing remain separate handoff steps.

| Type | Category | Stored subject |
|---|---|---|
| `research` | `vision` | Territory investigation closed by coverage |
| `evidence` | `knowledge` | One external material with its locator and extract |
| `rnd` | `vision` | Decision-bound investigation closed by a recommendation |

The required sections and ISO profile for the two new types live in @cli/templates/precision.go. The Archcore MCP server explains their status and source-tag conventions in @cli/internal/mcp/server.go.

## Actor-subject vocabulary in the CLI

The pair follows `concepts/scenario-and-journey-types` in the `archcore` global source, still `draft` there. Both take the actor as the subject of every step and carry no modal; the rules stay in the `spec`.

| Type | Category | Stored subject |
|---|---|---|
| `scenario` | `knowledge` | Realized actor flows and Given/When/Then examples illustrating one linked `spec` |
| `journey` | `vision` | Intended path of one user type before a `spec` covering that interaction exists |

The routing test between the pair is whether a `spec` covering the interaction exists, not whether the project holds any `spec`. The precision canon, the body cap table, and the injection rank live in @cli/templates/precision.go and @cli/internal/advisory/code_alignment.go; the two local specs under `document-types/` state the contracts.
