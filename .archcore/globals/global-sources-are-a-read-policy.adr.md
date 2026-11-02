---
title: "Global Sources Are a Read Policy — a Write Addresses a Project by project, and No Mount Is Read-Only"
status: draft
tags:
  - "component:cli"
  - "component:plugin"
  - "config"
  - "globals"
  - "hooks"
  - "mcp"
---

## Context

Since commit 27cf4f81 every MCP tool takes a `project` argument, so a session in project A writes any project's `.archcore/` by its absolute root. Measured on 2026-11-02 on a consumer A that declares `../g/.archcore` as the global source `org`: `update_document(project=G)`, `create_document(project=G)`, and `remove_document(project=G)` succeed, the same document is `read_only: true` through the mount and `source_id: local` through `project=G`, a write through the mounted path `../g/.archcore/x.md` fails with `invalid path: must start with ".archcore/"` rather than a read-only message, and the pre-tool-use hook that denies a direct edit of G's document answers `pass project="G"` (@cli/cmd/hook_project_root.go, @cli/internal/docs/guard.go). The read-only mount of the earlier model therefore guards nothing, while five accepted documents, the server instructions (@cli/internal/mcp/server.go), and the managed instruction block (@cli/internal/agents/instructions.go) still state that a global is never written.

## Decision

Adopted, for CLI 0.15.0, the model in which a declared global source is a read policy only: a read of project P covers P and the sources P declares in `settings.json`, every document has one address — `project` plus its path under `.archcore/` — every row and every write response carries `project` and `source`, the fields `read_only` and `global` are removed, the reserved directory `.archcore/global/` is removed, a source path may not sit inside the consumer's own `.archcore/`, and a write to any project goes through its `project` with no read-only refusal.

## Alternatives Considered

1. Keep the read-only mount and add a `global_source_of_session` field to a write response that names a declared source — rejected because it keeps two addresses for one document (`../g/.archcore/x.md` read-only and `project=G` writable) and keeps a refusal whose only effect is that the agent rewrites the call with `project`.
2. Writable mounts through the mounted path, as @.archcore/globals/writable-global-sources.idea.md proposed — rejected because `create_document` cannot express a directory outside the primary, and the hook guard would have to stay strict while the tools relax, which splits one predicate into two.
3. Cross-project relation edges stored in the referencing project's manifest with `source_in` and `target_in` fields — rejected because the manifest validator (@cli/internal/sync/manifest.go) rejects unknown fields only at the root, so a CLI of 0.14 would drop the fields on its next write; the user ruled on 2026-11-02 that no edge may point at another project.
4. Keep `.archcore/global/` as the one in-tree exception — rejected because an in-tree source would stay unwritable through `project` (it holds no `.archcore/` of its own) while an external source is writable, which leaves two kinds of source with different behavior.

## Consequences

Positive:

- One write address per document: the row's `source_root` plus its path under `.archcore/`; a write through a row path that starts with `../` is refused and names the source, measured by the integration suite. [verified 2026-11-02]
- The write guards in @cli/internal/docs/guard.go lose the reserved-directory and declared-global layers; `IsReservedGlobalDir`, `IsExternalGlobalDocument`, and the five `read-only global source` messages are deleted. [expected]
- A consumer's `archcore sync` push needs no global exclusion, because no source can sit under the consumer's `.archcore/`. [expected]
- The 27 Go tests and 4 plugin tests that encode the read-only mount fail for the recorded reason and no other test changes, per the plan's acceptance criteria. [expected]

Negative:

- An agent in project A changes a source document with one call; the only signals are `source: "<id>"` in the response and the git diff in the source repository. An unattended run has no human gate. [expected]
- A project with a `globals` path under its own `.archcore/`, such as @cli/examples/08-global-in-archcore, fails to start the MCP server until the source moves to a directory with its own `.archcore/`.
- The managed instruction block moves to version 3; a host keeps the version 2 text, which forbids writing to a global, until `archcore init` rewrites it.
- A plugin older than 0.15.0 reads `source_kind` and keeps refusing to write to a source on its own; it never sees `read_only` again.

## Superseded when

- A host or the MCP protocol carries a per-call root, which would make `project` and the `source` field protocol-level instead of tool-level.
- A consumer needs a relation edge into a source document for review traceability, which reopens alternative 3 with a manifest version bump.
- A source repository needs to refuse writes from consumers, which would require a marker on the source side that the declaration rule forbids today.

## Clarifications

- 2026-11-02 — Every row and write response carries both `project` and `source` (user choice).
- 2026-11-02 — `source` is `"local"` for the session root, the declared id for a declared source, and absent for a project the session root does not declare (user choice, recommended).
- 2026-11-02 — No cross-project relation edges; the manifest stays at version 1 (user choice).
- 2026-11-02 — `.archcore/global/` is removed together with example 08; a `globals` path under the consumer's `.archcore/` is a startup error with a migration message, no warning period (user choice, recommended).
- 2026-11-02 — The write-time signal is the `source` field only; no question to the user before a write to a source (user choice).
- 2026-11-02 — Shipped in v0.15.0 together with the `project` argument (user choice).
- 2026-11-02 — A write through a path that starts with `..` is refused with a message naming the owning project and `project` (user choice, recommended).
- 2026-11-02 — The managed instruction block moves to version 3 (user choice, recommended).
- 2026-11-02 — Implementation deviations, pending the user's review: (1) a global row keeps its 0.14 read path relative to the call root, `../g/.archcore/x.md`, so a 0.14 plugin and its fixtures keep working; (2) a row carries `source_root`, not `project`, because a row-level `project` collides with the response-level key in `get_document`; (3) `source` is a response-level field only; (4) `add_relation` and `remove_relation` responses carry no `source`, because their payload uses that key for the relation endpoint; (5) the hook's external-global branch and the sync exclusion of declared sources stay as defense in depth.
