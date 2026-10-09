---
title: "File Context Is a search_documents Mode That Shares Its Result With the Pre-Edit Hook"
status: draft
tags:
  - "component:cli"
  - "hooks"
  - "integrations"
  - "mcp"
---

## Context

An agent that knows which file it will edit has no single call that returns the documents constraining that file. `search_documents(path_ref="FullLayout.tsx")` returned 183 matches in the Litres monorepo with a draft spec first, and the pre-edit hint in @cli/internal/advisory/code_alignment.go ranks by its own rules, shows 3 documents, and prints paths that `get_document` refuses (`invalid path: must start with ".archcore/"`, 2026-10-09). The `search_documents` description holds 2 032 of the 2 048 characters Claude Code documents as its cap, so any new guidance for that tool needs room freed in the same change.

## Decision

Adopted a `for_path` parameter on `search_documents` that returns the file-context result — the ranked documents for one file plus the general rules — computed by the same function whose output the pre-edit hook renders, chosen by the user on 2026-10-09 over a dedicated tool.

## Alternatives Considered

1. A dedicated read tool `get_file_context(path)` — rejected by the user on 2026-10-09 to keep the MCP surface at 11 tools; it would have given the context address a one-argument call and a description budget of its own.
2. Re-ranking `path_ref` only — rejected because `path_ref` matches documents that mention a path, while a file context also needs documents that apply by directory and rules that name no path.
3. Delivering file context through the pre-edit hook only — rejected because the hook fires only on `Write` and `Edit`, Codex caps hook context at 2 500 tokens, and an agent planning an edit needs the context before it writes.

## Consequences

Positive:

- One ranking feeds both the hook and the agent's call, so the two channels cannot disagree about which documents constrain a file.
- The context address names one call with one argument the agent already knows: the file path.
- The tool count stays at 11.

Negative:

- The `search_documents` description must explain `for_path` inside 2 048 characters, so the rewrite has to remove at least as much text as it adds.
- `for_path` returns a different row shape from the content filters, and the search contract has to state how `for_path` combines with `content`, `types`, and `status`.
- [expected] An agent can still call `path_ref` where `for_path` was meant; the two names differ by one word.

## Superseded when

- The rewritten `search_documents` description cannot carry the `for_path` guidance within 2 048 characters.
- A bench run shows agents choosing `path_ref` over `for_path` before an edit in more than half of the runs.
