---
title: "A Go Module Version Ships Only After 30 Days"
status: draft
tags:
  - "code-quality"
  - "component:cli"
  - "release"
---

## Rule

1. WHEN an author raises a module version in `cli/go.mod`, the author MUST select a version published at least 30 days before the change.
2. IF the version is a pseudo-version, THEN the author MUST count its age from the commit time in the version string.
3. IF every version older than 30 days lacks a fix for a defect that reaches the archcore binary, THEN the author MAY select a younger version.
4. WHEN the author selects a younger version, the author MUST write the reason as a line comment on its `require` line in `cli/go.mod`.
5. WHEN a dependency requires a younger version through minimum version selection, the author MUST treat that version as a younger version under statement 4.

## Rationale

The unattended update replaces the binary on every machine that runs one qualifying session (`.archcore/update/unattended-update.spec.md`), so a defect in a new dependency release reaches the whole install base. A release that has been public for 30 days has had time to be reported and retracted [assumption]. On 2026-11-06 the upgrade selected `github.com/xo/terminfo v1.3.1`, published two days earlier, only because the main module raised it above what any dependency required.

## Examples

### Good

```text
require (
	github.com/mark3labs/mcp-go v1.5.0 // under 30 days: v1.2.0-v1.5.0 fix the stdio server (panic recovery, serialized writes, queued calls)
	github.com/xo/terminfo v1.2.0 // indirect
)
```

`v1.2.0` was 54 days old at the change; the `mcp-go` line names the defect that justifies the younger version.

### Bad

```text
require (
	github.com/mark3labs/mcp-go v1.5.0
	github.com/xo/terminfo v1.3.1 // indirect
)
```

`go get -u ./...` selected both versions. Neither line records why a release younger than 30 days ships.

## Enforcement

manual review. To read publish times, run `go list -m -f '{{.Path}} {{.Version}} {{.Time}}' all` in `cli/`. No CI step checks the age.