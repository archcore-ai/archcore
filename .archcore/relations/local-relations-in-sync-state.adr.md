---
title: "Store Local Document Relations in .sync-state.json"
status: accepted
tags:
  - "relations"
---

## Context

Documents in `.archcore/` often have meaningful relationships — an ADR implements a PRD, a guide extends another guide, a rule depends on a decision. Currently there is no mechanism to capture these connections locally.

The server side uses GraphRAG for intelligent relationship discovery, but explicit, hand-curated (or agent-curated) links between documents are valuable for:

- Navigating related decisions without server access
- Providing agents with a local knowledge graph
- Preserving intentional relationships that automated discovery might miss

We need a storage format that is simple, git-friendly, and doesn't pollute individual document files.

## Decision

Store document-to-document relations in the existing `.archcore/.sync-state.json` file under a top-level `relations` array.

```json
{
  "version": 1,
  "files": { "auth/jwt-strategy.adr.md": "sha256..." },
  "relations": [
    {
      "source": "auth/jwt-strategy.adr.md",
      "target": "auth/oauth-flow.guide.md",
      "type": "related"
    },
    {
      "source": "payments/stripe.adr.md",
      "target": "payments/billing-requirements.prd.md",
      "type": "implements"
    }
  ]
}
```

### Relation types

| Type         | Meaning                                                |
| ------------ | ------------------------------------------------------ |
| `related`    | General association between two documents              |
| `implements` | Source document implements what target describes       |
| `extends`    | Source document extends or builds upon target          |
| `depends_on` | Source document depends on target being accepted/valid |

### Key parameters

| Parameter        | Value                                                       |
| ---------------- | ----------------------------------------------------------- |
| Storage location | `.archcore/.sync-state.json` → `relations` field            |
| Git tracking     | Tracked (same as sync hashes)                               |
| Relation scope   | Document ↔ Document only                                    |
| Sync strategy    | Deferred — will be decided when sync is re-enabled          |
| Path format      | Relative to `.archcore/`, forward slashes, no leading slash |

### Agent integration

When an agent reads a document via `get_document`, the response is enriched with both outgoing and incoming relations from the graph. This eliminates the need for a separate lookup or full scan:

```json
{
  "path": ".archcore/auth/jwt-strategy.adr.md",
  "title": "JWT Strategy",
  "status": "accepted",
  "content": "...",
  "relations": [
    { "target": "auth/oauth-flow.guide.md", "type": "implements", "direction": "outgoing" },
    { "target": "auth/auth-redesign.prd.md", "type": "implements", "direction": "incoming" }
  ]
}
```

Dedicated MCP tools (`add_relation`, `remove_relation`, `list_relations`) provide full CRUD access to the relations graph.

## Alternatives Considered

### 1. Frontmatter `relations` field in each document

```yaml
---
title: JWT Strategy
relations:
  - target: auth/oauth-flow.guide.md
    type: related
---
```

**Rejected because:**

- Relations are properties of a pair, not of a single document — storing in one document means the other doesn't know about the link without a full scan
- Bidirectional awareness requires either duplicating relations in both files (maintenance burden) or scanning all documents (negates the benefit)
- Scatters relation data across many files — hard to query or validate holistically
- Modifying a relation requires editing document files, inflating diffs

### 2. Separate `.archcore/.relations.json` file

A dedicated file only for relations, separate from sync state.

**Rejected because:**

- Adds another tracked state file to manage
- `.sync-state.json` already serves as the local state file — splitting state across multiple files increases complexity
- No meaningful benefit over embedding in the existing state file
- Two files to merge-conflict with instead of one

### 3. Inline comments or links in document body

Using markdown links or special comment syntax to denote relations.

**Rejected because:**

- Not machine-readable without fragile parsing
- Mixes content with metadata
- No standard format — each author would link differently

## Consequences

**Positive:**

- Single state file for all local metadata (sync hashes + relations) — simple mental model
- Centralized store means both sides of a relation are queryable from one place — no full scan needed
- Agents see incoming and outgoing relations inline when reading any document via `get_document`
- Tracked in git → shared across team members automatically
- JSON format is easy to read, validate, and manipulate programmatically
- Relation types are specific enough to be useful but few enough to be manageable

**Negative:**

- Merge conflicts in `.sync-state.json` become more likely as the file grows with relations — this is an accepted trade-off, same as for sync hashes
- No server-side sync strategy yet — local relations and server GraphRAG will need reconciliation later
- Relations can reference documents that have been renamed or deleted — validation/cleanup tooling will be needed

**Neutral:**

- GraphRAG on the server remains the authoritative source for "smart" relationship discovery; local relations are explicit, intentional links only
- The `version` field in `.sync-state.json` can be bumped if the relations schema needs to evolve