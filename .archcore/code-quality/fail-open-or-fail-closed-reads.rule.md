---
title: "Every Read of External State Declares Fail-Open or Fail-Closed"
status: accepted
tags:
  - "code-quality"
  - "golang"
---

## Rule

1. Every read of state outside the program — `settings.json`, a host config, the filesystem, git —
   MUST be classified as serving a **guard** or an **advisory**, and the classification MUST be named
   in a comment at the branch that handles the error.
2. IF the reader is a guard, whose result decides whether an operation is permitted, THEN an
   unreadable or ambiguous state MUST refuse the operation.
3. IF the reader is an advisory, whose result cannot change a permission decision, THEN any error
   MUST yield an empty result rather than a refusal.
4. WHEN one read serves both modes, the developer MUST expose two functions with names that say
   which is which, and MUST NOT expose one function with a flag.
5. A guard MUST NOT call the advisory variant. The reverse is permitted and costs only a refusal the
   caller can ignore.

## Rationale

The two modes are opposite and both are correct. An advisory that refuses turns a missing hint into a
blocked edit; a guard that degrades turns an unreadable config into an open door. Neither failure is
visible at the call site, because both look like an ordinary error branch.

The pairing in requirement 4 is what makes the choice hard to get wrong. `config.ReadGlobals` and
`config.LoadGlobals` read the same file: the first swallows a parse error and reports no globals, the
second returns it. A single function taking `strict bool` would put the decision at every call site,
which is where it gets copied from the nearest neighbour instead of decided.

Two JSON files in this repository take opposite decisions on the same question, and both are right:
`@internal/config/config.go` tolerates unknown fields in `settings.json` because the user owns that
file and a newer release may have written it; `@internal/sync/manifest.go` rejects them because the
manifest is machine-owned and an unknown field there means corruption.

Three defects on record came from crossing the modes:

- A document reachable through a symlink out of the store stayed writable because a guard defaulted
  to allow on an error class it did not enumerate (`@cmd/hook_write_guard.go`).
- `archcore status` reported a healthy global source whose every read failed, because the inspection
  discarded the error (`@internal/docs/inspect.go`).
- The MCP write tools were denied on a host whose tool-name spelling was unrecognized, because an
  unfolded name fell through to a path the guard read as a direct edit.

## Examples

### Good

```go
// Guard: a settings.json that cannot be parsed must not be read as "no globals
// declared", because that is also what an empty file looks like.
globals, err := config.LoadGlobals(baseDir)
if err != nil {
    return fmt.Errorf("cannot verify global sources: %w", err)
}

// Advisory: the recap is context, not permission. No globals is a fine answer
// to "which globals are declared" when the file is unreadable.
globals := config.ReadGlobals(baseDir)
```

### Bad

```go
// A guard reading the degrading variant. An invalid settings.json now means
// "no globals are declared", so a write into a global source is permitted.
if docs.IsExternalGlobalDocument(baseDir, path, config.ReadGlobals(baseDir)) {
    return denyHook(reason)
}

// One function, one flag, and the decision pushed to every call site.
func LoadGlobals(baseDir string, strict bool) ([]GlobalSource, error)
```

## Enforcement

- Code review: any new `Read*`/`Load*` pair, and any call to one from a guard.
- `@internal/config/config.go` — `ReadGlobals` / `LoadGlobals`, the reference pair.
- `@internal/mcp/tools/common.go` — `loadGlobalsFailClosed`, the name states the mode.
- `@cmd/hook_write_guard.go` — enumerates the allow cases and denies everything else, rather than the
  reverse.
- `@cmd/hook_command.go` — `safeHandle` converts a panic into an allow, which is the advisory
  direction for the process as a whole; the write guard runs before it can matter.
