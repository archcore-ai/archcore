---
title: "Version Must Be Injected via ldflags at Build Time"
status: accepted
tags:
  - "release"
---

## Rule

1. The build MUST set the CLI version through `-ldflags -X` at build time.
2. The developer MUST NOT hardcode a version string in source.
3. `main.go` MUST declare the package-level variables `version` and `commit` with the development defaults `"dev"` and `"none"`.
4. `main.go` MUST pass both variables to `cmd.NewRootCmd(version, commit)`, which sets cobra's `Version` field.
5. The developer MUST NOT add a separate `version` subcommand.

## Current behavior

- GoReleaser injects the released values with `-X main.version={{.Version}} -X main.commit={{.Commit}}` (`@.goreleaser.yaml`).
- Cobra serves `--version` from the `Version` field, so no command code handles the flag.

## Rationale

The git tag drives the version everywhere: the binary, the GitHub release, and the install script read the same value. A development build reports `dev`, which keeps it distinguishable from a released binary.

## Examples

Non-normative examples.

**Good — development defaults with ldflags injection:**

```go
// main.go
var (
    version = "dev"
    commit  = "none"
)
```

**Bad — hardcoded version string:**

```go
const Version = "1.2.3"
```

**Bad — separate version command:**

```go
newVersionCmd() // cobra already provides --version
```

## Enforcement

- `NewRootCmd` requires the `(version, commit string)` parameters, so an omission fails compilation.
- `.goreleaser.yaml` defines the ldflags. IF a developer renames either variable without updating that file, THEN the release build stops injecting the version.
