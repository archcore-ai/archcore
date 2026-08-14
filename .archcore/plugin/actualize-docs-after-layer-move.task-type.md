---
title: "Actualize Documents After a Layer Moves Out of the Plugin"
status: draft
tags:
  - "development"
  - "plugin"
  - "validation"
---

## What

Repoint the accepted documents that name artifacts a refactor deleted. A commit moves a layer out of the plugin, deletes the scripts and tests that carried it, and leaves every `spec`, `rule`, and `guide` that named them describing a tree that no longer exists. Nothing fails, because those documents are prose rather than code — the drift is silent until an agent reads one as a live contract.

## When

- A commit deletes plugin artifacts because their policy moved elsewhere. `ca6dfb4` moved hook policy into the `archcore` CLI and deleted six `bin/check-*` scripts with their tests; `2f99997` removed the bundled launcher and three more tests.
- `/archcore:review --deep` reports documents citing paths absent from the tree.
- A `[Archcore Cascade]` notice names a source document whose target moved in a refactor commit.

Scale observed on 2026-08-14: 21 accepted documents cited artifacts deleted in `ca6dfb4` alone, across four specs, three rules, one guide, and thirteen claim-recording documents.

## Steps

1. Find the deletion commit: `git log --oneline -1 --diff-filter=D -- <path>`.
2. List every artifact that commit deleted with `git show --stat <sha>`.
3. Grep `.archcore/` for each deleted name; group the hits by document status.
4. Read the successor in the current tree — the launcher, the CLI leaf, the replacement test.
5. Classify each hit as a present-tense claim or a past-tense record.
6. Rewrite each present-tense claim in a `spec`, `rule`, or `guide` to name the successor.
7. Add one sentence recording the removal with its commit hash.
8. Apply each document through `update_document`, one user confirmation per document.
9. Run the repository's test suite and verification gate.

## Example

Grouping the hits by status is what keeps the edit small. From the 2026-08-14 pass over `ca6dfb4`:

```
[accepted] spec  host-adapter-contract.spec.md   -> check-archcore-write, check-code-alignment, validate-archcore
[accepted] rule  mcp-only-operations.rule.md     -> check-archcore-write, validate-archcore
[rejected] plan  development-roadmap.plan.md     -> check-cascade, check-precision, validate-archcore
```

The first two carried present-tense obligations and were rewritten. The third is a rejected plan — a historical record, left untouched. Of 36 documents citing the deleted names, 8 needed an edit.

The rewrite keeps the obligation and swaps the artifact:

```
7. An adapter MUST route the pre-mutation guard event to `bin/check-archcore-write`.   <- before
7. An adapter MUST route the pre-mutation guard event to `bin/pre-tool-use`.           <- after
```

## Pitfalls

- A raw path-existence scan across all documents returns mostly false positives — bare filenames resolve relative to `skills/_shared/`, and many cited paths belong to other repositories. Grep for the specific deleted names instead.
- Date arithmetic over the relation graph flags every source whose target moved later, same-commit pairs included. Filter to sources with status `accepted` or `draft` and a lag of at least one day.
- A `rejected` document, and a `prd`, `idea`, or `rnd` recording intent at the time, are history. Leaving them unchanged is the correct outcome, not an omission.
- An anti-pattern code example naming the artifact is not a claim. Read the surrounding block before rewriting.
- A deleted test sometimes guarded a budget that nothing replaces. State the gap; deleting the sentence hides it.
- `update_document` replaces the whole body, so a one-line correction still resends the full text.
- The precision hook reads the whole document, so an edit surfaces that document's pre-existing findings alongside any the edit introduced. Separate the two before acting.
