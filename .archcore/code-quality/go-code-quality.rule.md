---
title: "Go Code Quality Conventions for Archcore CLI"
status: accepted
tags:
  - "code-quality"
  - "code-review"
  - "golang"
---

## Rule

The developer is the obligated actor for every convention below. Each item states a MUST unless it names `SHOULD` or `MAY`. IF a change deviates from a convention, THEN the developer MUST add a comment that names the reason.

### Error handling

- Wrap errors with context: `fmt.Errorf("context: %w", err)`. Never return a bare error from a public function.
- Use `errors.New` for a constant message, not `fmt.Errorf` without format verbs.
- Treat a non-critical operation as a soft failure: print the error and continue instead of returning it. Example: an unreachable server during `archcore init` prints a warning and initialization completes.
- Treat the failure of the operation a command exists to perform as hard: `archcore update` returns an error with a non-zero exit code when the update fails, so a script can detect it.
- Return `cmd.ErrAlreadyReported` from a command that already printed its own failure summary. `main` then only sets the exit code and prints nothing more.
- Return MCP tool errors as `mcp.NewToolResultError(msg)`, not as Go errors — the MCP protocol expects error results, not handler failures. Sanitize operating-system errors through `sanitizeError` before they reach a tool result; the related rule on MCP error messages states the path-disclosure constraint.
- Validate then act: check every precondition and return the error before any side effect.
- Bound error-response reads: use `io.ReadAll(io.LimitReader(...))` when reading an error body from an HTTP response, for example 512 bytes maximum.

### Naming

- Use single-letter receiver names: `c` for Client, `s` for Settings, `m` for Manifest.
- Name factory functions `NewXxx()`, returning a pointer or an interface value.
- Name private helpers in lowercase with descriptive verbs: `validateTags`, `stripFrontmatter`, `buildDocumentFile`, `normalizeRelPath`.
- Group cobra flags in a separate struct type, such as `syncFlags`, not in inline fields.
- Declare typed string enums as `type DocumentType string` with a `const` block, and validate through a map lookup rather than a switch default.
- Compile regex patterns into package-level `var` values with `regexp.MustCompile`.

### Import organization

Three groups separated by blank lines, in this order:

1. Standard library (`fmt`, `os`, `path/filepath`).
2. Local packages (`archcore-cli/*`).
3. Third-party packages (`github.com/*`, `gopkg.in/*`).

Alias an import only when a name collision exists, for example `archsync` for `internal/sync` against the standard library `sync`.

History: an earlier version of this rule listed third-party before local while its own example showed the opposite, and the codebase drifted both ways. The July 2026 audit standardized on the order above — the dominant actual pattern — and aligned the outlier files.

### Output and display

- Route all styled terminal output through `internal/display`. Never write inline ANSI escape codes.
- Keep prefix symbols consistent: `CheckLine` (success), `FailLine` (failure), `WarnLine` (warning), `HintLine` (hints and details).
- Write initialization and setup messages, such as MCP server startup, and error output from `main` to stderr. Write normal command output to stdout.
- Set `SilenceErrors: true` and `SilenceUsage: true` on the root command, and format errors through `FormatExecuteError()`.
- Use no logging framework. All output goes through `fmt` and the display helpers.

### Cobra commands

- Use `RunE`, never `Run`.
- Follow the factory pattern: `newXxxCmd()` returns a `*cobra.Command` with the full `RunE` closure.
- Pass the version to a command factory as an argument, not through a package-level global. This includes the MCP server version: `mcpserver.NewServer(baseDir, version)`.
- Propagate `cmd.Context()` to every downstream function for cancellation, including `exec.CommandContext` for a subprocess.
- Register flags inside the command factory function, not globally.

### File I/O and path safety

- Validate every user-provided path against directory traversal: reject `..` segments and absolute paths.
- Apply `filepath.Clean()`, then check that the prefix is still inside `.archcore/`.
- Apply `filepath.ToSlash()` for a consistent cross-platform path representation in stored data.
- Set file permissions to `0o644` and directory permissions to `0o755`.
- Write critical state files atomically: write a `.tmp` file, then `os.Rename`. Shared helpers: `jsonfile.WriteAtomic` for config-style files, `writeFileAtomic` in `internal/mcp/tools` for documents, `sync.SaveManifest` for the manifest. The self-update binary is additionally fsynced before the rename.
- Edit user-owned config files (agent settings, MCP configs) surgically through `internal/jsonfile`: unknown fields and key order round-trip as opaque `json.RawMessage`, and the file is written only when something changed.
- Back up a corrupted config file before overwriting it: save a `.bak` copy before replacing it with valid content, and abort when the backup itself cannot be written. The related ADR records that decision.

