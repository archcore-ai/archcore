---
title: "Telemetry Provider and Approach for CLI"
status: accepted
tags:
  - "component:cli"
  - "telemetry"
---

## Summary

This RFC compares providers for anonymous CLI usage telemetry and selects PostHog. The provider choice stands. The rest of the original design was not adopted; see "Detailed Design".

## Motivation

Without telemetry, product decisions about commands, OS/arch priorities and feature adoption rest on guesswork. The goal is a lightweight, privacy-respecting signal.

## Detailed Design

The shipped design differs from the first proposal in four points. The contract is `cli-update-telemetry.spec`.

- Identifier: the shared `install-id` file written by the installer and the CLI. The proposed SHA-256 hash of hostname, user name and machine id was not adopted.
- Opt-out: the `DO_NOT_TRACK` and `ARCHCORE_TELEMETRY_OPTOUT` environment variables. The proposed `"telemetry": false` setting was not adopted, because `archcore update` runs outside any project.
- Events: the three update events only. The proposed `cli_command`, `init_completed`, `sync_*`, `mcp_tool_call` and `hooks_installed` events were never built.
- Transport: a plain HTTP capture request. The proposed `posthog-go` SDK is not a dependency.

The API key is injected through ldflags (option B of the first proposal). The reason: it keeps the key out of source and reuses the `version` injection path, and an unkeyed build stays inert.

## Drawbacks

- User trust: some developers distrust any telemetry. A clear opt-out and a minimal payload mitigate this.
- GDPR: an anonymous random identifier is not PII under most readings, but the privacy page must say so.

## Alternatives

| Aspect | PostHog (chosen) | Mixpanel | Amplitude |
|---|---|---|---|
| Go SDK | Official | Community, less active | Official |
| Self-hosted option | Yes | No | No |
| Free tier | 1M events/mo | 20M events/mo | 50M events/mo, with limits |
| Hosting | EU available | US only | US-primary |
| OSS | Yes (MIT) | No | No |

- Mixpanel: larger free tier, but no self-hosted option and a weaker Go SDK.
- Amplitude: built for web and mobile apps; too much overhead for CLI telemetry.
- Plausible or Umami: no SDK needed and privacy-first, but no funnels, user properties or retention analysis. It answers "how many use the CLI", not "how do they use it".
- Own endpoint: full control, but dashboards, aggregation and retention analysis must be built from scratch. Not justified at current scale.
- No telemetry: no quantitative data, and no error-rate visibility as the user base grows.

## Recommendation

PostHog, with an opt-out model and ldflags key injection. It has a self-hosted path for a later migration, OSS alignment and a sufficient free tier.
