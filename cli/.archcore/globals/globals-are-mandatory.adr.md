---
title: "All Global Sources Are Mandatory (Removed the required Flag)"
status: accepted
tags:
  - "config"
  - "globals"
  - "mcp"
---

## Context

Global sources — read-only knowledge bases mounted from `settings.json` (see @.archcore/globals/global-sources-via-settings.adr.md) — originally carried a per-entry `required` boolean. `required: true` made a missing source abort MCP startup; `required: false` (the default when omitted) silently skipped a missing source.

The optional/skip mode is a foot-gun. A typo in `path`, or a source that was never cloned, yields zero global documents with no signal — the agent then runs against a silently incomplete context, which is exactly the failure the feature exists to prevent. The whole `local → global` value proposition assumes the global is actually present.

## Decision

Remove the `required` field. **Every declared global source is mandatory.**

If a declared source's directory is absent on disk, the MCP server fails fast — both during the scan and at startup (`checkGlobals`) — with:

```
global source "<id>" not found at "<path>" — clone it before starting the MCP server
```

A global is declared only when a project deliberately adds it to its `globals` array; a declared dependency that cannot be found is a misconfiguration worth surfacing loudly, not degrading silently.

## Alternatives Considered

1. **Keep `required`, default `false` (status quo).** Rejected: silent-skip hides typos and un-cloned sources behind an empty result.
2. **Remove `required`, make every global optional (silent skip).** Rejected: never blocks startup, but reintroduces the silently-incomplete-context risk and makes `path` typos invisible.
3. **Express optionality some other way (env, glob).** Rejected: more surface for no clear gain. "Declared = needed" is the simplest mental model.

## Consequences

- The schema simplifies: each entry is `{ id, path }` (the `Required` field is removed from `GlobalSource` in @internal/config/config.go).
- `docs.Scan` (@internal/docs/scan.go) and `checkGlobals` (@cmd/mcp.go) always error on a missing source; the `if gs.Required` branch is gone.
- Distribution and lifecycle — getting the source onto disk — become more pressing, since there is no "tolerate missing" mode. This stays an open question; the interim answer is in-tree vendoring (@.archcore/globals/vendoring-a-global.guide.md), which is self-contained.
- This supersedes the `required` specifics in @.archcore/globals/global-sources-via-settings.adr.md (Decision point 6) and is reflected in @.archcore/globals/global-sources.spec.md and @.archcore/globals/declaring-global-sources.rule.md.
- Backward compatible at load time: an old `settings.json` that still carries `"required": …` keeps working — the unknown field is carried through per @.archcore/cli/forward-compatible-settings-parsing.rule.md — but it has no effect.
- Non-server surfaces (`archcore status`, the SessionStart hook) must not block a session, so they do **not** fail-fast on a missing global: they degrade to a local-only scan and surface a visible warning naming the source, rather than silently blanking local context. "Loud, not silent" holds without aborting. See @.archcore/globals/global-sources.spec.md §6.4 and @cmd/hooks_common.go.
