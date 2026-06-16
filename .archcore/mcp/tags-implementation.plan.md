---
title: "Implementation Plan: Tags in Document Frontmatter"
status: accepted
tags:
  - "mcp"
---

## Goal

Add optional `tags` field to document frontmatter enabling cross-cutting categorization (team ownership, domain, topic) that the free-form directory hierarchy cannot express alone.

**Problem:** A document like `e2e-auth-flow.rule.md` is relevant to both frontend and backend teams. Directory structure is one-dimensional — the file can only live in one folder. Tags allow multi-axis annotation without duplicating directories.

## Sprint Status

### ✅ Sprint 1 — Tags in frontmatter — DONE

Shipped in commit `1558381 feat: tags to documents in frontmatter` (and follow-ups).

Delivered (mapping to Phases below):

- **Phase 1 (Parser + data structures)** — `Frontmatter` struct with `Title`, `Status`, `Tags` in `templates/`; `SplitDocument` returns `(Frontmatter, body)`; `LocalDocument.Tags` with `json:"tags,omitempty"`; `validateTags` / `parseTags` / `normalizeTags` helpers in `internal/mcp/tools/common.go`; `buildDocumentFile` emits sorted/deduplicated YAML block. ✅
- **Phase 2 (MCP tools)** — `tags` parameter on `create_document`, `update_document`, `list_documents` (OR semantics for filter). `update_document` evolved beyond the original "replace" semantics into a three-way contract (omit / `[]` / `[values]`) — codified separately as `mcp/tag-update-semantics.rule.md`. ✅ (with documented evolution)
- **Phase 3 (Session context)** — `EXISTING TAGS:` line injected via `cmd/hooks_common.go`. ✅
- **Phase 4 (Sync)** — `SyncFrontmatter.Tags` plumbed through `internal/sync/payload.go`. ✅
- **Phase 5 (MCP server instructions)** — TAGS block added to `mcpServerInstructions` in `internal/mcp/server.go`. ✅
- **Phase 6 (Doctor hygiene checks)** — **PARTIAL.** Tag format is validated inline by `cmd/status.go:185` (rejects malformed tags). The remaining hygiene checks proposed in Task 13 — singleton-tag warnings (possible typos) and a unique tag count summary — are NOT implemented in `cmd/doctor.go`. Track as a follow-up if tag sprawl becomes observable.

### Follow-Ups (deferred, not blocking)

- Singleton-tag warnings + unique tag count in `archcore doctor` (Phase 6 remainder).
- Tag-aware merge semantics for future two-way sync (set-union per the original Risks section).

## Key Design Decisions

- **No `list_tags` MCP tool** — tags surface via `list_documents` filter + session context injection at session start. Rationale: avoids tool proliferation (9→8 tools), eliminates redundant filesystem scan, session context already provides tag discoverability.
- **Reject invalid tags** with error + "did you mean?" hint. No silent normalization. Explicit is better than implicit.
- **Hybrid YAML parsing** — keep existing `---` delimiter detection, use `yaml.v3` (already a dependency) for frontmatter block parsing. Avoids hand-rolling multiline YAML list parser.
- **Flat tags** — format `^[a-z][a-z0-9_:|-]*$` (lowercase, hyphens, underscores, colons, pipes), stored sorted + deduplicated. Colons enable lightweight namespace-like conventions (e.g. `team:payments`) without enforced hierarchy.
- **Session context** — top 30 tags by frequency, no counts, single compact line.
- **OR semantics** for tag filtering in `list_documents` — document matches if it has at least one of the requested tags.

## Frontmatter Format

```yaml
---
title: "E2E Auth Flow Testing"
status: accepted
tags:
  - frontend
  - team:backend
  - payment_flow
---
```

Tags are optional. When absent, field is omitted from JSON output (`omitempty`).

## Tasks

> The following task list is preserved as historical record. See **Sprint Status** above for what actually shipped.

### Phase 1: Parser + Data Structures

