---
title: "Sync Engine Contract (internal/sync)"
status: accepted
tags:
  - "component:cli"
  - "relations"
  - "sync"
---

## Purpose & Scope

Normative contract of the `internal/sync` package: the local sync-engine primitives — document scanning and hashing, the `.sync-state.json` manifest (file hashes + document relations), diff classification, and payload construction for the push-sync request. Dependents: the `sync` command (@cli/cmd/sync.go) and the MCP server store, which persists document relations through `Manifest`. Out of scope: HTTP transport (@cli/internal/api), settings and sync-mode validation (@cli/internal/config), server-side processing.

## Surface

- Scan/hash: @cli/internal/sync/hash.go
- Manifest: @cli/internal/sync/manifest.go
- Diff: @cli/internal/sync/diff.go
- Payload: @cli/internal/sync/payload.go

State file: `.archcore/.sync-state.json` — `version`, `files` (path relative to `.archcore/` → SHA-256), `relations`.

## Normative Behavior

1. `ScanFiles` MUST skip hidden directories, `settings.json`, `.sync-state.json`, and every document under a declared global source.
2. IF `ScanFiles` cannot load the declared global sources, THEN it MUST stop with an error. A misdeclared source must not leak into the push.
3. WHEN the manifest file exists, `LoadManifest` MUST run raw-JSON and semantic validation before returning the parsed manifest.
4. `BuildPayload` MUST exclude `unchanged` entries from the payload.
5. `SaveManifest` MUST write atomically: marshal to a temp file, then rename over the target.
6. `AddRelation` MUST NOT append a `(source, target, type)` triple that already exists.

## Constraints & Invariants

- Manifest `version` MUST equal 1; any other version is rejected on load (migration requires a version bump, not silent rewrite).
- The manifest MUST NOT exceed 10,000 file entries or 50,000 relations (bounds memory use on corrupt or hostile input).
- Every manifest and payload path MUST be relative and confined to `.archcore/` — no `..` escape, no absolute paths.
- The document walk MUST NOT follow symlinks, so nothing outside `.archcore/` is hashed or read. A sync payload carries full file content that the server writes and indexes, so one escaped path exposes a file outside the project.
- Code review: the reviewer verifies that a new sync boundary calls one of the path validators before it reads a file or writes a manifest entry. No test covers a boundary that does not exist yet.
- The manifest validator MUST accept only `related`, `implements`, `extends`, `depends_on`, `supports`, `contradicts`, and `supersedes` as relation types.
- The manifest validator MUST reject a relation whose source equals its target, and duplicate relation triples.
- Unknown manifest root fields fail closed — the manifest is machine-owned state, and rewriting it MUST NOT silently drop fields written by a newer binary.
- Older binaries reject manifests containing the three evidential and temporal relation values. The expanded enum retains manifest version 1; no downgrade conversion is provided.

## Failure Behavior

- IF manifest bytes are empty, malformed JSON, carry unknown root fields, or fail semantic validation, THEN `LoadManifest` MUST return an error and no manifest.
- IF a diff entry's path escapes the base directory, THEN `BuildPayload` MUST return an error and no payload.
- IF the rename step fails, THEN `SaveManifest` MUST remove the temp file and return an error.
- IF a relation's source or target file no longer exists on disk, THEN `CleanupRelations` MUST drop that relation and return the removed count.

## Conformance

An implementation is correct when it satisfies all MUST requirements, all invariants, and all failure rules above. The executable suite sits beside each source file in @cli/internal/sync/.
