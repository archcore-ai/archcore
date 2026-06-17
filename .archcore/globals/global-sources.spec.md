---
title: "Global Sources Contract"
status: accepted
tags:
  - "config"
  - "globals"
  - "mcp"
---

## Purpose

This specification defines the canonical contract for **global sources** — read-only external knowledge bases mounted into a local `.archcore/` project for read operations.

It is normative for:

- The declaration schema in `Settings.Globals` (@internal/config/config.go).
- Path resolution and source-health classification in @internal/config/globals.go (`ResolveGlobalPath`, `CheckGlobalDir`, `DescribeGlobalDirError`).
- The two-phase scan in @internal/mcp/tools/common.go (`ScanDocuments`, `ScanDocumentsFull`).
- Source-health reporting in @internal/mcp/tools/globals_inspect.go (`InspectGlobals`, `GlobalState`).
- Source annotation and read-only enforcement across the MCP read/write tools.
- MCP startup validation of declared globals in @cmd/mcp.go.

## Scope

### Covers

- The `globals` array schema and per-entry validation.
- Path resolution (relative, `../`, absolute) and the reserved `.archcore/global/` directory.
- The two-phase document scan and source tagging (`source_id`, `source_kind`, `global`, `read_only`).
- Read-only enforcement on `create_document`, `update_document`, `remove_document`, and `add_relation` (relation endpoints), plus the reserved `.archcore/global/` tree.
- Source-health classification at scan time and at MCP startup (missing / not-a-directory / unreadable / self-overlap / duplicate / empty), and startup rejection of an invalid `settings.json`.
- Error messages and conditions.

### Does Not Cover

- Distribution / lifecycle (clone, submodule, `archcore globals pull`, lockfiles) — deferred; see @.archcore/features/globals-prototype-fixes.plan.md.
- Remote / hosted globals over a non-filesystem transport — out of scope.
- Multi-writable projects in one session — explicitly excluded; one writable primary only.
- Cross-project relations — no relation edge may reference a global document (enforced in §5.7); a global is never a relation endpoint.

## Authority

This document is the normative specification for global-source behavior. If the implementation, tests, or consumers diverge from it, this specification takes precedence until amended. The originating decision is @.archcore/globals/global-sources-via-settings.adr.md; that every declared source is mandatory (no `required` flag) is @.archcore/globals/globals-are-mandatory.adr.md. The consolidated, plain-language statement of what globals may and may not do is @.archcore/globals/globals-are-read-only-everywhere.rule.md.

### Related Artifacts

- Schema & validation: @internal/config/config.go (`GlobalSource`, `Settings.Globals`, `Validate`, `ReadGlobals`, `LoadGlobals`, `globalIDRe`)
- Resolution & health classification: @internal/config/globals.go (`ResolveGlobalPath`, `CheckGlobalDir`, `DescribeGlobalDirError`, `ErrGlobalMissing`/`ErrGlobalNotDir`/`ErrGlobalUnreadable`/`ErrGlobalSelfOverlap`)
- Scan & annotation: @internal/mcp/tools/common.go (`scanDocuments`, `resolveGlobalPath`, `matchGlobal`, `isGlobalPath`, `isReservedGlobalDir`, `isReadOnlyGlobalPath`, `annotateSource`, `LocalDocument`)
- Source-health reporter: @internal/mcp/tools/globals_inspect.go (`InspectGlobals`, `GlobalInspection`, `GlobalState`) — single source of truth for the startup, status, and session-start surfaces
- Directory walk: @templates/templates.go (`WalkArchcoreFilesSkipping`, `IsValidType`, `ExtractDocType`)
- Read-only guards: @internal/mcp/tools/create_document.go, @internal/mcp/tools/update_document.go, @internal/mcp/tools/remove_document.go
- Relation guard: @internal/mcp/tools/add_relation.go (`isReadOnlyGlobalPath` on both endpoints)
- Local-only scan: @internal/mcp/tools/common.go (`ScanLocalDocuments`, `scanLocalDocuments`) — phase 1 only; never fails on a broken global
- Read-path validation: @internal/mcp/tools/common.go (`validateReadPath`) — `get_document` only; admits declared external globals, hardened (§4.4)
- Source annotation on read: @internal/mcp/tools/get_document.go
- Local-only CLI surfaces: @cmd/status.go (`checkGlobalSources`), @cmd/hooks_common.go (globals excluded from status + session context; degrade to local-only on a broken global, warn on empty/invalid settings, §6.4)
- Startup validation: @cmd/mcp.go (`checkGlobals`)
- Consolidated rule: @.archcore/globals/globals-are-read-only-everywhere.rule.md

