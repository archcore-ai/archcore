---
title: "Tag Hygiene Checks in archcore doctor"
status: draft
tags:
  - "cli"
  - "mcp"
---

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

## Status

Not started. Deferred deliberately, tracked as a follow-up if tag sprawl becomes observable.
