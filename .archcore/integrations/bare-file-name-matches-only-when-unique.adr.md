---
title: "A Shared Bare File Name Matches Only Through an Accepted Rule"
status: accepted
tags:
  - "component:cli"
  - "hooks"
  - "integrations"
  - "mcp"
---

## Context

File-context resolution gave reason `file` to every document whose body held the base name of the edited file, unless the stem was in `genericFileStems` (@cli/internal/advisory/code_alignment.go). On 2026-10-15 this repository tracked 21 files named `settings.json` and 18 named `AGENTS.md`, so a document about one `settings.json` reached the edit of all 21, and `cli/examples/01-minimal/.archcore/settings.json` collected 45 candidate documents for 5 matched rows. A simulation over the 775 tracked files outside `.archcore/` and the 254 local and global documents of the ranked types counted 3 557 bare-name rows: 3 071 on 148 files whose base name another tracked file carries, and 486 on 189 files with a unique base name; 72 of the 486 matched only inside a longer name, such as `document.go` inside `create_document.go`.

## Decision

Adopted three changes in the file-context engine: a bare file name gives reason `file` only when `git ls-files` lists no other tracked file with that base name; a shared base name reaches an edit only through an accepted `rule` or `cpat` that names it, with reason `kind`; and a document names a base name only where no file-name character and no `.` precede it — a full-path or `@`-path match still gives reason `file`, and a project whose tracked files cannot be listed keeps only the `genericFileStems` test.

## Alternatives Considered

1. A document-count threshold N, where a base name that more than N documents name stops matching by name — rejected because the count measures how central a file is, not whether its name is ambiguous: at N = 20 the threshold cuts 21 base names on 2026-10-15, among them `server.go` and `session-start`, which one tracked file each carries.
2. Add `settings`, `agents`, `claude`, and `skill` to `genericFileStems` — rejected because a shared name is a property of one project: `settings.json` is shared here and unique in a project with one settings file, and each new shared name would need a CLI release.
3. Keep only the `genericFileStems` test — rejected because 3 071 of the 3 557 bare-name rows on this repository land on files whose base name another tracked file carries.
4. Drop every row on a shared base name, the rule included — rejected because `declaring-global-sources.rule` and `forward-compatible-settings-parsing.rule` govern the format of every `settings.json` and stopped reaching `cli/examples/01-minimal/.archcore/settings.json`; the user chose the `kind` reason on 2026-10-16.

## Consequences

Positive:

- Measured with @plugin/test/behavioral/delivery-bench.py over 406 files of this repository on 2026-10-16, a HEAD build against this change: files at the cap of 5 matched rows fell from 100 to 76, mean candidates per file from 6.01 to 3.73, bare-name rows on shared names from 140 to 0, and bare-name rows on unique names from 165 to 152; the delivery count of accepted documents per type did not change.
- The format rules on `settings.json` reach every `settings.json` again, as `kind` rows.
- The rule has no threshold and no per-project setting to tune.
- The test costs one `git ls-files` query with a name pathspec: at most 20 ms on this repository (1 075 tracked files) and on a 4 712-file repository, on 2026-10-15.

Negative:

- An `adr`, `spec`, `guide`, or `doc` that names a shared file only by its base name no longer reaches the edit of that file; the author restores the row by writing the full path.
- A file at the project root keeps every bare mention, because its base name is its full path: root `AGENTS.md` still collects a document that writes `AGENTS.md` about `cli/AGENTS.md`.
- Delivery changes without a document edit: a second tracked `hooks.json` turns the bare-name rows of the first into `kind` rows for rules and removes them for other types.
- The query cost above 100 000 tracked files is unmeasured [assumption].

## Superseded when

- The delivery measurement records a `git ls-files` query above 200 ms on a bench project, one fifth of the one-second hook budget.
- More than 5 accepted rules name one shared base name, so `kind` rows alone fill the matched cap of a file that no document names by full path.
- The kernel records which document describes which file, so the engine no longer infers it from file names.
