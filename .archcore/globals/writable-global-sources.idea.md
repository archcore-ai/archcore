---
title: "Writable Global Sources (Content-Only, Relations Stay Forbidden)"
status: draft
tags:
  - "config"
  - "globals"
  - "mcp"
---

## Idea

Allow **writing** to declared global sources through the MCP write tools — regardless
of the global's location (in-tree `.archcore/global/<id>` or external `../`/absolute) —
while keeping **relations strictly forbidden** on globals (either endpoint), exactly as
today. This overturns the current read-only-everywhere model in
@.archcore/globals/globals-are-read-only-everywhere.rule.md and §5 of
@.archcore/globals/global-sources.spec.md for *content*, but leaves the relations
invariant untouched.

Founding premise (user): **declaring a global in `settings.json` IS the consent.**
Read-only-via-mount was never a lock — the user can always open the global directly as
its own primary and edit it (the global's own MCP server doesn't even know it's mounted).
So read-only is a *router*, not a guard; the friction it imposes (switch sessions to edit)
isn't real protection. Git is the safety net; the only hard requirement is that the write
goes **through the tools** (validation, path-hardening, annotation), not via raw edits.

Why it's safe to relax for content (and was NOT for relations): a relation edge written
into a shared global's manifest **propagates to every consumer** the moment it exists. A
content write lands in the global repo's *working tree* as an uncommitted diff and reaches
other consumers only after a deliberate `commit + push` **in that repo** — a human gate
that read-only never provided. With relations kept forbidden, the propagating channel
stays closed.

## Value

- **No session switch for the common workflow**: from a local session that mounts a
  global, author/refine org-wide context in place. Both directions work in one session —
  writing to local is native (local *is* the primary), writing "up" to the global is the
  new soft-writable path. "From a global idea spin off a local doc" and "promote a local
  rule up to the global" both become single-session.
- Removes friction that wasn't buying protection (the open-directly bypass already exists).
- Roles are relative to the session's single primary: the same repo is writable-local when
  it's the primary and write-mounted-global when another primary mounts it.

## Possible Implementation

- **Split the predicate.** Today both write guards and the relation guard use one
  `isReadOnlyGlobalPath` (@internal/mcp/tools/common.go). Keep it on `add_relation`
  **unchanged** (relations stay forbidden — it's path-based, independent of `read_only`).
  Introduce a narrower *write* predicate: writable = local OR **declared** global; the
  undeclared `__global__` reserved tree stays read-only (no `id`, nowhere to target).
- **Autonomy: soft only.** Guidance in the three write-tool descriptions + system
  instructions: "write to a global only when the user explicitly asked; never as a
  proactive side effect." No `confirm_global_write` flag, no `globals_writable` master
  switch (declaration = consent).
- **Scope: create + update only; `remove` stays read-only.** Deleting a doc in a
  shared/external repo is higher-stakes and orphans the global's *own* internal relations,
  which this session cannot clean (manifest is the global's; relations-to-globals
  forbidden). See Risks / F2.
- **New addressing scheme for external writes.** `create_document`'s `directory` param is
  relative-within-primary and rejects `..` (@internal/mcp/tools/create_document.go) — it
  *cannot* express an external global. Need e.g. `source_id` + subpath. `update_document`
  is path-addressed and works once `validateArchcorePath` is relaxed.
- **Write-path hardening** ported from `validateReadPath` §4.4: `.md`-only, lexical
  containment under the declared global root (block `../` traversal), symlink-evaluated
  containment **both directions** (block a symlink escaping the global, and a local symlink
  resolving *into* a global).
- **Atomic write** (temp + rename within one FS) to mitigate torn reads under
  last-writer-wins.
- **Provenance without relations**: a non-relation frontmatter marker
  (e.g. `derived_from: "<source_id>/<slug>"`) to record a local↔global link without a
  manifest edge — OPEN. Otherwise provenance survives only via same-slug override
  (@.archcore/globals/local-overrides-global.rule.md) + prose `@`-refs.
- **Notice on success**: `wrote to global "<id>" — commit it in that repository`. The
  primary **never syncs** global writes (sync/status are local-only, §6.4).

## Risks

Accepted (by the user, as MVP trade-offs):
- **Concurrency**: no lock, last-writer-wins; the global repo's git surfaces conflicts.
- **sync/status**: primary never syncs/validates global writes — only the success notice.
- **Behavior change for existing globals users**: read-only → writable shipped *without*
  opt-in. Tolerated because relations stay locked and git is the net.

Central tension:
- **A1 — provenance gap**: the very workflow (derive local from global / promote local to
  global) wants a *link*, but relations to globals are forbidden. Content moves both ways;
  the relationship can't be a manifest relation. Motivates `derived_from`.

Code-grounded findings:
- **F1**: `create_document` structurally can't address an external global (directory param
  rejects `..`). Needs the new addressing scheme.
- **F2**: `remove_document` cleans the *primary's* manifest, not the global's
  (@internal/mcp/tools/remove_document.go) → cross-repo dangling relations in the global's
  own manifest. → keep `remove` read-only.
- **F3**: writes are non-atomic (`os.WriteFile`) → torn reads under concurrency.
- **F4**: structural validation on write exists; semantic/health of the global is not
  checked by the primary (status is local-only); `MkdirAll` creates dirs in the foreign tree.

Other corner cases to design against:
- Same-slug divergence (editing both the local override and the global it overrides).
- Path aliasing: a global declared at `../primary/.archcore/sub` is a *descendant* of own
  `.archcore` — NOT caught by self-overlap (which only flags own-or-ancestor) — aliases part
  of own tree as a writable "global".
- Nested/overlapping globals: `matchGlobal` returns the first match → nondeterministic
  ownership of the overlap for writes/annotation.
- Stale mount (global moved ahead, no "pull first"); cross-FS / OS-readonly external
  global (mid-write failure, must keep clean message — no absolute-path leak).
- Soft-autonomy failure modes: weak/non-Claude agent writes proactively; ambiguous target
  (agent in global context writes "up" by mistake); silent local→global content duplication.

## Open questions

- Keep `remove` read-only for globals? (recommended: yes)
- Adopt the `derived_from` non-relation provenance marker, or rely on same-slug + prose?
- Exact write-addressing scheme (`source_id` + subpath shape).

When ready, promote to an RFC (`/archcore:decide`) — it supersedes accepted docs
(globals-are-read-only-everywhere.rule, global-sources.spec §5, the "exactly one writable
primary" invariant) and the write-hardening warrants a `/security-review`.
