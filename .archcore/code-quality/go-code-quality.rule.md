---
title: "Go Code Quality Conventions for Archcore CLI"
status: accepted
tags:
  - "code-quality"
  - "code-review"
  - "golang"
---

## Rule

### Error Handling

- Wrap errors with context: `fmt.Errorf("context: %w", err)` — never return bare errors from public functions.
- Constant messages use `errors.New`, not `fmt.Errorf` without format verbs.
- Soft failures for non-critical operations: print the error and continue instead of returning it. Example: server unreachable during `archcore init` prints a warning but completes initialization.
- Failures a command exists to perform are NOT soft: `archcore update` returns an error (non-zero exit) when the update fails, so scripts can detect it.
- A command that has already printed its own failure summary returns `cmd.ErrAlreadyReported`; `main` then only sets the exit code (no double printing).
- MCP tools return errors as `mcp.NewToolResultError(msg)`, not as Go errors — the MCP protocol expects error results, not handler failures. OS-level errors are sanitized through `sanitizeError` before reaching a tool result (see no-absolute-paths-in-mcp-errors.rule).
- Validate-then-act: check all preconditions early and return errors before performing any side effects.
- Bound error response reads: use `io.ReadAll(io.LimitReader(...))` when reading error bodies from HTTP responses (e.g., 512 bytes max).

### Naming

- Single-letter receiver names: `c` for Client, `s` for Settings, `m` for Manifest.
- Factory functions: `NewXxx()` returns a pointer or interface value.
- Private helpers: lowercase, descriptive verbs — `validateTags`, `stripFrontmatter`, `buildDocumentFile`, `normalizeRelPath`.
- Flag structs: separate struct type for cobra flag groups — `syncFlags`, not inline fields.
- Typed string enums: `type DocumentType string` with a `const` block. Validation via map lookup, not switch-default.
- Regex patterns as package-level `var` compiled with `regexp.MustCompile`.

### Import Organization

Three groups separated by blank lines, in this order:

1. Standard library (`fmt`, `os`, `path/filepath`, etc.)
2. Local packages (`archcore-cli/*`)
3. Third-party packages (`github.com/*`, `gopkg.in/*`)

Alias imports only when a name collision exists — e.g., `archsync` for `internal/sync` to avoid collision with stdlib `sync`.

(History: an earlier version of this rule listed third-party before local while its own example showed the opposite, and the codebase drifted both ways. The July 2026 audit standardized on the order above — the dominant actual pattern — and aligned the outlier files.)

### Output and Display

- All styled terminal output goes through `internal/display` — never use inline ANSI escape codes.
- Prefix symbols are consistent: `CheckLine` (success), `FailLine` (failure), `WarnLine` (warning), `HintLine` (hints/details).
- stderr for initialization and setup messages (e.g., MCP server startup) and for error output from `main`. stdout for normal command output.
- Root command sets `SilenceErrors: true` and `SilenceUsage: true` — custom error formatting via `FormatExecuteError()`.
- No logging framework. All output uses `fmt` and `display` helpers.

### Cobra Commands

- Always use `RunE` (error-returning), never `Run`.
- Factory function pattern: `newXxxCmd()` returns `*cobra.Command` with full `RunE` closure.
- Pass version as an argument to command factories — not as a package-level global. This includes the MCP server version (`mcpserver.NewServer(baseDir, version)`).
- Propagate `cmd.Context()` to all downstream functions for cancellation support, including `exec.CommandContext` for subprocesses.
- Register flags inside the command factory function, not globally.

### File I/O and Path Safety

- Validate all user-provided paths against directory traversal: reject `..` segments and absolute paths.
- Use `filepath.Clean()` then check prefix is still within `.archcore/`.
- `filepath.ToSlash()` for consistent cross-platform path representation in stored data.
- File permissions: `0o644` for files, `0o755` for directories.
- Atomic writes for critical state files: write to a `.tmp` file, then `os.Rename`. Shared helpers: `jsonfile.WriteAtomic` (config-style files), `writeFileAtomic` in `internal/mcp/tools` (documents), `sync.SaveManifest` (manifest). The self-update binary is additionally fsynced before rename.
- User-owned config files (agent settings, MCP configs) are surgically edited via `internal/jsonfile`: unknown fields and key order round-trip as opaque `json.RawMessage`; the file is only written when something actually changed.
- Backup corrupted config files before overwriting: save as `.bak` before replacing with valid content, and abort if the backup itself cannot be written (see backup-invalid-configs.adr).

