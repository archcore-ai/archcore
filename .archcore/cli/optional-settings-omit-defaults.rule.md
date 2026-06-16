---
title: "Optional Settings Fields Use omitempty and Code-Level Defaults"
status: accepted
tags:
  - "cli"
---

## Rule

- Optional settings fields (like `language`) MUST use the `json:"...,omitempty"` tag so they are omitted from `settings.json` when not explicitly set by the user.
- Default values for optional fields MUST be resolved at read time in the command layer (e.g., `getSettingsValue` returns `"en"` when `Language` is empty), not by writing defaults into the JSON file.
- The `init` command MUST NOT write optional fields with default values into `settings.json`. Only user-configured values appear in the file.
- When adding a new optional field, it MUST be added to `allowedFields` for all sync types, but MUST NOT be added to `requiredFields`.

## Rationale

- Keeping `settings.json` minimal makes it clear which values the user has explicitly configured vs. which are defaults.
- Avoids migration issues when defaults change — users who never set the field automatically get the new default.
- Consistent with how `project_id` already works (omitted when nil).

## Examples

### Good

```go
// Struct field with omitempty
Language string `json:"language,omitempty"`

// Default resolved at read time
case "language":
    if s.Language == "" {
        return "en", nil
    }
    return s.Language, nil
```

### Bad

```go
// Writing default into settings.json during init
settings := &config.Settings{Sync: "none", Language: "en"}

// Struct field without omitempty (forces field into JSON even when empty)
Language string `json:"language"`
```

## Enforcement

- Code review: verify new optional fields follow this pattern.
- Tests: roundtrip tests should confirm that unset optional fields do not appear in marshaled JSON.