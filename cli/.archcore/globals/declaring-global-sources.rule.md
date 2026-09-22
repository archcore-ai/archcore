---
title: "Declaring Global Sources in settings.json"
status: accepted
tags:
  - "config"
  - "globals"
---

## Rule

1. WHEN a project consumes a global source, the project MUST declare that source in the `globals` array of its own `.archcore/settings.json`.
2. The global target MUST NOT carry a marker of its own. A source is global only because a consumer references it.
3. Each `globals` entry MUST have the form `{ "id": …, "path": … }`.
4. `id` MUST match `^[a-z0-9][a-z0-9-]*$`.
5. `id` MUST NOT be `local`, which is reserved for the consuming project.
6. `id` MUST be unique within the array. It becomes the `source_id` of every document from that source.
7. `path` MUST point at the `.archcore` directory of the global source, not at the project root.
8. `path` MAY be relative, including `../` for a sibling or parent, MAY be absolute, and MAY be in-tree under the reserved `.archcore/global/` directory.
9. IF a declared global directory is absent at startup, THEN the MCP server MUST fail fast instead of degrading silently. Every declared global is mandatory.
10. The author MUST NOT name a local document directory `global`. `.archcore/global/` is reserved and the local scan skips it.
11. `.mcp.json` MUST stay generic: `{"command": "archcore", "args": ["mcp"]}`. Global wiring lives in `settings.json`, never in launch flags.

A global source is read-only everywhere outside the MCP read tools. The related rule on read-only globals carries the full statement.

## Rationale

- A declaration in the consumer's committed `settings.json` is versioned, reviewable, and travels with the repository, so the same `.mcp.json` works on every machine.
- An explicit, validated, unique `id` keeps sources distinguishable. An id derived from the path basename would collide for two repositories both named `standards`.
- Pointing `path` at the `.archcore` directory, with no auto-appended segment, keeps resolution literal and predictable.
- Treating every declared global as mandatory turns "the sibling repository is not cloned here" from a silent empty result into an actionable startup error. A declared dependency that cannot be found is a misconfiguration.

## Examples

Non-normative examples.

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

// path points at the project root, not at its .archcore directory
{ "globals": [ { "id": "company", "path": "../company-global" } ] }

// id reserved / duplicated
{ "globals": [
    { "id": "local",     "path": "../a/.archcore" },              // reserved
    { "id": "standards", "path": "../a/standards/.archcore" },
    { "id": "standards", "path": "../b/standards/.archcore" }     // duplicate id
  ] }
```

## Enforcement

- `Settings.Validate` (`@internal/config/config.go`) rejects an empty, malformed, reserved, or duplicate `id`, and an empty `path`. `config.Load` fails on an invalid `settings.json`.
- `checkGlobals` (`@cmd/mcp.go`) aborts MCP startup when a declared source is absent or `settings.json` is invalid.
- Tests: the globals cases in `@internal/config/config_test.go` and `@internal/mcp/tools/globals_test.go`.
