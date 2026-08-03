---
title: "Always Search .archcore/ Documents Before Codebase or External Sources"
status: accepted
tags:
  - "cli"
---

## Rule

1. WHEN an agent researches a pattern, decision, convention, or implementation approach in this project, the agent MUST search `.archcore/` first with `list_documents` or `search_documents`.
2. WHEN a search returns a relevant document, the agent MUST read it with `get_document` before it forms an answer.
3. The agent MUST read only the documents that the current task needs.
4. IF `.archcore/` holds no relevant document, THEN the agent MUST search the codebase with `Grep` or `Glob` before it uses an external source.
5. IF neither `.archcore/` nor the codebase answers the question, THEN the agent MAY use an external source such as a web search or a library documentation lookup.

## Rationale

`.archcore/` holds the distilled decisions (`adr`), standards (`rule`), and how-tos (`guide`) that are canonical for this project. A codebase search shows the current implementation but not the reasoning that constrains it. An external source knows neither.

## Examples

Non-normative examples.

### Good

```
User: "How do we disable a feature?"
Agent: search_documents → finds feature-gating-at-command-layer.rule.md → reads it → answers
```

```
User: "How does sync work?"
Agent: search_documents → finds sync-how-it-works.guide.md and sync-engine.spec.md → reads them → answers
```

### Bad

```
User: "How do we disable a feature?"
Agent: greps the codebase for "Hidden" → launches a research agent → reaches the archcore document last
```

## Enforcement

- `@CLAUDE.md` and `@AGENTS.md` state the search order, so every session loads it.
- The SessionStart hook injects the project context that names the MCP read tools.
- Code review rejects answers that contradict an accepted `.archcore/` document.
