---
title: "The MCP Server Root Follows the Session Through roots/list, Behind an Acceptance Gate"
status: accepted
tags:
  - "cli"
  - "integrations"
  - "mcp"
---

## Context

`resolveProjectRoot` picks the project root once, at process start, from `--project`, then `ARCHCORE_PROJECT_ROOT`, then `os.Getwd()` (@cmd/mcp_root.go). Every host wiring except Cursor spawns `archcore mcp` with no root argument, so the root is the host process working directory at spawn time. Before this decision `baseDir` was captured in ten tool handler closures at construction time (@internal/mcp/server.go).

When a session enters a git worktree mid-session, the host process working directory does not change and the MCP server is not restarted. Every MCP tool call kept reading and writing the checkout the session started from. `CLAUDE_PROJECT_DIR` carries the same start-time value.

A probe session on Claude Code 2.1.258 measured the client side. The client declares `"roots": {"listChanged": true}`, and `roots/list` returns the worktree after an `EnterWorktree` call while the server's own working directory stays behind. No `notifications/roots/list_changed` arrived at any point in that session, including across the switch that changed the result.

`github.com/mark3labs/mcp-go` v0.49.0 carries the needed API: `MCPServer.RequestRoots`, the `SessionWithRoots` interface, `ClientSessionFromContext`, and `stdioSession.ListRoots`. `handleInitialize` records the client's declared capabilities on the session, so the server can tell whether a host supports roots at all. `stdioSession.ListRoots` has no internal timeout — the caller's context is the only bound — and the tool worker pool holds five workers.

Three accepted documents constrain the change. @.archcore/integrations/host-cwd-misrouting.adr.md treats an implicitly sourced root as untrusted and rejects a plugin install cache. @.archcore/mcp/mcp-server-starts-without-archcore-dir.adr.md guarantees that the server starts for a root with no `.archcore/` and exposes `init_project`. @.archcore/globals/global-sources.spec.md §6 makes a broken global fatal to the scan while the write tools stay usable, so a root whose globals do not resolve produces a state where `create_document` succeeds and `list_documents` fails.

## Decision

The captured `baseDir` string is replaced by a root provider evaluated per tool call.

1. `--project` and `ARCHCORE_PROJECT_ROOT` pin the root. A pinned server never re-roots; Cursor already pins with `--project ${workspaceFolder}` (@internal/agents/mcp_helpers.go).
2. Otherwise the provider queries `roots/list` under its own timeout and caches the answer. IF the client declared no `roots` capability, THEN the provider skips the round trip and serves the start-time root.
3. A candidate root is accepted only when it passes every check: the `file://` URI parses to an absolute path, the path exists and is a directory, `IsPluginCachePath` does not match it, it contains `.archcore/`, and `docs.InspectGlobals` reports no fatal state for it.
4. IF exactly one candidate passes, THEN the provider adopts it. IF zero or more than one candidate passes, THEN the provider keeps the current root.
5. IF a candidate fails a check, THEN the provider keeps the current root and writes one line to stderr naming the reason. It writes that line once per distinct reason, not once per tool call.
6. The start-time root stays valid without `.archcore/`, so `init_project` keeps working. The containment requirement in check 3 governs only a move away from the start-time root.
7. On a timeout, a transport error, or an absent client session, the provider serves the current root.

The provider is the single place that treats a reported root as untrusted, on the same footing as `os.Getwd()`. The clause-level contract is @.archcore/mcp/project-root-resolution.spec.md.

## Alternatives

- **Accept any candidate that contains `.archcore/`, without the globals check.** Rejected: in a repository with a relative `globals` path it produces the measured half-broken state — `get_document`, the write tools, and the relation tools succeed while `list_documents` and `search_documents` fail for the whole corpus, so the agent cannot see what it is duplicating.
- **Drive re-rooting from `notifications/roots/list_changed`.** Rejected: the notification did not arrive in the probe session even though the client declares `listChanged: true`. A handler may be added later as a cache invalidator, but correctness may not depend on it.
- **Add a `project_root` parameter to every tool.** Rejected here and recorded in @.archcore/cli/multi-project-mcp-access.idea.md: it moves the choice to the model on every call and changes the wire contract of ten tools.
- **Restart the server on a root change.** Rejected: the server does not own its process lifecycle; the host spawns it.
- **Walk up from cwd for a `.archcore/` directory.** Rejected in @.archcore/integrations/host-cwd-misrouting.adr.md — a wrong root that looks right is worse than a refusal, and cwd does not move on a worktree switch anyway.
- **Keep `HandleX(baseDir string)` alongside a provider-taking variant.** Rejected: two constructors per tool for one behavior. The handlers take `tools.RootProvider`, and `tools.StaticRoot` carries the pinned and unit-test cases.

## Consequences

- Eleven handler constructors take `tools.RootProvider` instead of a string and resolve the root at the top of the handler body. `tools.StaticRoot` keeps every per-tool unit test a one-token change.
- `HostWiringFunc` gains `baseDir` as its first parameter and `hostWiringExecutor` loses its closure over the root (@cmd/host_wiring.go), so `install_host_config` wires the project the session is on now.
- `isPluginCachePath` moved from `package cmd` to @internal/projectroot/, which the start-time resolver and the provider both consult. The three fragment properties @.archcore/integrations/host-cwd-misrouting.adr.md names move with it.
- The `initialize` instructions cannot be refreshed. `WithInstructions` is applied once at construction (@internal/mcp/server.go) and MCP defines no update notification, so the language directive and the global source list stay derived from the start-time root. The `coverage` map of `search_documents` remains the per-call source of truth for which sources actually answered.
- Process-global state keyed by project root stays correct: the scan cache keys by absolute file path, the manifest store and the main-checkout memo key by root. The "one MCP server process serves one primary" comment in @internal/mcp/tools/manifest_store.go was corrected rather than the code.
- A cached decision means a switch is followed by the call after next at the latest, not by the very next call. The window is the cache lifetime the specification states.
- The in-process integration harness answers `roots/list` through a client built with both `transport.WithRootsHandler` and `client.WithRootsHandler`: the first registers the session the server reaches the client through, the second is what makes `Initialize` declare the capability. `client.NewInProcessClient` registers no session at all, so before this the new path was unreachable from a test.
- Measured end to end against the built binary: the server started in the main checkout, a probe client reported a linked worktree over `roots/list`, and the next `create_document` wrote into the worktree, not into the checkout the process started in.
