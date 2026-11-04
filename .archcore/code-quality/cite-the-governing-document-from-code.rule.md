---
title: "Code That Exists Because of a Recorded Decision Cites It by Slug"
status: accepted
tags:
  - "code-quality"
  - "component:cli"
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
failure mode `@cli/internal/docs/inspect.go` describes as already having happened.

Requirement 1 fixes the form because three were in use — bare slug, slug with `.md`, and a full
relative path — and a search that matches one misses the others. The `.md`-less form is what
`search_documents` indexes as a reference.

Requirement 3 is not hypothetical: `@cli/internal/agents/copilot.go` cited `copilot-adapter-design.adr`,
which has never existed in this repository. A citation to nothing is worse than none, because it
reads as evidence the decision was recorded.

Requirement 5 keeps the comment useful to someone who will not open the document. The repository's
comments carry the reasoning; the slug is the footnote, not the argument.

## Enforcement

- Code review: any comment naming a `.archcore/` document.
- `archcore doctor` reports orphaned relations, not orphaned citations; a citation to a missing
  document is caught by review and by the removal step of requirement 4.
