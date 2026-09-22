---
title: "Code That Exists Because of a Recorded Decision Cites It by Slug"
status: accepted
tags:
  - "code-quality"
  - "docs-style"
  - "golang"
---

## Rule

1. WHEN a line of code exists because of a decision recorded in `.archcore/`, the comment MUST name
   that document as `<slug>.<type>` — no directory, no `.md`.
2. IF the comment points at one clause, THEN it MUST use the form `<slug>.<type> §N`.
3. The developer MUST NOT cite a document that is not in `.archcore/`.
4. WHEN a document is removed or renamed, the same change MUST update or delete every citation of it.
5. A citation MUST NOT replace the reason. The comment states why the code is as it is; the slug says
   where the decision is recorded.

## Rationale

The citation is a back-reference. `search_documents` on a slug returns every place in the code that
implements that decision, which is the only way to answer "what does this rule actually govern"
without reading the whole tree. Without it, a rule and its implementation drift silently — the
failure mode `@internal/docs/inspect.go` describes as already having happened.

Requirement 1 fixes the form because three were in use — bare slug, slug with `.md`, and a full
relative path — and a search that matches one misses the others. The `.md`-less form is what
`search_documents` indexes as a reference.

Requirement 3 is not hypothetical: `@internal/agents/copilot.go` cited `copilot-adapter-design.adr`,
which has never existed in this repository. A citation to nothing is worse than none, because it
reads as evidence the decision was recorded.

Requirement 5 keeps the comment useful to someone who will not open the document. The repository's
comments carry the reasoning; the slug is the footnote, not the argument.

## Examples

### Good

```go
// Sanitized because an MCP error crosses into a model's context, where an
// absolute path leaks the user's home directory —
// no-absolute-paths-in-mcp-errors.rule.
return errorResult(sanitizeError("writing "+relPath, err)), nil
```

```go
// The recap holds at most 24 document lines, so its size is a function of the
// budget rather than of corpus size — session-start-context.spec §4.
const maxRecapDocs = 24
```

### Bad

```go
// See backup-invalid-configs.adr.md
// ^ carries .md, so a path_ref search for the slug does not match it.

// See ../integrations/host-cwd-misrouting.adr
// ^ a path, which breaks the moment the document moves.

// copilot-adapter-design.adr explicitly scopes this out.
// ^ cites a document that does not exist.

// Per session-start-context.spec.
// ^ the citation replaced the reason: the reader learns nothing without opening it.
```

## Enforcement

- Code review: any comment naming a `.archcore/` document.
- `archcore doctor` reports orphaned relations, not orphaned citations; a citation to a missing
  document is caught by review and by the removal step of requirement 4.
- Reference sites: `@internal/mcp/tools/common.go`, `@internal/docs/guard.go`, `@cmd/hooks_common.go`,
  `@cmd/mcp_root.go`, `@internal/wiring/hooks_install.go`.
