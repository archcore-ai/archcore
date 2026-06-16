---
title: "Version Check Strategy for archcore update"
status: draft
tags:
  - "update"
---

## Summary

Define how `archcore update` resolves the latest available version without requiring user credentials and without hitting GitHub API rate limits.

## Motivation

Current implementation calls `api.github.com/repos/archcore-ai/cli/releases/latest` directly. GitHub's unauthenticated rate limit is 60 requests/hour per IP. This is fine for individual developers but problematic for:

- Teams behind corporate NAT (shared IP)
- CI environments running version checks
- Offices/coworking spaces with many archcore users

We explicitly decided not to require `GITHUB_TOKEN` from users — the CLI must work with zero configuration.

## Detailed Design

### Option A: Proxy through archcore.ai (recommended)

```
CLI  →  GET https://archcore.ai/api/v1/releases/latest
         ↓
Server  →  GET api.github.com (with server-side token, cached 5 min)
         ↓
CLI  ←  { "tag_name": "v1.2.3" }
         ↓
CLI  →  GET github.com/releases/download/v1.2.3/archcore_*.tar.gz  (direct, no rate limit)
```

**Pros:** Full control, can add rollout logic, analytics, deprecation notices.
**Cons:** Dependency on archcore.ai uptime for version checks.

### Option B: Static version file on archcore.ai

Publish a static file at `https://archcore.ai/version.txt` (or `/releases/latest.json`) updated by CI after each release.

```
CLI  →  GET https://archcore.ai/version.txt
CLI  ←  "v1.2.3"
```

**Pros:** Simplest. No server logic — just a static file served by CDN. No rate limits. Fastest response.
**Cons:** Requires CI step to update the file on each release. If CI fails to update, users don't see new version.

Implementation: Add a step to `.github/workflows/release.yml` that writes the version to a file and deploys/commits it.

### Option C: Keep direct GitHub API (current)

```
CLI  →  GET api.github.com/repos/archcore-ai/cli/releases/latest
```

**Pros:** Zero infrastructure. Already working.
**Cons:** 60 req/hr rate limit. Can fail for shared IPs.

### Option D: Conditional requests with ETag caching

Cache the GitHub API response locally (`~/.archcore/cache/latest-release.json`) with the `ETag` header. Subsequent requests use `If-None-Match` — GitHub returns `304 Not Modified` without counting against the rate limit.

```
CLI  →  GET api.github.com (If-None-Match: "etag-from-last-check")
CLI  ←  304 Not Modified (doesn't count against rate limit)
         or
CLI  ←  200 + new data (counts as 1 request)
```

**Pros:** No server infrastructure. Dramatically reduces actual rate limit consumption. Works offline (uses cached version).
**Cons:** First request per IP still counts. Doesn't fully solve shared-IP NAT problem (each user's first request counts). Adds local file cache management.

### Option E: Hybrid — ETag + fallback to proxy

Use Option D (ETag caching) as primary, fall back to Option A (proxy) if GitHub returns 403/429 (rate limited).

**Pros:** Best of both worlds — usually zero server load, graceful degradation.
**Cons:** Most complex. Two code paths to maintain.

## Recommendation

**Option B (static version file)** for simplicity, with Option A as future upgrade path if we need richer metadata (release notes, minimum version, deprecation notices).

Option B requires:
1. A static file at `https://archcore.ai/version.txt`
2. One CI step: after GoReleaser, write version to file and deploy
3. CLI change: one URL swap in `CheckLatest()`

## Drawbacks

- Any approach using archcore.ai adds a dependency on that domain's availability
- Static file (Option B) has a slight delay between release and file update (CI pipeline time)

## Alternatives

All five options are detailed above. The "do nothing" alternative (Option C) is viable for the current user base but doesn't scale to teams.