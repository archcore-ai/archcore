---
title: "Display Version in CLI Banners"
status: accepted
tags:
  - "cli-ui"
---

## Idea

Show the CLI version in `Banner()` and `WelcomeBanner()` output so users see the version at a glance when running `archcore`, `archcore --help`, or `archcore doctor`.

Examples:
- `Banner()`: `Archcore v0.0.1-alpha.5 — Git-native context for AI coding agents`
- `WelcomeBanner()`: add a version line below the title

## Value

- Users immediately know which version they're running without `--version`
- Easier to reference in bug reports and screenshots
- Consistent with common CLI tools that show version in help/banner output

## Delivered

`@internal/display/display.go` renders the version through `versionSuffix`, `Banner(version)`, and `WelcomeBanner(version)`. `versionSuffix` returns an empty string for an empty version and for `dev`, so an unstamped build renders the original title.

The shipped form takes the version as a function parameter. It does not use the package variable `display.Version` that step 1 below proposes, so steps 1, 2, and 5 did not apply and `cleanVersion` stayed in `@cmd/root.go`. `@internal/display/display_test.go` covers the with-version and without-version cases.

## Possible Implementation

1. Add `var Version string` to `internal/display/display.go`
2. Set `display.Version = version` in `cmd/root.go` (`NewRootCmd`)
3. Update `Banner()` to include version in title: `Archcore v1.2.3 — Git-native context for AI coding agents`
4. Update `WelcomeBanner()` to add a dim version line below the title
5. Use `cleanVersion()` (already in `cmd/root.go`) to strip pseudo-version suffixes
6. Add tests in `internal/display/display_test.go` for with/without version

## Risks and Constraints

- Minor: version string length could affect banner alignment in `WelcomeBanner()` — needs visual testing with long pseudo-versions
- `cleanVersion` logic currently lives in `cmd/root.go` — may need to move to `display` package or a shared util if display needs it directly
