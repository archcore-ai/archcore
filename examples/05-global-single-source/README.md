# 05 · global-single-source

> A project that reuses a **shared standards** knowledge base instead of copying
> the rules into every repo.

This billing service keeps only its own docs. The company-wide engineering
standards — error handling, commits, API versioning, logging — come from the
shared [`company-standards`](../_global_/company-standards/) source it reuses. So
when you ask the agent about error handling, it answers from the company standard
even though that document doesn't live in this repo.

**What's in `.archcore/` here**

- `service-overview.doc.md` — what this service is
- `local-conventions.rule.md` — billing-only rules (money in cents, idempotent
  charges, signed webhooks)
- *(reused)* the shared company standards

The link lives in `settings.json`:

```jsonc
{ "globals": [ { "id": "company-standards", "path": "../_global_/company-standards/.archcore" } ] }
```

## Try it

Install the CLI (see the [examples README](../README.md)), then open it with your agent —
**Claude Code:** `cd 05-global-single-source && claude` · **Cursor:** open the folder ·
**Codex CLI:** `cd 05-global-single-source && codex`.

## Ask your agent

- "What are our error-handling and logging standards?"
- "How should I version a new API endpoint?"
- "What's specific to the billing service here?"
- "How do we handle money values in this service?"
