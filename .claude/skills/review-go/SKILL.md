---
name: review-go
description: Run a Go code review on git-changed files in the archcore CLI project. Retrieves project-specific code quality agreements from .archcore/ via MCP, fetches library docs via Context7, then delegates structured review to the golang-pro agent. Use when the user runs `/review-go` or asks to review Go code changes.
---

# Archcore CLI — Go Code Review

## Step 1: Identify Changed Go Files

```bash
git diff --name-only HEAD -- '*.go'
git diff --cached --name-only -- '*.go'
```

Combine and deduplicate. Split into two lists:
- **Source files**: `*.go` excluding `_test.go`
- **Test files**: `*_test.go`

If no changed `.go` files found, ask the user which files or directories to review.

## Step 2: Retrieve Project Code Quality Agreements from Archcore

Use the archcore MCP tools to load project-specific conventions that the review must check against.

1. Call `list_documents` with `tags: ["code-quality"]` to find every binding coding agreement. Do not
   narrow the tag: `code-review` returns only a subset of `.archcore/code-quality/`.
2. For each returned document, call `get_document` with its path to retrieve the full content.
3. Collect all document contents — these are the **project-specific review rules** that supplement the general Go review areas.

These documents contain established patterns extracted from the actual codebase (error handling conventions, naming rules, testing patterns, etc.). The reviewer agent must check changed code against these agreements.

## Step 3: Pre-fetch Library Documentation via Context7 MCP

The `golang-pro` agent does NOT have Context7 MCP tools — fetch docs now.

1. Read `go.mod` for third-party dependencies.
2. Scan changed Go files for `import` statements. Identify which third-party libraries are used.
3. For each non-trivial library (skip stdlib, skip obvious ones):
   a. Call `resolve-library-id` with the module path (e.g., `github.com/spf13/cobra`)
   b. If resolved, call `query-docs` with the resolved ID and a topic query relevant to usage in the changed code
4. Limit to 3-5 most important libraries.

## Step 4: Launch golang-pro Agent for Review

Use the `Agent` tool with `subagent_type: "ai-kit:golang-pro"`. Pass a prompt that includes:

- The list of changed `.go` file paths (source and test files)
- The **full text** of all archcore code review documents from Step 2
- The pre-fetched library documentation from Step 3
- The review instructions below

### Review Prompt Template

```
You are reviewing Go code changes in the archcore CLI project. Read each file listed below and produce a structured review.

## Files to Review
{list of file paths — source files first, then test files}

## Project Code Quality Agreements
The following documents define the established conventions for this project. Check changed code against these rules. Flag any violations.

{full content of each archcore document retrieved with tag "code-quality"}

## Library Documentation Context
{pre-fetched Context7 docs for third-party libraries used in these files}

## Review Areas

Analyze each file across these areas:

1. **Project Convention Compliance** — Check changed code against ALL rules from the Project Code Quality Agreements above. Flag any deviations from established patterns (error handling, naming, imports, display usage, cobra patterns, file I/O, data structures, JSON serialization, concurrency).
2. **Test Convention Compliance** — For test files: check against the testing patterns guide. Verify table-driven tests, assertion style, t.TempDir() usage, mock patterns, edge case coverage.
3. **Go Idioms & Code Quality** — Non-idiomatic patterns, naming violations, struct design issues not covered by project conventions.
4. **Error Handling** — Swallowed errors, missing error checks, incorrect error wrapping.
5. **Memory & Allocations** — Unnecessary allocations, missing pre-allocation for known-size slices, string concatenation in loops. Only practical issues.
6. **Dead Code** — Unused functions, unreachable branches, commented-out code.
7. **Code Duplication** — Repeated logic that would benefit from extraction (3+ occurrences or complex blocks). Do NOT suggest extraction for simple 2-3 line patterns.
8. **Library Usage** — Incorrect or suboptimal use of stdlib and third-party libraries. Use the library documentation provided above to verify correct API usage.

## Review Rules

- Check project conventions FIRST — these are the most important review criteria
- Be minimal and optimal — don't overcomplicate suggestions
- Be context-specific — no generic advice, only what matters for THIS code
- Simple > complex — never suggest an overengineered solution
- Every finding MUST include a file:line reference
- If a review area has no findings, omit it entirely
- Do NOT suggest adding comments, docstrings, or type annotations unless there is a clear correctness issue
- Do NOT suggest error handling for impossible scenarios
- Do NOT suggest abstractions for one-time code

## Output Format

Group findings by review area. For each finding:

**[Area Name]**
- `file.go:42` — Description of the issue and suggested fix

If everything looks good, say so briefly. Do not pad the review with praise or filler.
```

## Step 5: Present Results

Present the golang-pro agent's review directly to the user. Do not re-analyze or add extra commentary.
