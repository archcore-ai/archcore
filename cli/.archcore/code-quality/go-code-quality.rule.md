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

- An error leaving a function MUST name the operation that failed and the subject it failed on.
- Wrap when the wrapper adds information the inner error lacks: `fmt.Errorf("context: %w", err)`. Use `%w`, so a caller can still `errors.Is` or `errors.As`.
- Pass an error through unwrapped when it already names the operation and the subject. Wrapping it again stutters — `verifying checksum: checksum not found for archcore_darwin.tar.gz` says one thing twice. A thin exported wrapper over a helper that already wrapped adds nothing.
- The bare `return err` to look for is the one where the caller cannot tell what failed. `internal/api.CheckHealth` was one: its inner errors said "server returned status 403", never which request produced it.
- Use `errors.New` for a constant message, not `fmt.Errorf` without format verbs.
- Treat a non-critical operation as a soft failure: print the error and continue instead of returning it. Example: an unreachable server during `archcore init` prints a warning and initialization completes.
- Treat the failure of the operation a command exists to perform as hard: `archcore update` returns an error with a non-zero exit code when the update fails, so a script can detect it.
- Return `cmd.ErrAlreadyReported` from a command that already printed its own failure summary. `main` then only sets the exit code and prints nothing more.
- Return MCP tool errors as `mcp.NewToolResultError(msg)`, not as Go errors — the MCP protocol expects error results, not handler failures. Sanitize operating-system errors through `sanitizeError` before they reach a tool result; the related rule on MCP error messages states the path-disclosure constraint.
- Validate then act: check every precondition and return the error before any side effect.
- Bound error-response reads: use `io.ReadAll(io.LimitReader(r, limit+1))` with `limit` in a named constant, and report the overflow rather than truncating silently. See the bounded-output rule §5.
- Assign a deliberately ignored error to `_`, so the decision is visible: `_ = os.Remove(tmp)`. An unassigned call reads as an oversight, and `errcheck` treats it as one.
- Classify every read of external state as guard or advisory before choosing its error branch — the fail-open-or-fail-closed rule states the obligation.
- Return a classified sentinel, not a rendered message, from a predicate two surfaces share — the shared-guard rule states the obligation.

### Naming

- Use single-letter receiver names: `c` for Client, `s` for Settings, `m` for Manifest.
- Name factory functions `NewXxx()`, returning a pointer or an interface value.
- Name private helpers in lowercase with descriptive verbs: `validateTags`, `stripFrontmatter`, `buildDocumentFile`, `normalizeRelPath`.
- Group cobra flags in a separate struct type, such as `syncFlags`, not in inline fields.
- Declare typed string enums as `type DocumentType string` with a `const` block, and validate through a map lookup rather than a switch default.
- Compile regex patterns into package-level `var` values with `regexp.MustCompile`.

The naming rule states the full set. This section names only what a reviewer reaches for first.

### Comments

The default is no comment. `comments-are-the-exception.rule` states when one is written and how long
it may be, and lists the comments other rules still require.

### Import organization

Three groups separated by blank lines, in this order:

1. Standard library (`fmt`, `os`, `path/filepath`).
2. Local packages (`archcore-cli/*`).
3. Third-party packages (`github.com/*`, `gopkg.in/*`).

Alias an import only when a name collision exists, for example `archsync` for `internal/sync` against the standard library `sync`.

History: an earlier version of this rule listed third-party before local while its own example showed the opposite, and the codebase drifted both ways. The July 2026 audit standardized on the order above — the dominant actual pattern — and aligned the outlier files. A September 2026 audit found six test files that had drifted back and aligned them.

### Standard-library idioms

- Use `slices` and `cmp`, not `sort`. `slices.Sort`, `slices.SortFunc`, `slices.SortStableFunc` with `cmp.Compare` cover every case in this codebase.
- Test for a missing file with `errors.Is(err, fs.ErrNotExist)`, not `os.IsNotExist`. The latter does not unwrap, so it silently returns false the first time a caller wraps the error with `%w`. Name the sentinel `fs.ErrNotExist`, not the `os` alias.
- Iterate a split lazily with `strings.SplitSeq` or `strings.Lines` when the input is bounded by a limit. `strings.Split` materializes every element before the limit can refuse any of them.

### Output and display

