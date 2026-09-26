---
title: "update_document edits Contract"
status: draft
tags:
  - "component:cli"
  - "mcp"
---

## Purpose & Scope

This spec defines the `edits` argument of the `update_document` MCP tool: exact-match replacements that change part of a document body without resending the whole body. It is normative for the handler in @cli/internal/mcp/tools/update_document.go. Agents on every host that calls `update_document` depend on it, and so do plugin skills that tell an agent how to change a document. The behavior is implemented and not yet released.

Out of scope: frontmatter retention and the other update arguments, which `document-update-frontmatter.spec` governs.

## Surface

- Argument `edits`: an array of `{old_string, new_string}` objects; the schema is in `NewUpdateDocumentTool` (@cli/internal/mcp/tools/update_document.go).
- `parseEdits` reads the argument; `applyEdits` applies it to the body that `templates.SplitDocument` returns (@cli/templates/templates.go).
- The response keeps its fields: `path`, `category`, `type`, `title`, `status`, and `tags` when present.

## Normative Behavior

1. WHEN `edits` is provided, the handler MUST apply the edits to the markdown body only.
2. WHEN `edits` holds more than one edit, the handler MUST apply the edits in array order.
3. WHEN the handler applies an edit, the handler MUST match `old_string` against the body as the earlier edits left it.
4. WHEN `old_string` occurs exactly once, the handler MUST replace that occurrence with `new_string`.
5. WHEN `new_string` is an empty string, the handler MUST delete the matched text.
6. WHEN an edit carries CRLF line endings, the handler MUST fold them to LF before matching.
7. WHEN every edit applies, the handler MUST write the document through the existing atomic write helper.
8. WHEN `edits` comes with `title`, `status`, or `tags`, the handler MUST apply every provided field in one write.

## Constraints & Invariants

- Constraint: a call MUST NOT carry both `edits` and `content`, because both define the new body.
- Invariant: an update applies every edit, or it writes nothing.
- Invariant: the frontmatter takes no part in matching, so an `old_string` that exists only in the frontmatter does not match.

## Failure Behavior

1. IF `edits` is not a non-empty array, THEN the handler MUST reject the call.
2. IF an item has no non-empty string `old_string`, THEN the handler MUST reject the call with an error that names the item index.
3. IF an item has no string `new_string`, THEN the handler MUST reject the call with an error that names the item index.
4. IF `old_string` does not occur in the body, THEN the handler MUST reject the call with an error that names the item index.
5. IF `old_string` occurs more than once, THEN the handler MUST reject the call with an error that names the match count.
6. IF a call carries both `content` and `edits`, THEN the handler MUST reject the call.
7. IF the handler rejects a call, THEN the handler MUST leave the file unchanged.

A rejection is not retriable as sent: the caller reads the document again with `get_document` and resends the edit.

## Conformance

An implementation is conformant when it satisfies behaviors 1-8, holds both invariants, and follows failure rules 1-7. `TestHandleUpdateDocument_Edits` in @cli/internal/mcp/tools/update_document_test.go and `TestUpdateDocumentEdits_RoundTrip` in @cli/internal/mcp/integration/edits_test.go check these clauses; each of 16 targeted mutants of the handler fails them (2026-09-26).

Given a CRLF document, When an agent sends a multi-line `old_string` copied from `get_document`, Then the handler applies the edit.
