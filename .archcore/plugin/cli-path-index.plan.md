---
title: "CLI — Path Index in .sync-state.json (deferred perf hardening)"
status: draft
tags:
  - "hooks"
  - "plugin"
  - "roadmap"
---

## Status — Deferred consumer (as of 2026-04-23)

**Scope update.** When first drafted, this plan was the CLI-side counterpart of plugin phase v0.4.0 task A1 (hook consumes `path_index`). Plugin v0.4.0 has since narrowed to delegated-coverage only (see `jtbd1-phase2-hardening-delegated.plan` — B1+B2). Hook performance hardening is deferred, so the plugin-side consumer of this CLI work ships in a later release rather than v0.4.0.

The CLI work itself remains valid and independently shippable — producing `path_index` in `.sync-state.json` is safe, additive, and carries no behavioural risk to current plugin versions (they ignore unknown keys). Shipping this plan early just means the consumer catches up later.

Promotion trigger for the consumer side: the first real-world repo whose `check-code-alignment` hook breaches the host's 1-second timeout, OR a deliberate decision to add perf hardening to a release for non-urgency reasons.

---

## Goal

Extend the Archcore CLI to maintain a reverse index from source-code path tokens to the `.archcore/` documents that reference them, persisted alongside the existing relation manifest in `.sync-state.json`. A future plugin release consumes the index in `bin/check-code-alignment` to replace the per-token grep scan with O(1) map lookup, bringing the hook well inside its 1-second timeout on repos with ~100 documents and scaling essentially flat beyond that.

Out of scope for this plan:

