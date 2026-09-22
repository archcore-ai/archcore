---
title: "Archcore CLI System Overview"
status: accepted
tags:
  - "architecture"
  - "cli"
  - "mcp"
---

## Overview

This document is the map of the system as a whole; every subsystem row points at the document that
owns the detail, and nothing here restates a contract.

`archcore-cli` is a modular monolith: one Go module, one binary, five surfaces, and an acyclic
package graph. For the procedures — how to add a command, a document type, an MCP tool, a hook, or an
agent — read @.archcore/cli-ui/building-the-cli.doc.md. This document holds the structure; that guide
holds the steps.

## Content

### Surfaces

Each surface is a way into the same core, and each runs in a process shape described by
@.archcore/architecture/process-and-concurrency-model.spec.md.

| Surface | Entry | Owning document |
|---|---|---|
| CLI | ten commands registered at @cmd/root.go | `cli-ui/building-the-cli.doc` |
| MCP | `archcore mcp`, 10 unconditional tools plus `install_host_config` | `mcp/project-root-resolution.spec`, `mcp/search-documents.spec`, `mcp/install-host-config-tool-contract.adr` |
| Hook | hidden leaves under `archcore hooks <host> <event>` | `integrations/hook-runtime.spec`, `integrations/hook-wire-protocol.spec`, `integrations/hook-payload-reading.spec` |
| Sync | `archcore sync`, one-way push | `sync/sync-engine.spec`, `sync/one-way-push-sync-strategy.adr` |
| Update | `archcore update`, plus the unattended run inside `archcore mcp` | `update/unattended-update.spec`, `update/mcp-background-update.spec`, `update/updating-the-plugin.spec` |

### Package structure

The packages form five tiers and the import direction is an obligation, not a habit.

The tier table and the three forbidden edges live in
@.archcore/architecture/package-dependency-direction.rule.md. `CLAUDE.md` carries the one-line role of
each of the 18 packages under `internal/`.

Two dependency inversions carry the structure. `internal/mcp` receives the unattended update as an
opaque `BackgroundTask func(context.Context)` — @internal/mcp/server.go — so the server never links
the updater. `internal/mcp/tools` receives host wiring as an injected `HostWiringFunc`, so the tool
layer never imports `cmd`.

### Data model

A document is a file; its type decides its category, and its directory decides nothing.

| Element | Shape | Owning document |
|---|---|---|
| document | `<slug>.<type>.md` anywhere under `.archcore/` | `cli/docs-package-owns-the-document-model.adr` |
| virtual category | derived from the type suffix | `dir/categories-and-document-types.doc`, `dir/free-form-directory-structure.adr` |
| relation | directed triple stored in the manifest | `relations/local-relations-in-sync-state.adr` |
| global source | read-only mount declared in `settings.json` | `globals/global-sources.spec` |

`internal/docs` owns this model and carries no MCP dependency, so the hook path and the MCP tool
layer read documents through the same code.

### State

Five stores hold everything the tool remembers, and only the first three travel with the repository.

| Store | Owner | Travels with the repo |
|---|---|---|
| `.archcore/**/*.md` | the user and the MCP write tools | yes |
| `.archcore/settings.json` | `internal/config` | yes |
| `.archcore/.sync-state.json` | `internal/sync` | yes |
| `${XDG_STATE_HOME:-~/.local/state}/archcore` | `internal/stamp`, `internal/update`, `internal/telemetry` | no |
| host config and plugin registries | `internal/wiring`, `internal/plugin` | per host |

### Accepted trade-offs

Two costs are deliberate and recorded here because no other document states them as decisions.

- A supported host is described by five registries in four packages — `internal/agents`,
  `cmd/hook_dialect.go`, `internal/wiring`, and `internal/plugin`. Unifying them would pull
  `internal/agents` up to the surface tier, so the split stands and cross-registry agreement tests
  guard it — `code-quality/registry-agreement-and-test-seams.guide`.
- The relation graph shares `.sync-state.json` with sync hashes, so a project with `sync: none` still
  writes a file named "sync state" — `relations/local-relations-in-sync-state.adr`.

## Examples

**Reading path for a new contributor.** Read this document, then
`architecture/package-dependency-direction.rule`, then `cli-ui/building-the-cli.doc`, then the one
surface document for the change at hand.

**Placing a new document.** A structural claim about the system as a whole belongs in
`.archcore/architecture/`. A claim about one surface belongs in that surface's directory.
