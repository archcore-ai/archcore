---
title: "Directory Structure and Document Naming Rules"
status: accepted
tags:
  - "directory-structure"
---

## Description

Rules governing directory structure and document naming inside `.archcore/`. These ensure consistent scanning, categorization, and sync behavior across all tools (CLI, MCP, server).

## Rule

1. Any directory structure is allowed inside `.archcore/` — directories can be nested to any depth
2. Document files must follow the `slug.type.md` naming convention:
   - **slug**: lowercase alphanumeric characters and hyphens only (e.g., `use-postgres`, `login-flow`)
   - **type**: one of the 18 valid document types: `adr`, `rfc`, `rule`, `guide`, `doc`, `spec`, `task-type`, `cpat`, `prd`, `idea`, `plan`, `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs`
   - **extension**: always `.md`
3. Category (vision / knowledge / experience) is always derived from the document type, never from the directory path
4. Hidden directories (`.`-prefixed, e.g., `.git/`) are ignored during scanning
5. Meta files (`settings.json`, `.sync-state.json`) are not documents and are skipped during scanning, validation, and sync
6. Documents without a recognized type segment in the filename default to category "knowledge"
7. The directories `vision/`, `knowledge/`, and `experience/` are valid but have no special meaning — they are treated like any other directory

## Rationale

Decoupling category from directory path allows teams to organize documents by domain, team, or feature while preserving virtual categories for filtering and display. The `slug.type.md` convention provides a single, unambiguous source of truth for both document type and category.

## Examples

### Good

```
.archcore/
├── settings.json
├── use-postgres.adr.md
├── auth/
│   ├── login-flow.guide.md
│   ├── jwt-tokens.adr.md
│   └── auth-api.doc.md
├── payments/
│   ├── stripe-integration.adr.md
│   └── refund-policy.rule.md
└── mvp-launch.plan.md
```

```
# Old-style layout — still valid, no migration needed
.archcore/
├── knowledge/
│   ├── use-postgres.adr.md
│   └── coding-standards.rule.md
├── vision/
│   └── mvp-launch.plan.md
└── experience/
    └── deploy-checklist.task-type.md
```

### Bad

```
# Missing type segment — will default to "knowledge" and may fail validation
.archcore/my-document.md

# Invalid type — "decision" is not a valid type
.archcore/use-postgres.decision.md

# Uppercase in slug — slugs must be lowercase
.archcore/Use-Postgres.adr.md
```

## Exceptions

- None. All document files inside `.archcore/` must follow these rules.

## Enforcement

- `archcore status` checks filename format and type validity for all `.md` files
- MCP `create_document` generates files with correct naming automatically
- MCP `list_documents` uses `ExtractDocType()` and `CategoryForType()` to derive category from filename

## References

- [ADR: Use Free-Form Directory Structure](./free-form-directory-structure.adr.md)
- [Guide: Organizing Your .archcore/ Directory](./archcore-directory-structure.guide.md)