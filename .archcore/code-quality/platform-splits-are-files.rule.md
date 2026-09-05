---
title: "Platform Splits Are Files, and the Weaker One States Its Residual Invariant"
status: accepted
tags:
  - "code-quality"
  - "golang"
---

## Rule

Behavior that differs by platform is expressed by which file the toolchain compiles, not by a branch
taken at run time.

1. WHEN behavior differs by `GOOS` or `GOARCH`, the developer MUST put each variant in its own file
   carrying a `//go:build` constraint.
2. The developer MUST name such a file `<subject>_<constraint>.go`.
3. The developer MUST NOT branch on `runtime.GOOS` in place of a missing variant file.
4. Every constraint the module builds for MUST have a variant.
5. IF one variant is weaker than the others, THEN its doc comment MUST state the residual invariant
   callers must uphold on that platform.
6. A caller relying on that residual invariant MUST carry a note pointing back at it.
7. A test pinning a platform-specific property MUST carry the same filename constraint.

## Rationale

A build constraint is checked by the compiler on the platform that matters, while a `runtime.GOOS`
branch ships every platform's code everywhere and fails only where nobody runs it. Clause 5 is the
part a split alone does not give: a weaker variant is still a correct build, so the gap has to be
written down or a caller assumes parity.

## Examples

**Good** — clauses 1 to 3. A two-line function split rather than branched:
@internal/mcp/dup2_dup3.go (`//go:build linux`, routes through `Dup3` because newer Linux ports ship
only `dup3(2)`) and @internal/mcp/dup2_classic.go (`//go:build unix && !linux`).

**Good** — clause 5 and 6. @internal/mcp/stdio_shield_windows.go states what the Windows shield does
not cover: the OS handle for stdout is not rerouted, so cgo, raw writes to fd 1, and child processes
inheriting the standard handle could still reach the protocol stream — therefore tool executors must
not spawn children on Windows. @internal/mcp/server.go carries the matching caller-side note.

**Good** — clause 7. @internal/jsonfile/jsonfile_umask_unix_test.go pins the half of the mode contract
a chmod cannot express, under `//go:build unix`.

**Bad** — a runtime branch standing in for a file.

> ```go
> if runtime.GOOS == "windows" {
>     return nil
> }
> return syscall.Dup2(oldfd, newfd)
> ```
>
> `syscall.Dup2` does not exist on every port, so this does not compile where it matters and silently
> does nothing where it does.

## Enforcement

- The Go toolchain holds clauses 1 to 4: a missing variant fails the build for that constraint.
- `go build` for each supported platform is the conformance check. CI covers the release matrix.
- Clauses 5 to 7 carry no automated check. Review holds them.
