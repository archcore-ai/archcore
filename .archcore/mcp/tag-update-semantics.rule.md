---
title: "Three-Way Tag Update Semantics in MCP Tools"
status: accepted
tags:
  - "mcp"
---

## Rule

The `update_document` MCP tool MUST maintain three-way semantics for the `tags` parameter:

1. **Omitted** — existing tags are preserved unchanged.
2. **Empty array `[]`** — all tags are cleared.
3. **Array with values** — existing tags are fully replaced (not appended or merged).

The distinction between "omitted" and "empty array" MUST be detected by checking argument presence in the raw request, NOT by checking for nil/empty on the parsed value.

This three-way semantic MUST be maintained for any future frontmatter array field that follows the same pattern.

## Rationale

"User didn't mention tags" and "user wants to clear all tags" are semantically different operations. Many serialization formats and naive nil-checks collapse these two cases into one, silently clearing tags when the caller simply didn't intend to modify them.

The implementation uses a `tagsProvided` boolean pattern in `internal/mcp/tools/update_document.go`:

```go
var newTags []string
tagsProvided := false
if _, ok := request.GetArguments()["tags"]; ok {
    tagsProvided = true
    // parse and validate...
}
```

This explicit presence check is the only reliable way to distinguish omission from an empty array.

## Examples

**Good** — check argument map presence:

```go
if _, ok := request.GetArguments()["tags"]; ok {
    tagsProvided = true
}
```

**Bad** — check parsed value (breaks clear-vs-preserve):

```go
if tags != nil {  // empty array [] becomes non-nil empty slice — clear works
    // BUT: omitted field also becomes nil — indistinguishable from "not provided"
}
```

**Bad** — check length (collapses omit and clear):

```go
if len(tags) > 0 {  // both omit and clear produce len 0
    // clear case is silently ignored
}
```

## Enforcement

Test cases in `internal/mcp/tools/update_document_test.go` verify the distinction:

- `TestHandleUpdateDocument_ClearTags` — empty array clears all tags
- `TestHandleUpdateDocument_PreserveTags` — omitted tags are preserved
- `TestHandleUpdateDocument_ReplaceTags` — provided tags fully replace existing
- `TestHandleUpdateDocument_AddTags` — tags added to previously untagged document