- Route all styled terminal output through `internal/display`. Never write inline ANSI escape codes.
- Keep prefix symbols consistent: `CheckLine` (success), `FailLine` (failure), `WarnLine` (warning), `HintLine` (hints and details).
- Write initialization and setup messages, such as MCP server startup, and error output from `main` to stderr. Write normal command output to stdout.
- Set `SilenceErrors: true` and `SilenceUsage: true` on the root command, and format errors through `FormatExecuteError()`.
- Use no logging framework. All output goes through `fmt` and the display helpers.
- Choose the stdout handle by what the command's output is, not by preference:
  - A command whose output is **one self-contained report a caller may redirect** prints through
    `cmd.OutOrStdout()`. Every line of the run belongs to that report, so a caller redirecting the
    command must see all of them or none. `plugin`, `update`, and the root command's banner are this
    branch — the root's banner shares its output with `cmd.Usage()`, which writes to the cobra writer
    and cannot be split from it.
  - A command that **interleaves its lines with helpers writing to the process handle** prints through
    `os.Stdout`. `init`, `doctor`, `hooks`, and `status` are this branch; routing them through the
    cobra writer would capture only half the run, and the command tests redirect the process handles.
- Build the report as data and render it in a separate function — the boundary-rendering rule states the obligation. `@cmd/status_report.go` records why the resolution has to happen at call time.

### Cobra commands

- Use `RunE`, never `Run`.
- Follow the factory pattern: `newXxxCmd()` returns a `*cobra.Command` with the full `RunE` closure.
- Pass the version to a command factory as an argument, not through a package-level global. This includes the MCP server version: `mcpserver.NewServer(baseDir, version)`.
- Propagate `cmd.Context()` to every downstream function for cancellation, including `exec.CommandContext` for a subprocess.
- Register flags inside the command factory function, not globally.
- Declare `Args` on every command. A command taking no positional argument uses `cobra.NoArgs`; a hook leaf uses `cobra.ArbitraryArgs` and carries a comment saying why, because on that path a non-zero exit is a verdict rather than a complaint.
- Resolve the project root through `resolveProjectRoot` and register a `--project` flag, never `os.Getwd()` directly. `TestCommands_OfferProjectFlag` walks the tree and fails on a command that does neither.

### File I/O and path safety

- Validate every user-provided path against directory traversal: reject `..` segments and absolute paths.
- Apply `filepath.Clean()`, then check that the prefix is still inside `.archcore/`.
- Apply `filepath.ToSlash()` for a consistent cross-platform path representation in stored data.
- Set file permissions to `0o644` and directory permissions to `0o755`.
- Write critical state atomically through a shared helper, never through a fresh temp-file-plus-rename sequence inline. The atomic-write rule owns the choice between the three and states the exceptions.
- Edit user-owned config files (agent settings, MCP configs) surgically through `internal/jsonfile`: unknown fields and key order round-trip as opaque `json.RawMessage`, and the file is written only when something changed.
- Back up a corrupted config file before overwriting it: save a `.bak` copy before replacing it with valid content, and abort when the backup itself cannot be written. The related ADR records that decision.

### Data structures

- Use `strings.Builder` for multi-part string construction. Never concatenate with `+` in a loop.
- Call `slices.Clone()` before sorting, so the caller's data stays unmutated.
- Deduplicate with `slices.Sort()` followed by `slices.Compact()`. Sort map keys before iterating when the output order is user-visible.
- Use maps for O(1) validation lookups: `validRelationTypes map[RelationType]bool`, `allowedFields map[string]map[string]bool`.
- Pre-allocate a slice when the size is known: `make([]T, len(source))`.
- Declare closed-set wire vocabularies as named constants, such as `sourceKindLocal` and `matchKindContent`, never as scattered string literals. Key a map on the typed enum with the constants, not with string literals — a typo in a literal key compiles and silently takes the default.
- Bound and order anything leaving the process: a named ceiling, and a deterministic sort before the cut. The bounded-output rule states the obligation.

### JSON serialization

- Implement custom `MarshalJSON` and `UnmarshalJSON` when serialization depends on runtime state, for example when the allowed fields vary by sync type.
- Use standard `json:"field,omitempty"` struct tags for simple data types.
- Build an MCP tool result as a `map[string]any` or a small result struct, marshal it to a JSON string, and return it as `mcp.TextContent`.

### Concurrency

