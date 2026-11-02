---
title: "Three-Way Tag Update Semantics in MCP Tools"
status: accepted
tags:
  - "component:cli"
  - "mcp"
---

## Rule

1. WHEN a caller omits the `tags` argument, `update_document` MUST preserve the existing tags unchanged.
2. WHEN a caller passes an empty array `[]` as `tags`, `update_document` MUST clear all tags.
3. WHEN a caller passes a non-empty array as `tags`, `update_document` MUST replace the existing tags with that array, never append to them and never merge them.
4. `update_document` MUST detect the difference between an omitted argument and an empty array by checking argument presence in the raw request, not by checking the parsed value for nil or zero length.
5. WHEN a developer adds another frontmatter array field with the same update semantics, the developer MUST apply requirements 1 to 4 to that field.

## Rationale

"The caller said nothing about tags" and "the caller wants no tags" are different operations. A nil check or a length check collapses them into one case, which silently clears tags that the caller never intended to touch.

Only an explicit presence check on the raw argument map separates the two. The `tagsProvided` pattern in `@cli/internal/mcp/tools/update_document.go` implements it.

## Examples

Good: check whether the key exists in `request.GetArguments()`. Bad: check the parsed slice for `nil` or for `len > 0`. An omitted field and an empty array then look the same, so one of the two cases is silently lost.

## Enforcement

The tests in `@cli/internal/mcp/tools/update_document_test.go` cover clear, preserve, and replace.
