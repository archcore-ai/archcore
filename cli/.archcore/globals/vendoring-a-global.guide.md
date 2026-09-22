---
title: "Vendoring a Global Source into .archcore/global/"
status: accepted
tags:
  - "config"
  - "globals"
---

## Purpose

Place a global source inside the project tree under `.archcore/global/<id>` and declare it, so that its documents mount read-only in this project.

## Prerequisites

- An initialized project with `.archcore/settings.json`.
- Access to the global source repository. Its documents live under its own `.archcore/`.
- The related rule on declaring global sources.

`.archcore/global/` is reserved: the local document scan skips any directory named `global`, so content placed there stays invisible until it is declared. The reservation needs no configuration.

## Inputs

- The global source repository URL.
- The `id` to give the source in `settings.json`.

## Procedure

### 1. Place the global content under `.archcore/global/<id>`

Choose one distribution mechanism.

Commit a vendored copy — reproducible, travels with the repository, updated by hand:

```bash
git clone <global-repo-url> .archcore/global/company
rm -rf .archcore/global/company/.git
git add .archcore/global/company && git commit -m "vendor company global"
```

Git submodule — pinned to a ref, refreshed with `git submodule update`:

```bash
git submodule add <global-repo-url> .archcore/global/company
```

Gitignore and clone on setup — not committed; a setup step clones it. Every declared global is mandatory, so a missing clone fails the server at startup:

```bash
echo ".archcore/global/" >> .gitignore
```

### 2. Point `path` at the directory that contains the documents

What was cloned decides the value:

- IF the whole global repository was cloned and it has its own `.archcore/`, THEN the documents sit at `.archcore/global/company/.archcore/knowledge/…`, so use `path: ".archcore/global/company/.archcore"`.
- IF only the document tree was vendored, with `knowledge/` directly under the folder, THEN use `path: ".archcore/global/company"`.

### 3. Declare the source in `settings.json`

```json
{
  "sync": "none",
  "globals": [ { "id": "company", "path": ".archcore/global/company" } ]
}
```

## Verification

Run the MCP server in the project and list the documents:

```bash
archcore mcp   # .mcp.json args: ["mcp"]
```

Expected result: `list_documents` shows the vendored documents with `source_kind: "global"`, `read_only: true`, `source_id: "company"`, and paths under `.archcore/global/company/…`.

Expected result: `update_document` on one of them returns `cannot update a read-only global source document`, because an in-tree path is a valid `.archcore/` path and reaches the read-only guard directly.

Expected result: `add_relation` with one of them as source or target is refused with `relations connect local documents only`.

Expected result: undeclared content elsewhere under `.archcore/global/`, such as a second folder that `globals` does not list, does not appear in `list_documents`, and stays read-only and non-linkable.

## Troubleshooting

- Documents do not appear. `path` points at the wrong level. It must reach the directory whose subtree holds the `*.type.md` files, which is often `…/.archcore` after cloning a whole repository. Re-check step 2.
- The server refuses to start with `global source "company" not found at "…"`. The vendored folder is missing, for example gitignored and not yet cloned. Clone it: every declared global is mandatory, so the server does not start without it.
- A local document folder disappeared. A local directory was named `global`, and the scan skips any directory named `global` at any depth. Rename it.
- The global was vendored outside `.archcore/global/`. An in-tree global under a folder not named `global`, such as the plural `.archcore/globals/<id>`, is de-duplicated by the scan: it appears once as a read-only global and never also as a writable local. It stays read-only and non-linkable. Vendoring under the reserved `.archcore/global/` directory remains the recommended, self-documenting layout.
- In-tree versus `../` sibling. In-tree vendoring is the more robust form: self-contained, with no assumption about the clone layout, and it yields the clean read-only message on a write. A `../` sibling instead fails a write with `invalid path: must start with ".archcore/"`. The related ADR on declaring globals in `settings.json` records this consequence.
- Updating the global. A vendored copy and a submodule are both snapshots. Re-clone, or run `git submodule update`, to pull upstream changes. An automated refresh command is planned, not implemented; the related plan document tracks it.
