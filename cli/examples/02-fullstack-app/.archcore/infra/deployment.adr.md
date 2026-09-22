---
title: "Container-per-Tier Deployment"
status: accepted
tags: [infra]
---

## Context

We want reproducible deploys without hand-managed servers.

## Decision

Each tier (frontend, API) ships as a container image, deployed via rolling
update behind a load balancer. The database is a managed PostgreSQL instance.
Config and secrets come from the environment.

## Consequences

Rollback is redeploying the previous image tag. No state lives in the container.
Latency and error budgets are tracked in @.archcore/infra/observability.doc.md.
