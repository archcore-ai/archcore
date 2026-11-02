---
title: "Output Leaving the Process Is Bounded by a Named Ceiling and Ordered Before It Is Cut"
status: accepted
tags:
  - "code-quality"
  - "component:cli"
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
(`@cli/cmd/hook_payload.go`). The same applies to anything crossing into a model's context, where the
cost is tokens the user pays for.

**Undetermined order.** Cutting in encounter order makes the output a function of where the input
happened to sit. `archcore status` printed an unchanged corpus's warnings in a different order every
run until `@cli/cmd/status.go` sorted the map keys. A session recap that varies run to run is not reproducible, and
a test that diffs it is flaky for a reason nobody can find.

Requirement 5 exists because truncation is silent and refusal is not. Reading exactly `limit` bytes
leaves a valid-looking prefix; reading `limit+1` turns an oversized input into an error the caller
must handle (`@cli/internal/update/update.go`).

Requirement 4 is the half that gets forgotten. Sorting by frequency alone leaves equal-frequency
elements in map order, which Go randomizes — the cut is stable but the set is not.

## Enforcement

Code review: any new constant that bounds output, and any slice expression that cuts a collection. Reference sites: the advisory engines in `@cli/internal/advisory/` (sort-then-cut with the tie-break), the session recap budget in `@cli/cmd/hooks_common.go`, the payload cap in `@cli/cmd/hook_payload.go`, and the `limit+1` reads in `@cli/internal/update/update.go`.
