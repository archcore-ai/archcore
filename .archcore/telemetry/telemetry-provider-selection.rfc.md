---
title: "Telemetry Provider and Approach for CLI"
status: draft
tags:
  - "telemetry"
---

## Summary

Proposal to add anonymous usage telemetry to the archcore CLI. This RFC evaluates provider options, opt-in/opt-out strategies, and data collection scope to select the best approach for understanding CLI adoption and usage patterns.

## Motivation

Currently we have zero visibility into how the CLI is used:
- Which commands are popular, which are never used
- Where users hit errors and drop off
- How many active installations exist
- Which OS/arch combinations to prioritize
- Whether new features get adoption

Without telemetry, product decisions are based on guesswork. A lightweight, privacy-respecting telemetry system solves this while maintaining user trust.

## Detailed Design

### Provider: PostHog

PostHog is the recommended provider. See **Alternatives** section for comparison.

**Integration:** `github.com/posthog/posthog-go` SDK — lightweight, supports async batching, no external dependencies.

**Data flow:**
1. CLI starts → `telemetry.New()` checks opt-out → creates PostHog client or no-op
2. Command runs → `Capture(event, props)` enqueues event (non-blocking)
3. CLI exits → `Close()` flushes batch to PostHog API
4. PostHog stores and aggregates on their cloud (or self-hosted instance)

**Distinct ID:** SHA-256 hash of `hostname + os.UserName + machine-id`. Anonymous, stable, not reversible.

### Opt-out mechanism

Three levels, any one disables telemetry:
1. **Env var `DO_NOT_TRACK=1`** — industry standard (consoledonottrack.com)
2. **Env var `ARCHCORE_TELEMETRY_OPTOUT=1`** — tool-specific override
3. **Settings field `"telemetry": false`** in `settings.json` — persistent per-project

Priority: env vars override settings. If any opt-out is active, the client is a no-op (zero network calls).

### Events and properties

Minimal event set — only what drives product decisions:

- `cli_command` — command name, success/error, duration
- `init_completed` — sync type, detected agents
- `sync_completed` / `sync_failed` — file counts, error category
- `update_check` — version comparison
- `mcp_tool_call` — tool name only
- `hooks_installed` — agent type

**Hard rules:** no file paths, no file content, no project names, no URLs, no error messages (only error type categories).

### API key handling

PostHog write-only API keys are designed to be public (like Segment write keys). Two options:

**A. Hardcoded in source** — simplest, used by most OSS projects (Next.js, PostHog CLI itself). The key can only write events, not read.

**B. Injected via ldflags** — keeps key out of source. Slightly more complex build, but aligns with existing `version` injection pattern.

Recommendation: **B (ldflags)** — consistent with existing build approach, trivially easy since the pattern is already in place.

## Drawbacks

- **Binary size increase** — PostHog Go SDK adds ~2-3 MB. Acceptable for a CLI tool.
- **Network dependency** — first-run latency if DNS is slow. Mitigated by async flush on exit.
- **User trust** — some developers distrust any telemetry. Mitigated by clear opt-out, notice banner, and minimal data collection.
- **GDPR / privacy** — anonymous machine hash is not PII under most interpretations, but should be documented clearly.

## Alternatives

### Alternative 1: Mixpanel

| Aspect | PostHog | Mixpanel |
|---|---|---|
| Go SDK | Official, maintained | Community, less active |
| Self-hosted option | Yes (Docker) | No |
| Free tier | 1M events/mo | 20M events/mo |
| Product analytics | Full (funnels, retention) | Full |
| Privacy | EU hosting available | US only |
| OSS | Yes (MIT) | No |

**Verdict:** Mixpanel has a larger free tier but no self-hosted option and weaker Go SDK. PostHog wins on self-hosted flexibility and OSS alignment.

### Alternative 2: Amplitude

| Aspect | PostHog | Amplitude |
|---|---|---|
| Go SDK | Official | Official |
| Self-hosted option | Yes | No |
| Free tier | 1M events/mo | 50M events/mo (with limits) |
| CLI focus | Neutral | Designed for apps, overkill for CLI |
| Privacy | EU hosting | US-primary |

**Verdict:** Amplitude is powerful but designed for web/mobile apps. Overhead and complexity are unnecessary for CLI telemetry. No self-hosted option.

### Alternative 3: Plausible / Umami (web-analytics repurposed)

Lightweight, privacy-focused web analytics used by some CLI tools (e.g. Astro uses a simple pixel).

**Pros:** Ultra-lightweight, no SDK needed (just HTTP POST), privacy-first by design.
**Cons:** Not designed for event analytics — no funnels, no user properties, no retention analysis. Would need custom dashboard for everything beyond page views.

**Verdict:** Too limited for product analytics. Good for "how many people use the CLI" but not "how do they use it".

### Alternative 4: Self-hosted solution (own endpoint)

Send events to `app.archcore.ai/api/v1/telemetry` and store in our own DB.

**Pros:** Full control, no third-party dependency, integrates with existing server.
**Cons:** Must build dashboards, aggregation, retention analysis from scratch. Significant engineering effort. Maintenance burden.

**Verdict:** Not justified at current scale. Can migrate later if needed — the telemetry package abstracts the provider behind an interface.

### Alternative 5: No telemetry

Rely on GitHub stars, issues, and user interviews.

**Pros:** Zero privacy concerns, zero implementation effort.
**Cons:** No quantitative data. Decisions based on anecdotes from vocal minority. No error rate visibility.

**Verdict:** Insufficient for informed product decisions as the user base grows.

## Recommendation

**PostHog** with opt-out model, ldflags key injection, and minimal event set. Best balance of:
- Mature Go SDK with async batching
- Self-hosted option for future migration
- OSS alignment
- Sufficient free tier for current scale
- Provider abstracted behind interface for easy swap later