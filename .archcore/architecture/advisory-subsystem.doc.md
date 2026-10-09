---
title: "The Advisory Subsystem"
status: accepted
tags:
  - "architecture"
  - "code-quality"
  - "component:cli"
  - "integrations"
---

## Overview

`internal/advisory` holds the four engines the hooks call; all four are advisory, all four fail open,
and none of them can block an edit.

The one thing that blocks is the write guard at @cli/cmd/hook_write_guard.go, which runs first and alone
and is not part of this subsystem. Everything described here degrades to silence on error.

## Content

### The four engines

| Engine | Call site | Trigger | Output |
|---|---|---|---|
| `CodeAlignment` | `preToolUseHandler` in @cli/cmd/hook_command.go and `search_documents` `for_path` | before a source edit, or on the agent's call | the file-context result: the documents that constrain the file |
| `Precision` | `postToolUseHandler` in @cli/cmd/hook_post_tool_use.go | after a document write | vague-requirement findings |
| `Restatement` | after a document write | a statement copied from a document the written one builds on | the duplicated statement |
| `Staleness` | `buildSessionContext` in @cli/cmd/hooks_common.go | session start | documents that mention directories that moved |

### Code alignment

`CodeAlignment` is the reason a rule reaches an agent that never searched for it.

An agent about to edit a file has no reason to know a document constrains it. `ResolveFileContext` in
@cli/internal/advisory/code_alignment.go gives each matching document a reason — `file` (the body names
the file), `kind` (an accepted `rule` or `cpat` names the compound extension, such as `.test.tsx`),
`directory` (the body names a directory of the file), or `general` (an accepted `rule` or `cpat` whose
references name no existing path). The hook renders the result before the edit; `search_documents`
returns the same result under `for_path`. The shared path-reference extractor lives in
@cli/internal/docs/pathref.go.

| Setting | Key | Default |
|---|---|---|
| source roots | `settings.json` → `codeAlignment.sourceRoots` | `src`, `lib`, `app`, `pkg`, `cmd`, `internal`, `apps`, `packages`, `modules`, `components` |
| kill switch | `ARCHCORE_DISABLE_INJECTION=1` | unset |

A file outside every source root gets only `file` and `general` rows. `config.CodeAlignment` preserves unknown nested
keys in `Extra`, so a newer binary's settings survive a write by an older one.

Seven document types are ranked by how much they constrain an edit — `rule`, `cpat`, `adr`, `spec`,
`scenario`, `guide`, then `doc`. Rows order by reason (`file`, `kind`, `directory`), then match depth,
then `accepted` before `draft`, then type, then mention count. A `scenario` reaches an edit through the
`Anchors:` line of its flows, below the `spec` it illustrates. A type absent from that map is not injected: a `plan` or an
`idea` is context for a discussion, not a constraint on a line of code. The accept-set is derived from
the ranking, so the allowlist has one definition.

The filter is a cost control, not a preference. `docs.ScanTypes` opens only the six ranked types, so
the walk rejects roughly three quarters of the corpus before reading anything, on a path that blocks
the user's edit under a one-second host budget.

| Bound | Value |
|---|---|
| matched rows | 5 |
| general rows | 10 |
| directory tokens walked | 5 |
| hook message length | 2048 runes |
| `for_path` response | 8000 bytes |

### Precision and restatement

Both run after a document write and both measure a document against a canon, not against code.

`Precision` measures the written document against the canon in `@cli/templates/precision.go`; the engine
and the canon are separate files so either can change alone. It is deliberately over-eager — a false
"look at this" costs a glance. Two of its checks read language data from that canon: a graded clause
with no BCP 14 keyword is reported with the native modal it used (`NativeModals`, one row per
language), and a code reference with a line number is reported when the path ends in one of
`CodeReferenceExtensions` — `plugin/language-neutral-structure-tokens.adr` and
`plugin/code-references-name-a-file-or-directory.adr`.

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
directory, and 10 lines total — @cli/internal/advisory/staleness.go.

## Examples

**A code-quality rule reaching an edit it was never searched for.** An agent edits
`internal/mcp/tools/search_documents.go`. `CodeAlignment` tokenizes the directory chain, matches the
`.archcore/code-quality/` rules that name `internal/mcp/tools`, and injects the top three ranked
documents before the write. Rules outrank every other type, so a `rule` wins the slot over an `adr`
that mentions the same directory.

**A rule that names no path.** A rule whose references name no existing path enters every result as a
`general` row, title only, after the matched rows; past 10 rows a remainder line names the call that
lists the rest. The `CLAUDE.md` tag load for `code-quality` stays, because a rule that cites one
example path is neither general nor a directory match.
