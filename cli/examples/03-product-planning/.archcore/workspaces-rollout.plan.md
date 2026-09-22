---
title: "Workspaces Rollout"
status: accepted
tags: [product]
---

## Phases

1. **Schema + API** behind a feature flag (@.archcore/workspaces-api.spec.md).
2. **Internal dogfood** — staff workspaces only.
3. **Beta** — opt-in for 10% of teams.
4. **GA** — default for new sign-ups, migration for existing accounts.

## Risk

Account → workspace migration is the riskiest step; it ships last with a
reversible data backfill.
