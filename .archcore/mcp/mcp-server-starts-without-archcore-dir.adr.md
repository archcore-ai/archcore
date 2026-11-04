---
title: "MCP Server Starts Without .archcore/ and Exposes init_project Tool"
status: accepted
tags:
  - "component:cli"
  - "golang"
  - "mcp"
---

## Context

`archcore mcp` used to refuse to start when `.archcore/` did not exist in the working directory. An agent configured to launch the server on an uninitialized project then crashed on stdio before any tool call, so it could not bootstrap the knowledge base. A user who installed the plugin globally had to drop to a terminal and run `archcore init` in every fresh repo, although the agent knows the project context and could initialize it in-session.

## Decision

1. **`archcore mcp` starts unconditionally for the default (cwd) project root.** The `.archcore/` existence check in `cmd/mcp.go` is removed. The server always boots on stdio and prints a hint in the startup banner when the directory is missing.
2. **Add an `init_project` MCP tool** (`internal/mcp/tools/init_project.go`). It creates `.archcore/` and writes `settings.json` from within the MCP server itself. It accepts `language`, `sync_mode`, and `archcore_url` and is idempotent — calling it on an already-initialized project returns the existing settings without overwriting them.
3. **Server instructions document the bootstrap flow.** The system prompt embedded in `internal/mcp/server.go` (`mcpServerInstructions`) explicitly tells the agent: "if `list_documents` returns empty AND the user wants to create documents, call `init_project` once, then proceed."
4. **Other tools continue to assume `.archcore/` exists.** `create_document`, `list_documents`, `update_document`, etc. are not changed to auto-init. The only tool safe to call on an uninitialized project is `init_project`.

## Alternatives Considered

- **Auto-init on any mutating tool call.** Rejected: makes the init contract implicit and fragile. If the user has not yet decided on sync mode, an auto-init would silently pick `none`, which is a product decision that should be explicit.
- **Keep the shell-only `archcore init` path and fail the MCP server loudly.** Rejected: defeats the whole point of MCP-first onboarding. The agent is exactly the right place to gather initialization preferences from the user conversationally.
- **Expose `archcore init` as a shell-out from the MCP server.** Rejected: couples the MCP server to subprocess execution and an installed CLI binary. `init_project` reuses the in-process `internal/config` helpers (`NewNoneSettings`, `InitDir`, `Save`) — no new code paths to maintain.

## Consequences

Positive:

- Zero-shell onboarding: installing the Archcore MCP config into an agent is enough to start using it in any repo.
- The server can be preconfigured in an agent plugin that targets *every* project by default, without breaking uninitialized projects.
- The init flow becomes conversational — the agent can ask the user about sync mode and language before calling the tool.

Negative / constraints:

- The MCP server now has a "degraded" mode (running, but most tools will fail). The startup banner's hint compensates, but agents that ignore system prompts may attempt `create_document` before `init_project` and see confusing errors. Mitigated by the explicit instruction block in `mcpServerInstructions`.
- `init_project` must stay idempotent. A regression that clobbers existing `settings.json` on the second call would destroy user configuration. Covered by `TestHandleInitProject_Idempotent`.
- Any future MCP tool that *also* safely works on an uninitialized project must be documented in the server instructions alongside `init_project`. Today that list is just `init_project`.

### Scope of the "unconditional start" property

The unconditional-start guarantee applies to the **default project root** (cwd) and to the case where `.archcore/` is missing inside an existing project directory. It does **not** override the project-root resolution layer:

- When `--project <path>` is passed to `archcore mcp`, or `ARCHCORE_PROJECT_ROOT` is set, the resolver (`cmd/mcp_root.go`) requires the path to exist and to be a directory. A missing or non-directory path causes the command to exit before the server starts.
- This is intentional: if the user explicitly names a project root, silently falling back to cwd would be more surprising than failing fast. `.archcore/` *inside* the resolved root may still be absent — the server still starts in that case and exposes `init_project`.

In short: the server tolerates a missing `.archcore/`, but does not tolerate a missing project directory the user explicitly named.
