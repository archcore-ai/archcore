---
title: "Declaring Global Sources in settings.json"
status: accepted
tags:
  - "config"
  - "globals"
---

## Rule

- A project that consumes a global source MUST declare it in its **own** `.archcore/settings.json` `globals` array. The global target MUST NOT carry any marker — globality comes only from being referenced.
- Each entry MUST be `{ "id": …, "path": … }`:
  - `id` MUST be lowercase alphanumeric with hyphens (`^[a-z0-9][a-z0-9-]*$`), MUST NOT be `local` (reserved), and MUST be unique within the array. It becomes the document `source_id`.
  - `path` MUST point at the global source's **`.archcore` directory** (not the project root). It MAY be relative (including `../` for siblings/parents), absolute, or in-tree under the reserved `.archcore/global/` directory.
- Every declared global is **mandatory**: if its directory is absent the MCP server fails fast at startup instead of silently degrading. There is no optional global (the former `required` flag was removed — see @.archcore/globals/globals-are-mandatory.adr.md).
- Local document directories MUST NOT be named `global` — `.archcore/global/` is reserved and skipped by the local scan.
- A global source is **read-only everywhere outside the MCP read tools** — not writable, not a relation endpoint (either direction), and absent from `archcore status` and the SessionStart context. The full statement is @.archcore/globals/globals-are-read-only-everywhere.rule.md.
- `.mcp.json` MUST stay generic (`{"command": "archcore", "args": ["mcp"]}`). Global wiring lives in `settings.json`, never in launch flags.

## Rationale

- Keeping the declaration in the consumer's committed `settings.json` makes it versioned, reviewable, and portable with the repo; the same `.mcp.json` works everywhere.
- An explicit, validated, unique `id` keeps sources distinguishable — unlike deriving an id from the path basename, where two repos named `standards` would collide.
- Pointing `path` at the `.archcore` directory (no auto-appended segment) keeps resolution literal and predictable.
- Treating every declared global as mandatory turns "the sibling repo isn't cloned here" from a silent empty result into an actionable startup error: a declared dependency that cannot be found is a misconfiguration, not a soft option.
- See @.archcore/globals/global-sources-via-settings.adr.md for why the marker and launch flags were removed, @.archcore/globals/globals-are-mandatory.adr.md for why the `required` flag was dropped, and @.archcore/globals/global-sources.spec.md §7 for the validation contract.

## Examples

### Good

```json
// Sibling repo
{ "sync": "none",
  "globals": [ { "id": "company-global", "path": "../company-global/.archcore" } ] }

// Multiple sources, distinct explicit ids
{ "sync": "none",
  "globals": [
    { "id": "company",  "path": "../company-global/.archcore" },
    { "id": "platform", "path": "../platform-standards/.archcore" }
  ] }

// In-tree vendored global under the reserved directory
{ "sync": "none",
  "globals": [ { "id": "company", "path": ".archcore/global/company" } ] }
```

### Bad

```json
// Marker on the target — there is no such field; globality is not a property
{ "sync": "none", "global": true }

// path points at the project root, not its .archcore directory
{ "globals": [ { "id": "company", "path": "../company-global" } ] }

// id derived implicitly / duplicated / reserved
{ "globals": [
    { "id": "local",    "path": "../a/.archcore" },   // reserved
    { "id": "standards", "path": "../a/standards/.archcore" },
    { "id": "standards", "path": "../b/standards/.archcore" }  // duplicate id
  ] }
```

## Enforcement

- `Settings.Validate` (@internal/config/config.go) rejects empty/malformed/reserved/duplicate `id` and empty `path`; `config.Load` fails on invalid `settings.json`.
- `checkGlobals` (@cmd/mcp.go) aborts MCP startup when any declared source is absent or `settings.json` is invalid.
- Tests: globals cases in @internal/config/config_test.go and @internal/mcp/tools/globals_test.go.