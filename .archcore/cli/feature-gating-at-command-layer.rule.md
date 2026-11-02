---
title: "Temporarily Disable Features at the Command Layer Only"
status: accepted
tags:
  - "cli"
  - "component:cli"
---

## Rule

WHEN the team temporarily disables a CLI feature:

1. The developer MUST block the feature in the cobra `RunE` handler and return a user-facing error.
2. The developer MUST NOT delete or change the internal logic, helper functions, or packages of the gated feature.
3. The developer MUST set `Hidden: true` on the cobra command struct.
4. The developer MUST keep the command registered on the root command.
5. The developer MUST keep the existing tests unchanged.
6. The developer MUST remove each interactive prompt that exposes the gated feature.
7. WHEN the developer removes such a prompt, the developer MUST replace it with a hardcoded default.
8. IF the gated feature owns configuration keys, THEN the `config set` handler MUST reject changes to those keys with a "not available yet" message.
9. The `config get` handler MUST keep read access to informational keys. Example: `config get sync` returns `"none"`.

Tests call internal functions directly (`runInit`, `doSync`, `getSettingsValue`), not cobra handlers, so a handler-level guard leaves them passing.

## Rationale

Gating at the command layer keeps the re-enablement diff small: the guards come out, the prompts come back, and nothing else moves. Internal packages, validation logic, and tests stay exercised while the feature is off.

Three shortcuts break this: deleting the gated package (breaks tests, large re-enablement diff), commenting out the registration in `root.go` (the command becomes unreachable even for testing), and changing internal validation to reject the gated modes (breaks unit tests that call internal functions directly).

## Enforcement

Code review. WHEN a feature is gated, the reviewer verifies that `go test ./...` passes with no change to any test file.
