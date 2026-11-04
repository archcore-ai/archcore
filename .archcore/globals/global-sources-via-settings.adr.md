---
title: "Global Sources Are Declared in settings.json (No Launch Flags, No Marker)"
status: accepted
tags:
  - "component:cli"
  - "config"
  - "globals"
  - "mcp"
---

## Context

A project's `.archcore/` knowledge base often needs to inherit context from another knowledge base — company-wide rules, platform conventions, a monorepo root's shared docs. These "global sources" must be visible to MCP read tools (`list_documents`, `get_document`, `search_documents`).

A hard invariant constrains the design: **a global may be referenced by a local, never the reverse — only `local → global`.** A shared global consumed by many independent repos must not accumulate back-references to any one consumer (relations, paths, markers), or it pollutes the shared resource for everyone. See @.archcore/relations/local-relations-in-sync-state.adr.md for how relations are stored per-project.

The feature went through three iterations:

1. **Committed `globals` array with an in-tree-only path restriction.** `Validate` rejected any `path` containing `../`, so a project could not reference a sibling or parent repo.
2. **Launch-flag MVP.** Repeatable `--project` flags; the first was the primary and any secondary whose `settings.json` had `global: true` was mounted. This introduced a marker on the target and positional magic on the command line.
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

1. **Primary = the project the server runs in** (cwd, the single `--project` flag, or `ARCHCORE_PROJECT_ROOT`). There is no positional "first project is primary" rule and no multi-`--project`.
2. **Globals come from the primary's `settings.json`.** Each entry is `{ id, path }`.
3. **The target stores nothing.** There is no `global: true` field on `Settings`. The same repo opened directly is an ordinary local project ("local to itself").
4. **`path` points at the global's `.archcore` directory** (e.g. `../company-global/.archcore`). It may be relative — including `../` for siblings and parents — or absolute.

The normative contract is @.archcore/globals/source-mounts.spec.md. Precedence is in @.archcore/globals/local-overrides-source.rule.md. The write model is in @.archcore/globals/global-sources-are-a-read-policy.adr.md. A missing source is a warning: @.archcore/globals/missing-global-degrades-to-local.adr.md.

## Alternatives Considered

### 1. Multi-`--project` flags + `global: true` marker (built as MVP, then rejected)

**Rejected because:**

- The marker allowed a referenced global to be the primary of another session, which opens the door to a `global → local` relation edge — directly breaking the one-directional invariant. settings.json-only makes the invariant airtight by construction: the target has nowhere to store a back-reference.
- `source_id` was derived from `filepath.Base(path)`, so two different repos named `standards` collided silently. The explicit `id` field eliminates this.
- A secondary `--project` without the marker was silently dropped — a confusing no-op.
- The marker conflated two unrelated notions: "a shared KB consumed by independent repos" vs. "a monorepo root versioned with its sub-apps."
- Positional flag order decided the role — implicit and easy to get wrong.

### 2. Dynamically spawning a second MCP server per global

**Rejected because:** it doubles the tool surface (and token cost) in the agent, and each server is blind to the other — merge, precedence, and dedup all collapse back onto the agent at query time. A separate server is only justified for a **remote/hosted** global (a different transport), which is out of scope here.

### 3. Per-call project parameter on tools

A per-call root was first judged a different concern from mounting a source. It was adopted later for another reason: @.archcore/mcp/explicit-target-project-per-call.adr.md.

## Consequences

**Positive:**

- The `local → global` invariant holds by construction — the global stores nothing about its consumers.
- The reference is committed and versioned: it travels with the repo, is reviewed in PRs, and `.mcp.json` stays generic (`["mcp"]`) for every project.
- `source_id` is explicit, unique (`Validate` rejects duplicates), and collision-free.
- "Monorepo root = global" and "root = local" collapse into one mechanism: the root is local-to-itself and global only to whoever references `../../.archcore`.
- This is largely a **removal** of code (flag plumbing, marker serialization, the variadic scan parameter), not an addition.

**Negative / constraints:**

- **"Where is the global on disk" is unsolved.** A relative `../company-global/.archcore` assumes a fixed clone layout; an absolute path is machine-specific. Distribution (clone, submodule, pull) is deferred with no accepted approach; the self-contained interim answer is in-tree vendoring (@.archcore/globals/vendoring-a-global.guide.md).
