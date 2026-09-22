---
title: "Output Leaving the Process Is Bounded by a Named Ceiling and Ordered Before It Is Cut"
status: accepted
tags:
  - "code-quality"
  - "golang"
  - "performance"
---

## Rule

1. Data leaving the process — session context, an MCP tool response, an HTTP request, a host config
   file — MUST be bounded by a named constant declared beside its use.
2. The constant's doc comment MUST name the budget it protects, not restate its value.
3. WHEN a collection is cut to a ceiling, the developer MUST give it a deterministic order first.
4. The ordering MUST break ties on a stable key, so that equal elements do not swap between runs.
5. WHEN reading a stream whose size the process does not control, the developer MUST read through
   `io.LimitReader(r, limit+1)` and MUST treat the extra byte as an error rather than truncating.
6. A ceiling that changes what a caller is guaranteed MUST be stated in the contract that caller
   reads, not only in the code.

## Rationale

Two different failures, one habit.

**Unbounded size.** A `PostToolUse` payload echoes the tool response, so its size is the host's
choice, not this program's; held as `map[string]any` it costs several times its bytes
(`@cmd/hook_payload.go`). The same applies to anything crossing into a model's context, where the
cost is tokens the user pays for.

**Undetermined order.** Cutting in encounter order makes the output a function of where the input
happened to sit. `archcore status` printed an unchanged corpus's warnings in a different order every
run until `@cmd/status.go` sorted the map keys; the precision advisory says it outright — capping in
encounter order makes the reported set a function of where the words appear
(`@internal/advisory/precision.go`). A session recap that varies run to run is not reproducible, and
a test that diffs it is flaky for a reason nobody can find.

Requirement 5 exists because truncation is silent and refusal is not. Reading exactly `limit` bytes
leaves a valid-looking prefix; reading `limit+1` turns an oversized input into an error the caller
must handle (`@internal/update/update.go`).

Requirement 4 is the half that gets forgotten. Sorting by frequency alone leaves equal-frequency
elements in map order, which Go randomizes — the cut is stable but the set is not.

## Examples

### Good

```go
// maxSessionTags caps the number of tags emitted in SessionStart context.
// Capped at 20 (top-N by frequency) to limit static token overhead per session
// while preserving enough coverage for projects with rich tag namespaces.
const maxSessionTags = 20

slices.SortFunc(sorted, func(a, b tagCount) int {
    if c := cmp.Compare(b.count, a.count); c != 0 {
        return c // most frequent first
    }
    return cmp.Compare(a.tag, b.tag) // ties broken by name, so the set is stable
})
tags := sorted[:min(maxSessionTags, len(sorted))]
```

```go
// One byte past the limit, so an oversized body is an error rather than a
// silently truncated document.
body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
if int64(len(body)) > maxBodyBytes {
    return fmt.Errorf("response exceeds %d bytes", maxBodyBytes)
}
```

### Bad

```go
const maxTags = 20 // maximum number of tags

// Cut in map order: the same corpus emits a different set on every run.
for tag := range tagFreq {
    tags = append(tags, tag)
    if len(tags) == maxTags {
        break
    }
}

// Sorted, then cut, but ties left in map order: the set is unstable at the edge.
slices.SortFunc(sorted, func(a, b tagCount) int { return cmp.Compare(b.count, a.count) })

// Reads exactly the limit, so an oversized body arrives as a valid-looking prefix.
body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
```

## Enforcement

- Code review: any new constant that bounds output, and any slice expression that cuts a collection.
- `@internal/advisory/precision.go`, `@internal/advisory/code_alignment.go`,
  `@internal/advisory/staleness.go` — sort-then-cut, each with the tie-break.
- `@cmd/hooks_common.go` — the session recap budget.
- `@cmd/hook_payload.go` — the payload cap and the patch line bound.
- `@internal/update/update.go` — the `limit+1` reads.
- `@cmd/status.go` — sorted map keys, with the reason on the line.
