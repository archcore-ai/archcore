---
title: "Vendoring a Global Source into .archcore/global/"
status: accepted
tags:
  - "config"
  - "globals"
---

## Prerequisites

- An initialized project with `.archcore/settings.json`.
- Access to the global source repository (its documents live under its own `.archcore/`).
- Familiarity with @.archcore/globals/declaring-global-sources.rule.md.

`.archcore/global/` is **reserved**: the local document scan skips any directory named `global`, so anything you place there is invisible until you declare it (see @.archcore/globals/global-sources.spec.md §3). You do not need to configure the reservation — it already exists.

## Steps

### 1. Place the global content under `.archcore/global/<id>`

Pick the distribution mechanism:

- **Commit a vendored copy** — reproducible, travels with the repo, updated by hand:
  ```bash
  git clone <global-repo-url> .archcore/global/company
  rm -rf .archcore/global/company/.git
  git add .archcore/global/company && git commit -m "vendor company global"
  ```
- **Git submodule** — pinned to a ref, refreshed with `git submodule update`:
  ```bash
  git submodule add <global-repo-url> .archcore/global/company
  ```
- **gitignore + clone on setup** — not committed; a setup step clones it. Because every declared global is mandatory, a missing clone fails the server fast:
  ```bash
  echo ".archcore/global/" >> .gitignore
  ```

### 2. Point `path` at the directory that contains the documents

What you cloned determines `path`:

- Cloned the whole global repo (it has its own `.archcore/`): documents are at `.archcore/global/company/.archcore/knowledge/…` → `path: ".archcore/global/company/.archcore"`.
- Vendored only the document tree (`knowledge/` directly under the folder): `path: ".archcore/global/company"`.

### 3. Declare it in settings.json

```json
{
  "sync": "none",
  "globals": [ { "id": "company", "path": ".archcore/global/company" } ]
}
```

## Verification

Run the MCP server in the project and list documents:

```bash
archcore mcp   # .mcp.json args: ["mcp"]
```

`list_documents` should show the vendored documents as `source_kind: "global"`, `read_only: true`, `source_id: "company"`, with paths under `.archcore/global/company/…`. A write attempt (`update_document`) on one of them returns the clean message `cannot update a read-only global source document` — because an in-tree path is a valid `.archcore/` path and reaches the read-only guard directly. A relation attempt (`add_relation`) with one as source or target is likewise refused (`relations connect local documents only`).

Undeclared content elsewhere under `.archcore/global/` (e.g. a second folder you have not listed in `globals`) MUST NOT appear in `list_documents`, and is still read-only and non-linkable.

## Common Issues

- **Documents don't appear.** `path` points at the wrong level — it must reach the directory whose subtree contains the `*.type.md` files (often `…/.archcore` when you clone a whole repo). Re-check Step 2.
- **Server refuses to start** with `global source "company" not found at "…"` — the vendored folder is missing (e.g. gitignored and not yet cloned). Clone it: every declared global is mandatory, so the server will not start without it.
- **A local doc folder disappeared.** You named a local directory `global`; the scan skips any directory named `global` at any depth. Rename it.
- **Vendored outside `.archcore/global/`.** If you place an in-tree global under a folder *not* named `global` (e.g. the plural `.archcore/globals/<id>`) and declare it, the scan de-duplicates it — it appears once as a read-only global, never also as a writable local. It is still read-only and non-linkable. Vendoring under the reserved `.archcore/global/` directory remains the recommended, self-documenting layout.
- **In-tree vs `../` sibling.** Vendoring in-tree is the more robust form: self-contained (no clone-layout assumption) and it yields the clean read-only write message. A `../` sibling instead fails writes with `invalid path: must start with ".archcore/"`. See @.archcore/globals/global-sources-via-settings.adr.md (Consequences).
- **Updating the global.** Vendored copies and submodules are snapshots — re-clone or `git submodule update` to pull upstream changes. Automated refresh (`archcore globals pull`) is deferred; see @.archcore/features/globals-prototype-fixes.plan.md.