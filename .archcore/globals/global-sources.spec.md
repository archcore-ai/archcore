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
- The two-phase scan in @internal/docs/scan.go (`Scan`, `ScanFull`).
- Source-health reporting in @internal/docs/inspect.go (`InspectGlobals`, `GlobalState`).
- Source annotation and read-only enforcement across the MCP read/write tools.
- The pre-write hook guard in @cmd/hook_write_guard.go, which calls the same write-path validator.
- MCP startup validation of declared globals in @cmd/mcp.go.

The document model, the scan, and the path guards live in `internal/docs`; @internal/mcp/tools/docs_bridge.go re-exports them under the short names the MCP tool handlers use (`ScanDocuments`, `ScanDocumentsFull`, `ScanLocalDocuments`, `guardWritablePath`, `validateReadPath`, `annotateSource`, `LocalDocument`). See @.archcore/cli/docs-package-owns-the-document-model.adr.md.

## Scope

### Covers

- The `globals` array schema and per-entry validation.
- Path resolution (relative, `../`, absolute) and the reserved `.archcore/global/` directory.
- The two-phase document scan and source tagging (`source_id`, `source_kind`, `global`, `read_only`).
- Read-only enforcement on `create_document`, `update_document`, `remove_document`, and `add_relation` (relation endpoints), plus the reserved `.archcore/global/` tree.
- Source-health classification at scan time and at MCP startup (missing / not-a-directory / unreadable / self-overlap / duplicate / empty), and startup rejection of an invalid `settings.json`.
- Error messages and conditions.

### Does Not Cover

- Distribution / lifecycle (clone, submodule, `archcore globals pull`, lockfiles) — deferred, no accepted approach yet. The interim answer is in-tree vendoring (@.archcore/globals/vendoring-a-global.guide.md).
- Remote / hosted globals over a non-filesystem transport — out of scope.
- Multi-writable projects in one session — explicitly excluded; one writable primary only.
- Cross-project relations — no relation edge may reference a global document (enforced in §5.7); a global is never a relation endpoint.
- The pre-write code-alignment injection, which MAY name a global document and marks it `[global]`. See @.archcore/globals/globals-are-read-only-everywhere.rule.md.

## Authority

This document is the normative specification for global-source behavior. If the implementation, tests, or consumers diverge from it, this specification takes precedence until amended. The originating decision is @.archcore/globals/global-sources-via-settings.adr.md; that every declared source is mandatory (no `required` flag) is @.archcore/globals/globals-are-mandatory.adr.md. The consolidated, plain-language statement of what globals may and may not do is @.archcore/globals/globals-are-read-only-everywhere.rule.md.

### Related Artifacts

- Schema & validation: @internal/config/config.go (`GlobalSource`, `Settings.Globals`, `Validate`, `ReadGlobals`, `LoadGlobals`, `globalIDRe`)
- Resolution & health classification: @internal/config/globals.go (`ResolveGlobalPath`, `CheckGlobalDir`, `DescribeGlobalDirError`, `ErrGlobalMissing`/`ErrGlobalNotDir`/`ErrGlobalUnreadable`/`ErrGlobalSelfOverlap`)
- Scan: @internal/docs/scan.go (`Scan`, `ScanFull`, `ScanLocal`, `BuildDoc`)
- Annotation & global matching: @internal/docs/globals.go (`IsGlobalPath`, `IsReservedGlobalDir`, `IsExternalGlobalDocument`, `AnnotateSource`)
- Document model: @internal/docs/document.go (`Document`, aliased as `LocalDocument` by @internal/mcp/tools/docs_bridge.go)
- Source-health reporter: @internal/docs/inspect.go (`InspectGlobals`, `GlobalInspection`, `GlobalState`) — single source of truth for the startup, status, and session-start surfaces
- Directory walk: @templates/templates.go (`WalkArchcoreFilesSkipping`, `IsValidType`, `ExtractDocType`)
- Write-path guard: @internal/docs/guard.go (`GuardWritablePath`, and the package-private `checkSymlinkContainment` it layers) — consumed by @internal/mcp/tools/create_document.go, @internal/mcp/tools/update_document.go, @internal/mcp/tools/remove_document.go, and @cmd/hook_write_guard.go
- Relation guard: @internal/mcp/tools/add_relation.go (`guardWritablePath` on both endpoints)
- Local-only scan: @internal/docs/scan.go (`ScanLocal`) — phase 1 only; never fails on a broken global
- Read-path validation: @internal/docs/guard.go (`ValidateReadPath`) — `get_document` and the post-write precision advisory; admits declared external globals, hardened (§4.4)
- Source annotation on read: @internal/mcp/tools/get_document.go
- Local-only CLI surfaces: @cmd/status.go (`checkGlobalSources`), @cmd/hooks_common.go (globals excluded from status + session context; degrade to local-only on a broken global, warn on empty/invalid settings, §6.4)
- Startup validation: @cmd/mcp.go (`checkGlobals`)
- Consolidated rule: @.archcore/globals/globals-are-read-only-everywhere.rule.md

