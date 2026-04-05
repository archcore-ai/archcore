---
title: "MCP-Only Document Operations"
status: accepted
tags:
  - "plugin"
  - "rule"
---

## Rule

1. All `.archcore/` document creation MUST use the `create_document` MCP tool.
2. All `.archcore/` document updates MUST use the `update_document` MCP tool.
3. All `.archcore/` document deletion MUST use the `remove_document` MCP tool.
4. All relation management MUST use `add_relation` and `remove_relation` MCP tools.
5. Never use Write, Edit, or Bash to directly create, modify, or delete files under `.archcore/`.
6. Never manually edit `.sync-state.json` — it is managed by MCP tools.
7. Reading `.archcore/` files via Read is allowed for context gathering, but modifications must go through MCP.

## Rationale

MCP tools enforce the full Archcore toolchain:

- **Validation**: slug format, frontmatter structure (title, status, tags), document type validity
- **Templates**: required sections generated automatically when content is omitted
- **Sync manifest**: `.sync-state.json` updated with file hashes and relations
- **Nearby relations**: auto-discovered when documents share a directory
- **Consistency**: single path for all operations eliminates edge cases

Direct file writes bypass all of these guarantees. A document created via Write may have invalid frontmatter, missing sections, no manifest entry, and no relation discovery. This creates drift that `archcore validate` will flag as errors.

## Examples

### Good

```
# Create a new ADR
create_document(type="adr", filename="use-postgres", title="Use PostgreSQL for Primary Persistence")

# Update an existing document
update_document(path="plugin/use-postgres.adr.md", status="accepted")

# Add a relation
add_relation(source="plugin/migration-rules.rule.md", target="plugin/use-postgres.adr.md", type="implements")

# Read for context (allowed)
Read(".archcore/plugin/use-postgres.adr.md")
```

### Bad

```
# Direct file creation — bypasses validation and templates
Write(".archcore/plugin/use-postgres.adr.md", "---\ntitle: Use PostgreSQL\nstatus: draft\n---\n...")

# Direct edit — bypasses frontmatter validation
Edit(".archcore/plugin/use-postgres.adr.md", old_string="status: draft", new_string="status: accepted")

# Manual manifest edit — corrupts sync state
Edit(".archcore/.sync-state.json", ...)

# Shell-based creation — bypasses everything
Bash("echo '---\ntitle: ...' > .archcore/plugin/use-postgres.adr.md")
```

## Enforcement

- **PreToolUse hook**: Intercepts Write/Edit calls targeting `.archcore/**/*.md` and blocks them with a redirect message
- **PostToolUse hook**: Runs `archcore validate` after any `.archcore/` file changes and reports issues
- **Skill instructions**: Every skill's Example Workflow section uses MCP tools exclusively
- **Agent tool restrictions**: The archcore-assistant agent has no access to Write/Edit
- **Command instructions**: Every command prompt references MCP tools for document operations
