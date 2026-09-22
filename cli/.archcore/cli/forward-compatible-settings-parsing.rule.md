---
title: "Unknown settings.json Fields Are Tolerated and Preserved"
status: accepted
tags:
  - "cli"
  - "config"
---

## Rule

An older `archcore` binary meets a `settings.json` written by a newer one whenever a release adds a
field. The parser therefore treats an unrecognized field as data to carry, not as an error.

1. WHEN `Settings.UnmarshalJSON` reads a field that this binary does not recognize, the CLI MUST
   capture the field into `Settings.Extra` and MUST NOT fail the load.
2. WHEN a field is known to this binary but not allowed for the active sync mode, the CLI MUST
   return the existing error for that field.
3. The CLI MUST keep strict value validation for every known field.
4. WHEN the CLI writes `settings.json`, the CLI MUST write back every field captured in
   `Settings.Extra`.
5. WHILE `Settings.Extra` is empty, `Settings.MarshalJSON` MUST produce the same bytes as the
   marshaller produces without the merge tail.
6. WHEN an author adds an optional field to `Settings`, the author MUST add it to all three
   per-sync-mode structs in `Settings.MarshalJSON`.
7. The CLI MUST report unknown field names on stderr from `mcp`, `config`, `doctor`, and `sync`.
8. `Settings.UnmarshalJSON` and `config.Load` MUST NOT print a warning.
9. WHEN a release adds a `settings.json` field, the maintainer MUST ship the tolerant parser in
   that release or earlier.

## Rationale

The parser used to reject every unknown field with `field %q is not allowed for sync type …`. One
new config field then broke every config-loading path on an older binary: `mcp`, `hooks`, `doctor`,
and `status`. An already released strict binary cannot be repaired, so the only available fix is a
tolerant parser shipped as early as possible.

Requirement 4 exists because tolerance alone is not enough. A binary that reads a field it does not
know, then rewrites the file without it, erases a newer version's configuration silently. Requirement
5 keeps that merge invisible for the common case, so existing byte-comparison tests stay valid.

Requirement 6 records a defect class rather than a preference: `MarshalJSON` builds a separate struct
per sync mode, so a field added to one struct disappears when the project switches modes.

Requirement 8 keeps the warning off the hot paths. `config.Load` runs on every hook invocation, where
stdout carries the host protocol and stderr is read as a diagnostic.

Accepted trade-off: protection against a typo in a field name is lost. This reverses the strictness
recorded in the ADR on backing up invalid configs, and the reversal is deliberate.

## Examples

Non-normative examples.

### Good

```go
// Unknown to this binary → captured, load continues.
if s.Extra == nil {
    s.Extra = make(map[string]json.RawMessage)
}
s.Extra[key] = raw[key]
```

```json
{ "sync": "none", "language": "ru", "somethingNewer": { "on": true } }
```

`archcore config set language en` leaves `somethingNewer` in the file unchanged.

### Bad

```go
// Rejects a field a newer release added, and breaks every command that loads settings.
return fmt.Errorf("field %q is not allowed for sync type %q", key, s.Sync)
```

## Enforcement

- `@internal/config/config.go` — `Settings.Extra`, `knownFields`, `UnmarshalJSON`, `MarshalJSON`,
  `UnknownFieldNames`.
- `@cmd/config_warn.go` — `warnUnknownConfigFields`, called from `mcp`, `config`, `doctor`, `sync`.
- Co-located tests cover an unknown field on load, round-trip preservation through `config set`, and
  the two retained error paths (wrong-mode field, malformed known value).
- The release guide carries the sequencing step for a release that adds a `settings.json` field.

## Status

Current behavior. One item from the originating work is deferred and not started: an opt-in
`ARCHCORE_AUTO_UPDATE=1` self-heal. Auto-running `archcore update` from a hook path is rejected as a
default because it replaces the global binary without asking, does not restart a running MCP server,
and fails in CI, offline, and on a pinned version.
