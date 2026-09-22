---
title: "Billing Service — Overview"
status: accepted
---

## Overview

The billing service owns subscriptions, invoices, and payment webhooks. It is
one of several repos that mount the shared **company-standards** global, so the
engineering baseline (errors, commits, API versioning, logging) lives there —
not duplicated here.

This repo keeps only what is specific to billing; see `local-conventions.rule.md`.
