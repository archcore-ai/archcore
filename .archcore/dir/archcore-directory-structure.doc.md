---
title: "Organizing Your .archcore/ Directory"
status: accepted
tags:
  - "directory-structure"
---

## Overview

Organize documents inside a `.archcore/` directory. The layout is free-form: a flat directory, domain folders, team folders, or any nesting that fits the project.

Intended readers:

- A developer setting up `.archcore/` for a new project.
- A team migrating from the old fixed `vision/`, `knowledge/`, and `experience/` layout.

## Structure basics

The filename convention is the only requirement: `slug.type.md`. The type in the filename decides the document's virtual category. The directory path has no effect on categorization.

```
.archcore/
├── settings.json           # config (not a document)
├── my-decision.adr.md      # category: knowledge (from type "adr")
├── feature/
│   └── mvp-scope.prd.md    # category: vision (from type "prd")
└── ops/
    └── deploy.task-type.md # category: experience (from type "task-type")
```

## Type to category mapping

| Type        | Category       |
|-------------|----------------|
| `adr`       | knowledge      |
| `rfc`       | knowledge      |
| `rule`      | knowledge      |
| `guide`     | knowledge      |
| `doc`       | knowledge      |
| `spec`      | knowledge      |
| `prd`       | vision         |
| `idea`      | vision         |
| `plan`      | vision         |
| `rnd`       | vision         |
| `mrd`       | vision         |
| `brd`       | vision         |
| `urd`       | vision         |
| `brs`       | vision         |
| `strs`      | vision         |
| `syrs`      | vision         |
| `srs`       | vision         |
| `task-type` | experience     |
| `cpat`      | experience     |

## Example layouts

Non-normative examples.

### Flat, for a small project under 10 documents

```
.archcore/
├── settings.json
├── use-postgres.adr.md
├── coding-standards.rule.md
├── api-reference.doc.md
└── mvp-launch.plan.md
```

Simple to browse, and a good starting point for any project.

### Domain-based, for a medium or large project

```
.archcore/
├── settings.json
├── auth/
│   ├── jwt-tokens.adr.md
│   ├── login-flow.guide.md
│   └── auth-api.doc.md
├── payments/
│   ├── stripe-integration.adr.md
│   ├── refund-policy.rule.md
│   └── payment-flow.guide.md
├── infra/
│   ├── deploy-checklist.task-type.md
│   └── aws-setup.guide.md
└── product/
    ├── mvp-scope.prd.md
    └── v2-features.idea.md
```

Group by feature or domain area. This layout helps when each domain carries its own decisions, guides, and rules.

### Team-based

```
.archcore/
├── settings.json
├── backend/
│   ├── api-versioning.adr.md
│   └── error-handling.rule.md
├── frontend/
│   ├── component-library.adr.md
│   └── accessibility.rule.md
└── platform/
    ├── ci-pipeline.guide.md
    └── monitoring-setup.guide.md
```

### Old layout, still supported

```
.archcore/
├── settings.json
├── knowledge/
│   ├── use-postgres.adr.md
│   └── coding-standards.rule.md
├── vision/
│   └── mvp-launch.plan.md
└── experience/
    └── deploy-checklist.task-type.md
```

The `vision/`, `knowledge/`, and `experience/` directories still work. The scanner treats them as ordinary directories with no special meaning, so no migration is needed.

## Tips

- Start flat. Introduce directories once the project passes about 10 documents and navigation becomes difficult.
- Group by what you look up together. WHEN you read auth decisions alongside auth guides, put both in `auth/`.
- Do not replicate categories as directories. The type already encodes the category, so a `knowledge/` directory adds no information.
- Nest sparingly. One level of directories covers most projects; deeper nesting lengthens paths without adding clarity.
- Run `archcore status` to check every document, or call the MCP tool `list_documents` to see documents with their virtual categories, independent of the directory layout.
