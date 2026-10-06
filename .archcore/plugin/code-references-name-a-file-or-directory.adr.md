---
title: "Code References Name a File or a Directory, Never a Line"
status: accepted
tags:
  - "component:cli"
  - "component:plugin"
  - "docs-style"
  - "precision"
---

## Context

A field audit of a 294-document corpus (frontend monorepo, 2026-10-06) checked line anchors such as `headers-middleware.ts:34-40` against `origin/master`: in every checked case across `spec`, `adr`, and `doc` the cited lines had moved, from 8 to 24 lines. No plugin prompt asks for a line number — the only "line range" in `@plugin/plugins/archcore/skills/_shared/grounding/convert-routing.md` describes import provenance, not a document body — so agents add the anchors on their own initiative. The engine reports no finding for them, because `@cli/internal/advisory/precision.go` has no line-anchor check.

## Decision

A document body cites code as `@path/to/file` or `@path/to/dir/`, adds a symbol in inline code when a narrower target is needed, and never carries a line number, a line range, or a `#L` fragment.

## Alternatives Considered

1. Keep line anchors and re-verify them on review — rejected because a line number changes on every unrelated edit above it, so review would report drift on documents whose claims still hold.
2. Allow anchors pinned to a commit (`path@<sha>:12`) — rejected because a pinned anchor stays valid while describing code that no longer exists, which hides the staleness the reference exists to expose.
3. Forbid symbol names as well and keep only paths — rejected because a 600-line file such as `AuthProvider.tsx` then gives the reader no way to find the cited behavior; a symbol name survives reordering and renames are caught by search.

## Consequences

- A reference stays valid until the file or directory is moved or deleted, and that failure is mechanically detectable by resolving the path.
- The engine gains one advisory finding for a line anchor next to a path; the finding never blocks a write.
- Existing documents that carry anchors report the finding on their next edit, because the post-write check reads the whole document.
- Cost: a reader of a long file searches for the symbol instead of jumping to a line.

## Superseded when

- A host renders `@path` references as live links that resolve a symbol to its current line, which makes a stored line number redundant rather than stale.
- Measured re-verification shows that anchors in the corpus stay correct across 90 days of normal commit activity.
