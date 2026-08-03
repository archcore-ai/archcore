---
title: "CLI — Path Index in .sync-state.json (deferred perf hardening)"
status: draft
tags:
  - "hooks"
  - "plugin"
  - "roadmap"
---

**Status — deferred consumer, as of 2026-04-23.** When first drafted this plan was the CLI-side counterpart of a plugin task that consumed the index. That plugin release narrowed to delegated coverage only, so hook performance hardening is deferred and the consumer ships later.

The CLI work stays valid and independently shippable: producing `path_index` in `.sync-state.json` is additive and carries no behavioral risk to a current plugin version, because readers treat an unknown key as opaque. Shipping early only means the consumer catches up later. The promotion trigger for the consumer side is the first real-world repository whose injection hook breaches the host's 1-second timeout, or a deliberate decision to add performance hardening to a release.

**CLI versioning note (2026-05-12).** The plugin no longer pins a CLI version, because that file was removed with the bundled launcher. Users install the CLI from the official installer, and its lifecycle is decoupled from plugin releases. When the consumer ships, the compatibility matrix below applies the moment a user upgrades past CLI 0.1.8, with no plugin-side pin to bump.

## Goal

Extend the CLI to maintain a reverse index from source-code path tokens to the `.archcore/` documents referencing them, persisted alongside the relation manifest in `.sync-state.json`. A future plugin release consumes it in `bin/check-code-alignment`, replacing the per-token grep scan with a constant-time map lookup, bringing the hook well inside its 1-second timeout at around 100 documents and scaling essentially flat beyond that.

Out of scope: violation detection and an `archcore check` subcommand, planned separately; a per-document mtime index, since the plugin already reads mtime from the filesystem when ranking; a full-text or inverted content index, since grep over bodies remains adequate for topic mode and the `search_documents` MCP tool already covers that path; and session-scoped state, which lives outside the manifest.

## Context

**What the plugin needs.** The injection hook fires on every mutation outside `.archcore/`. For a path like `src/api/handlers/users.ts` it generates tokens longest first, capped at 5 levels, and runs one `grep -rlF` per token, skipping a document already matched by a longer token. At about 40 documents across 5 tokens that is 200 grep invocations at roughly 2–5 ms each, so 400 ms to 1 s in total; at 100 documents it breaches the timeout, and at 500 it fails reliably. What each grep actually answers is which document files mention this directory prefix — which is a reverse index, answerable in a tight shell loop from a JSON map with no body read at all.

**What the CLI already produces.** `.sync-state.json` carries the relation manifest and some metadata, is written by `archcore sync`, is read by `archcore doctor` and the MCP server, and is the git-committed truth for relations. Adding `path_index` on the same write path is a pure extension: same sync cadence, same atomic write, same tracked artifact, with no new command and no new file.

**Why not query at read time.** Two reasons. The hook shares a 1-second budget with the host, and even a fast Go subprocess — with exec cost, shell setup, and result marshalling — will not reliably beat a precomputed lookup, whereas a plain shell read of a JSON map costs hundreds of microseconds. And the hook ships as POSIX shell that does not assume the CLI is on PATH during execution, so a file-based index keeps its dependency surface unchanged.

## Design

### Schema

```jsonc
{
  "version": 1,                 // existing — bumped if structure changes
  "manifest_version": "0.2.0",  // existing CLI-defined schema version
  "relations": [ /* existing */ ],

  // NEW — additive, opt-in for old plugins.
  "path_index": {
    "schema": "v1",
    "built_at": "2026-04-23T17:30:00Z",  // ISO 8601 UTC
    "built_by": "archcore 0.1.8",
    "roots": ["src", "lib", "app", "pkg", "cmd", "internal",
              "apps", "packages", "modules", "components"],
    "tokens": {
      "src/api/handlers/": [".archcore/<domain>/api-handlers.rule.md",
                            ".archcore/<domain>/rest-conventions.adr.md"],
      "src/api/":          [".archcore/<domain>/rest-conventions.adr.md",
                            ".archcore/<domain>/api-auth.spec.md"]
    },
    "docs": {
      ".archcore/<domain>/rest-conventions.adr.md": {
        "type": "adr",
        "title": "REST over HTTP for the public surface",
        "tokens": ["src/api/", "src/"]
      }
    }
  }
}
```

Four decisions shape it. `tokens` is the forward lookup the plugin reads to get candidates, whose value arrays preserve indexing order — stable by document path — so the plugin can tiebreak lexicographically. `docs` is a metadata sidecar carrying type, title, and optional tags, so the plugin gets the type it needs for ranking and the title it needs for rendering without a second stat and frontmatter parse per candidate. `roots` is captured so the plugin can compare its configured set against what the CLI indexed against; divergence is fine, because the plugin filters by its own roots, but the record makes debugging easier. And `schema` with `built_by` carry forward compatibility, so an unrecognized schema label sends the plugin back to the grep fallback.

