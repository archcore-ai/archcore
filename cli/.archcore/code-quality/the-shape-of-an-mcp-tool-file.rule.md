---
title: "The Shape of an MCP Tool File"
status: accepted
tags:
  - "code-quality"
  - "golang"
  - "mcp"
---

## Rule

Every file under `internal/mcp/tools/` that adds a tool follows one shape. All eleven tools already
do.

1. A tool file MUST export exactly two symbols: `New<Name>Tool() mcp.Tool` and
   `Handle<Name>(root RootProvider) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)`.
2. The handler MUST close over a `RootProvider`, never over a `baseDir` string.
3. The returned closure MUST resolve the root by calling `root.Root(ctx)`, not at construction time.
4. A handler needing work owned by the `cmd` layer MUST take it as a function type declared in
   `internal/mcp/tools`.
5. That tool MUST register through a `ServerOption`, so a server built without the option does not
   expose it.
6. A refusal the user caused MUST be returned as `errorResult(...)` with a nil Go error.
7. A Go error MUST be returned only when the failure is the process's own, such as marshalling the
   result.
8. An operating-system error reaching a tool result MUST pass through `sanitizeError`.
9. The developer MUST add the tool name to `archcoreMCPTools` in `@cmd/hook_payload.go`.

## Rationale

Clause 2 is the one that bites: a handler that captures `baseDir` at construction pins the server to
the root it started in, and a session that moves into a git worktree then reads the wrong project
without any error. Clause 5 keeps `internal/mcp/tools` from importing `cmd`, which is the import cycle
the executor seam exists to prevent. Clauses 6 and 7 keep a user mistake off the JSON-RPC error
channel, where a host renders it as a server fault.

## Examples

**Good** — the pair, with the root resolved inside the closure.

> `NewGetDocumentTool()` at @internal/mcp/tools/get_document.go:16 and
> `HandleGetDocument(root RootProvider)` at @internal/mcp/tools/get_document.go:38.

**Good** — clause 4 and 5. `HandleInstallHostConfig(root RootProvider, wire HostWiringFunc)` at
@internal/mcp/tools/install_host_config.go:55 takes the cmd-layer work as a function type;
@cmd/mcp.go supplies it through `WithHostWiring`.

**Bad** — a single constructor returning `(mcp.Tool, server.ToolHandlerFunc)` and closing over a
`baseDir`. No tool has this shape. It does not register against `NewServer`, and it breaks
session-following roots.

## Enforcement

- `TestArchcoreMCPTools_MatchesTheServer` holds clause 9.
- The Go compiler holds clause 5, because `cmd` imports `internal/mcp/tools`.
- Clauses 1 to 4 and 6 to 8 carry no automated check. Review holds them.
- [assumption] A test that reflects over the package and asserts the `New*Tool` / `Handle*` pairing
  would hold clause 1 mechanically.
