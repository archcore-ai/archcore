---
title: "Release Process"
status: accepted
---

## Rule

Apps are released independently from `main`. A release is a tagged commit;
changelogs come from Conventional Commit history. A broken release is rolled
back by redeploying the previous tag, never by force-pushing.

## Rationale

Independent, reversible releases keep one app's deploy from blocking another's.
