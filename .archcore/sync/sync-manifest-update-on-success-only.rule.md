---
title: "Manifest Must Only Update After Confirmed Server Response"
status: accepted
tags:
  - "sync"
---

## Rule

1. The CLI MUST update the sync manifest (`.archcore/.sync-state.json`) only after the server confirms that it accepted the synced files.
2. The CLI MUST NOT update the manifest before it receives the server response.
3. WHEN the server returns HTTP 207, the CLI MUST update the manifest only for the files listed in the `accepted` array and MUST leave every other entry at its previous state.
4. IF the sync request fails through a network error, a server error, or a timeout, THEN the CLI MUST leave the manifest unchanged.
5. The CLI MUST write the manifest atomically, through a temporary file and a rename.

## Current behavior

A failed sync leaves the manifest at its last confirmed state, so the next `archcore sync` run re-sends the same changes without an extra flag.

## Rationale

This is the safety guarantee of the sync system. An optimistic manifest update would let a failed sync lose documents silently: the CLI would treat unsent files as sent. Updating only after confirmation keeps every failure recoverable by re-running the command.

## Enforcement

- Code review: the reviewer verifies that the manifest write follows the server response in the sync command path.
- Atomic write: the temporary-file-and-rename sequence prevents a partially written manifest if the process stops mid-write.