## Definitions

| Term | Definition |
| ---- | ---------- |
| Primary | The project the MCP server runs against: cwd, the single `--project` value, or `ARCHCORE_PROJECT_ROOT`. Always writable. |
| Global source | A read-only knowledge base declared by the primary in its `settings.json` `globals` array. |
| External global source | A declared source whose resolved directory sits outside the primary's `.archcore/`, because its `path` is `../…` or absolute. Its documents render with a leading `..`, which the write-path validator rejects lexically (§5.5). |
| `source_id` | The explicit `id` of a global source, the literal `"local"` for primary documents, or `"__global__"` for undeclared reserved-tree content. |
| Reserved directory | Any directory named `global` under `.archcore/`, **at any depth**, and anything inside it — read-only global mount space: skipped in the local scan, and never a write target or relation endpoint, even for undeclared content under it. The match is on whole path segments, so a sibling like `.archcore/global-ish/` is not reserved. The read scan matches the name exactly; the write guard additionally matches it **case-insensitively** (§5.4). |
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

Resolution lives in one place, `config.ResolveGlobalPath(baseDir, path)`, shared by the scan, the read-path validation, and the startup check.

1. `ResolveGlobalPath` MUST return `filepath.Clean(path)` when `path` is absolute.
2. Otherwise it MUST return `filepath.Clean(filepath.Join(baseDir, path))`. Relative `../` segments are permitted and resolve outside `baseDir`.
3. There MUST NOT be a path-escape restriction in `Settings.Validate` — `../` and absolute paths are valid by design.

### §2 Two-phase scan

`docs.Scan(baseDir)` and `docs.ScanFull(baseDir)` MUST proceed in two phases:

1. **Local phase.** Walk `baseDir/.archcore` via `WalkArchcoreFilesSkipping(archcoreDir, []string{"global"}, …)`. The directory named `global` MUST be skipped, and any document whose resolved path falls under a declared global source (§3.3) MUST be skipped. Every remaining document is tagged `source_id="local"`, `source_kind="local"`. If the walk returns `fs.ErrNotExist`, the function MUST return `(nil, nil)` — an uninitialized primary yields no documents (and therefore no globals).
2. **Global phase.** For each entry from `config.ReadGlobals(baseDir)`, resolve its directory via §1, reject a duplicate resolved path, and classify it via `config.CheckGlobalDir` — any fatal state (§6) aborts the scan with a message built only from the declared id/path. Then walk it with no skip list. **Only files whose suffix is a recognized document type (`templates.IsValidType(templates.ExtractDocType(name))`) are mounted**, so a misconfigured path cannot surface stray `.md` files as malformed documents. Every mounted document MUST be tagged `source_id=<id>`, `source_kind="global"`, `global=true`, `read_only=true`. A mid-walk I/O error MUST NOT be returned raw (it embeds an absolute path); it maps to `global source "<id>" at "<path>" is not readable`.

Documents from both phases are returned in a single flat list, in walk order (local first, then each global in declaration order).

### §3 Reserved directory

1. `.archcore/global/` MUST be excluded from the local phase (§2.1), so content vendored there is invisible unless explicitly declared in `globals`.
2. The skip matches **any** directory whose base name is `global`, at any depth of the walk. Local document directories MUST NOT be named `global` — in any case variant. The write/relation guard (§5.4) matches the same any-depth segment case-insensitively, so the skip and the guard agree: a nested `global/` directory is both invisible to reads and read-only — there is no invisible-but-writable gap.
3. Beyond the `global/`-name skip, the local phase MUST also skip any document whose resolved path falls under a declared global source (`docs.IsGlobalPath`). A global vendored in-tree **outside** `.archcore/global/` (e.g. `.archcore/globals/<id>`) is therefore scanned once — as global in phase 2 — never also as a writable local, so "exactly one `source_id` per document" holds by construction.

### §4 Source annotation

