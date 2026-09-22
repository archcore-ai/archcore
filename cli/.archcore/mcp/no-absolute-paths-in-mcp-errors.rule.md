---
title: "Never Expose Absolute Filesystem Paths in MCP Tool Error Messages"
status: accepted
tags:
  - "mcp"
---

## Rule

1. An MCP tool MUST NOT include an absolute filesystem path in an error message.
2. An MCP tool MUST express every path it returns relative to the project root or to the `.archcore/` directory.
3. IF an operating-system error embeds an absolute path in its own message, THEN the MCP tool MUST replace that message with one that names only the relative path.

## Rationale

MCP tool responses reach external AI agents, which may log, display, or relay them.

- An absolute path such as `/Users/dev/projects/foo/.archcore/auth/` discloses the system username, the directory layout, and the operating system.
- A relative path carries everything the client needs to locate the file and to explain the failure.

## Examples

Non-normative examples.

### Bad

```go
dir := filepath.Join(baseDir, ".archcore", directory)
if err := os.MkdirAll(dir, 0o755); err != nil {
    // Leaks an absolute path such as "/Users/dev/project/.archcore/auth"
    return errorResult(fmt.Sprintf("creating directory %s: %v", dir, err)), nil
}
```

### Good

```go
dir := filepath.Join(baseDir, ".archcore", directory)
if err := os.MkdirAll(dir, 0o755); err != nil {
    // Shows only the relative directory segment
    return errorResult(fmt.Sprintf("creating directory %q: %v", directory, err)), nil
}
```

```go
// Operating-system error replaced by a clean relative-path message
if err := os.WriteFile(outputFile, data, 0o644); err != nil {
    return errorResult(fmt.Sprintf("writing %s: failed to write file", relPath)), nil
}
```

## Enforcement

- Code review: the reviewer checks every `errorResult()` and `mcp.NewToolResultError()` call in `@internal/mcp/tools/` for path content.
- Code review: the reviewer greps for `baseDir` inside error format strings in the MCP tool handlers.
