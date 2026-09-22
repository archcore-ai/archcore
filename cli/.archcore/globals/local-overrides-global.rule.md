---
title: "Local Documents Take Precedence Over Global Sources"
status: accepted
tags:
  - "globals"
  - "mcp"
---

## Rule

1. WHEN a local document and a global source cover the same topic, the agent MUST treat the local document as authoritative for this project.
2. The agent MUST read `source_kind`, `source_id`, and `read_only` on each result to tell a local document from a global one.
3. The agent MUST NOT infer a document's authority from its path or its title.
4. WHEN a local document and a global document share a slug, the agent MUST apply the local document as the effective rule and MUST present the global document as background.
5. The agent MUST target every write at a local document.
6. IF a global document needs a correction, THEN the agent MUST leave the change to the upstream source repository.
7. The MCP read tools MUST return both same-slug documents. Precedence is a reading convention that the agent applies, not a deduplication that the server performs.

## Rationale

Globals exist for inheritance with local override: an organization rule sets the default, and a repository tightens or relaxes it for its own context.

Returning both same-slug documents keeps the lineage visible. Collapsing them would hide that a local override exists, and an agent could then apply organization-wide guidance that this repository deliberately superseded.

Read-only enforcement on globals backs this rule mechanically: the write tools refuse a global path, so "edit it locally" is not available as a shortcut.

## Examples

Non-normative examples.

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

Agent tries update_document on the global document to "align" it → rejected
("cannot update a read-only global source document"); the fix belonged upstream.
```

## Enforcement

- Read-only guards in `create_document`, `update_document`, and `remove_document` (`@internal/mcp/tools/`) prevent writes to globals.
- `source_kind` and `read_only` are always present on scanned documents (`docs.Document` in `@internal/docs/document.go`, aliased as `LocalDocument` in `@internal/mcp/tools/docs_bridge.go`), and `search_documents` results carry the same four source fields (`source_id`, `source_kind`, `global`, `read_only`), so the distinction is machine-checkable across all three read tools.
- Reviewer and agent behavior: WHEN answering from a global document, the responder states whether a local override exists.
- The fixture `@examples/07-local-overrides-global/` exercises this precedence: the local `error-handling.rule.md` overrides the same-slug rule from `@examples/_global_/company-standards/`.
