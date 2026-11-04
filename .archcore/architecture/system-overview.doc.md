---
title: "Archcore CLI System Overview"
status: accepted
tags:
  - "architecture"
  - "cli"
  - "component:cli"
  - "mcp"
---

## Overview

This document is the map of the system as a whole; it points at the documents that own the detail and restates no contract.

`archcore-cli` is a modular monolith: one Go module, one binary, five surfaces (CLI, MCP, hook, sync, update), and an acyclic
package graph. Each surface runs in a process shape described by
@.archcore/architecture/process-and-concurrency-model.spec.md. For the procedures — how to add a command, a document type, an MCP tool, a hook, or an
agent — read @.archcore/cli-ui/building-the-cli.doc.md. This document holds the structure; that guide
holds the steps.

## Content

### Package structure

The packages form five tiers and the import direction is an obligation, not a habit.

The tier table and the three forbidden edges live in
@.archcore/architecture/package-dependency-direction.rule.md. `CLAUDE.md` carries the one-line role of
each package under `internal/`.

Two dependency inversions carry the structure. `internal/mcp` receives the unattended update as an
opaque `BackgroundTask func(context.Context)` — @cli/internal/mcp/server.go — so the server never links
the updater. `internal/mcp/tools` receives host wiring as an injected `HostWiringFunc`, so the tool
layer never imports `cmd`.

### Data model

A document is a file; its type decides its category, and its directory decides nothing (`dir/free-form-directory-structure.adr`). `internal/docs` owns this model and carries no MCP dependency, so the hook path and the MCP tool
layer read documents through the same code (`cli/docs-package-owns-the-document-model.adr`). Relations live in the manifest (`relations/local-relations-in-sync-state.adr`). A global source is a mount declared in `settings.json` (`globals/source-mounts.spec`).

### State

Only the project files travel with the repository: `.archcore/**/*.md`, `.archcore/settings.json`, and `.archcore/.sync-state.json`. The per-user state under `${XDG_STATE_HOME:-~/.local/state}/archcore` and the host config and plugin registries do not.

### Accepted trade-offs

Two costs are deliberate and recorded here because no other document states them as decisions.

- A supported host is described by five registries in four packages — `internal/agents`,
  `cmd/hook_dialect.go`, `internal/wiring`, and `internal/plugin`. Unifying them would pull
  `internal/agents` up to the surface tier, so the split stands and cross-registry agreement tests
  guard it — `code-quality/registry-agreement-and-test-seams.guide`.
- The relation graph shares `.sync-state.json` with sync hashes, so a project with `sync: none` still
  writes a file named "sync state" — `relations/local-relations-in-sync-state.adr`.
