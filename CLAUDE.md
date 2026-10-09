# Claude Code Repository Instructions

Read and follow `AGENTS.md` before creating or editing repository documentation, Archcore documents, skills, agents, rules, or user-facing Markdown.

## Shared agent instructions

`AGENTS.md` is the shared instruction file for every host, including the `## Integrations` section. It is imported below so Claude Code receives it natively.

@AGENTS.md

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
2. Read the relevant content contract under `plugin/plugins/archcore/skills/_shared/`.
3. Read `plugin/plugins/archcore/skills/_shared/precision-rules.md`.
4. Apply the controlled technical writing policy in `AGENTS.md`.

When modifying files under `plugin/plugins/archcore/skills/`, preserve existing routing terminology, document-type names, tool names, state names, and contract semantics.

Do not edit content inside an Archcore-managed block:

```text
<!-- archcore:start --> managed by `archcore init`, block v2 — edit outside these markers
## Archcore — project context for this repo

`.archcore/` holds this project's recorded context as typed Markdown files,
`<slug>.<type>.md`, in three categories: knowledge (decisions, rules, contracts,
reference), vision (requirements, plans, research), and experience (lessons from
past work). `list_documents` filters by `category` and `types`.

Before you change, test, or explain code here, call `search_documents` with `for_path` set to the file, and read the rules, decisions, and specs it returns with `get_document`. Accepted rules and decisions bind the change. Without the Archcore MCP tools, read `.archcore/` as Markdown: the `*.rule.md` files first, then grep `.archcore/` for the file name.

1. Once per session, before the first code edit, call `list_documents` with
   `types: ["rule", "cpat"]` and `status: "accepted"`, and read every rule that
   applies to the code you will write. A rule that names no path reaches you this way.
2. Before you state how this system behaves, search the topic. Cite the document,
   or say that none exists.
3. If an accepted document conflicts with the task, tell the user before you edit.
4. If a search is empty, read `near_misses` and retry with fewer words before you
   conclude that no document exists.
5. When a decision is made ("we'll use X", "from now on Y"), record it.
6. When a module, API, or system you touched has no document, offer to capture it.

If the Archcore MCP tools are missing or fail to connect, tell the user once, then
take the file route above. Do not write `.archcore/` files by hand.

A `.archcore/` may also mount read-only **global sources** — shared, org-wide
context. `list_documents` / `search_documents` surface them alongside local docs,
tagged `source_kind: "global"`. When present, treat them as defaults a local doc
can override — never edit or relate to one.

Skip these steps only for turns this repo would have no opinion on: syntax
trivia, throwaway snippets, pure mechanics.
<!-- archcore:end -->
```

Keep repository-specific instructions outside that block.
