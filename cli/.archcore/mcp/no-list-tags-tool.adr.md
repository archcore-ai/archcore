---
title: "No Dedicated list_tags MCP Tool"
status: accepted
tags:
  - "mcp"
---

## Context

When adding tags as a cross-cutting categorization mechanism, the natural design instinct is to add a dedicated `list_tags` MCP tool for discoverability — analogous to how other MCP tool suites expose list/get pairs for each entity type.

At the time of the tags implementation, the MCP server had 8 tools. Adding `list_tags` would bring it to 9.

## Decision

No dedicated `list_tags` tool. Tags are surfaced through two existing mechanisms:

1. **Session context injection** — `buildSessionContext()` in `cmd/hooks_common.go` injects `EXISTING TAGS: backend, auth, frontend, ...` at MCP session start. Top tags by frequency, no counts, comma-separated; the cap is `maxSessionTags` in `@cmd/hooks_common.go`. Provides proactive discoverability without the agent needing to know to ask.
2. **`list_documents` filter** — the `tags` parameter on `list_documents` accepts an array of tags with OR semantics. Documents matching any requested tag are returned with their full tag list visible in the response.

## Alternatives Considered

- **Dedicated `list_tags` tool** — would return all tags with counts. Rejected: adds tool proliferation (agents must discover and learn another tool), triggers a redundant `ScanDocuments()` filesystem walk that `list_documents` already performs, and session context already covers the primary use case of "what tags exist?"
- **Tags only in `get_document` response** — no filtering, no session context. Rejected: makes tags undiscoverable until you already know which document to read.

## Consequences

- Tag discoverability depends on session context injection working correctly. If `buildSessionContext()` breaks or is bypassed, agents lose awareness of available tags.
- If tag analytics (counts, usage trends, orphan detection) become needed, a tool or CLI command can be added then. The `archcore doctor` command already reports singleton tags and total unique count.
- This pattern (surface metadata via existing tools + session context rather than dedicated tools) should be considered for any future cross-cutting metadata field (e.g., owners, priority).

## History

The tag cap was written here as a literal "top 30" while the code capped at 20. Naming the constant instead of restating its value keeps the two from drifting again.
