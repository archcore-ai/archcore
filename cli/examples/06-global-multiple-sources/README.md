# 06 · global-multiple-sources

> A project that reuses **several** shared knowledge bases at once.

This web app inherits three shared sources instead of restating them:
engineering standards, the design system, and the security baseline. Its own
`.archcore/` holds just the app-specific bits.

**What's in `.archcore/` here**

- `app-overview.doc.md` — what the app is
- `frontend-stack.rule.md` — app-level frontend choices
- *(reused)* company-standards, design-system, security-baseline

```jsonc
{ "globals": [
    { "id": "company-standards", "path": "../_global_/company-standards/.archcore" },
    { "id": "design-system",     "path": "../_global_/design-system/.archcore" },
    { "id": "security-baseline", "path": "../_global_/security-baseline/.archcore" }
] }
```

## Try it

Install the CLI (see the [examples README](../README.md)), then open it with your agent —
**Claude Code:** `cd 06-global-multiple-sources && claude` · **Cursor:** open the folder ·
**Codex CLI:** `cd 06-global-multiple-sources && codex`.

## Ask your agent

- "What frontend standards apply here?"
- "What security rules must I follow?"
- "How should I fetch data from the server?"
- "What's specific to this app vs shared across the company?"
