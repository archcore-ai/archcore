# 08 · global-in-archcore

> Keep the shared standards **inside your own `.archcore/`**, committed with the
> repo — no sibling folder or separate package to clone.

Sometimes you want the shared context to live right in the project, so it travels
with the code and there's nothing external to set up. You place it under
`.archcore/global/<name>/` and reuse it like any other shared source — a committed
copy you refresh from the original when it changes.

**What's in `.archcore/` here**

- `service-overview.doc.md` — your own editable docs
- `global/company/` — the shared company standards, kept inside this repo
  - `coding-standards.rule.md`, `review-checklist.doc.md`

```jsonc
{ "globals": [ { "id": "company", "path": ".archcore/global/company" } ] }
```

## Try it

Install the CLI (see the [examples README](../README.md)), then open it with your agent —
**Claude Code:** `cd 08-global-in-archcore && claude` · **Cursor:** open the folder ·
**Codex CLI:** `cd 08-global-in-archcore && codex`.

## Ask your agent

- "What coding standards apply here?"
- "What's on the code-review checklist?"
- "What's specific to this service vs the shared standards?"
