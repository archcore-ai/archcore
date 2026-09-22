---
title: "Worktree-Aware Globals Resolution and a Session-Following MCP Root"
status: accepted
tags:
  - "config"
  - "globals"
  - "golang"
  - "mcp"
---

## Goal

Make the Archcore MCP server usable from a git worktree. Phase 1 closed issue #30 so a worktree resolves its declared global sources and the server starts. Phase 2 closed issue #31 so a session that enters a worktree mid-session moves the server's root with it.

Both phases have landed. This plan is kept as the record of what was built and in what order.

## Declared Delta

- `creates`: `session-following-mcp-root` — contract in @.archcore/mcp/project-root-resolution.spec.md.
- `modifies`: `global-source-path-resolution` — covered by @.archcore/globals/global-sources.spec.md §1, amended by task 6.
- `retires`: none.
- `decision`: @.archcore/globals/relative-globals-resolve-from-main-checkout.adr.md and @.archcore/mcp/session-following-project-root.adr.md.
- `intent_gap`: no.
- Route: `capability`, size L. Raised by M=stone (@.archcore/globals/global-sources.spec.md is accepted and carries eight dependents) and R=external-contract (`settings.json` resolution semantics and MCP tool behavior across hosts).

## Tasks

### Phase 1 — worktree-aware globals resolution (#30) — done

1. [x] Add the working-tree queries to @internal/git/git.go: `Toplevel`, `MainCheckout`, and the `WorktreeRoots` pair, all over the existing `run` with its 500 ms bound.
2. [x] Add `ResolveGlobalPathFrom` to @internal/config/globals.go as the pure core, and classify a cleaned relative path as in-tree or escaping in `ResolveGlobalPath`.
3. [x] Derive the anchor as the primary's position inside the main checkout, evaluate symlinks on both sides, and accept it only when it holds `.archcore/`.
4. [x] Memoize the anchor per project root behind the `lookupWorktreeRoots` seam, and reach it only for an escaping path.
5. [x] Leave the scan, the health reporter, the write guard, and the sync hash untouched — each already resolves through `config.ResolveGlobalPath`.
6. [x] Amend §1 of @.archcore/globals/global-sources.spec.md with the classification rule, the anchor, and the fallback.
7. [x] Add table-driven cases to @internal/config/globals_anchor_test.go and @internal/git/main_checkout_test.go for a main checkout, a linked worktree, a nested project, a submodule, a non-git directory, and a machine without git.
8. [x] Add the worktree fixture in @internal/mcp/integration/globals_worktree_test.go beside `setupPrimaryWithSiblingGlobal`.
9. [x] Add the startup cases in @cmd/mcp_worktree_test.go: a worktree that resolves only from the main checkout, and a source absent from both anchors that stays fatal.

Two things the implementation corrected against the original task list. Anchoring on the main checkout root broke every nested fixture under @examples/, so the anchor maps the primary's relative position instead. Threading the anchor through the scan and the reporter proved unnecessary: every consumer already funnels through `config.ResolveGlobalPath`, so a memoized lookup behind one seam replaced tasks 4 and 5 as originally written.

### Phase 2 — session-following root (#31) — done

10. [x] Add `sessionRootProvider` in @internal/mcp/root_provider.go, querying `roots/list` through `server.ClientSessionFromContext` and `SessionWithRoots`, under a 500 ms timeout and a 2 s decision cache.
11. [x] Read the client's declared `roots` capability through `SessionWithClientInfo.GetClientCapabilities` and skip the query when it is absent.
12. [x] Implement the acceptance checks of @.archcore/mcp/project-root-resolution.spec.md, and move `isPluginCachePath` to @internal/projectroot/ so the start-time resolver and the provider share one guard.
13. [x] Add `file://` parsing with percent-decoding, host rejection, and Windows drive handling.
14. [x] Change the ten tool constructors in @internal/mcp/server.go to take `tools.RootProvider`, and resolve the root at the top of each handler body.
15. [x] Add `tools.StaticRoot` so every per-tool unit test is a one-token change rather than a rewrite.
16. [x] Add `baseDir` to `HostWiringFunc` in @internal/mcp/tools/install_host_config.go and drop the closure in @cmd/host_wiring.go.
17. [x] Add `WithPinnedRoot` and `WithRootWarnings`, and pin the root in @cmd/mcp.go when `--project` or `ARCHCORE_PROJECT_ROOT` is set.
18. [x] Emit one warning line per distinct refusal reason, with no absolute path.
19. [x] Add a roots-capable in-process client in @internal/mcp/integration/roots_test.go, built with both `transport.WithRootsHandler` and `client.WithRootsHandler`, and drive every conformance case the spec lists.
20. [x] Correct the "one MCP server process serves one primary" comment in @internal/mcp/tools/manifest_store.go.
21. [x] Confirm @cmd/project_root_flag_test.go still holds: `resolveProjectRoot` keeps its shape, and the provider sits above it.
22. [x] Confirm no invalidation is needed for the main-checkout memo of @internal/config/globals.go: it is keyed by project root, and a root's anchor cannot change while the process runs.

### Phase 3 — documentation — done

23. [x] Set @.archcore/mcp/project-root-resolution.spec.md, @.archcore/mcp/session-following-project-root.adr.md, and @.archcore/globals/relative-globals-resolve-from-main-checkout.adr.md to `accepted`.
24. [x] Correct the root-resolution paragraph in @.archcore/integrations/supported-ai-agents.doc.md, add the per-host `roots` notes, and relate @.archcore/cli-ui/building-the-cli.doc.md to the spec rather than restating it.

## Acceptance Criteria

1. [x] `archcore mcp` started in a worktree of this repository serves 108 local and 43 global documents. Measured: `list_documents` returned `by_source: {"archcore": 43, "local": 108}`.
2. [x] `archcore status` in that worktree reports zero issues and exits zero.
3. [x] A session that enters a worktree mid-session has its next `create_document` write into the worktree. Measured against the built binary: the server started in the main checkout, a probe client reported a linked worktree over `roots/list`, and the document landed in the worktree.
4. [x] A candidate root whose declared global source does not resolve is refused, the server keeps the previous root, and one warning line names the source id.
5. [x] `init_project` still works for a start-time root that holds no `.archcore/`.
6. [x] A host that declares no `roots` capability produces no `roots/list` request.
7. [x] `go test ./...`, `go vet ./...`, and `golangci-lint run` pass. Removing the anchor fails the two worktree tests with the issue's own error text; returning the current root unconditionally fails five of the roots tests.

## Dependencies

- Phase 2 depended on phase 1 for usefulness, not for safety: the acceptance gate keeps phase 2 correct on its own.
- `github.com/mark3labs/mcp-go` v0.49.0 already carried `RequestRoots`, `SessionWithRoots`, `ClientSessionFromContext`, and `stdioSession.ListRoots`. No dependency upgrade was needed.
- @.archcore/integrations/host-cwd-misrouting.adr.md documents the three properties every plugin-cache fragment must keep; they moved intact into @internal/projectroot/.
- Out of scope: refreshing the `initialize` instructions after a re-root, which the protocol does not allow; and global source distribution, which @.archcore/globals/global-sources.spec.md excludes.
