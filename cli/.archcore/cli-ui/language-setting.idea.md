---
title: "'\"Add Language Configuration to Settings\"'"
status: accepted
tags:
  - "cli-ui"
---

## Idea

Add a `language` field to `.archcore/settings.json` that controls the language used by the archcore MCP server when generating document content. When an AI agent creates or updates documents via MCP tools (`create_document`, `update_document`), all generated text — section headers, placeholders, descriptions — is produced in the configured language.

Default is `"en"` (English). Example:

```json
{
  "sync": "none",
  "language": "ru"
}
```

With `"language": "ru"`, every document created through MCP will have Russian content. With `"en"` or when omitted, content is in English.

## Value

- Makes archcore fully usable for non-English-speaking teams — all generated `.archcore/` content appears in their language
- Per-project setting, so multilingual organizations can configure each project independently
- Simple, obvious configuration — one field, ISO 639-1 codes

## Possible Implementation

1. Add `Language string` field to the `Settings` struct with `json:"language,omitempty"` — defaults to `"en"` when empty
2. Add `"language"` to `allowedFields` for **all** sync types (none, cloud, on-prem) since it's a local content concern
3. Update `MarshalJSON` / `UnmarshalJSON` to handle the field
4. Update `getSettingsValue` / `setSettingsValue` in `cmd/config.go` to support CLI access
5. The MCP server reads this setting and includes the language directive in its prompts/templates when generating content

## Risks and Constraints

- **MCP integration**: The MCP server must read settings.json and use the language value when constructing content — this is the primary consumer
- **Validation scope**: Accepting any short lowercase string is simpler than validating against a strict ISO 639-1 list
- **No CLI i18n**: This does NOT change CLI UI strings (error messages, prompts) — only the language of generated document content in `.archcore/`