### Data Structures

- `strings.Builder` for multi-part string construction — never `+` concatenation in loops.
- `slices.Clone()` before sorting to avoid mutating the caller's data.
- `slices.Sort()` + `slices.Compact()` for deduplication. Sort map keys before iterating when output order is user-visible.
- Maps for O(1) validation lookups: `validRelationTypes map[RelationType]bool`, `allowedFields map[string]map[string]bool`.
- Pre-allocate slices when size is known: `make([]T, len(source))`.
- Closed-set wire vocabularies are named constants (e.g. `sourceKindLocal`, `matchKindContent`), never scattered string literals.

### JSON Serialization

- Custom `MarshalJSON`/`UnmarshalJSON` when serialization depends on runtime state (e.g., which fields are allowed varies by sync type).
- Standard `json:"field,omitempty"` struct tags for simple data types.
- MCP tool results: build a `map[string]any` or a small result struct, marshal to JSON string, return as `mcp.TextContent`.

### Concurrency

- No goroutines in business logic — the CLI uses a synchronous, single-threaded architecture.
- The MCP server is the exception by necessity: mcp-go serves tools/call on a worker pool, so shared mutable state in `internal/mcp/tools` (manifest store, scan cache) is mutex-protected, and manifest mutations are serialized through `manifestStore.mutate` (clone-and-swap).
- Context propagation: `cmd.Context()` flows through the call chain to HTTP calls via `http.NewRequestWithContext`.
- `signal.NotifyContext()` at the top level (`main.go`) for graceful shutdown on SIGINT/SIGTERM.

### Dependencies

- No assertion libraries (testify, etc.) — use stdlib `testing` only.
- No logging frameworks (slog, zerolog, etc.) — use `fmt` and display helpers.
- `charmbracelet/huh` for interactive CLI forms; `charmbracelet/lipgloss` for styled terminal output.
- `spf13/cobra` for CLI command framework.
- `gopkg.in/yaml.v3` for YAML frontmatter parsing.
- `mark3labs/mcp-go` for MCP protocol implementation.
- `wk8/go-ordered-map/v2` for order-preserving JSON object surgery (`internal/jsonfile`).

## Rationale

These conventions emerged organically across the codebase and are consistently applied. Documenting them serves two purposes:

1. **Reduce review overhead**: AI coding agents and human reviewers can check new code against explicit rules instead of pattern-matching against the full codebase each time.
2. **Prevent drift**: As the codebase grows, explicit conventions prevent gradual inconsistency.

Every rule above reflects an actual pattern verified in the current source — none are aspirational.

## Examples

### Good

```go
// Error wrapping with context
settings, err := config.Load(baseDir)
if err != nil {
    return fmt.Errorf("invalid settings: %w", err)
}

// Soft failure for non-critical operation
if err := client.CheckHealth(ctx); err != nil {
    fmt.Fprintln(os.Stderr, display.WarnLine("Server unreachable"))
} else {
    fmt.Fprintln(os.Stderr, display.CheckLine("Server reachable"))
}

// Typed enum with validation map
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

// Slice clone before sort
out := slices.Clone(tags)
slices.Sort(out)
out = slices.Compact(out)
```

### Bad

```go
// Bare error return — no context for caller
return err

// Inline ANSI codes instead of display package
fmt.Println("\033[31mError!\033[0m")

// Global variable for version
var Version string // set by ldflags

// String concatenation in loop
result := ""
for _, s := range items {
    result += s + ", "
}

// Sorting caller's slice directly
slices.Sort(tags) // mutates caller's data

// Constant message through fmt.Errorf
return fmt.Errorf("server is not ready") // use errors.New
```

## Enforcement

- The `review-go` skill checks changed Go files against these conventions during code review.
- New code must follow these patterns. Deviations require an explicit comment explaining why.
- Run `go vet ./...` and `go build ./...` as baseline checks — they catch a subset of these rules.