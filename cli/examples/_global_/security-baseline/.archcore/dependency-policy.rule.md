---
title: "Dependency Review & Pinning"
status: accepted
tags: [security]
---

## Rule

Production dependencies are pinned (lockfile committed). A new dependency needs
a short review note — why it's needed and a license check. The vulnerability
scanner runs in CI; a high or critical advisory blocks the merge.

## Rationale

Supply-chain risk is real. Pinning makes builds reproducible; review keeps the
dependency tree small and auditable.