### Token extraction rules

For each document the indexer scans, it takes four steps. It scans the body, including code fences, for a reference beginning with one of the configured roots, and also scans any advisory frontmatter path array as literal tokens. It normalizes each hit by stripping quotes, trimming trailing punctuation, keeping a trailing slash to mark a directory reference, dropping relative segments, and rejecting anything with a space or control character. It expands each concrete reference into its prefix set, capped at 5 levels to mirror the plugin's cap. And it deduplicates per document, so a document referencing one directory three times contributes once.

The token set is deliberately a superset of what the plugin derives from a file path: the plugin still matches the file under edit against these tokens, and the CLI never needs to know any specific file.

### Source corpus

Only the five types the plugin injects contribute: `rule`, `cpat`, `adr`, `spec`, and `guide`. The vision and requirements types are excluded even when they mention paths, because the plugin filters them out to avoid injecting aspirational content on a code edit. `doc` is excluded for now, since it rarely references a path specifically, and can be promoted later without a schema bump. `rfc` is excluded as pre-decision. Only documents with status `accepted` or `draft` are indexed; a rejected document never appears, and further status filtering is the plugin's responsibility.

### Indexer module

A new Go package `internal/pathindex/` exposes three functions. `Extract` is pure and I/O-free, returning the token set for a parsed document body, type, and frontmatter, which makes it unit-testable against fixture bodies. `Build` is pure over a pre-loaded corpus and returns the full index matching the schema. `Merge` is not required for a first version, because `archcore sync` always rebuilds, and is reserved for a future incremental mode.

Integration lands in the existing sync package: after relations are reconciled, call `Build` and set the result on the manifest writer. That is one extra call in an existing code path, with no new subcommand. The manifest write is already atomic through a temp file and rename, and the index is assembled in memory beforehand, so the existing atomicity guarantee covers the new field and no consumer sees a torn manifest.

### Roots configuration

The CLI indexes against the same root list the plugin uses by default, which is the invariant that makes the plugin's root filter work against the index. `.archcore/settings.json` at `codeAlignment.sourceRoots` is consumed by both sides, so a mismatch is impossible when both respect it, and a `--roots` flag on sync is an advisory override for tests that is not persisted.

### Size and performance targets

The index budget is at most 5% of the summed body bytes of the included documents, which at roughly 40 documents and a 400 KB corpus is at most 20 KB, and at ten times that scale is still around 200 KB. Build time is linear in corpus size at no more than 50 ms per 100 documents on commodity hardware, validated by a benchmark. And the plugin's read cost is a single `jq` invocation or POSIX-shell parse at no more than 20 ms, measured in the bats suite.

## Tasks

**Phase CLI-1 — the indexer module.** Create the package with the public `Extract` and `Build` functions plus the internal normalization helpers. Port the plugin's token-extraction algorithm into Go with identical semantics for prefix generation, the 5-level cap, and the trailing-slash directory marker. Add unit tests covering a bare reference, a quoted reference, a frontmatter-array reference, six-level truncation, a reference inside a code fence against one in prose, type-filter exclusion, rejected-status exclusion, and a roots override. Add a benchmark over synthetic corpora at 40, 400, and 4000 documents.

**Phase CLI-2 — sync integration.** Call `Build` after relation reconciliation and attach the result to the manifest struct before serialization. Thread the active roots list through the settings package, resolving the settings file, then the flag, then the default. Update the serializer to include the new field while keeping the existing fields byte-identical when it is absent. Add a `--no-path-index` flag for emergency disablement, so an operator can drop the index without downgrading. And teach `archcore doctor` to accept but not require the field, checking the schema lightly and never hard-failing, because the index is advisory while manifest correctness is load-bearing.

**Phase CLI-3 — tests and documentation.** Add a sync test asserting the field appears after sync and disappears under the flag, a snapshot test locking the JSON shape, a documentation subsection covering the schema, the rebuild cadence, and the opt-out, and a changelog entry.

**Phase CLI-4 — release coordination.** Cut a minor CLI release as an additive feature with no breaking change, confirmed by running `archcore doctor` against a manifest written by the previous version, which must pass unchanged. The plugin consumer ships later and, with no version pin to bump, simply detects the schema label at runtime and switches paths, so coordination is version-free and the consumer's release notes point users at `archcore update` for the fast path.

## Plugin integration, for the future consumer

**The reader algorithm.** The hook checks whether the manifest is readable, whether `jq` is available, and whether the schema label matches; when all three hold it takes the fast path, reading the candidate list for each token straight out of the token map, and otherwise it runs the existing grep for that token. Everything after that — deduplication across tokens, ranking, and the top-3 cap — is unchanged.

