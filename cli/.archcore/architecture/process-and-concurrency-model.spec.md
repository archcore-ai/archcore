---
title: "Process Shapes and the Concurrency Model"
status: accepted
tags:
  - "architecture"
  - "golang"
  - "mcp"
  - "performance"
---

## Purpose & Scope

This contract states the three process shapes the binary runs in, and the invariants that hold for
state shared inside a process and across processes.

In scope: the process shapes, the three process-global caches, the in-process mutation seam, and the
cross-process claim. Out of scope: the unattended update policy (`unattended-update.spec`), the
update trigger (`mcp-background-update.spec`), and the manifest schema (`sync-engine.spec`).

## Surface

| Shape | Entry | Lifetime | Concurrency | stdout |
|---|---|---|---|---|
| CLI command | `archcore <command>` | one operation | single goroutine | user output |
| Hook leaf | `archcore hooks <host> <event>` | sub-second | single goroutine | host protocol |
| MCP server | `archcore mcp` | the session | mcp-go worker pool plus one update goroutine | JSON-RPC stream |

| State | Location | Scope |
|---|---|---|
| `sharedScanCache` | @internal/docs/cache.go:49 | process, keyed by absolute file path |
| `sharedManifestStore` | @internal/mcp/tools/manifest_store.go:41 | process, keyed by `baseDir` |
| `mainCheckoutCache` | @internal/config/globals.go:102 | process, keyed by project root |
| claim stamp | `internal/stamp` under the XDG state directory | machine, across processes |

## Normative Behavior

1. The MCP server MUST treat `tools/call` handlers as concurrent, because mcp-go dispatches them on
   a worker pool.
2. A handler MUST mutate the manifest through `manifestStore.mutate`, never through a direct
   load-modify-save.
3. `manifestStore.mutate` MUST apply the change to a deep clone and publish the clone.
4. A caller of `manifestStore.load` MUST NOT modify the returned manifest.
5. Process-global state that varies per project MUST be keyed by a value that names one project root.
6. A goroutine started by the MCP server MUST NOT write to stdout, and MUST report to stderr instead.
7. WHEN two processes on one machine must not both act, the developer MUST establish exclusivity
   through an `internal/stamp` claim.
8. `sync.SaveManifest` MUST remain the only cross-process atomicity boundary for the manifest.
9. A hook leaf MUST stay cheap on a cold cache, because a short-lived process never warms one.
10. A lock that every handler contends for MUST NOT be held across a subprocess, a client round trip,
    or a filesystem walk.

## Constraints & Invariants

1. `sharedScanCache` retains at most `maxCachedContentBytes` (32 MiB) of document bodies. Past the
   cap an entry keeps its frontmatter and drops its body — @internal/docs/cache.go:31.
2. `sharedScanCache` runs no eviction policy by decision, because the flat cap already bounds it.
3. `sharedManifestStore.entries` carries no size bound today. One session that visits many git
   worktrees retains one parsed manifest per root for the life of the process.
4. Keying by `baseDir` is load-bearing, not a test convenience: one MCP process serves several
   project roots over its life (`project-root-resolution.spec`).
5. The background update starts after `backgroundUpdateDelay` (60 s) — @cmd/mcp.go:87.
6. `mainCheckoutCache` memoizes the git worktree anchor per project root
   (`relative-globals-resolve-from-main-checkout.adr`). It carries no size bound, for the reason
   invariant 3 gives, and holds one short string per root. A project declaring no escaping relative
   global source never populates it.
7. `mainCheckoutCache` fills under double-checked locking: the lookup runs with the mutex released,
   and a race on a cold key costs one duplicate git query rather than a serialized handler
   (clause 10). Both callers derive the same answer, so the published value does not depend on which
   one wins.
8. The session root provider serializes its refresh on a mutex separate from the one guarding its
   state, for the same reason. A call arriving while a refresh is in flight serves the root it
   already has rather than queuing (`project-root-resolution.spec` §9).

## Failure Behavior

1. IF the manifest file is missing, THEN `manifestStore.load` returns a fresh empty manifest.
2. IF `stat` on the manifest fails after a save, THEN the store drops the cache entry for that root.
3. IF a background update fails, THEN the MCP session continues and the failure reaches stderr only.
4. IF the git query behind `mainCheckoutCache` fails, THEN the anchor is empty.
5. WHEN the anchor is empty, an escaping relative global source resolves against the project root.

## Conformance

- Clause 3 is verified by the clone-and-swap in @internal/mcp/tools/manifest_store.go:79.
- Clause 6 is verified by the stdio shield in `internal/mcp` and the stderr comment at @cmd/mcp.go:50.
- Clause 10 is verified for the root provider by
  `TestSessionRootProvider_ConcurrentCallsIssueOneQuery`, and for the anchor memo by
  `TestResolveGlobalPath_LookupMemoized`.
- Clauses 1, 2, 4, 5, 7, 8, and 9 carry no automated check. Review holds them.
- Invariant 3 is a known gap. [assumption] An LRU bound of a few roots would close it at the cost of
  one JSON re-read per eviction.