1. `docs.Document` MUST carry `source_id` (always), `source_kind` (always), `global` (omit when false), and `read_only` (omit when false). The same four fields MUST also appear on `search_documents` result rows, so all three read tools (`list_documents`, `get_document`, `search_documents`) expose the local/global distinction identically.
2. `get_document` MUST call `docs.AnnotateSource(&doc, baseDir, globals)`, which matches the document's resolved absolute path against each declared global's resolved directory **first**; on a prefix match it sets the global tags with that source's `id`. A path in the reserved `global/` tree (§3) that is NOT declared is still annotated `source_id="__global__"`, `source_kind="global"`, `global=true`, `read_only=true`, so the read label matches the write guard. The `__global__` sentinel carries underscores, which the `id` pattern (§7.2) forbids, so it can never collide with a declared source id. Otherwise `source_id="local"`, `source_kind="local"`.
3. `AnnotateSource` and the scan MUST agree: a document listed as global by one MUST be annotated global by the other. Annotation keeps **exact-case** matching (`IsReservedGlobalDir`, global path matching) — case-folding on the read path would reclassify scan results on case-sensitive filesystems; only the write guard folds (§5.4).
4. `get_document` MUST validate its `path` with `docs.ValidateReadPath(baseDir, path, ReadGlobals(baseDir))`, which accepts every path `ValidateArchcorePath` accepts **and additionally** a document that resolves strictly inside a declared external global. The external-global branch MUST be hardened: relative-only input, `.md`-only, lexical containment under a declared global root (blocks `../` traversal), and symlink-evaluated containment (blocks a symlink inside the mount from escaping it). A path under a declared global pointing at a missing file MUST yield an ordinary `document not found`. The write tools MUST NOT use this relaxation — they keep the strict guard (§5.4), so an external global stays unwritable and non-linkable (§5.5).
5. Every hook surface that opens a document at a host-supplied path MUST validate it with `docs.ValidateReadPath` as well. Lexical validation alone lets a symlinked ancestor (`.archcore/escape -> /elsewhere`) resolve out of the store, and the post-write precision advisory reports the file's own wording and its document links — so an escape puts an outside document in front of the model. The advisory passes nil globals: it fires only after a write, and a global is never written.

### §5 Read-only enforcement

1. `create_document` MUST reject a target directory under any declared global with `cannot create document in a read-only global source`.
2. `update_document` MUST reject a path under any declared global with `cannot update a read-only global source document`.
3. `remove_document` MUST reject a path under any declared global with `cannot remove a read-only global source document`.
4. The write tools MUST use the single shared validator `docs.GuardWritablePath(baseDir, relPath, globals)` (@internal/docs/guard.go), which layers, in order:
   - **(a) lexical** — `ValidateArchcorePath` (relative, `.archcore/`-prefixed, no traversal);
   - **(b) document-only** — the basename MUST end in `.md` and MUST NOT be a meta file (`templates.SkipFiles`): `settings.json` and `.sync-state.json` are never write or remove targets;
   - **(c) reserved tree, case-folded** — any path segment equal to `global` under case folding: on case-insensitive filesystems (APFS, NTFS) `.archcore/Global/x` resolves to the reserved tree on disk, so the guard MUST fold case;
   - **(d) declared globals, exact and case-folded** — fail-closed: on a case-sensitive filesystem a local directory differing from a declared global only by case is also rejected;
   - **(e) symlink containment** — the deepest existing ancestor of the target MUST resolve (via `EvalSymlinks`) inside the real `.archcore/` root; the write-side mirror of §4.4, checked BEFORE any `MkdirAll` so a symlinked directory can never route a write outside the tree.
   The reserved directory is therefore read-only even for a path under it that is not declared. Global matching runs in absolute space, so `../` and absolute global paths resolve correctly.
