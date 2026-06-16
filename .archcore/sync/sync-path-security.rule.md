---
title: "Sync Paths Must Be Validated Against Traversal Attacks"
status: accepted
tags:
  - "sync"
---

## Rule

All file paths in sync payloads and manifests MUST be validated to prevent path traversal. The `validateRelPath()` function enforces this at every boundary.

## Validation Requirements

- Paths MUST NOT contain `..` segments (after `filepath.Clean`)
- Paths MUST NOT be absolute (no leading `/`)
- Paths MUST start with a valid category prefix: `vision/`, `knowledge/`, or `experience/`
- Paths MUST NOT contain `//` (double slashes) or end with `/`
- Only `.md` files in the top level of category directories are scanned — no subdirectory traversal

## Where Validation Occurs

- **Manifest loading** — every path in `files` map is validated via `validateFileEntry`
- **Payload construction** — every path in diff entries is validated via `validateRelPath`
- **File scanning** — `ScanFiles` only reads from the three known category directories using `os.ReadDir` (non-recursive)

## Rationale

The sync payload includes full file content that gets written/indexed on the server. A crafted path like `../../etc/passwd` or an absolute path could trick the server into reading or writing outside the expected document scope. Validating at every boundary (local scan, manifest, payload) provides defense in depth.