---
title: "Design Tokens"
status: accepted
tags: [frontend]
---

## Overview

The design system exposes **tokens** — never raw values — for color, spacing,
radius, and typography. Consume them via CSS variables or the `tokens` package.

| Group  | Example token  | Meaning        |
| ------ | -------------- | -------------- |
| color  | `--color-bg`   | system surface |
| space  | `--space-4`    | 16px           |
| radius | `--radius-md`  | 8px            |

Hard-coded hex values and pixel literals in application code are rejected in
review.
