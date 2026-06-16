---
title: "Manifest Must Only Update After Confirmed Server Response"
status: accepted
tags:
  - "sync"
---

## Rule

The sync manifest (`.archcore/.sync-state.json`) MUST only be updated **after** the server has confirmed acceptance of the synced files. Never update the manifest optimistically before receiving a response.

## Implications

- If the sync request fails (network error, server error, timeout), the manifest remains unchanged. The next sync will automatically retry the same changes.
- On partial success (HTTP 207), only files explicitly confirmed in the `accepted` response array are updated in the manifest. Files with errors retain their previous manifest state.
- Manifest writes are atomic (temp file + rename) to prevent corruption if the process crashes mid-write.

## Rationale

This is the fundamental safety guarantee of the sync system. If the manifest were updated before server confirmation, a failed sync could silently lose documents — the CLI would believe files were synced when they weren't. By updating only on confirmed success, sync failures are always recoverable by simply re-running the command.