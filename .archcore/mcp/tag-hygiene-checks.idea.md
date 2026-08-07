---
title: "Tag Hygiene Checks in archcore doctor"
status: accepted
tags:
  - "cli"
  - "mcp"
---

## Status

**Shipped 2026-04-07** in commit `df1015e`. Both checks live in `@cmd/status.go`. They run in
`archcore status`, and `archcore doctor` reports the same lines because it calls `runStatusChecks`.

- The singleton-tag warning emits `tag %q is used only once (possible typo)`.
- The unique tag count emits `Tag hygiene OK (%d unique tag(s))`.

The shipped form diverges from the sketch below in two ways:

1. The unique tag count is a standalone report line, not a field beside a document count.
   `archcore status` reports no total local document count; its only document count is per global
   source.
2. Neither containment measure named under "Risks" shipped. There is no corpus-size threshold and no
   `--strict` gate, so every tag carried by exactly one document produces a warning.

A singleton warning is not an issue. The report still states `Tag hygiene OK` when singletons are the
only finding, which `@cmd/status.go` records in an inline comment.

The predicted noise is observable. In `@examples/`, the projects `03-product-planning`,
`04-experience-playbook`, `06-global-multiple-sources`, `07-local-overrides-global`, and
`10-monorepo-root-global` each warn on at least one singleton tag while still reporting
`Tag hygiene OK`.

Tag counting covers local documents only, as required by
`globals/globals-are-read-only-everywhere.rule.md`.

The text below preserves the original idea as historical record. Its "Idea", "Possible
Implementation", and "Risks" sections describe the state before implementation, not current behavior.

## Idea

Add two tag hygiene checks to `archcore doctor`:

- a singleton-tag warning, which names every tag carried by exactly one document;
- a unique tag count in the summary.

Tag format is already validated inline by `@cmd/status.go`, which rejects a malformed tag. Neither
hygiene check is implemented. This is the remaining part of the tags work; everything else shipped.

## Value

A tag typed twice with two spellings splits a set silently. `list_documents` with the misspelled tag
returns one document, and the reader concludes the set is small rather than concluding the tag is
wrong. A singleton tag is the observable symptom, so reporting singletons turns a silent split into a
visible one.

The unique tag count makes sprawl measurable. Without it, no surface answers "how many tags does this
corpus carry?", so there is no signal for when a tag vocabulary needs pruning.

## Possible Implementation

- Aggregate tag frequency over the scan `archcore doctor` already runs, so the check adds no
  filesystem work.
- Report a singleton as a warning, not a failure. A tag carried by one document is often correct on
  a young corpus.
- Put the unique tag count in the doctor summary next to the document count.

## Risks

- On a small corpus most tags are singletons, so the warning is noise until the corpus grows. A
  threshold on corpus size, or a `--strict` gate, would contain it.
- The check has no way to tell a typo from a deliberate narrow tag. It reports; it does not correct.
