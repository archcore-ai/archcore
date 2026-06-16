---
title: "Secrets Never in Source"
status: accepted
tags: [security]
---

## Rule

Secrets (API keys, DB passwords, tokens) are read from the environment or the
secrets manager at runtime — never committed, never shipped in client bundles.
A leaked secret is **rotated** immediately, not just removed from history.

## Rationale

Anything in git history is compromised forever. Runtime injection keeps secrets
out of the build artifact.
