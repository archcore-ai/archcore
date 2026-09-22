---
title: "Error Handling (stricter local policy)"
status: accepted
tags: [backend]
---

## Rule

This service overrides the company error-handling baseline with a stricter
policy: in addition to wrapping with context, **every** wrapped error must carry
a typed `code` and be classified as retryable or terminal at the boundary.
Returning an unclassified error fails review.

## Why this overrides the global

The payments domain needs precise retry behavior, so the shared
`company-standards` error rule is not enough here. A local document on the same
topic **shadows** the global one — local always wins.
