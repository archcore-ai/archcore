---
title: "Version Must Be Injected via ldflags at Build Time"
status: accepted
tags:
  - "component:cli"
  - "release"
---

## Rule

1. The build MUST set the CLI version through `-ldflags -X main.version` at build time.
2. The developer MUST NOT hardcode a version string in source.
3. `main.go` MUST declare the package-level variable `version` with the development default `"dev"`.
4. `main.go` MUST pass `version` to `cmd.NewRootCmd(version)`, which sets cobra's `Version` field.
5. The developer MUST NOT add a separate `version` subcommand.

## Rationale

The git tag drives the version everywhere: the binary, the GitHub release, and the install script read the same value. A development build reports `dev`, which keeps it distinguishable from a released binary. Cobra serves `--version` from the `Version` field, so no command code handles the flag.

## Enforcement

- `NewRootCmd` requires the `version string` parameter, so an omission fails compilation.
- `@cli/.goreleaser.yaml` defines the ldflags. IF a developer renames the variable without updating that file, THEN the release build stops injecting the version.
