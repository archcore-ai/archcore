# 09 · monorepo-shared-global

> A monorepo where several apps share one set of standards kept in a shared
> package.

The shared rules (monorepo conventions, release process) live once in
`packages/shared-standards`, and both `apps/web` and `apps/api` reuse them — so
each app keeps only what's specific to it. Work inside the app you're on and the
agent sees that app's docs together with the shared standards.

```
09-monorepo-shared-global/
├── packages/shared-standards/.archcore/   shared rules
└── apps/
    ├── web/.archcore/   web-specific docs
    └── api/.archcore/   api-specific docs
```

## Try it

Install the CLI (see the [examples README](../README.md)), then open the app you
want to work in — each is wired up for all three agents:

- **Claude Code:** `cd 09-monorepo-shared-global/apps/web && claude`
- **Cursor:** open `apps/web` (or `apps/api`)
- **Codex CLI:** `cd 09-monorepo-shared-global/apps/api && codex`

## Ask your agent

- (from `apps/web`) "What standards apply across this monorepo?"
- (from `apps/api`) "What's our release process?"
- "What's specific to this app vs shared across the repo?"
