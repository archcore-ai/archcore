---
title: "Document Update Frontmatter Retention"
status: draft
tags:
  - "document-types"
  - "mcp"
---

## Purpose & Scope

This contract covers frontmatter retention during `update_document` for authors who store custom metadata. Existing field-update and refusal behavior comes from @internal/mcp/tools/update_document.go. The implementation retains unknown keys as required by the accepted implementation plan.

## Surface

The parser is `SplitDocument` in @templates/templates.go. The update handler and serializer live in @internal/mcp/tools/update_document.go and @internal/mcp/tools/common.go. The create handler shares the serializer through @internal/mcp/tools/create_document.go.

| Field set | Owner | Handling |
|---|---|---|
| `title`, `status`, `tags` | Archcore MCP server | Existing field-specific update semantics |
| Other top-level YAML keys | Document author | Opaque retained values |
| Markdown body | Document author | Existing content replacement semantics |

The parser's `Frontmatter` also participates in the sync payload — @templates/templates.go, @internal/sync/payload.go. Retention is not a new structured sync field.

## Normative Behavior

1. WHEN an update succeeds, the handler MUST retain every unowned top-level frontmatter key.
2. WHEN an update succeeds, the handler MUST preserve each retained value's YAML meaning.
3. WHEN serializing retained keys, the serializer MUST preserve their relative order.
4. The serializer MUST place retained keys after the owned fields.
5. WHEN an owned field is omitted, the handler MUST preserve its existing value.
6. WHEN `tags` is an empty array, the handler MUST clear the existing tags.
7. WHEN `tags` is omitted, the handler MUST preserve the existing tag order.
8. WHEN `content` is provided, the handler MUST retain metadata from the existing file.
9. The serializer MUST emit each owned field at most once.
10. WHEN writing succeeds, the handler MUST invalidate the document cache.

## Constraints & Invariants

The supported update arguments remain `title`, `status`, `tags`, and `content`. Unknown-key retention adds no editing API. Existing guards in @internal/docs/guard.go continue to govern writes.

1. The handler MUST NOT interpret retained keys as Archcore configuration.
2. The serializer MUST NOT add retained-key storage to the structured JSON frontmatter payload.
3. The handler MUST keep atomic replacement through the existing write helper.
4. The handler MUST reject writes to global source documents.

Retention promises YAML value semantics and relative extra-key order, not byte-identical formatting, comments, or positions among owned fields. Serialization needs to preserve anchor dependencies or refuse the update before writing.

## Failure Behavior

1. WHEN existing frontmatter is invalid YAML, the handler MUST reject the update.
2. WHEN frontmatter cannot be preserved, the handler MUST leave the file unchanged.
3. WHEN rejecting malformed frontmatter, the handler MUST direct the user to repair the file manually.
4. WHEN an update supplies no accepted field, the handler MUST return the existing missing-update error.
5. WHEN the requested status is invalid, the handler MUST reject the update.
6. WHEN an operation fails, the handler MUST omit absolute paths from its error response.

## Conformance

Conformance requires every numbered obligation above. The parser retains unowned YAML nodes; the serializer preserves their values and relative order. Alias dependencies on reconstructed owned fields cause refusal before writing.

Regression coverage includes @templates/templates_test.go, @internal/mcp/tools/common_test.go, @internal/mcp/tools/update_document_test.go, and @internal/sync/payload_test.go. Fixtures cover scalar strings with YAML-sensitive text, nulls, booleans, numbers, sequences, nested mappings, multiline text, aliases, and interleaved top-level keys. The matrix exercises body, title, status, tag replacement, and tag clearing independently, plus consecutive updates. Parser rejection cases verify unchanged file bytes.
