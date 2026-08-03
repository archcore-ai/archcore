# Claude Code Repository Instructions

Read and follow `AGENTS.md` before creating or editing repository documentation, Archcore documents, skills, agents, rules, or user-facing Markdown.

## Archcore operations

Use Archcore MCP tools for all `.archcore/` document operations.

- Create documents with `create_document`.
- Update documents with `update_document`.
- Remove documents with `remove_document`.
- Read documents with `list_documents` and `get_document`.
- Manage document relations with `add_relation`, `remove_relation`, and `list_relations`.

Do not use direct file-writing tools to modify `.archcore/` documents.

Before creating an Archcore document:

1. Check existing documents for duplicates.
2. Read the relevant content contract under `plugins/archcore/skills/_shared/`.
3. Read `plugins/archcore/skills/_shared/precision-rules.md`.
4. Apply the controlled technical writing policy in `AGENTS.md`.

When modifying files under `plugins/archcore/skills/`, preserve existing routing terminology, document-type names, tool names, state names, and contract semantics.

Do not edit content inside an Archcore-managed block:

```text
<!-- archcore:start --> managed by `archcore init` — edit outside these markers
## Archcore — project context for this repo

This repo's architecture, decisions, rules, specs and patterns live in `.archcore/`,
reachable through the Archcore MCP tools. Consult them even on code you think you
know — a decision or rule may already constrain it.

- Touching this repo's real code or behavior → search first; read only what matches.
- A decision was made ("we'll use X", "from now on Y") → record it.
- A module / API / system has no doc — or a search comes back empty → capture it.
- Planning a feature or refactor → scope it against what's already decided.

A `.archcore/` may also mount read-only **global sources** — shared, org-wide
context not shown in the session-start list. `list_documents` / `search_documents`
surface them alongside local docs, tagged `source_kind: "global"`. When present,
treat them as defaults a local doc can override — never edit or relate to one.

The search is cheap — lean on it. Skip it only for turns this repo would have no
opinion on: syntax trivia, throwaway snippets, pure mechanics.
<!-- archcore:end -->
```

Keep repository-specific instructions outside that block.
