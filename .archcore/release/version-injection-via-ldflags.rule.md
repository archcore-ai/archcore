---
title: "Version Must Be Injected via ldflags at Build Time"
status: accepted
tags:
  - "release"
---

## Rule

- The CLI version **must** be set via `-ldflags -X` at build time, never hardcoded in source.
- `main.go` declares two package-level variables with dev defaults:
  ```go
  var (
      version = "dev"
      commit  = "none"
  )
  ```
- These are passed to `cmd.NewRootCmd(version, commit)` which sets cobra's `Version` field.
- GoReleaser injects real values via `-X main.version={{.Version}} -X main.commit={{.Commit}}`.
- The `--version` flag is handled by cobra automatically — do not add a separate `version` subcommand.

## Rationale

- Single source of truth: the git tag drives the version everywhere (binary, GitHub release, install script).
- Dev builds self-identify as `dev` so there is no confusion with released binaries.
- Using cobra's built-in `Version` field avoids custom code and gives users the standard `--version` flag.

## Examples

**Good — dev defaults with ldflags injection:**
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

- `NewRootCmd` requires `(version, commit string)` parameters — compile error if omitted.
- GoReleaser config (`.goreleaser.yaml`) defines the ldflags; changing the variable names without updating the config breaks the release.