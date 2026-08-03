---
title: "Sync Mode Strictly Controls Allowed Settings Fields"
status: accepted
tags:
  - "config"
  - "sync"
---

## Rule

The `sync` field in `settings.json` decides which other fields are allowed, required, or forbidden. Two layers enforce the constraints: JSON deserialization and semantic validation.

1. WHILE `sync` is `"none"`, `Settings.Validate` MUST reject a `project_id` value.
2. WHILE `sync` is `"none"`, `Settings.Validate` MUST reject an `archcore_url` value.
3. WHILE `sync` is `"cloud"`, `Settings.Validate` MUST reject an `archcore_url` value, because the cloud mode uses the hardcoded URL.
4. WHILE `sync` is `"on-prem"`, `Settings.Validate` MUST require an `archcore_url` value.
5. `Settings.Validate` MUST reject an unknown `sync` value.
6. `MarshalJSON` MUST serialize only the fields that the current sync mode allows, even while other fields hold values in memory.
7. The developer MUST add a sync-independent field, such as `language`, to `allowedFields` for every sync mode.
8. The developer MUST NOT make a sync-independent field required in any sync mode.

## Field matrix

| Field          | `none`        | `cloud`       | `on-prem`     | Notes                          |
| -------------- | ------------- | ------------- | ------------- | ------------------------------ |
| `project_id`   | **forbidden** | optional      | optional      | —                              |
| `archcore_url` | **forbidden** | **forbidden** | **required**  | cloud uses the hardcoded URL   |
| `language`     | optional      | optional      | optional      | sync-independent; default `en` |

A sync-independent field uses `omitempty`, so it stays out of the JSON file while the user has not set it.

## Rationale

Strict field validation prevents silent misconfiguration. Without it, a user could switch to `sync: "cloud"` while a stale `archcore_url` from a previous on-prem configuration remains in the file, and the CLI would quietly ignore one of the two values. The forbidden-field error forces a clean configuration state instead.

## Enforcement

- `Settings.Validate` and the custom `MarshalJSON` in `@internal/config/config.go` implement the matrix.
- Tests: the validation cases in `@internal/config/config_test.go`.
