---
title: "Resolve the Latest Release via the github.com Redirect, Not the REST API"
status: accepted
tags:
  - "release"
  - "update"
---

## Context

`archcore update` and both install scripts must resolve "what is the newest published release" before they can download anything. All three originally asked the GitHub REST API — `GET https://api.github.com/repos/archcore-ai/cli/releases/latest` — and parsed `tag_name` out of the JSON body.

### Current State

Before this decision, the same lookup was implemented three times:

- `internal/update/update.go` — `CheckLatest` decoded a `releaseResponse{TagName string}` struct
- `install.sh` — `parse_version_from_json` with a jq → python3 → grep/sed fallback cascade
- `install.ps1` — `Invoke-RestMethod`, then read `.tag_name`

All three optionally attached `Authorization: Bearer $GITHUB_TOKEN` when the variable was set.

### Problem Statement

Unauthenticated REST API requests are capped at **60 per hour per IP**. That budget belongs to the egress address, not to the user, so it is shared by everyone behind it:

- teams on corporate NAT or CGNAT draw from one 60/hour pool
- CI runners on shared cloud IP ranges exhaust it quickly
- the background version check in `cmd/update.go` fires from editor SessionStart hooks on every installed machine, multiplying request volume per user

Once the budget is spent GitHub answers `403`, and the failure is quiet in the worst way: users simply stop being told that updates exist, or a fresh install fails outright. Requiring `GITHUB_TOKEN` was explicitly ruled out — the CLI must work with zero configuration.

## Decision

Resolve the latest release tag from the **github.com web redirect** instead of the REST API.

`GET https://github.com/archcore-ai/cli/releases/latest` answers with a redirect whose `Location` already carries the tag:

```
Location: https://github.com/archcore-ai/cli/releases/tag/v1.2.3
```

Read the tag from the path segment after `/releases/tag/`, and do not follow the redirect.

### Rationale

- **No rate-limit budget** — plain `github.com` carries no `x-ratelimit-*` accounting, unlike `api.github.com`
- **No token** — zero-configuration UX is preserved
- **No infrastructure** — nothing to host, no CI step, no cache to invalidate, unlike every archcore.ai-based alternative
- **No JSON parser** — drops the jq/python3/grep cascade from `install.sh` and `encoding/json` from `internal/update`
- **No new trust boundary** — release assets already come from `github.com`

### Implementation Notes

- `CheckLatest` halts redirects on a **copy** of the HTTP client (`CheckRedirect` → `http.ErrUseLastResponse`). Copying matters: `Apply` reuses the caller's client and must still follow the asset redirect chain. A test asserts `CheckRedirect` is still `nil` on the caller's client after the call.
- **Any `3xx` is accepted.** GitHub answers `302` today, but `301`/`303`/`307`/`308` are equally legitimate ways to point at the tag page, and the `Location` check below is the real gate — a redirect that does not land on a tag page fails regardless of which status carried it. Anything outside `3xx` is an error.
- **The tag comes from `resp.Location()`, not the raw header.** That resolves a relative `Location` against the request URL and normalizes dot-segments, and reading `.Path` drops query and fragment. Parsing the raw header string instead lets `…/releases/tag/v1.2.3?a=b#frag` and `…/releases/tag/../../../evil` both smuggle a bogus tag past the marker search — verified by reverting the block and watching the tests fail. `resp.Location()` also returns `http.ErrNoLocation` when the header is absent, which gives missing-Location its own error branch instead of collapsing onto "not a tag page".
- **A repo with no published release still answers `302`** — to the bare `/releases` page rather than a tag — so it fails the tag-segment check, not the status check. Verified unauthenticated against `github/gitignore` and `golang/go`, both of which redirect to `https://github.com/<repo>/releases`.
- The redirect body is drained through `io.LimitReader` (`maxRedirectBodyDrain`, 4 KiB) before close. A `3xx` body is empty so connection reuse is preserved, but an unexpected HTML error page must not stall the 2s budget the editor-hook check runs under.
- `install.sh` reads the header with `curl -fsS -I -o /dev/null -w '%{redirect_url}'`.
- `install.ps1` must read the `Location` off a **thrown** exception. Both stacks treat a 3xx as a terminating error when `-MaximumRedirection 0` is set — PowerShell 7 raises `HttpResponseException`, Windows PowerShell 5.1 a `WebException` — so the non-throwing branch is effectively unreachable and kept only for a stack that returns the response instead. The two stacks then expose `Location` through mutually exclusive shapes: PS7's `HttpResponseHeaders` offers only the typed `.Location` (Uri) property and throws on a string index, while PS5.1's `WebHeaderCollection` has no `.Location` property but does support the indexer. `Set-StrictMode` turns each mismatch into a terminating error, so both attempts must be guarded — reading only one silently yields an empty version and fails every unpinned install on that stack. PS5.1 may additionally hand back a `string[]` for a repeated header.

