---
title: "Never Expose Absolute Filesystem Paths in MCP Tool Error Messages"
status: accepted
tags:
  - "mcp"
---

## Rule

Never include absolute filesystem paths in MCP tool error messages. All paths returned to MCP clients must be relative to the project root or the `.archcore/` directory.

MCP tool responses are consumed by external AI agents. Leaking server-side absolute paths (e.g., `/Users/dev/projects/foo/.archcore/auth/`) exposes internal directory structure and usernames to the client.

## Rationale

- MCP tools run locally but their responses are sent to LLM clients that may log, display, or relay the content.
- Absolute paths reveal system usernames, directory layout, and OS details — unnecessary information disclosure.
- Relative paths are sufficient for the client to understand the context and for the user to locate files.

## Examples

### Bad

```go
dir := filepath.Join(baseDir, ".archcore", directory)
if err := os.MkdirAll(dir, 0o755); err != nil {
    // Leaks absolute path like "/Users/dev/project/.archcore/auth"
    return errorResult(fmt.Sprintf("creating directory %s: %v", dir, err)), nil
}
```

### Good

```go
dir := filepath.Join(baseDir, ".archcore", directory)
if err := os.MkdirAll(dir, 0o755); err != nil {
    // Only shows the relative directory segment
    return errorResult(fmt.Sprintf("creating directory %q: %v", directory, err)), nil
}
```

For OS errors that embed the full path in their message, wrap them with a clean relative-path message:

```go
if err := os.WriteFile(outputFile, data, 0o644); err != nil {
    return errorResult(fmt.Sprintf("writing %s: failed to write file", relPath)), nil
}
```

## Enforcement

- Code review: check all `errorResult()` and `mcp.NewToolResultError()` calls in `internal/mcp/tools/` for path content.
- Grep for `baseDir` appearing inside error format strings in MCP tool handlers.