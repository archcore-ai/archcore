---
title: "Go Code Quality Conventions for Archcore CLI"
status: accepted
tags:
  - "code-quality"
  - "code-review"
  - "component:cli"
  - "golang"
---

## Rule

The developer is the obligated actor for every convention below. Each item states a MUST unless it names `SHOULD` or `MAY`. IF a change deviates from a convention, THEN the developer MUST add a comment that names the reason. Naming is in `strict-go-naming-conventions.rule`; comments are in `comments-are-the-exception.rule`.

### Error handling

- An error leaving a function MUST name the operation that failed and the subject it failed on.
- Wrap when the wrapper adds information the inner error lacks, with `%w`, so a caller can still `errors.Is` or `errors.As`.
- Pass an error through unwrapped when it already names the operation and the subject. Wrapping it again stutters — `verifying checksum: checksum not found for archcore_darwin.tar.gz` says one thing twice. A thin exported wrapper over a helper that already wrapped adds nothing.
- The bare `return err` to look for is the one where the caller cannot tell what failed.
- Treat a non-critical operation as a soft failure: print the error and continue. Example: an unreachable server during `archcore init` prints a warning and initialization completes.
- Treat the failure of the operation a command exists to perform as hard: `archcore update` returns a non-zero exit code when the update fails, so a script can detect it.
- Return `cmd.ErrAlreadyReported` from a command that already printed its own failure summary. `main` then only sets the exit code and prints nothing more.
- Return MCP tool errors as `mcp.NewToolResultError(msg)`, not as Go errors — the MCP protocol expects error results, not handler failures. Sanitize operating-system errors through `sanitizeError` before they reach a tool result.
- Validate then act: check every precondition and return the error before any side effect.
- Assign a deliberately ignored error to `_`, so the decision is visible. An unassigned call reads as an oversight, and `errcheck` treats it as one.
- Classify every read of external state as guard or advisory before choosing its error branch — `fail-open-or-fail-closed-reads.rule`.
- Return a classified sentinel, not a rendered message, from a predicate two surfaces share — `shared-guards-return-classified-sentinels.rule`.

### Standard-library idioms

- Test for a missing file with `errors.Is(err, fs.ErrNotExist)`, not `os.IsNotExist`. The latter does not unwrap, so it silently returns false the first time a caller wraps the error with `%w`.
- Iterate a split lazily with `strings.SplitSeq` or `strings.Lines` when the input is bounded by a limit. `strings.Split` materializes every element before the limit can refuse any of them.

### Output and display

- Route all styled terminal output through `internal/display`. Never write inline ANSI escape codes.
- Write initialization and setup messages, such as MCP server startup, and error output from `main` to stderr. Write normal command output to stdout.
- Set `SilenceErrors: true` and `SilenceUsage: true` on the root command, and format errors through `FormatExecuteError()`.
- Choose the stdout handle by what the command's output is, not by preference:
  - A command whose output is **one self-contained report a caller may redirect** prints through
    `cmd.OutOrStdout()`. Every line of the run belongs to that report, so a caller redirecting the
    command must see all of them or none. `plugin`, `update`, and the root command's banner are this
    branch — the root's banner shares its output with `cmd.Usage()`, which writes to the cobra writer
    and cannot be split from it.
  - A command that **interleaves its lines with helpers writing to the process handle** prints through
    `os.Stdout`. `init`, `doctor`, `hooks`, and `status` are this branch; routing them through the
    cobra writer would capture only half the run, and the command tests redirect the process handles.
- Build the report as data and render it in a separate function — `rendering-happens-at-the-boundary.rule`. `@cli/cmd/status_report.go` records why the resolution has to happen at call time.

### Cobra commands

- Pass the version to a command factory as an argument, not through a package-level global. This includes the MCP server version: `mcpserver.NewServer(baseDir, version)`.
- Propagate `cmd.Context()` to every downstream function for cancellation, including `exec.CommandContext` for a subprocess.
- Declare `Args` on every command. A command taking no positional argument uses `cobra.NoArgs`; a hook leaf uses `cobra.ArbitraryArgs` and carries a comment saying why, because on that path a non-zero exit is a verdict rather than a complaint.
- Resolve the project root through `resolveProjectRoot` and register a `--project` flag, never `os.Getwd()` directly. `TestCommands_OfferProjectFlag` walks the tree and fails on a command that does neither.

### File I/O and path safety

- Validate every user-provided path against directory traversal: reject `..` segments and absolute paths, then check that the cleaned path is still inside `.archcore/`.
- Write critical state atomically through a shared helper — `choosing-an-atomic-write.rule` owns the choice and the exceptions.
- Edit user-owned config files (agent settings, MCP configs) surgically through `internal/jsonfile`: unknown fields and key order round-trip as opaque `json.RawMessage`, and the file is written only when something changed.
- Back up a corrupted config file as `.bak` before overwriting it, and abort when the backup itself cannot be written — `backup-invalid-configs.adr`.

### Data structures

- Declare closed-set wire vocabularies as named constants, such as `sourceKindLocal` and `matchKindContent`, never as scattered string literals. Key a map on the typed enum with the constants, not with string literals — a typo in a literal key compiles and silently takes the default.
- Bound and order anything leaving the process: `bounded-and-deterministic-output.rule`.

### Concurrency

- Use no goroutines in business logic. The CLI has a synchronous, single-threaded architecture.
- Shared mutable state reachable from an MCP tool call MUST be mutex-protected. mcp-go serves
  `tools/call` on a worker pool, so a handler is concurrent by default. Manifest mutations go through
  `manifestStore.mutate` (clone-and-swap); the guarded state is not confined to `internal/mcp/tools`.
- An unwaited goroutine is permitted only where a spec states its non-join contract and names where
  its output goes. The one that exists is the background update started by `RunStdio`, which must
  never write to stdout — `mcp-background-update.spec`.
- `architecture/process-and-concurrency-model.spec` states the invariants and the known bounds. Read
  it before adding shared state or a goroutine.
- Call `signal.NotifyContext()` at the top level (`@cli/main.go`) for graceful shutdown on SIGINT and SIGTERM.
- Bound every subprocess and network call with an explicit timeout held in a named constant whose comment states the budget it protects. Never use `http.DefaultClient` or a context-free `exec.Command`.

## Rationale

Explicit conventions let an AI coding agent and a human reviewer check new code without pattern-matching the whole codebase, and keep new code from drifting. The error-wrapping clause states the criterion the code actually follows: an audit of all twelve public bare-return sites found eleven were correct pass-throughs of an already-contextual error, so "never return a bare error" was wrong.

## Enforcement

- `golangci-lint run ./...` runs `errcheck`, `errname`, `exhaustive`, `revive`, `staticcheck`, and `unconvert`. The config is `.golangci.yml`; CI runs the same step.
- The `review-go` skill checks changed Go files against what the linters cannot see. It loads every document tagged `code-quality`.
- No linter catches the error-wrapping criterion, unnamed timeouts, or unbounded reads. Review holds them.
- A lint deviation MUST use the analyzer's own directive with a reason, never a blanket exclusion.
- An `exhaustive` directive MUST stand alone on the last comment line before its `switch`. Its reason MUST be on the comment lines directly above the directive: `// <why>`, then `//exhaustive:ignore`. `exhaustive` v0.15 and later rejects any text after the directive.
- A `//nolint:<linter>` directive keeps its reason on the same line: `//nolint:<linter> // <why>`.
