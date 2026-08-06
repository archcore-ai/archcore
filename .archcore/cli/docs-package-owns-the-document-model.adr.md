---
title: "internal/docs Owns the Document Model, Scan, and Path Guards"
status: accepted
tags:
  - "cli"
  - "code-quality"
  - "mcp"
---

## Context

The document model lived in `internal/mcp/tools/common.go`: the `LocalDocument` struct, the two-phase
scan, the per-file cache, the global-source resolver, and every path guard the write tools consult.
The file had grown past 500 lines and its package name said MCP.

The hook handlers in `cmd/` need the same things. A pre-write guard must answer "is this path a
document the MCP tools would accept?", and the session recap and the code-alignment injection both
need the local scan. Importing `internal/mcp/tools` from `cmd/` to reach them would tie the hook path
to the MCP tool layer, and the tool layer already imports nothing from `cmd/` on purpose — that is
what the executor-injection pattern exists to avoid.

## Decision

Move the document model out of the MCP tool layer into `internal/docs`, and reach it from
`internal/mcp/tools` through one seam file.

- `@internal/docs/document.go` — `Document`, `EnrichedDocument`, `DocumentRelation`,
  `ReadDocumentContent`, `NormalizeRelPath`, `WriteFileAtomic`.
- `@internal/docs/scan.go` — `Scan`, `ScanFull`, `ScanLocal`, `BuildDoc`.
- `@internal/docs/cache.go` — the mtime-and-size-keyed per-file cache and `InvalidateCache`.
- `@internal/docs/guard.go` — `GuardWritablePath`, `ValidateReadPath`, `ValidateArchcorePath`,
  `CheckSymlinkContainment`.
- `@internal/docs/globals.go` — `IsGlobalPath`, `IsReservedGlobalDir`, `IsReadOnlyGlobalPath`,
  `AnnotateSource`.
- `@internal/docs/inspect.go` — `InspectGlobals`, `GlobalState`, `GlobalInspection`.
- `@internal/mcp/tools/docs_bridge.go` — the seam. It aliases `LocalDocument` to `docs.Document` and
  re-exports the helpers under the short names the tool handlers already call.

`LocalDocument` stays the name on the MCP wire and in every caller outside the package;
`docs.Document` is the domain name. A type alias keeps both true without a conversion layer.

## Alternatives

**Leave the model in `internal/mcp/tools` and import it from `cmd/`.** Rejected: it makes the hook
path depend on the MCP tool layer, and it keeps a package named for one consumer holding the model
that three consumers need.

**Duplicate a smaller scan inside `cmd/`.** Rejected: the write guard must agree with the MCP write
tools exactly. Two implementations of one predicate is the defect, not the fix.

**Rename `LocalDocument` to `Document` everywhere.** Rejected as a separate concern: the field name
is part of the MCP response shape that the plugin reads.

## Consequences

- The hook handlers, the MCP tools, and the status report share one scan, one cache, and one set of
  path guards.
- A path the MCP write tools refuse cannot be reached by editing the file directly, because the write
  guard calls the same function.
- `internal/mcp/tools` keeps its short call sites. Only `docs_bridge.go` knows where the helpers now
  live, so a further move touches one file.
- Documents that reference `internal/mcp/tools/common.go`, `scan_cache.go`, or `globals_inspect.go`
  point at paths that no longer exist and need updating when they are next touched.
