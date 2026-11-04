---
title: "Never Expose Absolute Filesystem Paths in MCP Tool Error Messages"
status: accepted
tags:
  - "component:cli"
  - "mcp"
---

## Rule

1. An MCP tool MUST NOT include an absolute filesystem path in an error message.
2. An MCP tool MUST express every path it returns relative to the project root or to the `.archcore/` directory.
3. IF an operating-system error embeds an absolute path in its own message, THEN the MCP tool MUST replace that message with one that names only the relative path.
4. An MCP tool MAY return the resolved project root as an absolute path in the `project` response field, and in no other field, per @.archcore/mcp/call-root-resolution.spec.md.
5. An MCP tool MUST NOT echo the value of a `project` argument or an absolute `for_path` argument in an error message.

## Rationale

MCP tool responses reach external AI agents, which may log, display, or relay them.

- An absolute path such as `/Users/dev/projects/foo/.archcore/auth/` discloses the system username, the directory layout, and the operating system.
- A relative path carries everything the client needs to locate the file and to explain the failure.
- The `project` field is the one exception: the caller supplied or already holds that root, and needs it verbatim to address the same project on the next call. An error message never needs it.

## Examples

Bad: an error that formats the joined directory (`creating directory <baseDir>/.archcore/auth: ...`) leaks the absolute path. Good: format the relative `directory` segment, or replace an OS error that embeds the path with `writing <relPath>: failed to write file`.

## Enforcement

- Code review: the reviewer checks every `errorResult()` and `mcp.NewToolResultError()` call in `@cli/internal/mcp/tools/` for path content.
- Code review: the reviewer greps for `baseDir` inside error format strings in the MCP tool handlers.
