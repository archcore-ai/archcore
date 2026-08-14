---
title: "Use Free-Form Directory Structure in .archcore/"
status: accepted
tags:
  - "directory-structure"
---

## Context

The `.archcore/` directory originally enforced a rigid 3-directory structure: `vision/`, `knowledge/`, and `experience/`. Every document had to live in one of these directories, and the directory determined the document's category.

### Current State

- Documents were organized into exactly three fixed directories
- The physical directory path was the source of truth for a document's category
- `archcore init` created all three directories upfront
- Teams couldn't organize documents by domain, feature, or team

### Problem Statement

The fixed directory structure didn't scale for projects with many documents. Teams wanted to organize by domain (e.g., `auth/`, `payments/`, `infra/`) or by team, but the rigid structure forced everything into three flat buckets. As document count grew, navigating `knowledge/` with 30+ files became unwieldy.

## Decision

Allow any directory structure inside `.archcore/`. Categories become **virtual** — derived from the document type in the filename, not from the physical directory.

The `slug.type.md` naming convention is the sole source of category information. For example, `auth/login-flow.adr.md` is categorized as "knowledge" because `adr` maps to the knowledge category, regardless of the `auth/` directory.

### Rationale

- **Domain-centric organization:** Teams can group related documents together (`auth/`, `payments/`, `infra/`) instead of scattering them across three category directories
- **Backward compatibility:** The old `vision/`, `knowledge/`, `experience/` directories still work — they're just regular directories now, with no special meaning
- **Simpler init:** `archcore init` only needs to create `.archcore/` itself, not three subdirectories
- **Mental model preserved:** Virtual categories still appear in MCP responses (`list_documents`), so the category abstraction is maintained without enforcing directory layout
- **Filename already encodes type:** The `slug.type.md` convention was already in use; deriving category from type is a natural extension

## Alternatives Considered

### Alternative 1: Keep Fixed Directories

- Continue requiring `vision/`, `knowledge/`, `experience/`
- Rejected because it doesn't scale and prevents domain-centric organization

### Alternative 2: Fixed + Custom Directories with Override

- Allow custom directories alongside the three fixed ones, with a config option to override category mapping
- Rejected because it adds unnecessary complexity — the type-based derivation achieves the same result without configuration

## Consequences

### Positive

- Flexible organization: any nesting depth, any directory names
- Full backward compatibility: existing projects with `vision/`/`knowledge/`/`experience/` continue to work without changes
- Simpler initialization: just `.archcore/` directory
- Category information remains available in all interfaces (CLI, MCP) via type derivation

### Negative

- Type segment in filename is now mandatory for category derivation — files without a recognized type default to "knowledge"
- Users must understand the `slug.type.md` naming convention to ensure correct categorization

## Implementation Notes

- `CategoryForType()` in `templates/templates.go` handles the type → category mapping
- File scanning uses `filepath.WalkDir` recursively instead of scanning three fixed directories
- Hidden directories (`.`-prefixed) and meta files (`settings.json`, `.sync-state.json`) are skipped during scanning
- `archcore status` checks filename format and type validity across all directories

## References

- [Rule: Directory Structure and Document Naming Rules](./free-form-directory-rules.rule.md)
- [Doc: Organizing Your .archcore/ Directory](./archcore-directory-structure.doc.md)
