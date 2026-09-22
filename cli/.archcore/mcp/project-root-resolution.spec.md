---
title: "MCP Project Root Resolution Contract"
status: accepted
tags:
  - "cli"
  - "config"
  - "mcp"
---

## Purpose &amp; Scope

This specification defines how the Archcore MCP server decides which project root each MCP tool call operates on, and when that root may change during a session.

It is normative for the root provider (@internal/mcp/root_provider.go), for the acceptance checks a candidate root must pass, and for the messages a refusal emits. It supersedes nothing in @.archcore/integrations/host-cwd-misrouting.adr.md; the plugin-cache guard stays a required check. The originating decision is @.archcore/mcp/session-following-project-root.adr.md.

### Does Not Cover

- Global source declaration, resolution, and health classification — @.archcore/globals/global-sources.spec.md.
- Root resolution for short-lived CLI commands and for hook invocations, which resolve per invocation and are unaffected.
- Serving more than one writable project in one session — @.archcore/cli/multi-project-mcp-access.idea.md.

## Surface

| Element | Shape |
| ------- | ----- |
| Pinned sources | `--project` flag, `ARCHCORE_PROJECT_ROOT` environment variable |
| Implicit source | the process working directory at start |
| Session source | the client's `roots/list` reply |
| Provider contract | `tools.RootProvider` — `Root(ctx) string`, which never fails |
| Pinned implementation | `tools.StaticRoot` |
| Session implementation | `mcp.sessionRootProvider`, built by `NewServer` |
| Per-call contract | every tool handler resolves the root from the request context before its first filesystem access |
| Shared guard | `projectroot.IsPluginCachePath`, consumed by the start-time resolver and the provider |

## Normative Behavior

1. WHEN `--project` or `ARCHCORE_PROJECT_ROOT` is non-empty, the server MUST serve that root for the process lifetime.
2. WHEN a root is pinned, the server MUST NOT query `roots/list`.
3. WHEN no root is pinned, the server MUST resolve the root once per tool call from the request context.
4. WHEN the client declared no `roots` capability at `initialize`, the provider MUST serve the start-time root without a query.
5. WHEN the provider queries `roots/list`, it MUST bound the call with its own timeout.
6. WHEN a query succeeds, the provider MUST cache the decision.
7. WHEN a query fails, the provider MUST hold the current root for the same cache lifetime.
8. WHILE a cached decision is current, the provider MUST NOT query again.
9. WHEN two tool calls need a root at once, the provider MUST issue at most one `roots/list` request.
10. WHEN a reported root equals the current root, the provider MUST keep it without running the acceptance checks.
11. The provider MUST accept a different candidate root only when every check in Acceptance passes.
12. WHEN exactly one candidate passes, the provider MUST adopt it as the current root.
13. WHEN no candidate passes, the provider MUST keep the current root.
14. WHEN more than one candidate passes, the provider MUST keep the current root.
15. WHEN the provider rejects a candidate, it MUST write one line naming the failed check to its warning stream.
16. The provider MUST write that line at most once per distinct reason.
17. A warning line MUST NOT contain an absolute filesystem path, per @.archcore/mcp/no-absolute-paths-in-mcp-errors.rule.md.
18. WHEN the current root changes, the provider MUST NOT emit an MCP error to the caller.

The warning stream is stderr: file descriptor 1 carries JSON-RPC frames. `WithRootWarnings` redirects it.

### Acceptance

A candidate root passes when all of these hold:

1. The reported URI parses as a `file://` URI and yields an absolute path after percent-decoding.
2. The path exists and is a directory.
3. `projectroot.IsPluginCachePath` does not match the path.
4. The path contains a `.archcore/` directory.
5. `docs.InspectGlobals` reports no fatal state for the path.

Check 4 governs a move away from the start-time root only. The start-time root MUST stay servable without `.archcore/`, so `init_project` keeps its guarantee from @.archcore/mcp/mcp-server-starts-without-archcore-dir.adr.md.

## Constraints &amp; Invariants

| Constraint | Value | Rationale |
| ---------- | ----- | --------- |
| Writable roots per call | exactly 1 | Preserves the single-primary model of @.archcore/globals/global-sources.spec.md |
| Root sources that may change mid-session | `roots/list` only | A pinned root states user intent |
| `roots/list` calls per cache lifetime | at most 1 | The transport writes outside the response mutex |
| Query timeout | 500 ms | The transport applies none of its own, and a stalled query holds one of five tool workers |
| Cache lifetime | 2 s [assumption] | A burst of calls costs one query; a worktree switch is picked up by the call after next at the latest |
| Trust level of a reported root | untrusted | Same treatment as the process working directory |

- A tool call reads and writes exactly one root; a root never changes inside one call.
- A root the provider serves has passed every acceptance check, or is the start-time root.
- The server never serves a root whose declared global sources hold a fatal state, except the start-time root, which the startup gate already validated.
- The `initialize` instructions describe the start-time root for the process lifetime.
- Process-global state that is keyed by project root — the document scan cache, the manifest store, the main-checkout memo of @.archcore/globals/global-sources.spec.md §1 — stays correct across a root change, because a key names one root.

## Failure Behavior

| Condition | Response |
| --------- | -------- |
| No client session on the context | serve the current root, say nothing |
| Client declared no `roots` capability | serve the current root, no query, say nothing |
| `roots/list` times out | serve the current root, one warning line |
| `roots/list` returns a transport error | serve the current root, one warning line |
| Candidate is not a `file://` URI, or not absolute | reject, one warning line |
| Candidate does not exist, or is not a directory | reject, one warning line |
| Candidate sits in a plugin install cache | reject, one warning line naming the cache |
| Candidate holds no `.archcore/` directory | reject, one warning line |
| Candidate's global source holds a fatal state | reject, one warning line naming the declared source id |
| Candidate's `settings.json` is unreadable | reject, one warning line |
| More than one candidate passes | keep the current root, one warning line |

## Conformance

An implementation conforms when it satisfies every clause above and the acceptance checks, and passes @internal/mcp/root_provider_test.go and @internal/mcp/integration/roots_test.go, whose cases are: no session; a session without the `roots` capability; an empty reply; a candidate without `.archcore/`; a candidate in a plugin install cache; a candidate whose globals resolve; a candidate whose globals do not resolve; several passing candidates; a query that errors; a query that never answers; a burst of five calls costing one query; a switch mid-session; and `init_project` on an uninitialized start-time root.
