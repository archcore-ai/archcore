---
title: "Temporarily Disable Features at the Command Layer Only"
status: accepted
tags:
  - "cli"
---

## Rule

When temporarily disabling a CLI feature:

1. **Block at the cobra command handler level only.** Add guards in `RunE` that return early with a user-facing error message. Never delete or modify internal logic, helper functions, or packages.
2. **Hide the command** using `Hidden: true` on the cobra command struct. Do not unregister it from the root command.
3. **Preserve all tests.** Tests call internal functions directly (`runInit`, `doSync`, `getSettingsValue`, etc.), not through cobra handlers. Command-level guards must not break them.
4. **Remove interactive prompts** that expose the disabled feature (e.g., sync type selector in `init`), replacing them with hardcoded defaults.
5. **Block config writes for gated fields.** If a feature has associated config keys, add guards in the `set` subcommand handler to reject changes with a "not available yet" message. Allow read-only access to informational keys (e.g., `config get sync` returns `"none"`).

## Rationale

This approach keeps the codebase ready for re-enablement with minimal diff. Internal packages, validation logic, and tests remain exercised and correct. The only changes needed to re-enable are removing the guards and restoring the prompts — a small, reviewable diff.

## Examples

### Good

```go
// cmd/sync.go — hide command, block at handler
cmd := &cobra.Command{
    Use:    "sync",
    Hidden: true,
    RunE: func(cmd *cobra.Command, args []string) error {
        return fmt.Errorf("sync is not available yet — this feature is coming soon")
    },
}
```

```go
// cmd/config.go — block config set for gated keys
case "set":
    if args[1] == "sync" || args[1] == "project_id" {
        return fmt.Errorf("%s is not available yet — sync features are coming soon", args[1])
    }
```

### Bad

```go
// Deleting the sync package or removing sync functions
// BAD: breaks tests, large re-enablement diff

// Commenting out command registration in root.go
// BAD: makes the command completely unreachable, even for testing

// Modifying internal validation logic to reject sync modes
// BAD: breaks unit tests that test internal functions directly
```

## Enforcement

Code review. When a feature is gated, verify that `go test ./...` passes with no changes to test files.