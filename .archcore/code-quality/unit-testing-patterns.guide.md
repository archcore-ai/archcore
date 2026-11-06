---
title: "Unit Testing Patterns for Archcore CLI"
status: accepted
tags:
  - "code-quality"
  - "code-review"
  - "component:cli"
  - "golang"
  - "testing"
---

## Prerequisites

- Go 1.26+
- Read `go-code-quality.rule.md` for general conventions
- Read `comments-are-the-exception.rule.md` before commenting a test

## Steps

### 1. Organize and write tests

Place tests in the same package as the source, so they reach unexported functions. Use pure `testing.T`, with no assertion library. Use `t.TempDir()` for every filesystem test, and table-driven subtests with `name` as the first field and `tt` as the loop variable. Use inline strings for test data, not fixture files.

### 2. Arm isolation for ambient state

`t.TempDir()` contains a test's own writes. It does not contain a write the code under test aims at
`$HOME`, an XDG state directory, or a host CLI on `PATH`.

If the package reaches any of those, arm the isolation in `TestMain` with `testsupport.IsolateGit()` and `testsupport.IsolateAmbientState()`, and exit through `isolation.Finish(m.Run())`. The full procedure, the incident behind it, and how to guard the guard
are in `isolating-the-machine-from-the-test-suite.guide`.

### 3. Mock with a minimal interface

Define a minimal interface in the source file and implement the mock in the test file. For a package-level variable that exists so a test can replace behavior, see
`registry-agreement-and-test-seams.guide` — a seam carries a comment saying production never
reassigns it.

### 4. Test MCP tools

A handler takes a `RootProvider`, not a base directory. Unit tests use `StaticRoot`. See
`the-shape-of-an-mcp-tool-file.rule`. A test that must cross tool boundaries — one handler writes, another reads — belongs in
`internal/mcp/integration/` instead. See `in-process-mcp-integration-tests.adr`.

### 5. Benchmarks

A benchmark over a document corpus builds it with `testsupport.BuildCorpus(tb, dir, n)`.

Do not hand-roll a corpus of one document type. `corpusTypes` in @cli/internal/testsupport/corpus.go spans
both halves of the code-alignment allowlist, so a benchmark measures the filter rather than a corpus that happens to be all one type. Every generated
body mentions `src/api/`, so code-alignment correlation has something to match.

Report allocations. A read-path regression in this repository has historically shown up as allocation
count before it showed up as wall clock.

### 6. Name tests

Use `Test<Function>` or `Test<Feature>_<Scenario>`. A test file that pins normative properties rather than coverage is named
`<subject>_spec_test.go` — `strict-go-naming-conventions.rule` §I.

## Verification

A test that passes is not yet a test that guards. Break the behavior deliberately and confirm the test
fails, then restore. A test that stays green against an injected fault pins nothing.

## Common Issues

### Test fails only in CI

- Check for `os.Getuid() == 0` skip guards — CI may run as root.
- Check for `t.Parallel()` races on shared state.

### The suite is green and the machine changed

The package is missing `TestMain`, or `Isolation.Finish` is not on the exit path. See step 2.

### Filesystem test pollution

- Never `os.Chdir()` without saving and restoring the original directory; prefer `t.Chdir`.
- When the working directory must move, do NOT use `t.Parallel()` — it is process-global.

### A test asserts a platform property and fails elsewhere

`filepath.ToSlash` and friends are no-ops off Windows. A property that depends on the host OS belongs
in a build-tagged file — see `platform-splits-are-files.rule` §7.
