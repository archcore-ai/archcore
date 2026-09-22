---
title: "Rendering Happens at the Boundary, Not in the Domain"
status: accepted
tags:
  - "cli"
  - "code-quality"
  - "golang"
  - "mcp"
---

## Rule

A domain package returns facts. Each surface turns those facts into what its own audience may see.

1. A domain package under `internal/` MUST return absolute paths and unwrapped errors in its result
   types.
2. The result type's doc comment MUST state that its paths are absolute and its errors are raw.
3. Each consuming boundary MUST perform its own transform before the value leaves the process.
4. The terminal boundary MUST render through `internal/display`.
5. The MCP boundary MUST make paths project-relative and MUST pass errors through `sanitizeError`.
6. A domain package MUST NOT import `internal/display`.
7. A domain package MUST NOT import an MCP protocol library.
8. A command that prints a report MUST build the report as data and render it in a separate function.

## Rationale

Two surfaces disagree about what is safe to show: a terminal user owns the filesystem the path names,
an MCP client does not. A domain package that rendered for one of them would either leak a path to
the other or force the other to unparse prose. Clause 8 is the same split one layer up — the hook path
needs the data form, because on a hook stdout carries the host protocol and nothing may print to it.

## Examples

**Good** — clause 1 and 2 stated on the package and on the type.

> `internal/wiring` — @internal/wiring/wiring.go: "Results carry raw errors and absolute paths:
> rendering is the caller's job at its own boundary." `AgentResult` repeats it: "Paths are absolute;
> errors are raw."

**Good** — clause 5. `hostWiringExecutor` at @cmd/host_wiring.go adapts `wiring.Apply` for the MCP
boundary: absolute paths become project-relative and per-agent errors are sanitized.

**Good** — clause 8. `collectStatus` returns a `statusReport`; `writeTo` renders it —
@cmd/status_report.go. `writeTo` is the only place `display.*Line` is called.

**Bad** — a domain function returning a pre-formatted line.

> ```go
> func Apply(...) (string, error) {
>     return display.OKLine("wrote " + absPath), nil
> }
> ```
>
> The MCP boundary now has to strip a decoration and a path out of prose, and the hook path cannot use
> the result at all.

## Enforcement

- The Go compiler holds clauses 6 and 7 indirectly through the import tiers in
  `architecture/package-dependency-direction.rule`.
- No `internal/display` or MCP library import exists outside `cmd/`, `internal/mcp/`, and
  `internal/display` itself. `go list -deps ./internal/<pkg>` reports the actual edges.
- Clauses 1 to 5 and 8 carry no automated check. Review holds them.
