---
title: "Observability"
status: accepted
tags: [infra]
---

## Overview

Three pillars, all keyed by `request_id`:

- **Logs** — structured JSON, shipped to the aggregator.
- **Metrics** — RED (rate, errors, duration) per endpoint; p99 latency is an SLO.
- **Traces** — sampled distributed traces across frontend → API → DB.

The Q3 latency goal (@.archcore/product/q3-roadmap.plan.md) is measured here.