- Use no goroutines in business logic. The CLI has a synchronous, single-threaded architecture.
- Shared mutable state reachable from an MCP tool call MUST be mutex-protected. mcp-go serves
  `tools/call` on a worker pool, so a handler is concurrent by default. Manifest mutations go through
  `manifestStore.mutate` (clone-and-swap); the guarded state is not confined to `internal/mcp/tools`.
- An unwaited goroutine is permitted only where a spec states its non-join contract and names where
  its output goes. The one that exists is the background update started by `RunStdio`, which must
  never write to stdout — `mcp-background-update.spec`.
- `architecture/process-and-concurrency-model.spec` states the invariants and the known bounds. Read
  it before adding shared state or a goroutine; this section states the conventions only.
- Propagate context: `cmd.Context()` flows through the call chain to HTTP calls via `http.NewRequestWithContext`.
- Call `signal.NotifyContext()` at the top level (`@main.go`) for graceful shutdown on SIGINT and SIGTERM.
- Bound every subprocess and network call with an explicit timeout held in a named constant whose comment states the budget it protects. Never use `http.DefaultClient` or a context-free `exec.Command`.

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

Every item above reflects a pattern verified in the current source. None is aspirational — the September 2026 audit rewrote the three clauses that had stopped being true, on error wrapping, output routing, and concurrency, rather than leaving the code to disagree with them. The error-wrapping clause previously read "never return a bare error from a public function"; an audit of all twelve such sites found eleven were correct pass-throughs of an already-contextual error, so the clause now states the criterion the code actually follows.

## Examples

Non-normative examples.

### Good

```go
// Wrap: the inner error says "server returned status 403" and never says which
// request produced it.
if err := c.get(ctx, "/status", &result); err != nil {
    return fmt.Errorf("checking server readiness: %w", err)
}

// Pass through: findChecksum already names the operation and the file.
expected, err := findChecksum(checksums, filename)
if err != nil {
    return err
}

// Pass through: requireArchcoreDir returns a complete user-facing sentence.
if err := requireArchcoreDir(baseDir); err != nil {
    return err
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

// A named timeout whose comment states the budget it protects
// healthTimeout bounds the readiness probe, which answers from memory on a
// healthy server and must not hold a command open on an unhealthy one.
const healthTimeout = 10 * time.Second

// A bounded read that reports the overflow instead of truncating
b, _ := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes+1))

// Clone before sort
out := slices.Clone(tags)
slices.Sort(out)
out = slices.Compact(out)

// A deliberately dropped error says so
_ = os.Remove(tmp)
```

### Bad

```go
// A bare return where the caller cannot tell what failed
data, err := json.MarshalIndent(s, "", "  ")
if err != nil {
    return err // which document? which operation?
}

// Wrapping an error that already says it: "verifying checksum: checksum not
// found for archcore_darwin.tar.gz"
if err := findChecksum(checksums, filename); err != nil {
    return fmt.Errorf("verifying checksum: %w", err)
}

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

// Does not unwrap: false the first time a caller wraps this error
if os.IsNotExist(err) { ... } // use errors.Is(err, fs.ErrNotExist)

// An unnamed timeout: nothing records what budget it protects
HTTPClient: &http.Client{Timeout: 10 * time.Second}

// Reads exactly the limit, so an oversized body arrives as a valid-looking prefix
b, _ := io.ReadAll(io.LimitReader(body, 512))

// Materializes every line to read the first one
for _, line := range strings.Split(out, "\n") {
    if p, ok := strings.CutPrefix(line, "worktree "); ok {
        return p
    }
}

// A typo compiles and silently takes the default
var typePriority = map[templates.DocumentType]int{"rul": 1} // use templates.TypeRule
```

## Enforcement

- `golangci-lint run ./...` runs `errcheck`, `errname`, `exhaustive`, `revive`, `staticcheck`, and `unconvert` against these conventions. The config is `.golangci.yml`; CI runs the same step.
- The `review-go` skill checks changed Go files against what the linters cannot see. It loads every document tagged `code-quality`, not a subset.
- No linter catches the error-wrapping criterion, the `fmt.Errorf`-without-verbs clause, unnamed timeouts, unbounded reads, or import groups. Review holds them.
- New code MUST follow these conventions. A deviation MUST carry a comment that explains why. A lint deviation MUST use the analyzer's own directive with a reason — `//exhaustive:ignore // <why>` — rather than a blanket exclusion.
- `go vet ./...` and `go build ./...` run as baseline checks and catch a subset of these conventions.
