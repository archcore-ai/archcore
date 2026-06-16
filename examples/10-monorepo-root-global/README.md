# 10 · monorepo-root-global

> A monorepo where the shared standards live at the **repo root**, and each app
> reuses them.

Same idea as [09-monorepo-shared-global](../09-monorepo-shared-global/), but the
shared context sits in the root `.archcore/` instead of a dedicated package —
handy when the standards describe the whole repo. Each app and package reuses the
root; open the root directly when you want to work on the standards themselves.

```
10-monorepo-root-global/
├── .archcore/            repo-wide standards (architecture, conventions, deps)
├── apps/       web · api
└── packages/   ui
```

## Try it

Install the CLI (see the [examples README](../README.md)), then open the app,
package, or the root:

- **Claude Code:** `cd 10-monorepo-root-global/apps/web && claude`
- **Cursor:** open `apps/web` (or `apps/api`, `packages/ui`, or the root)
- **Codex CLI:** `cd 10-monorepo-root-global/apps/api && codex`

## Ask your agent

- (from `apps/web`) "What conventions apply across the whole repo?"
- "Can apps depend on each other, or only on packages?"
- "What's specific to this app vs repo-wide?"
