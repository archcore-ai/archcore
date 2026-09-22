---
title: "Package Dependency Direction Is Layered and Acyclic"
status: accepted
tags:
  - "architecture"
  - "code-quality"
  - "golang"
---

## Rule

The Go packages of this repository form five tiers. A package imports its own tier or a lower tier,
never a higher one. Three named edges are forbidden outright.

| Tier | Packages | Definition |
|---|---|---|
| T0 leaf | `internal/display`, `internal/git`, `internal/jsonfile`, `internal/projectroot`, `internal/xdg`, `templates` | imports no other repository package |
| T1 domain | `internal/agents`, `internal/config`, `internal/docs`, `internal/sync` | owns a domain model |
| T2 capability | `internal/advisory`, `internal/api`, `internal/plugin`, `internal/stamp`, `internal/telemetry`, `internal/update`, `internal/wiring` | performs work for a surface |
| T3 surface | `cmd`, `internal/mcp`, `internal/mcp/tools` | speaks a protocol to a user or a host |
| T4 entry | `main` | wires the binary |

1. A package MUST NOT import a package of a higher tier.
2. A T0 package MUST import the standard library and third-party modules only.
3. `internal/mcp` MUST NOT link `internal/update`.
4. `internal/update` MUST NOT link `internal/plugin`.
5. `internal/mcp/tools` MUST NOT import `cmd`.
6. The developer MUST pass a higher-tier capability into a lower tier as an injected function type.
7. The production code of `internal/testsupport` MUST import no other repository package.
8. The developer MUST guard a new forbidden edge with a test named `TestPackage_DoesNotLink<Surface>`.
9. WHEN a change adds a package, the developer MUST place it in a tier in the table above.

## Rationale

The tiers keep the import graph acyclic without a tool, and they keep a surface replaceable: a second
front end reuses T0 through T2 unchanged. Clauses 3 to 5 protect three properties that a plain layer
check misses — an MCP server that links the updater gains an unwanted binary-replacement path, an
updater that links the plugin surface loses the guarantee that plugin work never changes the CLI, and
a tool package that imports `cmd` creates the import cycle the executor seam exists to prevent.

## Examples

**Good** — the surface hands the server an opaque capability instead of importing it.

> `RunStdio` takes `BackgroundTask func(context.Context)` — @internal/mcp/server.go:238 — and
> `cmd/mcp.go` supplies the unattended-update run. `internal/mcp` never names `internal/update`.

**Good** — `install_host_config` registers only when the cmd layer injects an executor through
`WithHostWiring(tools.HostWiringFunc)` — @internal/mcp/server.go:220.

**Bad** — a tool handler that calls a helper in `cmd` directly. Go refuses the build, because `cmd`
already imports `internal/mcp/tools`. The seam is not a preference; the cycle is the reason it exists.

## Enforcement

- `TestPackage_DoesNotLinkTheUpdateStack` — @internal/mcp/server_contract_test.go:67 — holds clause 3.
- `TestPackage_DoesNotLinkThePluginSurface` — @internal/update/plugin_link_test.go:37 — holds clause 4.
- The Go compiler holds clause 5, because `cmd` imports `internal/mcp/tools`.
- Clauses 1, 2, 6, 7, and 9 carry no automated check today. `go list -deps ./internal/<pkg>` reports
  the actual edges of one package, and review compares them against the table. [assumption] A single
  test that walks every package and asserts the tier table would close the gap.
