---
title: "Source Mounts — Declaration, Read Scope, Source Annotation, and Addressed Writes"
status: draft
tags:
  - "component:cli"
  - "config"
  - "globals"
  - "hooks"
  - "mcp"
---

## Purpose & Scope

This spec is normative for source mounts: the `globals` array a project declares in `.archcore/settings.json`, the read scope that array gives a call, the `project` and `source` fields every document row and write response carries, and the verdict the write tools and the pre-tool-use hook reach for a document of a declared source. Dependents: the scan and guards in @cli/internal/docs/, the eleven tool handlers in @cli/internal/mcp/tools/, the hook leaves in @cli/cmd/, and the plugin skills that read `source`. Out of scope: the call root resolution of `project`, the session-start `GLOBALS` block, and sync.

## Surface

- `GlobalSource{id, path}` and `Settings.Globals` — @cli/internal/config/config.go.
- `ResolveGlobalPath`, `CheckGlobalDir` — @cli/internal/config/globals.go.
- `Scan`, `ScanFull`, `ScanLocal` — @cli/internal/docs/scan.go; `Document.SourceID`, `Document.SourceKind`, `Document.SourceRoot` — @cli/internal/docs/document.go.
- `GuardWritablePath`, `ValidateArchcorePath`, `ValidateReadPath` — @cli/internal/docs/guard.go.
- Row fields: `source_kind` (`"local"` or `"global"`), `source_id`, and on a global row `source_root` (the absolute root of the source project).
- Response fields: `project` (the call root) and `source` (`"local"`, a declared id, or absent), appended by the server per @.archcore/mcp/call-root-resolution.spec.md behaviors 19–22.
- Hook verdict: `writeGuard.decide` — @cli/cmd/hook_write_guard.go; root per target — @cli/cmd/hook_project_root.go.

## Normative Behavior

1. WHEN a project declares a `globals` entry, the server MUST resolve `path` to a directory that holds documents, relative, `../`, or absolute.
2. WHEN a call resolves root P, the server MUST read P's own `.archcore/` and every source P declares, in declaration order.
3. The server MUST NOT read the sources a declared source declares.
4. WHEN a document belongs to a declared source, the server MUST tag it `source_kind: "global"` and `source_id: <id>`.
5. WHEN a document belongs to the call root, the server MUST tag it `source_kind: "local"` and `source_id: "local"`.
6. WHEN a source resolves to a directory named `.archcore`, the server MUST set `source_root` on each row of that source to the parent of that directory.
7. The server MUST NOT set `source_root` on a local row.
8. WHEN a global row is returned, the server MUST keep its path relative to the call root, so a row path of an external source starts with `../` as in 0.14.
9. The server MUST NOT emit `project` or `source` on a row; both are response-level fields.
10. Every successful response MUST carry `project` and `source` per @.archcore/mcp/call-root-resolution.spec.md behaviors 19–22.
11. WHEN a write tool receives a path under the call root's `.archcore/`, the server MUST accept it without a source check.
12. IF a write path starts with `../`, THEN the server MUST reject it with a message that names `project`.
12a. IF that path lies under a declared source, THEN the message MUST also name the source id.
13. WHEN a hook payload names a document under any project's `.archcore/`, the hook MUST deny the edit and name that project's root as `project`.
13a. WHEN the session root declares that project as a source, the hook MUST also name the source id.
14. `add_relation` MUST accept two endpoints under one call root only.
15. The server MUST NOT emit a `read_only` or `global` field.
15a. The server MUST NOT read a `read_only` or `global` field from any document.
16. WHEN a `globals` path resolves under the declaring project's own `.archcore/`, the server MUST refuse to start and name the source id and the migration.

## Constraints & Invariants

- Constraint: `id` MUST match `^[a-z0-9][a-z0-9-]*$` and MUST NOT be `local`; `path` MUST NOT be empty. Rationale: `source_id` and `source` carry the id verbatim.
- Constraint: an escaping relative `path` resolves against the repository main checkout. Rationale: a linked worktree reads the same corpus.
- Invariant: one document has one write address, its project root plus a path under `.archcore/`. A global row also shows a read path relative to the call root; that path keeps the 0.14 form, and a write tool refuses it per behavior 12.
- Invariant: read-only is not a property of any document; the same repository read as a source and read as a call root carries the same body.
- Invariant: every document carries exactly one `source_id`.
- Invariant: no source directory lies under a consumer's `.archcore/`, so a consumer's local scan and its sync push never meet a source document.
- Invariant: `source_kind`, the `source` filter, `coverage`, `hits`, and `by_source` keep their 0.14 values and shapes.

## Failure Behavior

1. IF a declared source directory does not exist, THEN the server MUST serve the other sources and warn with the source id.
2. IF a declared source is a file, is unreadable, or shares a resolved path with another source, THEN the server MUST refuse to start with a message built from the id and the declared path.
3. IF a write names a document under a path that starts with `..`, THEN the server MUST answer per behavior 12 and change nothing.
4. IF `add_relation` or `remove_relation` receives endpoints under two roots, THEN the server MUST refuse and state that a relation stays inside one project.
5. IF `settings.json` is present and invalid, THEN the server MUST refuse to start with `invalid .archcore/settings.json`.
6. IF `project` names a project whose declared sources hold a fatal state, THEN the tool MUST return an error naming the source id.
7. IF a `globals` path lies under the declaring project's `.archcore/`, THEN the startup message MUST name the directory with its own `.archcore/` the source moves to.

## Conformance

An implementation is conformant when it satisfies behaviors 1–16, holds every constraint and invariant, and degrades per the failure rules. The unit cases live beside the scan, the guards, and the hook; the in-process integration suite adds a session in A reading, writing, and relating inside a source G through `project`, and a write through `../g/.archcore/x.md` refused per behavior 12.

Given A declares `{ "id": "org", "path": "../g/.archcore" }` and the session root is A,
When the agent calls `update_document(project="<root of G>", path=".archcore/concepts/logging.rule.md", …)`,
Then the file under G changes and the response carries `project: "<root of G>"` and `source: "org"`.
