---
title: "Short-Lived Access Tokens + Refresh Rotation"
status: accepted
tags: [security]
---

## Context

Services need a uniform auth scheme that limits the blast radius of a stolen
token.

## Decision

Access tokens are JWTs with a 15-minute TTL. Refresh tokens are opaque,
single-use, and rotated on every refresh; reuse of a consumed refresh token
revokes the entire session family.

## Consequences

Stolen access tokens expire fast and refresh reuse is detectable. Every service
validates tokens the same way through the shared middleware.
