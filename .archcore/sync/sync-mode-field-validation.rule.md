---
title: "Sync Mode Strictly Controls Allowed Settings Fields"
status: accepted
tags:
  - "config"
  - "sync"
---

## Rule

The `sync` field in `settings.json` defines which other fields are allowed, required, or forbidden. These constraints are enforced at two layers: JSON deserialization and semantic validation.

Some fields are **sync-independent** — allowed in all sync modes and not governed by `sync`.

## Field Matrix

| Field          | `none`        | `cloud`       | `on-prem`     | Notes                          |
| -------------- | ------------- | ------------- | ------------- | ------------------------------ |
| `project_id`   | **forbidden** | optional      | optional      | —                              |
| `archcore_url` | **forbidden** | **forbidden** | **required**  | cloud uses hardcoded URL       |
| `language`     | optional      | optional      | optional      | sync-independent; default `en` |

## Implications

- Setting `sync: "none"` with a `project_id` present is a validation error.
- Setting `sync: "cloud"` with an `archcore_url` is a validation error — cloud always uses the hardcoded URL.
- Setting `sync: "on-prem"` without `archcore_url` is a validation error.
- `MarshalJSON` only serializes fields legal for the current sync mode, even if other fields are set in memory.
- Unknown sync type values are rejected.
- Sync-independent fields (like `language`) are allowed in all modes and included in `allowedFields` for every sync type. They use `omitempty` and are omitted from JSON when not explicitly set.

## Rationale

Strict field validation prevents silent misconfiguration. A user cannot accidentally set `sync: "cloud"` while having a stale `archcore_url` from a previous on-prem config — the presence of the forbidden field triggers an explicit error, forcing a clean configuration state.