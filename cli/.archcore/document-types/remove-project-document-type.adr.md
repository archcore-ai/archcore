---
title: "Remove project Document Type"
status: accepted
tags:
  - "document-types"
---

## Context

The `project` document type shipped in the initial set of Archcore document types. It was meant to capture a project overview: architecture, components, and getting-started information.

### State at the time of the decision

- No real `.archcore/` directory used the `project` type.
- Its purpose overlapped with `doc`, which already covers reference material, and with repository-level files such as `README.md` and `CLAUDE.md`.
- Its template content — architecture overview, getting started, project structure — is normally served better by documentation outside `.archcore/`.

### Problem

An unused document type adds a choice for every author and agent selecting a type, and it adds maintenance surface in the codebase without returning value.

## Decision

Remove the `project` document type from the CLI.

### Rationale

- No adoption: no known usage in any project.
- Overlapping purpose: `doc` covers reference material, and `README.md` and `CLAUDE.md` serve the project overview.
- Simpler type selection: fewer types mean less disambiguation for users and AI agents.
- Maintenance cost: removing unused code removes the template, constants, mappings, and tests that went with it.

## Alternatives Considered

### Keep the type but deprecate it

Mark it deprecated and remove it later. Rejected: with no adoption there is no migration path to protect, so immediate removal is simpler.

### Merge it into doc

Redirect `project` to `doc`. Rejected: no document of this type exists, so the redirect would add complexity with nothing to redirect.

## Consequences

### Positive

- Cleaner type selection for users and AI agents.
- Less code to maintain: template, constants, mappings, and tests.
- Shorter MCP tool descriptions.

### Negative

- A later need for a project-overview type is served by `doc` instead. This trade-off was accepted.

### Changes made

- Removed the `TypeProject` constant and its `categoryMap` entry from `@templates/templates.go`.
- Removed `generateProjectTemplate()`.
- Removed `project` from `ValidTypes()`.
- Removed `project` from the MCP tool descriptions in `@internal/mcp/server.go` and `@internal/mcp/tools/create_document.go`.
- Removed the related test cases from `@templates/templates_test.go` and `@internal/mcp/tools/create_document_test.go`.
- Updated the related reference document on categories and document types.
