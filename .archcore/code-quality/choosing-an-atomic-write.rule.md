---
title: "Which Atomic Write to Call, and Never a Fourth Copy"
status: accepted
tags:
  - "code-quality"
  - "golang"
---

## Rule

1. A write that must not be observable half-finished MUST go through one of the three helpers below,
   chosen by what is being written.
2. The developer MUST NOT write a fourth temp-file-plus-rename sequence inline.
3. IF the target is archcore-owned state — the sync manifest, a stamp, a config archcore writes in
   full — THEN the developer MUST call `jsonfile.WriteAtomic`.
4. IF the target is an `.archcore/` document, THEN the developer MUST call `docs.WriteFileAtomic`,
   which additionally invalidates the scan cache for that path.
5. IF the target is a file the user owns — `CLAUDE.md`, `AGENTS.md`, `GEMINI.md`, a host config —
   THEN the developer MUST call the writer in `@internal/agents/instructions.go`, which preserves the
   existing mode and resolves a symlink before replacing.
6. WHEN a new target fits none of the three, the developer MUST extend the closest helper rather than
   add a variant, and MUST state in its doc comment what the new case needs that the others do not.

## Rationale

Crash safety is one primitive, and a copy of it is a place a fix can fail to land. The manifest
carried a byte-for-byte duplicate of `jsonfile.WriteAtomic` for no reason other than that
`internal/sync` did not import `internal/jsonfile`; `jsonfile`'s own doc comment admitted the
duplication by pointing at it.

The three that remain are not redundant — they differ in what they must preserve:

| Target | Helper | What it adds |
|---|---|---|
| archcore-owned state | `jsonfile.WriteAtomic` | nothing; temp + rename at 0o644 |
| `.archcore/` document | `docs.WriteFileAtomic` | invalidates the per-file scan cache |
| user-owned file | `agents.writeFileAtomic` | keeps the existing mode, resolves a symlink |

Requirement 5 is the one that is easy to get wrong, and the consequence is the least visible: writing
a user's `CLAUDE.md` through the plain helper resets its permissions to 0o644 and replaces a symlink
with a regular file. Neither shows up in a test that only reads the content back.

The self-update binary replacement is a fourth sequence and stays one: it fsyncs before close and
carries a Windows rename-aside with rollback, which no other target needs. It is a documented
exception, not a precedent (`@internal/update/update.go`).

## Examples

### Good

```go
// Machine-owned state: the plain helper is the whole requirement.
if err := jsonfile.WriteAtomic(manifestPath(baseDir), data); err != nil {
    return fmt.Errorf("saving manifest: %w", err)
}
```

### Bad

```go
// A fourth copy of the primitive, inline. Identical to jsonfile.WriteAtomic
// today, and silently divergent from it after the first fix lands there.
tmp := target + ".tmp"
if err := os.WriteFile(tmp, data, 0o644); err != nil {
    return err
}
if err := os.Rename(tmp, target); err != nil {
    _ = os.Remove(tmp)
    return err
}
```

```go
// A user-owned file through the machine-owned helper: mode reset to 0o644,
// symlink replaced by a regular file, content assertions still green.
jsonfile.WriteAtomic(filepath.Join(baseDir, "CLAUDE.md"), data)
```

## Enforcement

- Code review: any `os.Rename` in a write path, and any new `*.tmp` path construction.
- `@internal/jsonfile/jsonfile.go` — `WriteAtomic`.
- `@internal/docs/document.go` — `WriteFileAtomic`.
- `@internal/agents/instructions.go` — the mode- and symlink-preserving writer.
- `@internal/update/update.go` — the documented binary-replacement exception.
