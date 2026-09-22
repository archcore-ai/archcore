# `_global_` — shared standards

These are **shared knowledge bases** that other examples reuse. They model
company-wide standards that many repositories follow but none of them owns:

- `company-standards/` — engineering baseline (errors, commits, API versioning, logging)
- `design-system/` — frontend standards (data fetching, naming, design tokens)
- `security-baseline/` — security baseline (secrets, auth, dependencies)

An example reuses one of these by listing it in its own `settings.json`:

```jsonc
// examples/05-global-single-source/.archcore/settings.json
{ "globals": [ { "id": "company-standards", "path": "../_global_/company-standards/.archcore" } ] }
```

When you open that example with your agent, these documents show up alongside the
project's own — so shared standards are always in context without copying them
into every repo.