## Definitions

| Term | Definition |
| ---- | ---------- |
| Primary | The project the MCP server runs against: cwd, the single `--project` value, or `ARCHCORE_PROJECT_ROOT`. Always writable. |
| Global source | A read-only knowledge base declared by the primary in its `settings.json` `globals` array. |
| `source_id` | The explicit `id` of a global source, the literal `"local"` for primary documents, or `"__global__"` for undeclared reserved-tree content. |
| Reserved directory | Any directory named `global` under `.archcore/`, **at any depth**, and anything inside it — read-only global mount space: skipped in the local scan, and never a write target or relation endpoint (`isReservedGlobalDir`), even for undeclared content under it. The match is on whole path segments, so a sibling like `.archcore/global-ish/` is not reserved. |
| Document root | The `.archcore` directory a `path` resolves to; documents live under it (`<root>/knowledge/x.rule.md`). |
| Fatal state | A declared source that is unusable — missing, not-a-directory, unreadable, self-overlapping, or a duplicate path. Aborts the MCP server and the scan; a visible issue on `status`. |
| Empty state | A declared source whose directory exists and is readable but holds no recognized-type documents. A warning, never fatal. |

## Declaration Schema

Each entry of the `globals` array is a `GlobalSource`:

| Field | Type | Required | Description |
| ----- | ---- | -------- | ----------- |
| `id` | string | yes | Stable identifier; becomes `source_id` on every document from this source. |
| `path` | string | yes | Points at the global source's `.archcore` directory. Relative (incl. `../`), absolute, or in-tree. |

Every declared source is **mandatory at runtime** — there is no per-entry opt-out; a fatal source fails fast (§6). The array is serialized with `json:"globals,omitempty"` and is permitted for every sync type (`allowedFields`), never required.

## Normative Behavior

### §1 Path resolution

Resolution lives in one place, `config.ResolveGlobalPath(baseDir, path)`, shared by the scan, the read-path validation, and the startup check (the `tools.resolveGlobalPath` wrapper delegates to it).

1. `ResolveGlobalPath` MUST return `filepath.Clean(path)` when `path` is absolute.
2. Otherwise it MUST return `filepath.Clean(filepath.Join(baseDir, path))`. Relative `../` segments are permitted and resolve outside `baseDir`.
3. There MUST NOT be a path-escape restriction in `Settings.Validate` — `../` and absolute paths are valid by design.

### §2 Two-phase scan

`scanDocuments(baseDir, includeContent)` MUST proceed in two phases:

1. **Local phase.** Walk `baseDir/.archcore` via `WalkArchcoreFilesSkipping(archcoreDir, []string{"global"}, …)`. The directory named `global` MUST be skipped, and any document whose resolved path falls under a declared global source (§3.3) MUST be skipped. Every remaining document is tagged `source_id="local"`, `source_kind="local"`. If the walk returns `fs.ErrNotExist`, the function MUST return `(nil, nil)` — an uninitialized primary yields no documents (and therefore no globals).
2. **Global phase.** For each entry from `config.ReadGlobals(baseDir)`, resolve its directory via §1, reject a duplicate resolved path, and classify it via `config.CheckGlobalDir` — any fatal state (§6) aborts the scan with a message built only from the declared id/path. Then walk it with no skip list. **Only files whose suffix is a recognized document type (`templates.IsValidType(templates.ExtractDocType(name))`) are mounted**, so a misconfigured path cannot surface stray `.md` files as malformed documents. Every mounted document MUST be tagged `source_id=<id>`, `source_kind="global"`, `global=true`, `read_only=true`. A mid-walk I/O error MUST NOT be returned raw (it embeds an absolute path); it maps to `global source "<id>" at "<path>" is not readable`.

Documents from both phases are returned in a single flat list, in walk order (local first, then each global in declaration order).

