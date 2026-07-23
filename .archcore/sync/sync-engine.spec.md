---
title: "Sync Engine Contract (internal/sync)"
status: accepted
tags:
  - "relations"
  - "sync"
---

## Purpose & Scope

Normative contract of the `internal/sync` package: the local sync-engine primitives — document scanning and hashing, the `.sync-state.json` manifest (file hashes + document relations), diff classification, and payload construction for the push-sync request. Dependents: the `sync` command (@cmd/sync.go) and the MCP server store, which persists document relations through `Manifest`. Out of scope: HTTP transport (@internal/api), settings and sync-mode validation (@internal/config), server-side processing.

## Surface

- Scan/hash: `FileState`, `HashFile`, `ScanFiles` — @internal/sync/hash.go
- Manifest: `Manifest`, `Relation`, `RelationType`, `ManifestFile`, `LoadManifest`, `SaveManifest`, `ValidateManifestJSON`, `ValidateManifest`, `Clone`, `AddRelation`, `RemoveRelation`, `RelationsFor`, `CleanupRelations` — @internal/sync/manifest.go
- Diff: `DiffAction` (`created`/`modified`/`deleted`/`unchanged`), `DiffEntry`, `Diff`, `HasChanges`, `FilterByAction` — @internal/sync/diff.go
- Payload: `Payload`, `FileEntry`, `BuildPayload` — @internal/sync/payload.go

State file: `.archcore/.sync-state.json` — `version`, `files` (path relative to `.archcore/` → SHA-256), `relations`.

## Normative Behavior

1. `ScanFiles` MUST return every Markdown document under `.archcore/` with its slash-separated path relative to `.archcore/` and its content hash.
2. `ScanFiles` MUST skip hidden directories, `settings.json`, `.sync-state.json`, the reserved `global/` tree, and every document under a declared global source.
3. `HashFile` MUST return the SHA-256 digest of the file content as 64 lowercase hex characters.
4. WHEN the manifest file is absent, `LoadManifest` MUST return a fresh empty manifest.
5. WHEN the manifest file exists, `LoadManifest` MUST run raw-JSON and semantic validation before returning the parsed manifest.
6. `Diff` MUST classify each scanned file against the manifest: absent from manifest → `created`, hash differs → `modified`, hash equal → `unchanged`.
7. `Diff` MUST report each manifest entry with no matching file on disk as `deleted`.
8. `Diff` MUST list `deleted` entries in ascending path order.
9. `BuildPayload` MUST exclude `unchanged` entries from the payload.
10. `BuildPayload` MUST place each `deleted` entry's path in the payload `deleted` list.
11. WHEN adding a created or modified file to the payload, `BuildPayload` MUST populate the entry with the file content, parsed frontmatter, filename-derived `doc_type`, and type-derived `category`.
12. `SaveManifest` MUST write atomically: marshal to a temp file, then rename over the target.
13. `AddRelation` MUST NOT append a `(source, target, type)` triple that already exists.
14. `Clone` MUST return a deep copy such that mutations of the clone are never observable through the original snapshot.

## Constraints & Invariants

- Manifest `version` MUST equal 1; any other version is rejected on load (migration requires a version bump, not silent rewrite).
- The manifest MUST NOT exceed 10,000 file entries or 50,000 relations (bounds memory use on corrupt or hostile input).
- Every stored hash MUST match `^[0-9a-f]{64}$`.
- Every manifest and payload path MUST be relative and confined to `.archcore/` — no `..` escape, no absolute paths.
- Relation `type` MUST be one of `related`, `implements`, `extends`, `depends_on`; source and target MUST differ; duplicate triples are invalid.
- Unknown manifest root fields fail closed — the manifest is machine-owned state, and rewriting it MUST NOT silently drop fields written by a newer binary.

## Failure Behavior

- IF manifest bytes are empty, malformed JSON, carry unknown root fields, or fail semantic validation, THEN `LoadManifest` MUST return an error and no manifest.
- IF hashing any scanned file fails, THEN `ScanFiles` MUST abort with an error naming the relative path.
- IF a diff entry's path escapes the base directory, THEN `BuildPayload` MUST return an error and no payload.
- IF a file's frontmatter cannot be split or its `status` is invalid, THEN `BuildPayload` MUST return an error naming the relative path.
- IF the rename step fails, THEN `SaveManifest` MUST remove the temp file and return an error.
- IF a relation's source or target file no longer exists on disk, THEN `CleanupRelations` MUST drop that relation and return the removed count.

## Conformance

An implementation is correct when it satisfies all MUST requirements, all invariants, and all failure rules above. Executable conformance suite: @internal/sync/hash_test.go, @internal/sync/manifest_test.go, @internal/sync/diff_test.go, @internal/sync/payload_test.go.

Example — Given a manifest entry `a.adr.md → h1`, When the file on disk hashes to `h2`, Then `Diff` reports `a.adr.md` as `modified` with hash `h2`.