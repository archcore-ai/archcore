---
title: "Three-Way Tag Update Semantics in MCP Tools"
status: accepted
tags:
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

Only an explicit presence check on the raw argument map separates the two. The `tagsProvided` pattern in `@internal/mcp/tools/update_document.go` implements it.

## Examples

Non-normative examples.

**Good** — check presence in the argument map:

```go
var newTags []string
tagsProvided := false
if _, ok := request.GetArguments()["tags"]; ok {
    tagsProvided = true
    // parse and validate...
}
```

**Bad** — check the parsed value; omission and clear become indistinguishable:

```go
if tags != nil {  // an empty array [] parses to a non-nil empty slice — clear works
    // BUT an omitted field also parses to nil — indistinguishable from "not provided"
}
```

**Bad** — check the length; omission and clear collapse into one case:

```go
if len(tags) > 0 {  // both omission and clear produce len 0
    // the clear case is silently ignored
}
```

## Enforcement

Test cases in `@internal/mcp/tools/update_document_test.go` verify the distinction:

- `TestHandleUpdateDocument_ClearTags` — an empty array clears all tags
- `TestHandleUpdateDocument_PreserveTags` — omitted tags stay unchanged
- `TestHandleUpdateDocument_ReplaceTags` — provided tags fully replace the existing tags
- `TestHandleUpdateDocument_AddTags` — tags are added to a previously untagged document