### §3 Reserved directory

1. `.archcore/global/` MUST be excluded from the local phase (§2.1), so content vendored there is invisible unless explicitly declared in `globals`.
2. The skip matches **any** directory whose base name is `global`, at any depth of the walk. Local document directories MUST NOT be named `global`. The write/relation guard (`isReservedGlobalDir`, §5.4) matches the same any-depth segment, so the skip and the guard agree: a nested `global/` directory is both invisible to reads and read-only — there is no invisible-but-writable gap.
3. Beyond the `global/`-name skip, the local phase MUST also skip any document whose resolved path falls under a declared global source (`isGlobalPath`). A global vendored in-tree **outside** `.archcore/global/` (e.g. `.archcore/globals/<id>`) is therefore scanned once — as global in phase 2 — never also as a writable local, so "exactly one `source_id` per document" holds by construction.

### §4 Source annotation

1. `LocalDocument` MUST carry `source_id` (always), `source_kind` (always), `global` (omit when false), and `read_only` (omit when false). The same four fields MUST also appear on `search_documents` result rows, so all three read tools (`list_documents`, `get_document`, `search_documents`) expose the local/global distinction identically.
2. `get_document` MUST call `annotateSource(&doc, baseDir)`, which matches the document's resolved absolute path against each declared global's resolved directory **first**; on a prefix match it sets the global tags with that source's `id`. A path in the reserved `global/` tree (§3) that is NOT declared is still annotated `source_id="__global__"`, `source_kind="global"`, `global=true`, `read_only=true`, so the read label matches the write guard (`isReadOnlyGlobalPath`). The `__global__` sentinel carries underscores, which the `id` pattern (§7.2) forbids, so it can never collide with a declared source id. Otherwise `source_id="local"`, `source_kind="local"`.
3. `annotateSource` and `scanDocuments` MUST agree: a document listed as global by one MUST be annotated global by the other.
4. `get_document` MUST validate its `path` with `validateReadPath(baseDir, path, ReadGlobals(baseDir))`, which accepts every path `validateArchcorePath` accepts **and additionally** a document that resolves strictly inside a declared external global (rendered with a leading `..` because its `path` is `../…` or absolute). The external-global branch MUST be hardened: relative-only input, `.md`-only, lexical containment under a declared global root (blocks `../` traversal), and symlink-evaluated containment (blocks a symlink inside the mount from escaping it). A path under a declared global pointing at a missing file MUST yield an ordinary `document not found`. The write tools MUST NOT use this relaxation — they keep `validateArchcorePath`, so an external global stays unwritable and non-linkable (§5.5).

### §5 Read-only enforcement

1. `create_document` MUST reject a target directory under any declared global with `cannot create document in a read-only global source`.
2. `update_document` MUST reject a path under any declared global with `cannot update a read-only global source document`.
3. `remove_document` MUST reject a path under any declared global with `cannot remove a read-only global source document`.
4. The guard MUST use the single predicate `isReadOnlyGlobalPath(baseDir, relPath, globals)` = declared global (`isGlobalPath`, which matches in absolute space so `../` and absolute global paths resolve correctly) **OR** inside a reserved `global/` tree — any directory named `global` under `.archcore/`, at any depth (`isReservedGlobalDir`). The reserved directory is therefore read-only even for a path under it that is not declared. This matches the any-depth skip the local scan applies (§3.2), so a `global/` segment hides a document from reads AND makes it read-only — never one without the other.
5. A path that does not begin with `.archcore/` is rejected earlier by `validateArchcorePath` with `invalid path: must start with ".archcore/"`. Consequence: writes targeting a global declared via a `../` path fail with this path-validation message rather than the read-only message; writes targeting an in-tree global under `.archcore/global/` reach the guard and return the read-only message. These are the **write** tools; `get_document` reads an external `../`/absolute global successfully via `validateReadPath` (§4.4) — read access is not blocked by this rule.
6. The guards MUST fail closed: if `config.LoadGlobals` returns an error (present-but-invalid `settings.json`), the write/relation tool MUST reject with `cannot verify global sources: settings.json is unreadable` rather than proceed.
7. **Relations.** `add_relation` MUST reject an edge whose source **or** target is read-only-global (`isReadOnlyGlobalPath`), in **either** direction — a declared global or anything in the reserved `.archcore/global/` tree — with `cannot add a relation involving a read-only global source document — relations connect local documents only`. No relation edge may reference a global document; relations connect local documents only. `remove_relation` is exempt (removing an edge never mutates a global, and must stay available to clean up a pre-existing edge).

