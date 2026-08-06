---
title: "Global Sources Are Declared in settings.json (No Launch Flags, No Marker)"
status: accepted
tags:
  - "config"
  - "globals"
  - "mcp"
---

## Context

A project's `.archcore/` knowledge base often needs to inherit read-only context from another knowledge base — company-wide rules, platform conventions, a monorepo root's shared docs. These "global sources" must be visible to MCP read tools (`list_documents`, `get_document`, `search_documents`) and protected from the write tools.

A hard invariant constrains the design: **a global may be referenced by a local, never the reverse — only `local → global`.** A shared global consumed by many independent repos must not accumulate back-references to any one consumer (relations, paths, markers), or it pollutes the shared resource for everyone. See @.archcore/relations/local-relations-in-sync-state.adr.md for how relations are stored per-project.

The feature went through three iterations:

1. **Committed `globals` array with an in-tree-only path restriction.** `Settings.Globals` existed but `Validate` rejected any `path` containing `../`, so a project could not reference a sibling or parent repo.
2. **Launch-flag MVP.** Repeatable `--project` flags; the first was the writable "primary" and any secondary whose `settings.json` had `global: true` was mounted read-only. This validated the "global/local is a role, not a property" model but introduced a marker on the target and positional magic on the command line.
3. **settings.json-only (this decision).** The marker and the multi-`--project` mechanism are removed; the committed `globals` array becomes the single declaration mechanism, with the `../` restriction lifted.

## Decision

Global sources are declared **only** in the consuming project's own `.archcore/settings.json`, in a `globals` array. The global target stores nothing — globality is purely "a local references it."

```json
{
  "sync": "none",
  "globals": [
    { "id": "company-global", "path": "../company-global/.archcore" }
  ]
}
```

Concretely:

1. **Primary = the project the server runs in** (cwd, or the single `--project` flag, or `ARCHCORE_PROJECT_ROOT`). It is always writable. There is no positional "first project is primary" rule and no multi-`--project`.
2. **Globals come from `config.ReadGlobals(baseDir)`** — the primary's `settings.json` `globals` array. Each entry is `{ id, path }` (see @internal/config/config.go).
3. **The target stores nothing.** There is no `global: true` field on `Settings`. The same repo opened directly is an ordinary writable local project ("local to itself").
4. **`path` points at the global's `.archcore` directory** (e.g. `../company-global/.archcore`). It may be relative — including `../` for siblings/parents — or absolute. The path-escape restriction in `Validate` is removed.
5. **Mounted read-only.** Scanned documents are tagged `source_kind: "global"`, `read_only: true`, `source_id: <id>`; write tools refuse paths under any global.
6. **Every declared global is mandatory.** A missing global aborts MCP startup; there is no optional/skip behavior. (The per-entry `required` flag from earlier iterations was later removed — see @.archcore/globals/globals-are-mandatory.adr.md.)

The normative contract is @.archcore/globals/global-sources.spec.md. Declaration standards are in @.archcore/globals/declaring-global-sources.rule.md; precedence in @.archcore/globals/local-overrides-global.rule.md.

## Alternatives Considered

### 1. Multi-`--project` flags + `global: true` marker (built as MVP, then rejected)

The first `--project` writable, secondaries with `global: true` mounted read-only.

**Rejected because:**

- The marker made a referenced global **writable in-session** when it was also the primary, which opens the door to creating a `global → local` relation edge — directly breaking the one-directional invariant. settings.json-only makes the invariant airtight by construction: the target has nowhere to store a back-reference.
- `source_id` was derived from `filepath.Base(path)`, so two different repos named `standards` collided silently. The explicit `id` field eliminates this.
- A secondary `--project` without the marker was silently dropped — a confusing no-op.
- The marker conflated two unrelated notions: "a shared KB consumed by independent repos" vs. "a monorepo root versioned with its sub-apps."
- Positional flag order decided writability — implicit and easy to get wrong.

### 2. Dynamically spawning a second MCP server per global

Run a separate `archcore` MCP server for the global source.

**Rejected because:** it doubles the tool surface (and token cost) in the agent, and each server is blind to the other — merge, precedence, dedup, and read-only enforcement all collapse back onto the agent at query time. A separate server is only justified for a **remote/hosted** global (a different transport), which is out of scope here.

### 3. Per-call `project_root` parameter on tools

From @.archcore/cli/multi-project-mcp-access.idea.md. That idea targets *querying another project's* graph ad hoc, a different concern from *mounting a read-only source*. Not adopted for globals; the idea remains open for cross-project querying.

## Consequences

**Positive:**

- The `local → global` invariant holds by construction — the global stores nothing about its consumers.
- The reference is committed and versioned: it travels with the repo, is reviewed in PRs, and `.mcp.json` stays generic (`["mcp"]`) for every project.
- `source_id` is explicit, unique (`Validate` rejects duplicates), and collision-free.
- "Monorepo root = global" and "root = local" collapse into one mechanism: the root is local-to-itself and global only to whoever references `../../.archcore`.
- This is largely a **removal** of code (flag plumbing, marker serialization, the variadic scan parameter), not an addition.

**Negative / constraints:**

- **No multi-write.** Exactly one writable primary per server. Writing two independent local projects in one session (the "workspace" case) is not supported. A future upgrade could make a referenced project writable only when it shares the primary's VCS root.
- **"Where is the global on disk" is unsolved.** A relative `../company-global/.archcore` assumes a fixed clone layout; an absolute path is machine-specific. Distribution (clone, submodule, pull) is deferred with no accepted approach; the self-contained interim answer is in-tree vendoring (@.archcore/globals/vendoring-a-global.guide.md).
- **Write-attempt error message is uneven.** A write to a global declared via a `../` path fails with `invalid path: must start with ".archcore/"` (the path-validation layer fires before the read-only guard, because the doc renders as `../company-global/...`). An in-tree global under `.archcore/global/` instead returns the clean `cannot update a read-only global source document`. This affects **writes only** — `get_document` reads such an external global successfully via `ValidateReadPath` (see @.archcore/globals/global-sources.spec.md §4.4); the uneven message is a write-path cosmetic, not a read limitation.

**Neutral:**

- In-tree vendoring under the reserved `.archcore/global/` directory remains fully supported and is the most self-contained distribution form (no clone-layout assumption, clean read-only message). See @.archcore/globals/vendoring-a-global.guide.md.
