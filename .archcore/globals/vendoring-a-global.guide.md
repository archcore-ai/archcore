---
title: "Vendoring a Global Source into the Project Tree"
status: accepted
tags:
  - "component:cli"
  - "config"
  - "globals"
---

## Purpose

Place a global source inside the project tree, outside the project's own `.archcore/`, and declare it, so that this project reads its documents with its own (@.archcore/globals/source-mounts.spec.md).

## Prerequisites

- An initialized project with `.archcore/settings.json`.
- Access to the global source repository. Its documents live under its own `.archcore/`.
- An archcore CLI 0.15.0 or newer.

A source path under the project's own `.archcore/` is fatal: the MCP server refuses to start and names the source id and the migration target. `.archcore/global/` is no longer reserved; a directory with that name is an ordinary local directory.

## Inputs

- The global source repository URL.
- The `id` to give the source in `settings.json`.
- A directory for the source outside `.archcore/`. This guide uses `vendor/company`.

## Procedure

### 1. Place the global repository under `vendor/<id>`

Choose one distribution mechanism.

Commit a vendored copy — reproducible, travels with the repository, updated by hand:

```bash
git clone <global-repo-url> vendor/company
rm -rf vendor/company/.git
git add vendor/company && git commit -m "vendor company global"
```

Git submodule — pinned to a ref, refreshed with `git submodule update`:

```bash
git submodule add <global-repo-url> vendor/company
```

Gitignore and clone on setup — not committed; a setup step clones it. Until the clone exists, the server starts with the local documents only and prints a warning that names the missing source:

```bash
echo "vendor/company/" >> .gitignore
```

### 2. Declare the source in `settings.json`

Point `path` at the `.archcore/` directory of the vendored repository:

```json
{
  "sync": "none",
  "globals": [ { "id": "company", "path": "vendor/company/.archcore" } ]
}
```

## Verification

Run the MCP server in the project and list the documents:

```bash
archcore mcp   # .mcp.json args: ["mcp"]
```

Expected result: `list_documents` shows the vendored documents with `source_kind: "global"`, `source_id: "company"`, and `source_root` set to the absolute path of `vendor/company`.

Expected result: `update_document` with `project` set to that `source_root` and a path that starts with `.archcore/` changes the vendored document, and the response carries `source: "company"`.

Expected result: `add_relation` between a local document and a vendored one is refused, because both endpoints must belong to one project.

## Migration from `.archcore/global/<id>`

1. Move the vendored folder out of `.archcore/`: `git mv .archcore/global/company vendor/company`.
2. IF the moved folder holds the documents directly, without its own `.archcore/`, THEN move them into `vendor/company/.archcore/`.
3. Set `path` to `vendor/company/.archcore`.

Expected result: the MCP server starts, and `list_documents` shows the documents with `source_id: "company"`.

## Troubleshooting

- The server refuses to start with `overlaps the project's own .archcore`. The declared `path` lies under `.archcore/`. Follow the migration above.
- Documents do not appear. `path` points at the wrong level. It must reach the vendored repository's `.archcore/` directory.
- The server prints `global source "company" not found at "…" — starting without it`, and no global documents appear. The vendored folder is missing, for example gitignored and not yet cloned. Clone it to read its documents; the local documents work without it.
- A global row carries no `source_root`. The declared `path` does not end in `.archcore`, so no project argument reaches the source. Point `path` at the repository's `.archcore/`.
- Updating the global. A vendored copy and a submodule are both snapshots. Re-clone, or run `git submodule update`, to pull upstream changes. A write through `project` changes the vendored copy only; send the change upstream as well.
