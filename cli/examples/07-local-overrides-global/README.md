# 07 · local-overrides-global

> Keep the company standards, but **override one rule** locally where this
> service needs something stricter.

This payments service reuses the shared `company-standards`, but ships its own
`error-handling.rule.md` with a stricter policy. When a local document covers the
same topic as a shared one, the local document is the source of truth — so the
agent follows the payments rule here, while every other company standard still
applies.

**What's in `.archcore/` here**

- `service-overview.doc.md` — what this service is
- `error-handling.rule.md` — the local, stricter error policy (overrides the
  company one)
- *(reused)* the rest of company-standards

## Try it

Install the CLI (see the [examples README](../README.md)), then open it with your agent —
**Claude Code:** `cd 07-local-overrides-global && claude` · **Cursor:** open the folder ·
**Codex CLI:** `cd 07-local-overrides-global && codex`.

## Ask your agent

- "What's our error-handling policy in this service?"
- "Does this service follow the company error standard, or something stricter? Why?"
- "Which standards here are company-wide and which are local?"
