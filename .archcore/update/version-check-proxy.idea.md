---
title: "Proxy GitHub Release Check Through archcore.ai to Avoid Rate Limits"
status: draft
tags:
  - "update"
---

## Idea

Route the "check latest version" request in `archcore update` through `archcore.ai` instead of hitting `api.github.com` directly. The server caches the GitHub API response (5–10 min TTL), shielding users from the 60 req/hour unauthenticated rate limit.

Download of archives and `checksums.txt` stays direct from `github.com/releases/download/` (CDN, no rate limit).

## Value

- Users never hit GitHub API rate limits, even behind corporate NAT where hundreds of developers share one external IP
- No `GITHUB_TOKEN` requirement — zero configuration for end users
- Archcore controls the endpoint — can add analytics, staged rollouts, or deprecation notices in the future

## Possible Implementation

1. Add an endpoint on the landing/server: `GET https://archcore.ai/api/v1/releases/latest`
2. Server fetches `https://api.github.com/repos/archcore-ai/cli/releases/latest` with a server-side `GITHUB_TOKEN` (5000 req/hr)
3. Cache response in memory with 5-minute TTL
4. Response format: `{ "tag_name": "v1.2.3" }` (same as GitHub API — drop-in replacement)
5. In CLI `CheckLatest()`: replace `api.github.com` URL with `archcore.ai/api/v1/releases/latest`

### Alternatives considered for this approach

- **Vercel Edge Function** — simple `fetch` + `Cache-Control` header, zero infrastructure
- **Vercel Serverless Function** — same but with explicit in-memory/KV cache
- **Cloudflare Worker** — if landing moves to Cloudflare

## Risks and Constraints

- Adds a dependency on `archcore.ai` availability for version checks (but update still works if user knows the version)
- Need to handle the case where proxy is down — fall back to direct GitHub API or show a clean error
- Cache staleness: 5-minute window where a just-published release isn't visible (acceptable for CLI updates)