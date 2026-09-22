---
title: "Conventional Commits"
status: accepted
---

## Rule

Commit subjects use Conventional Commits: `type(scope): summary`.
Allowed types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`.

```
feat(billing): add proration on plan downgrade
fix(auth): reject expired refresh tokens
```

Keep the subject under 72 characters; put the *why* in the body.

## Rationale

Machine-readable history powers changelog generation and semver decisions.