1. **Refactor `SplitDocument` in @templates/templates.go** — change return from `(title, status, body string)` to `(Frontmatter, string)` struct. Use `yaml.v3.Unmarshal` for frontmatter block (keep `---` delimiter detection as-is). Add `Frontmatter` struct with `Title`, `Status`, `Tags` fields.

2. **Update call sites** (compiler catches all):
   - @internal/mcp/tools/common.go — `extractFrontmatter()`, `ReadDocumentContent()`, `ScanDocuments()`
   - @internal/sync/payload.go — `ParseFrontmatter()`
   - @internal/mcp/tools/update_document.go

3. **Add `Tags []string` to `LocalDocument` struct** in @internal/mcp/tools/common.go with `json:"tags,omitempty"`.

4. **Bump `extractFrontmatter` buffer** from 1024 → 2048 bytes (tags may extend frontmatter).

5. **Update `buildDocumentFile`** in @internal/mcp/tools/common.go — write sorted, deduplicated `tags:` YAML block when tags are non-empty.

6. **Add tag validation helpers** — `validateTags(tags []string) error` with regex check + "did you mean?" lowercase hint. `normalizeTags(tags []string) []string` for sort + dedup (no case change).

### Phase 2: MCP Tools

7. **`create_document`** @internal/mcp/tools/create_document.go — add optional `tags` array parameter. Validate → reject invalid → pass to `buildDocumentFile`. Include tags in response JSON.

8. **`update_document`** @internal/mcp/tools/update_document.go — add optional `tags` array parameter. Replace semantics (not append). Preserve existing tags when param not provided.

9. **`list_documents`** @internal/mcp/tools/list_documents.go — add `tags` filter parameter with OR semantics. Add `hasAnyTag()` helper to filter loop. Tags already visible in response via `LocalDocument.Tags`.

### Phase 3: Session Context

10. **`buildSessionContext`** in @cmd/hooks_common.go — after scanning documents, aggregate unique tags with frequency. Inject `EXISTING TAGS: backend, auth, frontend, ...` line (top 30 by frequency, no counts, comma-separated).

### Phase 4: Sync

11. **`SyncFrontmatter`** in @internal/sync/payload.go — add `Tags []string` with `json:"tags,omitempty"`. Update `BuildPayload()` to extract and populate tags.

### Phase 5: MCP Server Instructions

12. **Update instructions** in @internal/mcp/server.go — add TAG-BASED SEARCH section: tags narrow search but don't guarantee completeness; combine with type filtering; fall back to type-only if tag query returns 0; treat co-named tags and directories as same topic.

### Phase 6: Doctor

13. **Tag hygiene checks** in @cmd/doctor.go — warn on invalid tag format, singleton tags (possible typos), report unique tag count.

## Acceptance Criteria

- Documents can be created/updated with tags via MCP tools
- Invalid tags (uppercase, spaces) are rejected with actionable error
- Tags support hyphens, underscores, colons, and pipes (`team-platform`, `payment_flow`, `team:payments`, `flag|beta`)
- `list_documents(tags=["x"])` returns documents with OR matching
- `list_documents(types=["rule"], tags=["frontend"])` combines filters correctly
- Tags appear in `list_documents` and `get_document` responses
- Session context shows `EXISTING TAGS:` line at MCP session start
- Old documents without tags parse correctly (nil tags, omitted from JSON)
- Sync payload includes tags in `SyncFrontmatter`
- `archcore doctor` reports tag hygiene issues
- All existing tests continue to pass

## Dependencies

- `gopkg.in/yaml.v3` — already in go.mod, no new dependency
- Server-side: must accept `tags` field in `SyncFrontmatter` (or ignore unknown fields)

## Risks

- **Partial tag coverage** — if only 60% of documents are tagged, tag-filtered search gives false sense of completeness. Mitigated by MCP instructions telling agents to treat tag results as starting point, not exhaustive.
- **Tag sprawl** — without controlled vocabulary, tags may diverge over time. Mitigated by `doctor` hygiene warnings and session context showing canonical tags.
- **Future multi-user sync** — two users adding different tags = different SHA-256 hashes. Current one-way push means last push wins. Future pull/two-way sync should implement set-union merge for tags field.
