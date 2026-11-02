---
title: "Every Read of External State Declares Fail-Open or Fail-Closed"
status: accepted
tags:
  - "code-quality"
  - "component:cli"
  - "golang"
---

## Rule

1. External state means `.archcore/settings.json`, a host config file, the filesystem, and git.
2. A read of external state MUST be classified as a guard or an advisory.
3. Clause 2 binds a read whose error branch decides whether an operation proceeds.
4. The developer MUST name the classification in a comment at the branch that handles the error.
5. A plainly advisory read MAY inherit the classification from its enclosing function's doc comment.
6. IF the reader is a guard, THEN an unreadable state MUST refuse the operation.
7. IF the reader is an advisory, THEN an error MUST yield an empty result.
8. WHEN one read serves both modes, the developer MUST expose two functions.
9. The two function names MUST say which mode each one serves.
10. The developer MUST NOT expose one function with a mode flag.
11. A guard MUST NOT call the advisory variant.
12. WHEN a guard's refusal would block the file that repairs the failing state, the developer MUST
    narrow the refusal.

The reverse of clause 11 is permitted. It costs only a refusal the caller can ignore.

## Rationale

The two modes are opposite and both are correct. An advisory that refuses turns a missing hint into a
blocked edit; a guard that degrades turns an unreadable config into an open door. Neither failure is
visible at the call site, because both look like an ordinary error branch.

The pairing in clause 8 is what makes the choice hard to get wrong. `config.ReadGlobals` and
`config.LoadGlobals` read the same file: the first swallows a parse error and reports no globals, the
second returns it. A single function taking `strict bool` would put the decision at every call site,
which is where it gets copied from the nearest neighbour instead of decided.

Clause 3 was added in September 2026. The rule previously demanded a classification comment at *every*
read of external state; an audit found roughly one in ten carried one, and a MUST that nine in ten
call sites ignore grades nothing. Clause 5 lets the five `config.ReadGlobals` calls inside
`@cli/internal/docs/scan.go` inherit the mode from the scan they serve.

Two JSON files in this repository take opposite decisions on the same question, and both are right:
`@cli/internal/config/config.go` tolerates unknown fields in `settings.json` because the user owns that
file and a newer release may have written it; `@cli/internal/sync/manifest.go` rejects them because the
manifest is machine-owned and an unknown field there means corruption.

Four defects on record came from crossing the modes:

- A document reachable through a symlink out of the store stayed writable because a guard defaulted
  to allow on an error class it did not enumerate (`@cli/cmd/hook_write_guard.go`).
- The same file later read the globals list through the advisory variant, so a write into an
  externally mounted global source was permitted exactly when the guard could not verify the mount
  list. Fixed September 2026.
- `archcore status` reported a healthy global source whose every read failed, because the inspection
  discarded the error (`@cli/internal/docs/inspect.go`).
- The MCP write tools were denied on a host whose tool-name spelling was unrecognized, because an
  unfolded name fell through to a path the guard read as a direct edit.

Clause 12 is the counterweight the write-guard fix needed. Failing closed on every Markdown path would
have blocked a `README.md` edit on a `settings.json` typo, and failing closed inside `.archcore/`
would have blocked editing `settings.json` itself — the one write that ends the failing state.

## Enforcement

- Code review: any new `Read*`/`Load*` pair, and any call to one from a guard.
- No linter distinguishes the modes. A `Read*` call inside a function whose name or doc comment says
  "guard" is the pattern to grep for.
- `@cli/internal/config/config.go` — `ReadGlobals` / `LoadGlobals`, the reference pair.
- `@cli/cmd/hook_write_guard.go` — enumerates the allow cases and denies everything else, and carries the
  clause 12 narrowing. `TestWriteGuard_UnreadableSettingsFailsClosedOutsideTheProject` and
  `TestWriteGuard_UnreadableSettingsStillAllowsRepairingIt` pin both halves.
- `@cli/cmd/hook_command.go` — `safeHandle` converts a panic into an allow, which is the advisory
  direction for the process as a whole; the write guard runs before it can matter.
