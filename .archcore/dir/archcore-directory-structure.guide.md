---
title: "Organizing Your .archcore/ Directory"
status: accepted
tags:
  - "directory-structure"
---

## Overview

How to organize documents inside your `.archcore/` directory. The directory structure is free-form — you can use flat layouts, domain-based folders, team-based folders, or any nesting that fits your project.

### Target Audience

- Developers setting up `.archcore/` for a new project
- Teams migrating from the old fixed `vision/`/`knowledge/`/`experience/` layout

## Structure Basics

The only requirement is the **filename convention**: `slug.type.md`. The type in the filename determines the document's virtual category. The directory path has no effect on categorization.

```
.archcore/
├── settings.json          # config (not a document)
├── my-decision.adr.md     # category: knowledge (from type "adr")
├── feature/
│   └── mvp-scope.prd.md   # category: vision (from type "prd")
└── ops/
    └── deploy.task-type.md # category: experience (from type "task-type")
```

## Type → Category Mapping

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
| `mrd`       | vision         |
| `brd`       | vision         |
| `urd`       | vision         |
| `brs`       | vision         |
| `strs`      | vision         |
| `syrs`      | vision         |
| `srs`       | vision         |
| `task-type` | experience     |
| `cpat`      | experience     |

## Example Layouts

### Flat (small projects, <10 documents)

```
.archcore/
├── settings.json
├── use-postgres.adr.md
├── coding-standards.rule.md
├── api-reference.doc.md
└── mvp-launch.plan.md
```

Simple and easy to browse. Good starting point for any project.

### Domain-Based (medium to large projects)

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

Group by feature or domain area. Useful when different domains have their own decisions, guides, and rules.

### Team-Based

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

### Old Layout (still works)

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

The old `vision/`, `knowledge/`, and `experience/` directories still work. They're treated as regular directories with no special meaning. No migration is needed.

## Tips

- **Start flat.** Only introduce directories when you have 10+ documents and navigating becomes difficult.
- **Group by what you look up together.** If you often read auth decisions alongside auth guides, put them in `auth/`.
- **Don't replicate categories as directories.** The category is already encoded in the type. A `knowledge/` directory adds no information.
- **Nest sparingly.** One level of directories covers most use cases. Deep nesting makes paths long without adding clarity.
- **Use `archcore status`** to check all documents, or **MCP `list_documents`** to see documents with their virtual categories, regardless of directory layout.

## References

- [ADR: Use Free-Form Directory Structure](./free-form-directory-structure.adr.md)
- [Rule: Directory Structure and Document Naming Rules](./free-form-directory-rules.rule.md)