---
name: doc
argument-hint: "[topic]"
description: Creates non-behavioral reference material like registries, glossaries, and lookup tables. Activates for cataloging, listing, or informational content that doesn't prescribe behavior.
---

# Doc — Reference Material

## When to use

- Creating a glossary, registry, or lookup table
- Cataloging services, APIs, or components

**Not Doc:**
- Mandatory standard → **rule**
- Step-by-step procedure → **guide**
- Normative contract → **spec**

## Quick create

1. `mcp__archcore__list_documents(types=["doc"])` — check duplicates
2. Use the `AskUserQuestion` tool to ask: "What reference material are you cataloging? Who is it for?"
3. Compose content covering Overview, Content (tables/lists), Examples — using user's answers for depth. Pass as `content` to `mcp__archcore__create_document`.
4. Suggest `mcp__archcore__add_relation` based on existing documents.

## Relations

| Direction | Type | Target | When |
|-----------|------|--------|------|
| Peer | `related` | rule | Doc lists things, rule governs them |
| Peer | `related` | guide | Doc catalogs, guide instructs |