- Violation detection or `archcore check` CLI subcommand (planned separately for v0.5.0+)
- Per-document mtime index (the plugin already reads mtime from the filesystem when it needs to rank).
- Full-text / inverted content index (grep over bodies is still adequate for topic mode; the plugin's `search_documents` MCP tool already handles that path).
- Session-scoped state (that lives in `/tmp`, not `.sync-state.json`).

## Context

### What the plugin needs (when the consumer eventually ships)

`bin/check-code-alignment` fires on every `Write|Edit` outside `.archcore/`. For a file path like `src/api/handlers/users.ts` the hook generates tokens longest-first: `src/api/handlers/`, `src/api/`, `src/`, capped at 5 levels. For each token it runs `grep -rlF <token> .archcore --include='*.md'` and walks the matches. A doc matched by a longer token is not re-scored by shorter ones.

The cost today at ~40 docs × 5 tokens is 200 grep invocations (~2–5 ms each, ~400 ms–1 s total). At 100 docs it breaches the 1-second host timeout. At 500 docs it fails reliably.

What the hook actually needs from each grep is: *which document files mention this directory prefix?* That is a reverse index. The plugin can answer it in a tight shell loop if the index is present as a JSON map, no bodies read at all.

### What the CLI already produces

`.sync-state.json` today carries the relation manifest (source/target/type triples) plus some metadata. The file is written by `archcore sync` and read by `archcore doctor` / other subcommands, and serves as the git-committed truth for relations. Readers treat unknown top-level keys as opaque, so additions are backward-compatible. The plugin's MCP server reads from the same file when listing relations.

Adding `path_index` on this same write path is a pure extension: same sync cadence, same atomic write, same git-tracked artifact. No new commands, no new files.

### Why not query at read-time

Two reasons.

1. **Budget.** The hook has a 1-second budget shared with the host. Even a fast Go subprocess with exec cost, shell setup, and result marshalling won't be reliably faster than a precomputed lookup. A plain shell read of a JSON map via the existing Unix pipeline (`jq`, or in-POSIX-shell fallback) is hundreds of microseconds.
2. **Portability.** The plugin deliberately ships the hook as POSIX shell (see `check-code-alignment`); it does not assume the CLI binary is on PATH during hook execution. A file-based index keeps the hook's dependency surface unchanged — read, parse, lookup, emit.

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
      "src/api/handlers/": [".archcore/auth/api-handlers.rule.md",
                            ".archcore/plugin/rest-conventions.adr.md"],
      "src/api/":          [".archcore/plugin/rest-conventions.adr.md",
                            ".archcore/security/api-auth.spec.md"],
      "src/":              [".archcore/plugin/rest-conventions.adr.md",
                            ".archcore/security/api-auth.spec.md",
                            ".archcore/monorepo-layout.doc.md"]
    },
    "docs": {
      ".archcore/plugin/rest-conventions.adr.md": {
        "type": "adr",
        "title": "REST over HTTP for the public surface",
        "tokens": ["src/api/", "src/"]
      }
    }
  }
}
```

Key decisions:

- **`tokens` is the forward lookup** — plugin reads `tokens[<dir>/]` to get ranked candidates. Value arrays preserve the order in which documents were indexed (stable by `.archcore/` path), so the plugin can tiebreak lexicographically.
- **`docs` is a sidecar for metadata** — type, title, optional tags. The plugin needs type for ranking (rule > cpat > adr > spec > guide) and title for rendering; reading it here avoids a second file stat + frontmatter parse per candidate.
- **`roots` is captured** — plugin can check its configured source-root set against the set the CLI indexed against. Divergence is fine (plugin filters by its own roots), but the captured list makes debugging easier.
- **`schema` and `built_by` carry forward-compatibility** — if schema evolves, the plugin can downgrade to grep fallback when it doesn't recognise the schema label.

### Token extraction rules

For every `.archcore/**/*.md` document the indexer scans:

1. **Source-root frontmatter and inline mentions.** Scan the body (and codefences within it) for patterns matching `^|[\s`'"(\[]<root>/(?:[\w.\-]+/)*` where `<root>` is one of the configured roots. Also scan YAML frontmatter `paths:` / `affects:` / `touches:` arrays if present (advisory — treated as literal tokens).
2. **Normalize** — strip quotes, trim trailing punctuation, keep trailing `/` to mark "directory reference", drop `.` / `..` segments, reject anything containing spaces or control chars.
3. **Expand to prefix set.** For each concrete reference `src/api/handlers/users.ts`, emit all prefix tokens `src/api/handlers/`, `src/api/`, `src/`. For a bare directory `src/api/` emit `src/api/`, `src/`. Cap at 5 levels (mirrors the plugin's cap).
4. **Deduplicate per document.** A doc referencing `src/api/` three times contributes once to `tokens['src/api/']`.

Token set is deliberately a superset of what the plugin would derive from a file path — the plugin still has to match the file under edit to these tokens, but the CLI does not need to know any specific file.

### Source corpus

Only eligible document types contribute to the index:

- `rule`, `cpat`, `adr`, `spec`, `guide` — mirrors the plugin's type allowlist.
- `prd`, `idea`, `plan`, `task-type`, `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs` — **excluded** from the index even when they mention paths, because the plugin explicitly filters those out to avoid injecting aspirational or vision content on code edits.
- `doc` — **excluded** for now (rarely references paths specifically; can be promoted to included later without a schema bump).
- `rfc` — **excluded** (pre-decision status).

Status filter: only documents with `status: accepted` or `status: draft` are indexed. `rejected` documents never appear in the index. The plugin is responsible for any further status filtering.

### Indexer module

New Go package `internal/pathindex/`:

- `pathindex.Extract(doc DocMeta) []string` — pure function, no I/O, given a parsed document body + type + frontmatter returns the token set. Unit-testable with fixture bodies.
- `pathindex.Build(corpus Corpus, roots []string) Index` — pure function over a pre-loaded corpus. Returns the full `Index` struct matching the JSON schema.
- `pathindex.Merge(prev, next Index) Index` — not strictly required in v1 because `archcore sync` always rebuilds; reserved for future incremental-update mode.

Integration in `internal/sync/` (existing): after relations are reconciled, call `pathindex.Build(corpus, defaultRoots)` and set the result on the `.sync-state.json` writer. One extra call in an existing code path; no new subcommand.

### Write ordering

`.sync-state.json` write is already atomic (write to temp file + rename). The `path_index` is assembled in memory before the write, so the existing atomicity guarantee covers the new field. Plugin consumers will never see a torn manifest.

### Roots configuration

The CLI indexes against the *same* root list the plugin uses by default — this is the invariant that makes the plugin's root filter work correctly against the index. Overrides:

- `.archcore/settings.json` → `codeAlignment.sourceRoots` — consumed by both CLI and plugin. If present, the CLI uses this list for indexing and the plugin uses it for filtering. Mismatch is impossible if both respect the same config.
- CLI flag `archcore sync --roots <list>` — advisory override, not persisted. Primarily for tests.

### Size and perf targets

- Index size budget: **≤ 5% of sum of `.archcore/` body bytes**. At ~40 docs / ~400 KB corpus this is ≤ 20 KB, which is negligible. At 10× scale we are still at 200 KB — well within any file-handling limit.
- Build time budget: **linear in corpus size, ≤ 50 ms per 100 documents** on commodity hardware. Validated by benchmark in `internal/pathindex/pathindex_bench_test.go`.
- Plugin read cost: **single `jq` invocation or POSIX-shell parse, ≤ 20 ms**. Measured in the plugin's bats suite.

## Tasks

### Phase CLI-1 — Indexer module

- Create `internal/pathindex/pathindex.go` with the public `Extract` / `Build` functions and internal normalization helpers.
- Port the plugin's token-extraction algorithm into Go with identical semantics (prefix generation, 5-level cap, trailing-slash directory marker).
- Create `internal/pathindex/pathindex_test.go` covering: bare reference, quoted reference, frontmatter-array reference, 6-level path truncation, doc referencing paths in codefence vs prose, type-filter exclusion, rejected-status exclusion, roots override.
- Create `internal/pathindex/pathindex_bench_test.go` with synthetic corpora at 40 / 400 / 4000 documents.

### Phase CLI-2 — Sync integration

- Modify `internal/sync/sync.go` (or equivalent): after relation reconciliation, call `pathindex.Build(corpus, activeRoots)` and attach the result to the manifest struct before serialization.
- Thread the active roots list through `internal/config/settings.go` — resolve priority `.archcore/settings.json` → CLI flag → default.
- Update the manifest serializer to include `path_index`; ensure existing manifest fields stay byte-identical when `path_index` is absent (unchanged behaviour when the indexer is disabled).
- Add `--no-path-index` CLI flag for emergency disablement. Operators can drop the index without a CLI downgrade.
- Update `archcore doctor` to accept (but not require) `path_index` and lightly check schema (`schema=="v1"` means required keys present). Never hard-fail on index anomalies — the index is advisory, manifest correctness is load-bearing.

### Phase CLI-3 — Tests and docs

- Update `internal/sync/sync_test.go` with a fixture that asserts `path_index` appears after sync and disappears under `--no-path-index`.
- Add a snapshot test (`testdata/sync/with-path-index/`) to lock the JSON shape.
- Update `docs/sync.md` or equivalent CLI docs with a "Path index" subsection documenting the schema, the rebuild cadence, and the opt-out flag.
- Add a `CHANGELOG.md` entry under the CLI's next release.

### Phase CLI-4 — Release coordination (when plugin consumer ships)

- Cut CLI release (target `0.1.8` from current `0.1.7` — minor bump, additive feature, no breaking changes). Confirm by running `archcore doctor` against a plugin repo with the *previous* CLI's `.sync-state.json` — must pass unchanged.
- Plugin consumer ships in a later release (not v0.4.0); coordination with that release's planning owns the `bin/CLI_VERSION` bump.
- Announce in CLI CHANGELOG; plugin-side changelog entry appears only when the consumer ships.

## Plugin integration (consumption details — future plugin release)

### Reader algorithm in `bin/check-code-alignment`

Pseudocode (actual implementation in POSIX shell with `jq` optional):

```sh
INDEX_FILE=".archcore/.sync-state.json"
USE_INDEX=0
if [ -r "$INDEX_FILE" ] && command -v jq >/dev/null 2>&1; then
  if jq -e '.path_index.schema == "v1"' "$INDEX_FILE" >/dev/null 2>&1; then
    USE_INDEX=1
  fi
fi

for TOKEN in $TOKENS; do
  if [ "$USE_INDEX" = "1" ]; then
    # Fast path — O(1) per token.
    DOCS=$(jq -r --arg t "$TOKEN" '.path_index.tokens[$t][]?' "$INDEX_FILE")
  else
    # Fallback — existing grep behaviour.
    DOCS=$(grep -rlF "$TOKEN" .archcore --include='*.md' 2>/dev/null)
  fi
  # Remainder (dedup across tokens, ranking, top-3) unchanged.
done
```

Rules the plugin enforces regardless of which path fires:

- Index absence OR `jq` absence OR schema mismatch → silent fallback to grep. User never sees an error.
- Index present but empty (`tokens: {}`) → no candidates → same exit-0-silent behaviour the hook has today when no docs match.
- If `docs` sidecar is present, use its cached `type` and `title` instead of re-reading frontmatter. If absent (older CLI), fall back to current behaviour of parsing title from the first 10 lines of each candidate.

### Fallback matrix

| State                        | Plugin behaviour                                      |
|------------------------------|-------------------------------------------------------|
| Old CLI → no `path_index`    | Current grep path; unchanged runtime                  |
| New CLI → v1 index present   | Index fast path; < 200 ms on 100-doc repos            |
| New CLI → schema bump to v2  | Fall back to grep; warn only in debug mode            |
| Index present, `jq` missing  | Fall back to grep (shell-only POSIX JSON parse is not worth the complexity for v1) |
| Index present but corrupt    | Fall back to grep; do not rewrite or delete the index |

### Environment and overrides

- `ARCHCORE_DISABLE_PATH_INDEX=1` — plugin forces the grep path regardless of index presence. Useful for debugging and for per-test cases in the bats suite.
- `ARCHCORE_DISABLE_INJECTION=1` — already exists; short-circuits the hook entirely before any index work, so unaffected.
- No CLI-owned env vars are added; the `--no-path-index` flag on the CLI is the equivalent on the write side.

### Version compatibility matrix

| Plugin (hook) | CLI          | Behaviour                                                              |
|---------------|--------------|------------------------------------------------------------------------|
| ≤ 0.4.x       | 0.1.7        | Current state — grep on every edit                                     |
| ≤ 0.4.x       | 0.1.8+       | Plugin still greps (unchanged); CLI writes the index but no consumer   |
| Future plugin | 0.1.7        | Plugin greps (fallback); no perf improvement but no regression         |
| Future plugin | 0.1.8+       | Fast path; target delivered                                            |

Upgrade order therefore does not matter — both directions are safe. The target state (consumer-shipped plugin + CLI 0.1.8) delivers the performance; any other combo is indistinguishable from v0.3.0 behaviour.

### Plugin tests added by the future consumer plan

- `test/unit/check-code-alignment-index.bats` — fixture with a prebuilt `.sync-state.json` containing a canned `path_index`. Asserts the hook emits the expected top-3 ranking without ever running `grep -rlF`. Instrumented by stubbing `grep` on PATH to a failing binary — if the hook tries to grep, the test fails.
- `test/unit/check-code-alignment-fallback.bats` — `.sync-state.json` present but with invalid `path_index.schema`. Asserts the hook falls back to grep and still emits the expected output.
- `test/unit/check-code-alignment.bats` (existing 13 cases) — none regress; index absence remains the default path in those tests.

## Acceptance Criteria

1. `archcore sync` on a fresh repo produces a `.sync-state.json` with a valid `path_index` object conforming to the schema above, verifiable via `jq '.path_index.schema' == "v1"`.
2. `archcore sync` on a pre-existing repo (without `path_index` in the prior manifest) does not alter any relation data; only the new `path_index` field appears.
3. `archcore doctor` passes on both pre-migration and post-migration manifests.
4. `archcore sync --no-path-index` produces a manifest without `path_index` even if one was present before; idempotent.
5. Benchmark in `internal/pathindex/pathindex_bench_test.go` shows build time ≤ 50 ms for a 100-document synthetic corpus on commodity hardware.
6. Index size ≤ 5% of the cumulative body size of included documents on a real-world test repo.
7. Plugin consumer (future release) validates the fast path: hook consumes the index when present and `jq` is available; falls back to grep otherwise. Both paths emit identical ranking output on the same fixture.
8. Plugin hook wall-clock time ≤ 200 ms on the 100-doc fixture when the index is used; ≥ 400 ms on the grep fallback (asserted to prove the fast path is actually being taken). Validated by the consumer plan.
9. `ARCHCORE_DISABLE_PATH_INDEX=1` forces the plugin to grep even when the index is present — verified by the consumer plan's dedicated bats test.
10. CLI `CHANGELOG.md` and `docs/sync.md` updated. When the plugin consumer ships, it cites this plan as its upstream dependency.

## Dependencies

- **Plugin repo (future release)** — the eventual consumer reads the index. CLI release precedes or ships in parallel with the consumer release; upgrade order does not matter for correctness (fallback path absorbs any ordering).
- **`jq` presence** — the plugin-side fast path assumes `jq` is available on the developer machine. If not, the plugin falls back to grep. Consider adding `jq` to the plugin's bundled runtime in a separate plan if fast-path adoption matters; out of scope here.
- **`.archcore/settings.json` schema** — `codeAlignment.sourceRoots` already exists for the plugin; no schema change needed. The CLI reads the same key when present.

## Risks

- **Index staleness between sync runs.** If a user edits a `.archcore/` document and does not run `archcore sync` before the next source-file edit, the index is stale relative to the document. Mitigation: the hook falls back to grep when the index does not contain a token it needs — no silent wrong answers. Longer-term: have the plugin's PostToolUse `validate-archcore` trigger an index refresh as part of its normal run, making the index eventually-consistent with zero user action.
- **Index growth at 10× repo scale.** At 400–500 documents the index may cross 100 KB, which is still fine for a git-tracked file. At 4000+ documents it is worth revisiting the single-file approach (possibly splitting out `path_index` into a sibling artifact). Out of scope for v1; revisit when a real repo approaches the limit.
- **Root-list drift between CLI and plugin.** If a user sets `codeAlignment.sourceRoots` in `.archcore/settings.json` but runs an old CLI that ignores the setting, the index would be built against the wrong root list. Mitigation: capture `roots` in the index header (`path_index.roots`) and have the plugin warn (in debug mode) when its active root list disagrees with the index's.
- **Codefence false positives.** Scanning document bodies for path tokens picks up path-like strings inside code examples that don't actually reference real source paths. Acceptable: the index overestimates candidates, but ranking (specificity + type priority) compensates. If noise becomes user-visible, add codefence exclusion to `pathindex.Extract` in a v2 schema bump.
- **Shipping without a consumer.** Shipping the CLI side before a plugin consumer means the index field sits unused. Risk: schema decisions ossify before a real consumer exercises them. Mitigation: a minimal end-to-end smoke on a fixture repo where the schema is read back with `jq` and structural expectations are asserted, independent of the plugin hook.

## Relations

- `implements` → `jtbd-alignment-analysis.idea` (continues Path B in the CLI layer)
- `implements` → `pre-code-context-injection.idea` (Phase 2 referenced in that idea)
- `related` → `pre-code-hook-implementation.plan` (Phase 1 predecessor whose output this plan optimises)
- `related` → `jtbd1-phase2-hardening-delegated.plan` (plugin-side plan; consumer deferred, not v0.4.0)
- `related` → `bundled-cli-launcher.adr` (CLI release coordination context)
