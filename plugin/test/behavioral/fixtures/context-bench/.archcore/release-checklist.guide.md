---
title: "Release Checklist"
status: accepted
---

## Prerequisites

- Write access to the package registry.

## Steps

1. Bump the version in @package.json.
2. Run `npm test`.
3. Publish with `npm publish`.

## Verification

The registry shows the new version.

## Common Issues

- A failed test blocks the release.
