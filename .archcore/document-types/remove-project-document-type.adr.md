---
title: "Remove project Document Type"
status: accepted
tags:
  - "document-types"
---

## Context

The `project` document type was included in the initial set of archcore document types as a way to capture project overviews with architecture, components, and getting-started info.

### Current State

- The `project` type has never been used in any real `.archcore/` directory
- Its purpose overlaps significantly with `doc` (reference material) and repository-level files like `README.md` or `CLAUDE.md`
- The template content (architecture overview, getting started, project structure) is typically better served by existing documentation outside `.archcore/`

### Problem Statement

An unused document type adds cognitive overhead for users choosing between types and increases maintenance surface in the codebase without providing value.

## Decision

Remove the `project` document type entirely from the CLI.

### Rationale

- **Zero adoption** — no known usage in any project
- **Overlapping purpose** — `doc` type covers reference material; `README.md` and `CLAUDE.md` serve project overview needs better
- **Simpler type selection** — fewer types means less disambiguation overhead for users and AI agents
- **Maintenance cost** — removing unused code reduces the surface area for bugs and testing

## Alternatives Considered

### Alternative 1: Keep but deprecate

- Mark as deprecated and remove later
- Rejected: no adoption means no migration path is needed — immediate removal is cleaner

### Alternative 2: Merge into doc

- Redirect `project` to `doc` type
- Rejected: unnecessary complexity since no documents of this type exist

## Consequences

### Positive

- Cleaner type selection for users and AI agents
- Less code to maintain (template, constants, mappings, tests)
- Simpler MCP tool descriptions

### Negative

- If someone later needs a project overview type, they'll use `doc` instead (acceptable trade-off)

### Changes Made

- Removed `TypeProject` constant and `categoryMap` entry from @templates/templates.go
- Removed `generateProjectTemplate()` function
- Removed `project` from `ValidTypes()` list
- Removed `project` from MCP tool descriptions in @internal/mcp/server.go and @internal/mcp/tools/create_document.go
- Removed related test cases from @templates/templates_test.go and @internal/mcp/tools/create_document_test.go
- Updated categories-and-document-types.doc.md
