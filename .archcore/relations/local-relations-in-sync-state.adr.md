---
title: "Store Local Document Relations in .sync-state.json"
status: accepted
tags:
  - "component:cli"
  - "relations"
---

## Context

Documents in `.archcore/` have meaningful relationships — an ADR implements a PRD, a guide extends another guide, a rule depends on a decision. Explicit, hand-curated (or agent-curated) links are valuable for:

- Navigating related decisions without server access
- Providing agents with a local knowledge graph
- Preserving intentional relationships that automated discovery might miss

The storage format must be simple, git-friendly, and must not pollute individual document files.

## Decision

Store document-to-document relations in the existing `.archcore/.sync-state.json` file under a top-level `relations` array. Each entry holds `source`, `target`, and `type`.

The relation vocabulary has seven types in three axes: structural (`related`, `implements`, `extends`, `depends_on`), evidential (`supports`, `contradicts`), and temporal (`supersedes`). The first four were the original set. The contract for the rest is `evidential-and-temporal-relations.spec`.

### Key parameters

| Parameter        | Value                                                       |
| ---------------- | ----------------------------------------------------------- |
| Storage location | `.archcore/.sync-state.json` → `relations` field            |
| Git tracking     | Tracked (same as sync hashes)                               |
| Relation scope   | Document ↔ Document only                                    |
| Path format      | Relative to `.archcore/`, forward slashes, no leading slash |

### Agent integration

`get_document` enriches its response with both outgoing and incoming relations from the graph, so an agent needs no separate lookup or full scan. `add_relation`, `remove_relation`, and `list_relations` provide full CRUD access to the graph.

## Alternatives Considered

### 1. Frontmatter `relations` field in each document

**Rejected because:**

- Relations are properties of a pair, not of a single document — storing in one document means the other doesn't know about the link without a full scan
- Bidirectional awareness requires either duplicating relations in both files (maintenance burden) or scanning all documents (negates the benefit)
- Scatters relation data across many files — hard to query or validate holistically
- Modifying a relation requires editing document files, inflating diffs

### 2. Separate `.archcore/.relations.json` file

**Rejected because:**

- Adds another tracked state file to manage
- `.sync-state.json` already serves as the local state file — splitting state across multiple files increases complexity
- Two files to merge-conflict with instead of one

### 3. Inline comments or links in document body

**Rejected because:**

- Not machine-readable without fragile parsing
- Mixes content with metadata
- No standard format — each author would link differently

## Consequences

**Positive:**

- Single state file for all local metadata (sync hashes + relations) — simple mental model
- Centralized store means both sides of a relation are queryable from one place — no full scan needed
- Tracked in git → shared across team members automatically

**Negative:**

- Merge conflicts in `.sync-state.json` become more likely as the file grows with relations — this is an accepted trade-off, same as for sync hashes
- Relations can reference documents that have been renamed or deleted — validation/cleanup tooling is needed

**Neutral:**

- The `version` field in `.sync-state.json` can be bumped if the relations schema needs to evolve
