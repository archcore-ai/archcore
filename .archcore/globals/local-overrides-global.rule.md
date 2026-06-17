---
title: "Local Documents Take Precedence Over Global Sources"
status: accepted
tags:
  - "globals"
  - "mcp"
---

## Rule

- When the same topic is covered by both a local document and a global source, the **local document is authoritative** for this project. Global sources are organization-wide defaults; the local project refines or overrides them.
- An agent reading documents MUST consult `source_kind` / `source_id` / `read_only` on each result to tell local from global. It MUST NOT guess a document's authority from its path or title.
- A same-slug pair (e.g. a local `error-handling.rule.md` and a global one) MUST be treated as: local is the effective rule; the global is background/context. Do not present the global rule as binding when a local one exists.
- All writes MUST target local documents. Global documents are read-only; an agent MUST NOT attempt to "fix" a global by editing it in the consuming project — corrections belong upstream in the source repository.
- The scan surfaces **both** documents (local and global) — it does not silently drop the global. Precedence is a reading convention the agent applies, not a deduplication the server performs.

## Rationale

- The whole point of globals is inheritance with local override: a company rule sets the default, a repo tightens or relaxes it for its own context.
- The server intentionally returns both same-slug documents so the agent can see the lineage; collapsing them would hide that a local override exists.
- Making the convention explicit prevents the failure mode where an agent applies a global rule that a local document has deliberately superseded.
- Read-only enforcement on globals (@.archcore/globals/global-sources.spec.md §5) backs this rule mechanically: the tools refuse to write a global, so "edit it locally" is not even possible.

## Examples

### Good

```text
list_documents → both appear:
  local  read_only=false  src=local              error-handling.rule.md   (this service's variant)
  global read_only=true   src=company-standards  …/error-handling.rule.md (company default)

Agent: "This repo overrides the company error-handling rule locally; the company rule is the
general default it refines." → applies the local rule.
```

### Bad

```text
Agent reads only the global error-handling rule, ignores source_kind, and applies the generic
company guidance even though the local rule overrides it.

Agent tries update_document on the global doc to "align" it → rejected
("cannot update a read-only global source document"); the fix belonged upstream.
```

## Enforcement

- Read-only guards in `create_document` / `update_document` / `remove_document` (@internal/mcp/tools/) prevent writing to globals.
- `source_kind` / `read_only` are always present on scanned documents (@internal/mcp/tools/common.go `LocalDocument`) so the distinction is machine-checkable — and on `search_documents` results, which now carry the same four source fields (`source_id`, `source_kind`, `global`, `read_only`), so the distinction holds across all three read tools.
- Reviewer / agent behavior: when answering from globals, state whether a local override exists. The fixture @examples/07-local-overrides-global/ exercises this precedence: a local @examples/07-local-overrides-global/.archcore/error-handling.rule.md overrides the same-slug rule from @examples/_global_/company-standards/.
