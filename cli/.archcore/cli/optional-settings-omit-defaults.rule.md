---
title: "Optional Settings Fields Use omitempty and Code-Level Defaults"
status: accepted
tags:
  - "cli"
---

## Rule

1. The developer MUST tag every optional settings field with `json:"...,omitempty"`, so `settings.json` omits the field while the user has not set it.
2. The command layer MUST resolve the default value of an optional field at read time. Example: `getSettingsValue` returns `"en"` while `Language` is empty.
3. The `init` command MUST NOT write an optional field with its default value into `settings.json`.
4. WHEN a developer adds an optional field, the developer MUST add it to `allowedFields` for every sync type.
5. WHEN a developer adds an optional field, the developer MUST NOT add it to `requiredFields`.

## Rationale

A minimal `settings.json` shows which values the user configured and which ones the code supplies. Resolving defaults in code also removes a migration step: when a default changes, every user who never set the field picks up the new value. `project_id` already behaves this way — it is omitted while nil.

## Examples

Non-normative examples.

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
// Writing the default into settings.json during init
settings := &config.Settings{Sync: "none", Language: "en"}

// Struct field without omitempty (forces the field into JSON even when empty)
Language string `json:"language"`
```

## Enforcement

- Code review: the reviewer verifies that each new optional field follows this pattern.
- Tests: a roundtrip test confirms that an unset optional field does not appear in the marshaled JSON.
