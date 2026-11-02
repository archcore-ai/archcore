---
title: "Document Update Frontmatter Retention"
status: accepted
tags:
  - "component:cli"
  - "document-types"
  - "mcp"
---

## Purpose & Scope

This contract covers frontmatter retention during `update_document` for authors who store custom metadata. The handler (@cli/internal/mcp/tools/update_document.go) owns `title`, `status`, and `tags`. Every other top-level YAML key belongs to the author and is retained as an opaque value. The parser is `SplitDocument` in @cli/templates/templates.go.

## Surface

The serializer in @cli/internal/mcp/tools/common.go is shared with `create_document`. The parsed `Frontmatter` also feeds the sync payload (@cli/internal/sync/payload.go); retention adds no structured sync field.

## Normative Behavior

1. WHEN an update succeeds, the handler MUST retain every unowned top-level frontmatter key and preserve each value's YAML meaning.
2. WHEN serializing retained keys, the serializer MUST preserve their relative order and place them after the owned fields.
3. WHEN an owned field is omitted, the handler MUST preserve its existing value.
4. WHEN `tags` is an empty array, the handler MUST clear the existing tags.
5. WHEN `tags` is omitted, the handler MUST preserve the existing tag order.
6. WHEN `content` or `edits` is provided, the handler MUST retain metadata from the existing file.
7. WHEN writing succeeds, the handler MUST invalidate the document cache.

## Constraints & Invariants

The supported update arguments are `title`, `status`, `tags`, and either `content` or `edits`; @.archcore/mcp/update-document-edits.adr.md governs `edits`. Unknown-key retention adds no editing API. Existing guards in @cli/internal/docs/guard.go continue to govern writes.

1. The handler MUST NOT interpret retained keys as Archcore configuration.
2. The serializer MUST NOT add retained-key storage to the structured JSON frontmatter payload of the sync request.
3. The handler MUST keep atomic replacement through the existing write helper.

Retention promises YAML value semantics and relative extra-key order, not byte-identical formatting, comments, or positions among owned fields. An alias that depends on a reconstructed owned field makes the handler refuse the update before writing.

## Failure Behavior

1. WHEN existing frontmatter is invalid YAML, the handler MUST reject the update and direct the user to repair the file manually.
2. WHEN frontmatter cannot be preserved, the handler MUST leave the file unchanged.
3. WHEN an update supplies no accepted field, the handler MUST return an error that names every accepted field.
4. WHEN the requested status is invalid, the handler MUST reject the update.
5. WHEN an operation fails, the handler MUST omit absolute paths from its error response.

## Conformance

Conformance requires every numbered obligation above. The regression tests sit beside the parser, the serializer, and the handler.