### §6 Source-health handling

Every declared global is mandatory; there is no optional source. A source is classified at resolution time — `config.CheckGlobalDir` for the filesystem state, plus a cross-entry duplicate-path check — into **fatal** (abort) and **warn** (surface, allow) states. The filesystem classification MUST live in one place (`config.ResolveGlobalPath` + `config.CheckGlobalDir`), consumed identically by the scan (`scanDocuments`) and by the startup/status/session-start reporter (`tools.InspectGlobals`), so startup and runtime never disagree.

**Fatal states** (a source is unusable; MUST abort the server and the scan):

- **Missing** — the resolved directory does not exist (`ErrGlobalMissing`).
- **Not a directory** — the resolved path exists but is a file (`ErrGlobalNotDir`).
- **Unreadable** — the directory exists but cannot be read (`ErrGlobalUnreadable`). A readability probe (`os.Open` + `Readdirnames(1)`) catches this so the startup gate agrees with the runtime walk, which would otherwise pass `os.Stat` and fail only on first read.
- **Self-overlap** — the path resolves to the primary's own `.archcore` or an ancestor of it (`ErrGlobalSelfOverlap`); it would re-mount the primary's local documents as read-only globals. Descendants (in-tree vendoring such as `.archcore/global/<id>`) are NOT flagged.
- **Duplicate path** — two declared sources resolve to the same directory; mounting both would double every document.

**Warn state** (surfaced, never fatal):

- **Empty** — the directory exists and is readable but contains zero recognized-type documents. It contributes no documents but does NOT block; the startup, status, and SessionStart surfaces emit a visible warning naming the source.

1. **Scan time.** `scanDocuments` MUST reject every fatal state with a message built only from the declared id/path (never an absolute path — see @.archcore/mcp/no-absolute-paths-in-mcp-errors.rule.md). A missing source MUST read `global source "<id>" not found at "<path>"`; the other fatal messages follow `config.DescribeGlobalDirError` (not-a-directory / not-readable / resolves-to-own-.archcore) and the duplicate-path form `global sources "<a>" and "<b>" resolve to the same path "<path>"`. An empty source MUST NOT error — it simply yields no documents.
2. **Startup time.** `checkGlobals(baseDir)` MUST, before serving, abort on any fatal state. Missing keeps `… — clone it before starting the MCP server`; the other fatal states append `… — fix .archcore/settings.json before starting the MCP server`. An empty source MUST be reported as a stderr warning and MUST NOT block startup.
3. **Invalid settings at startup.** `checkGlobals` MUST distinguish a missing `settings.json` (no globals declared — OK) from a present-but-invalid one. A parse or validation error MUST abort startup with `invalid .archcore/settings.json: …`, rather than start with globals silently dropped from the read path (`ReadGlobals` degrades to "no globals" on a parse error — exactly the silent-incomplete-context failure mandatory globals exist to prevent).
4. **Non-server surfaces MUST surface a broken or empty global, never blank silently — and without fail-fast (they must not block a session).** The SessionStart hook MUST degrade to a local-only scan (`ScanLocalDocuments`, which never reads a global) on a fatal scan error and inject a visible warning naming the source, so local documents are never dropped; it MUST additionally warn on an empty source and on an invalid `settings.json` — both invisible to `ScanDocuments` (`ReadGlobals` silently drops globals on invalid settings) — via `tools.InspectGlobals` (@cmd/hooks_common.go). `archcore status` MUST report every fatal state as a visible failure (counted as an issue, non-zero exit) and an empty source as a warning (not an issue) via `checkGlobalSources`; its structural local-file checks and tag hygiene scan local documents only (`ScanLocalDocuments`) and never depend on the global scan. Neither aborts. (The MCP server still fails fast per §6.1–§6.2.)

The decision to drop the per-entry `required` flag is @.archcore/globals/globals-are-mandatory.adr.md.