## Alternatives Considered

Five options were weighed before this decision. All were rejected in favour of the redirect.

### Alternative 1: Proxy through archcore.ai

A server-side endpoint that calls the API with its own token and caches for 5 minutes.

- Rejected: it adds a service to operate and a hard dependency on archcore.ai availability for a lookup a static redirect already answers for free.

### Alternative 2: Static version file on archcore.ai

Publish `version.txt` from CI after each release. This was the recommendation of the earlier version-check proposal.

- Rejected: still infrastructure, plus a release-pipeline step that can fail silently and leave every user pinned to a stale version. The redirect is in sync with the actual release by construction.

### Alternative 3: Keep the REST API

- Rejected: this is the problem being solved.

### Alternative 4: ETag conditional requests

Cache locally and send `If-None-Match`; a `304` does not count against the budget.

- Rejected: a mitigation, not a fix. Each IP's first request still counts, so the shared-NAT case — the actual failure mode — remains. It also adds local cache management.

### Alternative 5: Hybrid ETag with proxy fallback

- Rejected: the most complex option, two code paths to maintain, and it still requires the proxy from Alternative 1.

## Consequences

### Positive

- Version checks and installs no longer fail on shared egress addresses
- `GITHUB_TOKEN` is no longer consulted for version resolution; it still applies to asset downloads in the install scripts
- One lookup mechanism, no JSON decoding in three languages

### Negative

- The decision rests on **stable but undocumented** github.com behavior rather than a versioned API contract. If GitHub changes the redirect shape, `CheckLatest` now fails cleanly — into one of four distinguishable errors (non-3xx status, no `Location`, unparseable `Location`, redirect not landing on a tag page), each pinned by its own test row — and both installers tell the user to pin `ARCHCORE_VERSION`. This only holds because the tag is taken from a *resolved* URL; the first implementation substring-matched the raw header and silently produced garbage tags instead.
- The redirect carries only the tag — no release notes, publish date, or asset list. Any future need for richer metadata (deprecation notices, staged rollouts, minimum-version gates) reopens this decision, with Alternative 1 as the upgrade path.
- Residual: `url.URL.Path` is percent-decoded, so a `Location` of `…/releases/tag/v1.2.3%2F..%2Fevil` still yields a tag string containing `/../`. Not exploitable in practice — it requires control of the TLS-protected redirect target, checksum verification still gates installation, and the download request re-escapes the path — but it is the one gap left by extracting from `.Path`.

### Changes Made

- @internal/update/update.go — `CheckLatest` resolves the tag via `resp.Location()` and searches the resolved path; accepts any `3xx`; `releaseResponse` and the `encoding/json` import removed; `tagPathMarker` and `maxRedirectBodyDrain` constants and a `client()` helper added; redirect body drain bounded
- @install.sh — `parse_version_from_json` deleted; `get_latest_version` reads `%{redirect_url}` and validates the `*/releases/tag/*` shape
- @install.ps1 — `Get-LatestVersion` reads `Location` off the thrown exception, guarding both the PS7 typed-property and PS5.1 indexer shapes
- @internal/update/update_test.go, @cmd/update_test.go, @cmd/update_check_test.go — test servers answer a redirect with `Location` instead of JSON; `TestCheckLatest` is table-driven over 302/301/307/308 plus relative-`Location` and dot-segment rows, each asserting no `CheckRedirect` leak; the error table asserts `errContains` per row and covers non-3xx-with-valid-tag, traversal segments, unparseable `Location`, missing `Location`, empty tag, and no-release-published
- Updated self-update-command.doc.md (retyped from guide on 2026-08-14) and install-script-usage.guide.md
