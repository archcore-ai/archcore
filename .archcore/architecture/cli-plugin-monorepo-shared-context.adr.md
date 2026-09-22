---
title: "Develop CLI and Plugin in One Monorepo with Shared Context"
status: draft
tags:
  - "architecture"
  - "component:cli"
  - "component:plugin"
---

## Context

On 2026-09-22, the owner chose to develop the CLI and plugin together on the plugin repository's dev branch and share one Archcore project. The source import preserves CLI commit 87a1b5d3f78a89b77ca76a3460ac6edabcf019e5 through subtree merge b40b4bd. Before the merge, the two corpora contained 138 and 124 documents with 357 and 655 relations, without conflicting document paths.

## Decision

Develop the CLI under `cli/` and the plugin under `plugin/` on dev, with one repository-root `.archcore/` for both components.

## Alternatives Considered

1. Keep a separate `.archcore/` per component — rejected because the owner requested a single project context and one relation graph.
2. Import CLI as a snapshot without Git history — rejected because a subtree merge preserves the original commits and authorship.
3. Publish the entire monorepo checkout as the plugin — rejected because it includes development context and CLI example projects in host plugin caches.
4. Unify versions and rename GitHub repositories in this change — deferred because the owner limited this stage to source layout and shared context.

## Consequences

### Enabled

- The shared corpus retains all 262 source documents, their statuses, and all 1012 original relations before this decision record is added.
- Documents retain their domain paths. Tags `component:cli` and `component:plugin` support component-scoped queries in the same project.
- Code references resolve from the monorepo root, including @cli/main.go and @plugin/plugins/archcore/bin/session-start.
- The root @Makefile runs component checks; @.github/workflows/test.yml exercises the plugin against both a pinned CLI and this commit's CLI source.
- @scripts/export-plugin.sh preserves the public marketplace layout while excluding CLI source and development context.

### Costs and boundaries

- CLI example `.archcore/` directories remain fixtures. Neither component owns a second development corpus.
- The external global source keeps the existing `../global/.archcore` setting; this change does not import the global repository.
- CLI publication remains a separate migration step. Imported legacy workflows under `cli/.github/workflows/` are reference files, not active root workflows.
- The plugin still publishes its existing version and repository IDs. The export's public paths differ from their source paths by the `plugin/` prefix.
- Root workflow and shared-context paths require explicit workspace-root resolution in plugin tests.

## Superseded when

- A subsequent release decision unifies component versions and their publication workflow.
- A component becomes an independently maintained product and requires its own writable context and release ownership.
