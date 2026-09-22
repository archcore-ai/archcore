---
title: "Payments Service — Overview"
status: accepted
---

## Overview

Handles charge capture and refunds against external processors. It mounts the
`company-standards` global but deliberately **overrides** one rule locally — see
`error-handling.rule.md`. Everything else (commits, logging, API versioning) is
inherited from the global unchanged.
