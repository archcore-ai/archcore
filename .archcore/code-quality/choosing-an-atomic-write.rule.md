---
title: "Which Atomic Write to Call, and Never a Fourth Copy"
status: accepted
tags:
  - "code-quality"
  - "component:cli"
  - "golang"
---

## Rule

1. A write that must not be observable half-finished MUST go through one of the three helpers below,
   chosen by what is being written.
2. The developer MUST NOT write a fourth temp-file-plus-rename sequence inline.
3. IF the target is archcore-owned state — the sync manifest, the freshness cache, a config archcore
   writes in full — THEN the developer MUST call `jsonfile.WriteAtomic`.
4. IF the target is an `.archcore/` document, THEN the developer MUST call `docs.WriteFileAtomic`,
   which additionally invalidates the scan cache for that path.
5. IF the target is a file the user owns — `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, a host config —
   THEN the developer MUST call `jsonfile.WriteThroughSymlink`, directly or through the writer in
   `@cli/internal/agents/instructions.go`, which follows a symlink before replacing.
6. WHEN a new target fits none of the three, the developer MUST extend the closest helper rather than
   add a variant, and MUST state in its doc comment what the new case needs that the others do not.
7. IF a write establishes exclusivity between processes, THEN the developer MUST create the file with
   `os.O_CREATE|os.O_EXCL`.
8. The developer MUST NOT convert a claim in `@cli/internal/stamp/` to a temp-file-plus-rename write.

## Rationale

Crash safety is one primitive, and a copy of it is a place a fix can fail to land. The manifest
carried a byte-for-byte duplicate of `jsonfile.WriteAtomic` for no reason other than that
`internal/sync` did not import `internal/jsonfile`; `jsonfile`'s own doc comment admitted the
duplication by pointing at it.

The three that remain are not redundant — they differ in what they must preserve:

| Target | Helper | What it adds |
|---|---|---|
| archcore-owned state | `jsonfile.WriteAtomic` | the primitive: per-attempt temp + rename, existing mode kept, a new file at 0o644 under the umask, stale own temps swept; a symlink at the path is replaced, never followed |
| `.archcore/` document | `docs.WriteFileAtomic` | calls `jsonfile.WriteAtomic`, then invalidates the per-file scan cache |
| user-owned file | `jsonfile.WriteThroughSymlink` (`agents.writeFileAtomic` wraps it) | follows the symlink first, so a link into a dotfiles repository survives |

Requirement 5 is the one that is easy to get wrong, and the consequence is the least visible: a
writer that renames over the path replaces a user's symlinked `CLAUDE.md` with a regular file, and
nothing that reads the content back notices. The opposite mistake is worse: `.archcore/settings.json`
and the manifest arrive with a cloned repository, so a committed link pointing outside the project
must be replaced, not written through. That is why following the link is a separate entry point and
not a property of the primitive. All three delegate to `jsonfile.WriteAtomic`, so the mode and umask
handling exists once.

Requirements 7 and 8 exist because a stamp looks like the case requirement 3 covers and is the
opposite of it. A stamp carries no content — it is a zero-byte file whose existence and mtime are the
whole message — and its claim is won by the create call failing for everyone but the first caller. A
temp-file-plus-rename has no losing caller: every writer creates its own temp name and every rename
succeeds, so converting a claim to the "safer" primitive lets two processes both believe they hold it.
The dedup stamp would then run the same hook twice, and the unattended update's claim would let two
processes replace one binary at once.

The `os.Rename` in `@cli/internal/stamp/stamp.go` is not a publish and is not the pattern requirement 2
forbids. It moves an abandoned lock aside so the reclaim is one step: a bare remove followed by a
create lets two callers interleave and both come away holding the lock.

The self-update binary replacement is a fourth sequence and stays one: it fsyncs before close and
carries a Windows rename-aside with rollback, which no other target needs. It is a documented
exception, not a precedent (`@cli/internal/update/replace_other.go`, `@cli/internal/update/replace_windows.go`).

## Enforcement

- Code review: any `os.Rename` in a write path, and any new `*.tmp` path construction.
- `@cli/internal/stamp/stamp.go` is the exclusivity exception. Its `os.Rename` is the lock steal, not a
  publish; `TestClaimStamp_RecoversAnAbandonedLockThroughTheAtomicSteal` fails if the reclaim stops
  going through it.
- `@cli/internal/update/replace_other.go` and `replace_windows.go` are the documented binary-replacement
  exception.
