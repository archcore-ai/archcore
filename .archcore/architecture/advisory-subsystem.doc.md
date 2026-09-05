---
title: "The Advisory Subsystem"
status: accepted
tags:
  - "architecture"
  - "code-quality"
  - "integrations"
---

## Overview

`internal/advisory` holds the four engines the hooks call; all four are advisory, all four fail open,
and none of them can block an edit.

The one thing that blocks is the write guard at @cmd/hook_write_guard.go, which runs first and alone
and is not part of this subsystem. Everything described here degrades to silence on error.

## Content

### The four engines

| Engine | Call site | Trigger | Output |
|---|---|---|---|
| `CodeAlignment` | @cmd/hook_command.go:187 | before a source edit | the documents that constrain the file |
| `Precision` | @cmd/hook_post_tool_use.go:41 | after a document write | vague-requirement findings |
| `Restatement` | after a document write | a statement copied from a document the written one builds on | the duplicated statement |
| `Staleness` | @cmd/hooks_common.go:82 | session start | documents that mention directories that moved |

### Code alignment

`CodeAlignment` is the reason a rule reaches an agent that never searched for it.

An agent about to edit a file has no reason to know a document constrains it. The engine tokenizes the
file's directory chain, finds the documents that mention it, ranks them, and puts the most specific
ones in front of the edit — @internal/advisory/code_alignment.go.

| Setting | Key | Default |
|---|---|---|
| source roots | `settings.json` → `codeAlignment.sourceRoots` | `src`, `lib`, `app`, `pkg`, `cmd`, `internal`, `apps`, `packages`, `modules`, `components` |
| kill switch | `ARCHCORE_DISABLE_INJECTION=1` | unset |

A file outside every source root gets no injection. `config.CodeAlignment` preserves unknown nested
keys in `Extra`, so a newer binary's settings survive a write by an older one.

Only five document types are ever injected, ranked by how much they constrain an edit — `rule` 5,
`cpat` 4, `adr` 3, `spec` 2, `guide` 1. A type absent from that map is not injected: a `plan` or an
`idea` is context for a discussion, not a constraint on a line of code. The accept-set is derived from
the ranking, so the allowlist has one definition.

The filter is a cost control, not a preference. `docs.ScanTypes` opens only the five ranked types, so
the walk rejects roughly three quarters of the corpus before reading anything, on a path that blocks
the user's edit under a one-second host budget.

| Bound | Value |
|---|---|
| documents injected | 3 |
| directory tokens walked | 5 |
| message length | 2048 runes |

### Precision and restatement

Both run after a document write and both measure a document against a canon, not against code.

`Precision` measures the written document against the canon in `@templates/precision.go`; the engine
and the canon are separate files so either can change alone. It is deliberately over-eager — a false
"look at this" costs a glance.

`Restatement` reads the documents the written one builds on through `implements` relations and reports
a statement that survived the move nearly word for word. It matches near-verbatim text only: a
paraphrase scores far under the threshold, because a `prd` requirement and the `spec` behavior that
grades it are meant to differ. The rule it enforces is ownership rule 1 of
`document-types/prd-spec-plan-content-ownership.adr`.

### Staleness

`Staleness` compares the last commit that touched `.archcore/` against everything committed since, and
names the documents that mention the directories that moved.

The correlation is by directory name, so it over-reports by design. It is rate-limited to 24 hours
through an `internal/stamp` claim, and bounded at 12 correlated directories, 5 documents per
directory, and 10 lines total — @internal/advisory/staleness.go:26.

## Examples

**A code-quality rule reaching an edit it was never searched for.** An agent edits
`internal/mcp/tools/search_documents.go`. `CodeAlignment` tokenizes the directory chain, matches the
`.archcore/code-quality/` rules that name `internal/mcp/tools`, and injects the top three ranked
documents before the write. Rules outrank every other type, so a `rule` wins the slot over an `adr`
that mentions the same directory.

**A rule that never reaches an edit.** A rule that names no directory matches no file, so injection
cannot deliver it. That is why `CLAUDE.md` also requires loading the `code-quality` tag at the start of
Go work: injection is a safety net over a directory match, not the delivery mechanism for the set.