5. A path that does not begin with `.archcore/` is rejected earlier by `ValidateArchcorePath` with `invalid path: must start with ".archcore/"`. Consequence: writes targeting an external global fail with this path-validation message rather than the read-only message; writes targeting an in-tree global under `.archcore/global/` reach the guard and return the read-only message. These are the **write** tools; `get_document` reads an external global successfully via `ValidateReadPath` (§4.4) — read access is not blocked by this rule.
6. The guards MUST fail closed: if `config.LoadGlobals` returns an error (present-but-invalid `settings.json`), the write/relation tool MUST reject with `cannot verify global sources: settings.json is unreadable` rather than proceed.
7. **Relations.** `add_relation` MUST apply the same guard to **both** endpoints, in **either** direction. An endpoint in read-only global space — a declared global or anything in the reserved `.archcore/global/` tree, including case variants — is rejected with `cannot add a relation involving a read-only global source document — relations connect local documents only`; a non-document endpoint (non-`.md`, meta file) is rejected with `relation endpoints must be .md document files`. No relation edge may reference a global document; relations connect local documents only. `remove_relation` is exempt (removing an edge never mutates a global, and must stay available to clean up a pre-existing edge).
8. **Direct writes.** The `PreToolUse` hook guard MUST decide through the same `docs.GuardWritablePath`, so an editor writing an `.archcore/` document directly is refused for the same reason the MCP tools refuse it. A path in read-only global space is blocked there as well.
9. **Direct writes to an external global.** `GuardWritablePath` never sees a path outside `.archcore/` (§5.5), and a host hands the hook an absolute path — so the guard MUST additionally test such a path against the declared global roots via `docs.IsExternalGlobalDocument` and MUST refuse a match. Without it the MCP tools and the hook diverge on one class of path: the tools cannot address an external global at all, while the hook reads "outside the project" and allows the write. The test is case-folded (§5.4d) and covers `.md` non-meta files only, so its verdict matches `GuardWritablePath`'s non-document step; it runs only after the lexical branch declines, so an ordinary source edit never reads `settings.json` for it.

### §6 Source-health handling

Every declared global is mandatory; there is no optional source. A source is classified at resolution time — `config.CheckGlobalDir` for the filesystem state, plus a cross-entry duplicate-path check — into **fatal** (abort) and **warn** (surface, allow) states. The filesystem classification MUST live in one place (`config.ResolveGlobalPath` + `config.CheckGlobalDir`), consumed identically by the scan and by the startup/status/session-start reporter (`docs.InspectGlobals`), so startup and runtime never disagree.

**Fatal states** (a source is unusable; MUST abort the server and the scan):

- **Missing** — the resolved directory does not exist (`ErrGlobalMissing`).
- **Not a directory** — the resolved path exists but is a file (`ErrGlobalNotDir`).
- **Unreadable** — the directory cannot be read (`ErrGlobalUnreadable`). A readability probe (`os.Open` + `Readdirnames(1)`) catches this at the top level so the startup gate agrees with the runtime walk, which would otherwise pass `os.Stat` and fail only on first read. The probe opens the top directory only, so `InspectGlobals` MUST also classify a **document-walk failure** as unreadable rather than derive a state from a partial count: an unreadable subdirectory is invisible to the probe and fatal to the scan.
- **Self-overlap** — the path resolves to the primary's own `.archcore` or an ancestor of it (`ErrGlobalSelfOverlap`); it would re-mount the primary's local documents as read-only globals. Descendants (in-tree vendoring such as `.archcore/global/<id>`) are NOT flagged.
- **Duplicate path** — two declared sources resolve to the same directory; mounting both would double every document.

**Warn state** (surfaced, never fatal):

- **Empty** — the directory exists, is walkable to the end, and contains zero recognized-type documents. It contributes no documents but does NOT block; the startup, status, and SessionStart surfaces emit a visible warning naming the source.

1. **Scan time.** The scan MUST reject every fatal state with a message built only from the declared id/path (never an absolute path — see @.archcore/mcp/no-absolute-paths-in-mcp-errors.rule.md). A missing source MUST read `global source "<id>" not found at "<path>"`; the other fatal messages follow `config.DescribeGlobalDirError` (not-a-directory / not-readable / resolves-to-own-.archcore) and the duplicate-path form `global sources "<a>" and "<b>" resolve to the same path "<path>"`. An empty source MUST NOT error — it simply yields no documents.
2. **Startup time.** `checkGlobals(baseDir)` MUST, before serving, abort on any fatal state. Missing keeps `… — clone it before starting the MCP server`; the other fatal states append `… — fix .archcore/settings.json before starting the MCP server`. An empty source MUST be reported as a stderr warning and MUST NOT block startup.
3. **Invalid settings at startup.** `checkGlobals` MUST distinguish a missing `settings.json` (no globals declared — OK) from a present-but-invalid one. A parse or validation error MUST abort startup with `invalid .archcore/settings.json: …`, rather than start with globals silently dropped from the read path (`ReadGlobals` degrades to "no globals" on a parse error — exactly the silent-incomplete-context failure mandatory globals exist to prevent).
4. **Non-server surfaces MUST surface a broken or empty global, never blank silently — and without fail-fast (they must not block a session).** The SessionStart hook MUST degrade to a local-only scan (`docs.ScanLocal`, which never reads a global) on a fatal scan error and inject a visible warning naming the source, so local documents are never dropped; it MUST additionally warn on an empty source and on an invalid `settings.json` — both invisible to the full scan — via `docs.InspectGlobals` (@cmd/hooks_common.go). `archcore status` MUST report every fatal state as a visible failure (counted as an issue, non-zero exit) and an empty source as a warning (not an issue) via `checkGlobalSources`; its structural local-file checks and tag hygiene scan local documents only and never depend on the global scan. Neither aborts. (The MCP server still fails fast per §6.1–§6.2.)

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
| Global mutability via MCP or direct write | none | All global documents are read-only through the tools and through the pre-write hook guard, in-tree and external alike, and are never relation endpoints. The write guard matches global space case-insensitively (§5.4). |
| Transitive globals | not followed | The global phase reads only the primary's `globals`; a global's own `globals` is ignored. |

