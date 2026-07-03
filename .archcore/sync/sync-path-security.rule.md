---
title: "Sync Paths Must Be Validated Against Traversal Attacks"
status: accepted
tags:
  - "sync"
---

## Rule

All file paths in sync payloads and manifests MUST be validated to prevent path traversal. The `validateRelPath()` function (@internal/sync/payload.go) enforces this at every boundary.

## Validation Requirements

- Paths MUST NOT be `..` or begin with `../` (checked after `filepath.Clean`; a legal filename that merely starts with two dots, e.g. `..notes.adr.md`, is permitted)
- Paths MUST NOT be absolute (no leading `/`)
- Paths are relative to `.archcore/` and MAY live in any subdirectory — the directory layout is free-form (see @.archcore/dir/free-form-directory-structure.adr.md); there is no category-prefix requirement
- Manifest entries additionally MUST NOT contain `//` (double slashes) or end with `/` (`validateFileEntry`, @internal/sync/manifest.go)
- `ScanFiles` (@internal/sync/hash.go) walks `.archcore/` recursively via the shared `WalkArchcoreFilesSkipping`, which skips hidden directories and meta files; it also skips the reserved `global/` tree and documents under declared global sources — read-only globals are never pushed as local documents (see @.archcore/globals/globals-are-read-only-everywhere.rule.md)

## Where Validation Occurs

- **Manifest loading** — every path in the `files` map and every relation endpoint is validated via `validateFileEntry` / `validateRelations`
- **Payload construction** — every path in diff entries is validated via `validateRelPath` before its file is read
- **File scanning** — `ScanFiles` reads only through the shared archcore document walk (`.md` documents, skip-list applied); nothing outside `.archcore/` is ever hashed or read

## Rationale

The sync payload includes full file content that gets written/indexed on the server. A crafted path like `../../etc/passwd` or an absolute path could trick the server into reading or writing outside the expected document scope. Validating at every boundary (local scan, manifest, payload) provides defense in depth.

## History

The original version of this rule mandated category-prefix validation (`vision/`, `knowledge/`, `experience/`) and a non-recursive top-level scan. That contract predates @.archcore/dir/free-form-directory-structure.adr.md and never matched the shipped recursive free-form implementation; this rule was aligned to the actual contract in the July 2026 audit follow-up.