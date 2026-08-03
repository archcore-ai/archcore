---
title: "Sync Paths Must Be Validated Against Traversal Attacks"
status: accepted
tags:
  - "sync"
---

## Rule

1. The CLI MUST validate every file path in a sync payload and in the manifest against path traversal. `validateRelPath()` in `@internal/sync/payload.go` performs this check at every boundary.
2. A path MUST NOT be `..` and MUST NOT begin with `../`, checked after `filepath.Clean`.
3. A path MUST NOT be absolute.
4. A path MUST be relative to `.archcore/` and MAY live in any subdirectory. The directory layout is free-form and carries no category-prefix requirement.
5. A manifest entry MUST NOT contain `//` and MUST NOT end with `/`. `validateFileEntry` in `@internal/sync/manifest.go` performs this check.
6. `ScanFiles` in `@internal/sync/hash.go` MUST walk `.archcore/` through the shared `WalkArchcoreFilesSkipping`, which skips hidden directories and meta files.
7. `ScanFiles` MUST skip the reserved `global/` tree and every document under a declared global source, so that a read-only global is never pushed as a local document.

A filename that merely starts with two dots, such as `..notes.adr.md`, is legal and passes requirement 2.

## Where validation occurs

- Manifest loading: `validateFileEntry` and `validateRelations` validate every path in the `files` map and every relation endpoint.
- Payload construction: `validateRelPath` validates the path of each diff entry before the CLI reads the file.
- File scanning: `ScanFiles` reads only through the shared archcore document walk, so nothing outside `.archcore/` is hashed or read.

## Rationale

A sync payload carries full file content that the server writes and indexes. A crafted path such as `../../etc/passwd`, or an absolute path, could make the server read or write outside the expected document scope. Validating at the local scan, at the manifest, and at the payload gives defense in depth.

## Enforcement

- `validateRelPath` in `@internal/sync/payload.go` and `validateFileEntry` in `@internal/sync/manifest.go` reject an invalid path before any file read or manifest write.
- Tests: `@internal/sync/payload_test.go`, `@internal/sync/manifest_test.go`, and `@internal/sync/hash_test.go` cover the traversal, absolute-path, and skip-list cases.
- Code review: the reviewer verifies that a new sync boundary calls one of the validators before it reads a file or writes a manifest entry.

## History

An earlier version of this rule required category-prefix validation (`vision/`, `knowledge/`, `experience/`) and a non-recursive top-level scan. That contract predates the accepted decision on the free-form directory structure and never matched the shipped recursive implementation. The July 2026 audit follow-up aligned this rule with the implemented contract.