## Invariants

- A global source's `settings.json` is never written or read for *its* globals during a primary's scan — only the primary's `globals` is consulted.
- The same repository scanned as a primary is fully writable; scanned as another project's global it is read-only. Read-only is a property of the *mount*, not the repository.
- Every document carries exactly one `source_id`; primary documents use `"local"`.
- Two identical scans against an unchanged tree (primary + globals) MUST produce the same document set and tags.
- No global document path is ever passed to a write tool successfully — including case-variant spellings of global space on any filesystem.
- The MCP write tools and the pre-write hook guard reach the same verdict for the same path. Under `.archcore/` they call one function; for an external global, which the tools cannot address at all, the hook refuses through `IsExternalGlobalDocument` (§5.9) rather than allowing what the tools reject.
- No relation edge references a global document, in either direction; `add_relation` rejects a global on either endpoint.
- The write tools never rewrite or remove a non-document file (`settings.json`, `.sync-state.json`, non-`.md`) and never follow a symlinked ancestor outside `.archcore/` (§5.4).
- No surface opens a document at a host-supplied path without symlink-evaluated containment (§4.4, §4.5).
- The startup gate and the runtime scan classify a source identically: a source that aborts the scan also aborts startup, and vice versa. Both reach the same answer for a directory that fails only partway through a walk (§6).
- No global scan error embeds an absolute filesystem path.
- Globals are surfaced only through the MCP read tools (`list_documents`, `get_document`, `search_documents`) and the pre-write code-alignment injection, which marks them `[global]`; `archcore status` and the SessionStart context operate on local documents only. When a declared global is broken or empty, neither blanks silently: the SessionStart hook degrades to a local-only scan with a warning, and `status` reports a fatal source as a visible failure (and an empty source as a warning) while still running its structural local checks (§6.4).

## Error Handling