### §7 Validation

`Settings.Validate` MUST enforce, per `globals` entry:

1. Non-empty `id` → else `globals[i]: "id" must not be empty`.
2. `id` matches `^[a-z0-9][a-z0-9-]*$` → else a lowercase-alphanumeric-with-hyphens error.
3. `id` is not the reserved `local` → else `globals[i]: "id" "local" is reserved`. (The reserved-tree sentinel `__global__` (§4) needs no separate reservation: it contains underscores, which rule 2's pattern already forbids in a declared `id`, so it can never be declared.)
4. Non-empty `path` → else `globals[i]: "path" must not be empty`.
5. No duplicate `id` across entries → else `globals[i] and globals[j]: duplicate id "<id>"`.

`Validate` MUST NOT reject `path` for containing `../` or being absolute. Self-overlap and duplicate-**path** detection are resolution-time checks (§6), not `Validate` checks — they require `baseDir` to resolve `path`, which `Settings.Validate` does not have.

## Constraints

| Constraint | Value | Rationale |
| ---------- | ----- | --------- |
| Writable projects per server | exactly 1 (the primary) | Preserves the `local → global` invariant; no `global → local` edges possible. |
| `source_id` uniqueness | enforced by `Validate` | Distinct sources remain distinguishable; no basename collision. |
| Resolved-path uniqueness | enforced at resolution (§6) | Two sources at one directory would double every document. |
| Path traversal | unrestricted (`../`, absolute) | Cross-project references are the core use case. |
| Self-overlap | rejected (§6) | A global may not re-mount the primary's own `.archcore`. |
| Declared source presence | mandatory | A fatal source (missing / not-a-directory / unreadable / self-overlap / duplicate) fails fast (§6); an empty source warns. No silent-skip. |
| Source classification | shared (`config.CheckGlobalDir`) | Startup and runtime classify identically; no startup-passes-but-runtime-fails gap. |
| Global mutability via MCP | none | All global documents are read-only through the tools, and are never relation endpoints. |
| Transitive globals | not followed | The global phase reads only the primary's `globals`; a global's own `globals` is ignored. |

## Invariants

- A global source's `settings.json` is never written or read for *its* globals during a primary's scan — only the primary's `globals` is consulted.
- The same repository scanned as a primary is fully writable; scanned as another project's global it is read-only. Read-only is a property of the *mount*, not the repository.
- Every document carries exactly one `source_id`; primary documents use `"local"`.
- Two identical scans against an unchanged tree (primary + globals) MUST produce the same document set and tags.
- No global document path is ever passed to a write tool successfully.
- No relation edge references a global document, in either direction; `add_relation` rejects a global on either endpoint.
- The startup gate and the runtime scan classify a source identically: a source that aborts the scan also aborts startup, and vice versa (both call `config.CheckGlobalDir`).
- No global scan error embeds an absolute filesystem path.
- Globals are surfaced only through the MCP read tools (`list_documents`, `get_document`, `search_documents`); `archcore status` and the SessionStart context operate on local documents only. When a declared global is broken or empty, neither blanks silently: the SessionStart hook degrades to a local-only scan with a warning, and `status` reports a fatal source as a visible failure (and an empty source as a warning) while still running its structural local checks (§6.4).

## Error Handling

| Condition | Response |
| --------- | -------- |
| Global source missing (startup) | `global source "<id>" not found at "<path>" — clone it before starting the MCP server` (server refuses to start) |
| Global source missing (scan) | `global source "<id>" not found at "<path>"` (scan error) |
| Global source path is a file / not a directory | `global source "<id>" at "<path>" is not a directory` (startup + scan abort) |
| Global source directory unreadable | `global source "<id>" at "<path>" is not readable` (startup + scan abort; no absolute path leaked) |
| Global resolves to the project's own `.archcore` (self-overlap) | `global source "<id>" at "<path>" resolves to the project's own .archcore` (startup + scan abort) |
| Two globals resolve to the same path (duplicate) | `global sources "<a>" and "<b>" resolve to the same path "<path>"` (startup + scan abort) |
| Global exists but holds no documents (empty) | visible warning naming the source on startup (stderr), `status`, and SessionStart; not an issue, not blocking |
| Invalid `settings.json` at MCP startup | `invalid .archcore/settings.json: …` (server refuses to start) |
| Write to global (in-tree path, including a nested `global/` directory at any depth) | `cannot {create,update,remove} … read-only global source …` |
| Write to global (`../` path) | `invalid path: must start with ".archcore/"` (write-path validation fires first; **reads succeed** — §4.4) |
| Read a global via `get_document` | success — body + `read_only`/`source_id`; external `../`/absolute paths admitted by `validateReadPath` (§4.4) |
| Read path escaping a global (traversal / symlink / non-`.md`) | `invalid path: …` — rejected by `validateReadPath` hardening |
| Broken global on SessionStart | local-only context + visible warning naming the source (§6.4); session not blocked |
| Broken global on `archcore status` | reported as a visible failure (issue, non-zero exit) by `checkGlobalSources`; structural local checks still run (§6.4) |
| Relation touching a global (either endpoint) | `cannot add a relation involving a read-only global source document — relations connect local documents only` |
| Unreadable `settings.json` during a write/relation | `cannot verify global sources: settings.json is unreadable` (fail closed) |
| Empty/invalid/duplicate `id`, empty `path` | `Settings.Validate` error; `settings.json` load fails |

## Conformance

An implementation conforms if it satisfies all MUST/MUST NOT statements in §§1–7, the invariants, and the error-handling rows, and passes the tests in @internal/mcp/tools/globals_test.go, @internal/mcp/tools/globals_edge_test.go, @internal/mcp/integration/globals_test.go, the globals cases in @internal/config/config_test.go and @internal/config/globals_test.go, and the CLI-surface tests in @cmd/mcp_test.go, @cmd/status_test.go, and @cmd/hooks_common_test.go.

## Examples

### Sibling global (the canonical fixture)

```json
// examples/05-global-single-source/.archcore/settings.json
{ "sync": "none",
  "globals": [ { "id": "company-standards", "path": "../_global_/company-standards/.archcore" } ] }
```

`archcore mcp` run in `05-global-single-source/` → `list_documents` returns 2 local (writable) + 4 global (`read_only: true`, `source_id: company-standards`). The same `company-standards` opened directly (`archcore mcp` in `_global_/company-standards/`) returns its 4 documents as writable local.

### In-tree vendored global

```json
{ "sync": "none",
  "globals": [ { "id": "company", "path": ".archcore/global/company" } ] }
```

Documents at `.archcore/global/company/knowledge/*.md` are mounted read-only as `source_id: company`. Undeclared content elsewhere under `.archcore/global/` stays invisible (§3) and is still read-only and non-linkable (§5) — annotated read-only with the reserved `source_id: __global__`. A write to such a document returns the clean read-only message (§5.5).

### Several globals (no collision)

```json
{ "globals": [
  { "id": "a-standards", "path": "../a/standards/.archcore" },
  { "id": "b-standards", "path": "../b/standards/.archcore" } ] }
```

Both directories are named `standards`, but `source_id` is the explicit `id`, so the two sources remain distinct.

### Broken vs empty source

```json
{ "globals": [ { "id": "company", "path": "../company/.archcore" } ] }
```

- The directory does not exist → **fatal** (missing): the MCP server refuses to start, the scan errors, and `status` reports an issue. The message names `company` and `../company/.archcore`, never an absolute path.
- The directory exists and is readable but holds no documents → **warn** (empty): the server starts, the scan yields zero global documents, and startup / `status` / SessionStart each emit a warning naming `company`.
- The path resolves to a file, an unreadable directory, the project's own `.archcore`, or the same directory as another declared source → **fatal**, with the corresponding message.

## Security Considerations

- Global sources are read-only through the MCP tools; no content can be created, modified, or deleted in them, and they are never relation endpoints.
- `path` may resolve outside the primary (`../`, absolute). This is intentional for cross-project references; mitigations are that the source is read-only, only recognized-type archcore document files are walked, the declaration is committed and PR-reviewed, and a source may not re-mount the primary's own `.archcore` (self-overlap is rejected).
- Source-health checks (missing / not-a-directory / unreadable / self-overlap / duplicate) run before serving so the agent never operates against a broken mount configuration. Error messages never embed an absolute filesystem path.
