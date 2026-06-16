---
title: "Always Search .archcore/ Documents Before Codebase or External Sources"
status: accepted
tags:
  - "cli"
---

## Rule

When researching patterns, decisions, conventions, or implementation approaches in this project:

1. **Search .archcore/ first.** Call `list_documents` → `get_document` to check if the topic is already documented before grepping the codebase or launching research agents.
2. **Use codebase search second.** Only fall back to `Grep`/`Glob`/agents if no relevant archcore document exists.
3. **Use external sources last.** Web searches, documentation lookups, and context7 are a last resort when neither archcore nor the codebase has the answer.

## Rationale

The .archcore/ knowledge base captures distilled decisions (ADRs), standards (rules), and how-tos (guides) that are the canonical source of truth for this project. Searching code first wastes time and may miss the reasoning behind patterns. External sources don't know project-specific conventions.

## Examples

### Good

```
User: "How do we disable a feature?"
Agent: list_documents → finds feature-gating-howto.guide.md → reads it → answers
```

```
User: "How does sync work?"
Agent: list_documents → finds sync-design.idea.md, sync-how-it-works.guide.md → reads them → answers
```

### Bad

```
User: "How do we disable a feature?"
Agent: Grep for "Hidden" across codebase → launches research agent → eventually finds archcore docs
```

## Enforcement

Code review and agent hooks. This rule should also be referenced in CLAUDE.md to ensure it is loaded into every session context.