| Condition | Response |
| --------- | -------- |
| Global source missing (startup) | `global source "<id>" not found at "<path>" — clone it before starting the MCP server` (server refuses to start) |
| Global source missing (scan) | `global source "<id>" not found at "<path>"` (scan error) |
| Global source path is a file / not a directory | `global source "<id>" at "<path>" is not a directory` (startup + scan abort) |
| Global source unreadable, at the top level or in a subdirectory | `global source "<id>" at "<path>" is not readable` (startup + scan abort; no absolute path leaked) |
| Global resolves to the project's own `.archcore` (self-overlap) | `global source "<id>" at "<path>" resolves to the project's own .archcore` (startup + scan abort) |
| Two globals resolve to the same path (duplicate) | `global sources "<a>" and "<b>" resolve to the same path "<path>"` (startup + scan abort) |
| Global exists but holds no documents (empty) | visible warning naming the source on startup (stderr), `status`, and SessionStart; not an issue, not blocking |
| Invalid `settings.json` at MCP startup | `invalid .archcore/settings.json: …` (server refuses to start) |
| Write to global (in-tree path, including a nested `global/` directory at any depth, in any case variant) | `cannot {create,update,remove} … read-only global source …` |
| Write to an external global through MCP | `invalid path: must start with ".archcore/"` (write-path validation fires first; **reads succeed** — §4.4) |
| Direct editor write to a document inside an external global | blocked by the `PreToolUse` guard (§5.9), with the message naming the MCP document tools |
| Direct editor write to an `.archcore/` document | blocked by the `PreToolUse` guard with the message naming `create_document` / `update_document` / `remove_document` |
| Write/remove of a non-document (`settings.json`, `.sync-state.json`, non-`.md` file) | `invalid path: not a document — only .md document files can be {created,updated,removed}`; relation endpoints: `relation endpoints must be .md document files` |
| Write path whose existing ancestor resolves outside `.archcore/` (symlink escape) | `invalid path: resolves outside .archcore/` |
| Read a global via `get_document` | success — body + `read_only`/`source_id`; external paths admitted by `ValidateReadPath` (§4.4) |
| Read path escaping a global or the store (traversal / symlink / non-`.md`) | `invalid path: …` — rejected by `ValidateReadPath` hardening; a hook advisory emits nothing at all |
| Broken global on SessionStart | local-only context + visible warning naming the source (§6.4); session not blocked |
| Broken global on `archcore status` | reported as a visible failure (issue, non-zero exit) by `checkGlobalSources`; structural local checks still run (§6.4) |
| Relation touching a global (either endpoint) | `cannot add a relation involving a read-only global source document — relations connect local documents only` |
| Unreadable `settings.json` during a write/relation | `cannot verify global sources: settings.json is unreadable` (fail closed) |
| Empty/invalid/duplicate `id`, empty `path` | `Settings.Validate` error; `settings.json` load fails |

## Conformance

An implementation conforms if it satisfies all MUST/MUST NOT statements in §§1–7, the invariants, and the error-handling rows, and passes the tests in @internal/docs/guard_test.go, @internal/docs/inspect_test.go, @internal/mcp/tools/globals_test.go, @internal/mcp/tools/globals_edge_test.go, @internal/mcp/tools/globals_reserved_test.go, @internal/mcp/tools/guard_writable_path_test.go, @internal/mcp/integration/globals_test.go, the globals cases in @internal/config/config_test.go and @internal/config/globals_test.go, and the surface tests in @cmd/mcp_test.go, @cmd/status_test.go, @cmd/hooks_common_test.go, @cmd/hook_write_guard_test.go, and @internal/advisory/precision_path_test.go.

## Examples

### Sibling global (the canonical fixture)

```json
// examples/05-global-single-source/.archcore/settings.json
{ "sync": "none",
  "globals": [ { "id": "company-standards", "path": "../_global_/company-standards/.archcore" } ] }
```

`archcore mcp` run in `05-global-single-source/` → `list_documents` returns 2 local (writable) + 4 global (`read_only: true`, `source_id: company-standards`). The same `company-standards` opened directly (`archcore mcp` in `_global_/company-standards/`) returns its 4 documents as writable local. An editor writing into `../_global_/company-standards/.archcore/` from the primary is refused by the hook guard (§5.9).

### In-tree vendored global

```json
{ "sync": "none",
  "globals": [ { "id": "company", "path": ".archcore/global/company" } ] }
```

Documents at `.archcore/global/company/knowledge/*.md` are mounted read-only as `source_id: company`. Undeclared content elsewhere under `.archcore/global/` stays invisible (§3) and is still read-only and non-linkable (§5) — annotated read-only with the reserved `source_id: __global__`. A write to such a document returns the clean read-only message (§5.5). A write addressed as `.archcore/Global/company/…` is rejected identically — the guard folds case (§5.4).

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
- The directory opens, but a subdirectory below it cannot be entered → **fatal** (unreadable), even though a partial count would look healthy.
- The path resolves to a file, an unreadable directory, the project's own `.archcore`, or the same directory as another declared source → **fatal**, with the corresponding message.

## Security Considerations

- Global sources are read-only through the MCP tools and through the pre-write hook guard; no content can be created, modified, or deleted in them, and they are never relation endpoints. The write guard matches global space case-insensitively, so a case-variant path cannot bypass it on APFS/NTFS, and it covers external sources the MCP tools cannot address (§5.9).
- `path` may resolve outside the primary (`../`, absolute). This is intentional for cross-project references; mitigations are that the source is read-only, only recognized-type archcore document files are walked, the declaration is committed and PR-reviewed, and a source may not re-mount the primary's own `.archcore` (self-overlap is rejected).
- Write tools additionally refuse to follow a symlinked ancestor outside the real `.archcore/` root (§5.4e) and refuse non-document targets (§5.4b), so a repo-shipped symlink or a meta-file path cannot route a mutation outside the knowledge base. The read surfaces apply the same containment (§4.4, §4.5), so a symlink cannot route a *read* outside it either — including the hook advisories, which report what they find to the model.
- Source-health checks (missing / not-a-directory / unreadable / self-overlap / duplicate) run before serving so the agent never operates against a broken mount configuration. Error messages never embed an absolute filesystem path.
