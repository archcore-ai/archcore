---
title: "MCP-Only Document Operations"
status: accepted
tags:
  - "plugin"
  - "rule"
---

## Rule

1. The agent MUST create every `.archcore/` document with the `create_document` MCP tool.
2. The agent MUST update every `.archcore/` document with the `update_document` MCP tool.
3. The agent MUST delete every `.archcore/` document with the `remove_document` MCP tool.
4. The agent MUST create every document relation with the `add_relation` MCP tool.
5. The agent MUST delete every document relation with the `remove_relation` MCP tool.
6. The agent MUST NOT use Write to create, modify, or delete a file under `.archcore/`.
7. The agent MUST NOT use Edit to create, modify, or delete a file under `.archcore/`.
8. The agent MUST NOT use Bash to create, modify, or delete a file under `.archcore/`.
9. The agent MUST NOT edit `.archcore/.sync-state.json` directly.
10. The agent MAY read a file under `.archcore/` with the Read tool for context gathering.

## Rationale

The MCP tools carry guarantees that a direct file write bypasses: slug-format and frontmatter validation, template generation for the required sections, `.archcore/.sync-state.json` updates with file hashes and relations, and nearby-relation discovery for documents that share a directory. A document written with Write can therefore carry invalid frontmatter, miss required sections, hold no manifest entry, and skip relation discovery — drift that `archcore doctor` reports as an error.

## Examples

### Good

```
# Create a new ADR
create_document(type="adr", filename="use-postgres", title="Use PostgreSQL for Primary Persistence")

# Update an existing document
update_document(path="plugin/use-postgres.adr.md", status="accepted")

# Add a relation
add_relation(source="plugin/migration-rules.rule.md", target="plugin/use-postgres.adr.md", type="implements")

# Read for context — permitted by item 10
Read(".archcore/<dir>/use-postgres.adr.md")
```

### Bad

```
# Direct file creation — bypasses validation and templates
Write(".archcore/<dir>/use-postgres.adr.md", "---\ntitle: Use PostgreSQL\nstatus: draft\n---\n...")

# Direct edit — bypasses frontmatter validation
Edit(".archcore/<dir>/use-postgres.adr.md", old_string="status: draft", new_string="status: accepted")

# Manual manifest edit — corrupts sync state
Edit(".archcore/.sync-state.json", ...)

# Shell-based creation — bypasses every guarantee named in Rationale
Bash("echo '---\ntitle: ...' > .archcore/<dir>/use-postgres.adr.md")
```

## Enforcement

- `@plugins/archcore/bin/check-archcore-write` (PreToolUse) intercepts a Write or Edit call targeting `.archcore/**/*.md` and blocks it with a redirect message.
- `@plugins/archcore/bin/validate-archcore` (PostToolUse) runs `archcore doctor` after a `.archcore/` file change and reports the issues it finds.
- Every skill's Example Workflow section uses MCP tools only.
- The `archcore-assistant` agent definition grants no Write tool and no Edit tool.
- Every command prompt names an MCP tool for document operations.
