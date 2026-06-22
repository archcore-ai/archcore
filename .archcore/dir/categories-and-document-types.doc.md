---
title: "Categories and Document Types"
status: accepted
tags:
  - "directory-structure"
---

## Overview

The full document-type reference — the per-type purpose tables, the "choosing the right type" matrix, and the requirements-track guidance — now lives in the `archcore` global source:

- `concepts/document-types-reference` — per-type tables + type-selection matrix + choosing a requirements track
- `concepts/requirements-layers` — the Sources-vs-Specifications two-layer model and relation conventions
- `concepts/core-concepts` / `concepts/document-tracks` — the high-level model and document flows

This file is retained as the CLI's local entry point to that model. The CLI enforces it through the MCP server type-selection instructions and the templates in `@templates/templates.go`. The category is derived from the `.type` suffix in the filename, not from the directory.