Three rules hold whichever path fires. A missing index, a missing `jq`, or a schema mismatch falls back to grep silently, so the user never sees an error. An index present but empty yields no candidates, which is the same silent exit the hook already takes when nothing matches. And where the `docs` sidecar is present, the hook uses its cached type and title instead of re-reading frontmatter, falling back to parsing the title from the first lines of each candidate when an older CLI wrote no sidecar.

| State | Plugin behavior |
|---|---|
| Old CLI, no index | The current grep path, with unchanged runtime |
| New CLI, v1 index present | The index fast path, under 200 ms on a 100-document repository |
| New CLI, schema bumped to v2 | Falls back to grep, warning only in debug mode |
| Index present, `jq` missing | Falls back to grep, since a shell-only JSON parse is not worth the complexity |
| Index present but corrupt | Falls back to grep, and neither rewrites nor deletes the index |

**Environment and overrides.** `ARCHCORE_DISABLE_PATH_INDEX=1` forces the grep path regardless of index presence, which is useful for debugging and for bats cases. The existing injection kill-switch short-circuits the hook before any index work and is unaffected. No CLI-owned environment variable is added; the sync flag is the write-side equivalent.

| Plugin | CLI | Behavior |
|---|---|---|
| Current | 0.1.7 | Grep on every edit |
| Current | 0.1.8+ | The plugin still greps; the CLI writes the index with no consumer |
| Future consumer | 0.1.7 | The plugin greps as a fallback: no improvement, no regression |
| Future consumer | 0.1.8+ | The fast path, delivering the target |

Upgrade order therefore does not matter, because both directions are safe. Only the target combination delivers the performance, and every other combination is indistinguishable from today.

**Tests the consumer plan adds.** One bats file with a prebuilt manifest containing a canned index, asserting the hook emits the expected ranking without ever running grep — instrumented by stubbing `grep` on PATH to a failing binary, so an attempted grep fails the test. One with an invalid schema label, asserting the fallback still emits the expected output. And the existing cases, none of which regress, since index absence remains their default.

## Acceptance Criteria

1. `archcore sync` on a fresh repository produces a manifest with a valid index object conforming to the schema, verifiable by reading its schema label.
2. `archcore sync` on a pre-existing repository alters no relation data; only the new field appears.
3. `archcore doctor` passes on both a pre-migration and a post-migration manifest.
4. `archcore sync --no-path-index` produces a manifest without the field even when one was present, idempotently.
5. The benchmark shows build time at or under 50 ms for a 100-document synthetic corpus on commodity hardware.
6. Index size stays at or under 5% of the cumulative body size of the included documents on a real test repository.
7. The future consumer validates the fast path: the hook consumes the index when it and `jq` are present and falls back otherwise, with both paths emitting identical ranking on the same fixture.
8. Hook wall-clock time is at or under 200 ms on the 100-document fixture with the index, and at or above 400 ms on the grep fallback, which proves the fast path is actually taken.
9. `ARCHCORE_DISABLE_PATH_INDEX=1` forces grep even when the index is present, verified by a dedicated bats test.
10. The CLI changelog and sync documentation are updated, and the consumer cites this plan as its upstream dependency.

## Dependencies

- The plugin repository's future release, which is the eventual consumer. The CLI release precedes or parallels it, and upgrade order does not affect correctness, because the fallback absorbs any ordering.
- `jq` on the developer machine for the fast path. Without it the plugin falls back to grep. Adding `jq` to the documented prerequisites is a separate decision.
- The `codeAlignment.sourceRoots` settings key, which already exists for the plugin and needs no schema change; the CLI reads the same key.

## Risks

- **Index staleness between sync runs.** A user who edits a document and does not sync before the next source edit leaves the index stale. Mitigated because the hook falls back to grep for a token the index lacks, so there is no silent wrong answer. The longer-term fix is to have the post-mutation validator trigger a refresh, making the index eventually consistent with no user action.
- **Index growth at ten times repository scale.** At 400 to 500 documents the index may cross 100 KB, which is still fine for a tracked file; at 4000 the single-file approach is worth revisiting, possibly by splitting the index into a sibling artifact. Out of scope until a real repository approaches the limit.
- **Root-list drift between CLI and plugin.** A user who sets the settings key but runs an older CLI that ignores it gets an index built against the wrong roots. Mitigated by capturing the roots in the index header, so the plugin can warn in debug mode when its active list disagrees.
- **Code-fence false positives.** Scanning bodies picks up path-like strings inside examples that reference no real source path. Acceptable, because the index overestimates candidates while ranking compensates; if the noise becomes user-visible, add code-fence exclusion in a schema bump.
- **Shipping without a consumer.** The field sits unused, so schema decisions could ossify before a real consumer exercises them. Mitigated by an end-to-end smoke on a fixture repository that reads the schema back and asserts structural expectations independently of the hook.
