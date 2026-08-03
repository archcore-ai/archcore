---
title: "Directory Structure and Document Naming Rules"
status: accepted
tags:
  - "directory-structure"
---

## Purpose

State how directories and document filenames inside `.archcore/` must be formed, so that the CLI, the MCP server, and the server scan, categorize, and sync the same set of documents.

## Rule

1. The author MAY create any directory structure inside `.archcore/` and MAY nest directories to any depth.
2. The author MUST name every document file `slug.type.md`.
3. The author MUST write the slug in lowercase alphanumeric characters and hyphens only. Examples: `use-postgres`, `login-flow`.
4. The author MUST use one of the 19 valid document types as the type segment: `adr`, `rfc`, `rule`, `guide`, `doc`, `spec`, `task-type`, `cpat`, `prd`, `idea`, `plan`, `rnd`, `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs`.
5. The author MUST use the `.md` extension.
6. The CLI and the MCP server MUST derive the category (`vision`, `knowledge`, `experience`) from the document type, never from the directory path.
7. WHEN a scan reaches a hidden directory (a `.`-prefixed directory such as `.git/`), the scanner MUST skip it.
8. The scanner MUST treat `settings.json` and `.sync-state.json` as meta files, not documents, during scanning, validation, and sync.
9. IF a filename carries no recognized type segment, THEN the scanner MUST categorize the document as `knowledge`.
10. The scanner MUST treat the directories `vision/`, `knowledge/`, and `experience/` as ordinary directories with no special meaning.

## Exceptions

None. Every document file inside `.archcore/` follows these rules.

## Rationale

Decoupling the category from the directory path lets a team organize documents by domain, team, or feature while the virtual categories stay available for filtering and display. The `slug.type.md` convention gives both the document type and the category one unambiguous source.

## Examples

Non-normative examples.

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
# Missing type segment — categorized as "knowledge" and may fail validation
my-document.md

# Invalid type — "decision" is not a valid document type
use-postgres.decision.md

# Uppercase in slug — slugs are lowercase
Use-Postgres.adr.md
```

## Enforcement

- `archcore status` checks the filename format and the type validity of every `.md` file.
- The MCP tool `create_document` generates filenames that follow the convention.
- The MCP tool `list_documents` derives the category with `ExtractDocType()` and `CategoryForType()`.