### Data structures

- Use `strings.Builder` for multi-part string construction. Never concatenate with `+` in a loop.
- Call `slices.Clone()` before sorting, so the caller's data stays unmutated.
- Deduplicate with `slices.Sort()` followed by `slices.Compact()`. Sort map keys before iterating when the output order is user-visible.
- Use maps for O(1) validation lookups: `validRelationTypes map[RelationType]bool`, `allowedFields map[string]map[string]bool`.
- Pre-allocate a slice when the size is known: `make([]T, len(source))`.
- Declare closed-set wire vocabularies as named constants, such as `sourceKindLocal` and `matchKindContent`, never as scattered string literals.

### JSON serialization

- Implement custom `MarshalJSON` and `UnmarshalJSON` when serialization depends on runtime state, for example when the allowed fields vary by sync type.
- Use standard `json:"field,omitempty"` struct tags for simple data types.
- Build an MCP tool result as a `map[string]any` or a small result struct, marshal it to a JSON string, and return it as `mcp.TextContent`.

### Concurrency

- Use no goroutines in business logic. The CLI has a synchronous, single-threaded architecture.
- The MCP server is the exception by necessity: mcp-go serves `tools/call` on a worker pool, so shared mutable state in `internal/mcp/tools` (manifest store, scan cache) is mutex-protected, and manifest mutations are serialized through `manifestStore.mutate` (clone-and-swap).
- Propagate context: `cmd.Context()` flows through the call chain to HTTP calls via `http.NewRequestWithContext`.
- Call `signal.NotifyContext()` at the top level (`@main.go`) for graceful shutdown on SIGINT and SIGTERM.

### Dependencies

- Use no assertion library such as testify. The standard library `testing` package covers the need.
- Use no logging framework such as slog or zerolog. `fmt` and the display helpers cover the need.
- `charmbracelet/huh` for interactive CLI forms; `charmbracelet/lipgloss` for styled terminal output.
- `spf13/cobra` for the CLI command framework.
- `gopkg.in/yaml.v3` for YAML frontmatter parsing.
- `mark3labs/mcp-go` for the MCP protocol implementation.
- `wk8/go-ordered-map/v2` for order-preserving JSON object surgery in `internal/jsonfile`.

## Rationale

These conventions emerged across the codebase and are applied consistently. Writing them down serves two purposes:

1. Lower review overhead: an AI coding agent and a human reviewer can check new code against explicit conventions instead of pattern-matching the whole codebase each time.
2. Prevent drift: as the codebase grows, explicit conventions keep new code from diverging gradually.

Every item above reflects a pattern verified in the current source. None is aspirational.

## Examples

Non-normative examples.

### Good

```go
// Error wrapping with context
settings, err := config.Load(baseDir)
if err != nil {
    return fmt.Errorf("invalid settings: %w", err)
}

// Soft failure for a non-critical operation
if err := client.CheckHealth(ctx); err != nil {
    fmt.Fprintln(os.Stderr, display.WarnLine("Server unreachable"))
} else {
    fmt.Fprintln(os.Stderr, display.CheckLine("Server reachable"))
}

// Typed enum with a validation map
type RelationType string
const (
    RelRelated    RelationType = "related"
    RelImplements RelationType = "implements"
)
var validRelationTypes = map[RelationType]bool{
    RelRelated: true, RelImplements: true,
}

// Import organization: stdlib → local → third-party
import (
    "fmt"
    "os"

    "archcore-cli/internal/config"

    "github.com/spf13/cobra"
)

// Clone before sort
out := slices.Clone(tags)
slices.Sort(out)
out = slices.Compact(out)
```

### Bad

```go
// Bare error return — no context for the caller
return err

// Inline ANSI codes instead of the display package
fmt.Println("\033[31mError!\033[0m")

// Global variable for the version
var Version string // set by ldflags

// String concatenation in a loop
result := ""
for _, s := range items {
    result += s + ", "
}

// Sorting the caller's slice directly
slices.Sort(tags) // mutates the caller's data

// Constant message through fmt.Errorf
return fmt.Errorf("server is not ready") // use errors.New
```

## Enforcement

- The `review-go` skill checks changed Go files against these conventions during code review.
- New code MUST follow these conventions. A deviation MUST carry a comment that explains why.
- `go vet ./...` and `go build ./...` run as baseline checks and catch a subset of these conventions.
