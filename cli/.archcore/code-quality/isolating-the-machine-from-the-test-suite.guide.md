---
title: "Isolating the Developer's Machine from the Test Suite"
status: accepted
tags:
  - "code-quality"
  - "golang"
  - "testing"
---

## Overview

A test in this repository can reach the machine running it. Nine packages arm an isolation in
`TestMain` so that it cannot.

The isolation exists because of an incident, not a policy. A test ran `archcore init --agent
claude-code` with neither `HOME` nor `PATH` overridden; on a machine with Claude Code installed it
executed three real host commands and wrote a marketplace entry into the developer's own
`~/.claude/settings.json`. The suite stayed green throughout — @internal/testsupport/isolate.go.

### Target audience

Anyone adding tests to a package that touches `$HOME`, an XDG state directory, a host CLI on `PATH`,
or global git configuration.

## Prerequisites

- Read `unit-testing-patterns.guide` for the table-driven form and `t.TempDir()` use.
- Know which of the four ambient surfaces the package under test actually reaches.

## Steps

### 1. Decide whether the package needs it

Arm the isolation if the package's code reads `$HOME`, resolves an XDG state directory, executes a
host CLI, or spawns git. Nine packages do today: `cmd`, `internal/advisory`, `internal/config`,
`internal/git`, `internal/plugin`, `internal/stamp`, `internal/telemetry`, `internal/update`,
`internal/wiring`.

The list grows when a package gains a dependency, not only when it gains a test.
`internal/config` joined it when `ResolveGlobalPath` started shelling out to git for the worktree
anchor (`relative-globals-resolve-from-main-checkout.adr`) — the tests did not change, the code
underneath them did. Check this step again whenever a package reaches a new ambient surface.

`internal/testsupport` is not on the list and must not join it: its own tests exercise the isolation
rather than run under it.

### 2. Add TestMain

```go
var isolation *testsupport.Isolation

func TestMain(m *testing.M) {
	testsupport.IsolateGit()
	isolation = testsupport.IsolateAmbientState()
	os.Exit(isolation.Finish(m.Run()))
}
```

Keep the `*Isolation` in a package-level variable. A guard test inspects it; a discarded return value
cannot be checked.

A package that reaches only git may arm `IsolateGit` alone — `internal/git` and `internal/config` do,
because neither touches `$HOME`, an XDG directory, or a host CLI.

### 3. Name the surfaces in the TestMain doc comment

State which of the four this package actually reaches, and why. @cmd/main_test.go names all four
separately: the state directory, the home directory, `PATH`, and git.

### 4. Exit through Finish

Pass `m.Run()` to `Isolation.Finish` and exit with its result. `Finish` reads the stand-in host CLI
record. A stand-in that only failed would be absorbed: `internal/plugin`'s executor captures a child's
stderr and reports a failure as data rather than an error, so the run would stay green.

### 5. Override a specific value with t.Setenv, not by skipping the isolation

A test that needs a particular home sets it with `t.Setenv`, including to the empty string for the
unresolvable-directory cases in `internal/update` and `internal/xdg`.

### 6. Do not move the working directory

`go test` runs each package in its source directory, and the tests that parse this repository's own
source depend on that. A test needing a different root passes one explicitly or calls `t.Chdir`.

### 7. Guard the guard

Add a test that asserts the isolation is applied. Assert **containment in the isolation root**, not
inequality with the real home: once `HOME` is overridden the real home is no longer observable from
inside the process, so comparing against it compares the isolated value with itself and passes no
matter what — @cmd/isolation_guard_test.go.

## Verification

```bash
go test ./cmd/ -run TestIsolation
go test ./internal/testsupport/
```

`internal/testsupport/isolate_test.go` checks the stand-in CLI list against `internal/plugin`'s real
host table. It may import `internal/plugin` because nothing imports it back.

## Common issues

- **A new host CLI escapes the trap.** `hostCLIs` in @internal/testsupport/isolate.go repeats the CLI
  column of `internal/plugin`'s host table rather than reading it, because `internal/plugin`'s own
  `TestMain` calls into this package and the import would cycle. The agreement test is what closes
  the gap — see `registry-agreement-and-test-seams.guide`.
- **The suite is green and the machine changed anyway.** That is the incident's signature. Check that
  `Finish` is on the exit path and that the package has a `TestMain` at all.
- **A pure-looking unit test spawns a subprocess.** A lexical-path case can reach a git query through
  a memo two calls down. Drive such a case through the package's lookup seam instead, so the test
  stays pure and leaves the process-wide memo alone.
- **`internal/testsupport` production code must import no other repository package.** An import there
  can cycle through any package that arms the isolation — `architecture/package-dependency-direction.rule` §7.
