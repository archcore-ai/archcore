---
title: "search_documents MCP Tool Contract"
status: accepted
tags:
  - "component:cli"
  - "mcp"
---

## Purpose

This specification defines the canonical contract for the `search_documents` MCP tool exposed by the archcore CLI server.

It is normative for:

- The tool implementation in @cli/internal/mcp/tools/search_documents.go.
- Any MCP client calling the tool (LLM agents, future hooks, CI scripts).
- Any downstream consumer that depends on the tool's response shape.

## Scope

### Covers

- Tool signature, parameter validation, enum constraints.
- Path-reference extraction algorithm (explicit `@path` + qualified bare mentions).
- Content matching algorithm and match modes (`all`, `any`, `exact`).
- Specificity and score computation.
- Source scoping (`source`) and the coverage envelope.
- Ranking modes (`relevance`, `mtime`).
- Per-source representation on the truncated page.
- Evidence caps, the response byte budget, and the summary-first envelope.
- Output modes (`snippets` excerpt windows vs. `full` inline document body).
- Excerpt construction (including UTF-8 safety).
- Relations enrichment from the manifest.
- Lazy body loading behavior.
- Response JSON shape.
- Error conditions and messages.

### Does Not Cover

- Persistent path index in `.sync-state.json` — deferred (out of scope for the current contract).
- Hook-time invocation semantics — deferred to a future hook spec.
- Semantic / embedding search — out of scope.

## Authority

This document is the normative specification for the behavior of `search_documents`.

If the implementation, tests, or downstream consumers diverge from this specification, this specification takes precedence until it is amended. The recall design (tokenized matching, coverage, representation) is decided in @.archcore/mcp/global-recall-guarantees.rfc.md. The byte budget, the evidence caps, and the envelope order are decided in @.archcore/mcp/read-tool-responses-survive-host-truncation.adr.md. The slug field and separator folding are decided in @.archcore/mcp/search-matches-the-slug-and-folds-separators.adr.md.

### Related Artifacts

- Implementation: @cli/internal/mcp/tools/search_documents.go, @cli/internal/mcp/tools/response_budget.go
- Tool registration: @cli/internal/mcp/server.go
- Tests: @cli/internal/mcp/tools/search_documents_test.go, @cli/internal/mcp/tools/search_recall_test.go, @cli/internal/mcp/tools/response_budget_test.go
- Shared scan helpers: @cli/internal/mcp/tools/docs_bridge.go (`scanDocuments`, `scanDocumentsFull`), which delegate to @cli/internal/docs/scan.go (`Scan`, `ScanFull`). Both are package-private to `internal/mcp/tools`; see @.archcore/cli/docs-package-owns-the-document-model.adr.md.
- Manifest loading: @cli/internal/mcp/tools/manifest_store.go (shared cached store, `sharedManifestStore.load`) over @cli/internal/sync/manifest.go
- Source-extension list: @cli/templates/source_extensions.go

## Subject

- **Name:** `search_documents`
- **Kind:** MCP tool (JSON-RPC over stdio transport).
- **Primary responsibility:** Return `.archcore/` documents matching the caller's filters, with deterministic matching, ranking, and per-match evidence.
- **Consumers / dependents:** LLM agents invoking MCP tools directly, future PreToolUse hooks, any third-party MCP client.

## Definitions

| Term               | Definition                                                                                                                                                      |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Document body      | Markdown content following the YAML frontmatter block of an `.archcore/*.md` file.                                                                              |
| Frontmatter        | YAML block bounded by `---` at the top of an `.archcore/*.md` file, carrying `title`, `status`, `tags`.                                                         |
| Explicit reference | A path reference in a document body matching the regex `@[\w./\-_]+`.                                                                                           |
| Bare mention       | A path-like token in a document body matching `[\w\-_]+/[\w\-_./]+`, accepted only when one of the heuristic conditions defined in Normative Behavior §5 holds. |
| Specificity        | A non-negative integer measure of how precisely a candidate match relates to the filter.                                                                        |
| Match token        | One unit the content query matches by: a whitespace-separated word with its separators folded (§6.1) under `all`/`any`, or the whole query under `exact`. |
| Separator          | One of the six characters `-`, `_`, `/`, `.`, `@`, `:`. |
| Admitted row       | A row that survives the `limit` cut of §8.1–§8.4. |
| Byte budget        | The ceiling on the serialized response: 40,000 bytes (`searchResponseByteBudget`). |
| Score              | The internal ranking key: `100 × (path-ref maximum specificity + Σ content-token specificities) + capped occurrence count`. Not serialized.                      |
| Manifest           | The JSON file at `.archcore/.sync-state.json` that stores document relations, loaded via the shared manifest store.                                             |
| Source extension   | An extension listed in `templates.IsSourceExtension` (e.g., `.go`, `.ts`, `.py`, `.md`).                                                                        |
| Output mode        | The `mode` parameter selecting payload detail: `snippets` (excerpt windows only) or `full` (also inline the matched document's body).                            |

## Contract Surface

### Interface

Exposed over MCP using `github.com/mark3labs/mcp-go`. Registered from `NewServer(baseDir)` in @cli/internal/mcp/server.go. Annotated `ReadOnlyHint: true`.

### Inputs

| Name          | Type     | Required    | Description                                                                                                                                   |
| ------------- | -------- | ----------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `path_ref`    | string   | conditional | Path reference to match in document bodies. Leading `@` is stripped during comparison.                                                        |
| `content`     | string   | conditional | Case-insensitive word search against `title + slug + body`. Split on whitespace; the `match` parameter decides how many words must occur. Separators fold under `all` and `any` (§6.1). No stemming, no fuzzy matching. |
| `match`       | string   | no          | How content words must match. Enum: `all` (default; every word occurs somewhere in the document, any order, any distance), `any` (at least one word occurs), `exact` (the whole content string as one literal substring — the pre-2026-08 behavior). Any other value maps to `all`. |
| `source`      | string   | no          | Scope: `local` (the primary's own documents), `global` (every mounted global source), `__global__` (undeclared reserved-tree content), or a declared global source id. Empty admits every source. Any other value is rejected (§1.9). |
| `types`       | string[] | conditional | Filter by document type. OR semantics across the list.                                                                                        |
| `status`      | string   | conditional | Filter by frontmatter status. Enum: `draft`, `accepted`, `rejected`.                                                                          |
| `mtime_after` | string   | no          | Inclusive lower bound on document mtime. Accepts RFC3339 (ISO-8601) or a positive relative duration: `<N>h`, `<N>d`, `<N>w`, `<N>mo`, `<N>y`. |
| `sort`        | string   | no          | Ordering mode. Enum: `relevance` (default), `mtime`.                                                                                          |
| `mode`        | string   | no          | Output detail. Enum: `snippets` (default), `full`. `snippets` returns only excerpt windows around matches. `full` additionally returns each matched document's body inline (frontmatter stripped); a body the byte budget cannot hold arrives shortened (§12.4). Any value other than `full` maps to `snippets`. |
| `limit`       | number   | no          | Maximum number of results. Mode-dependent: `snippets` = default 50 / max 200; `full` = default 3 / max 20. Values above the cap are clamped; `0` or omitted maps to the mode default. The byte budget can return fewer rows than `limit` (§8.6). |

At least one of `path_ref`, `content`, `types`, or `status` MUST be provided.

### Outputs

A JSON object whose keys serialize in this order: `{"coverage": {...}, "hits": {...}, "truncated": false, "index": [...], "results": [...]}`. The summary comes first because a host that stores an oversized result shows the caller only its first bytes.

- `coverage` maps each searched source id to its scanned document count (e.g. `{"local": 102, "org": 42}`), computed after the `source` scope and before the query filters. An empty `results` next to a populated `coverage` is a verified absence — the corpus was searched and holds no match.
- `hits` maps each searched source id to its matched document count, before the `limit` cut. A searched source without a match carries `0`.
- `truncated` is `true` when the byte budget returned fewer rows, or fewer `index` entries, than `limit` admitted (§8.6, §8.10).
- `index` lists `path`, `title`, and `source_id` of the admitted rows, in §7 order, inside its own byte ceiling (§8.10).
- `results` is the array of `searchResult` objects (fields below).

(Behavior change 2026-08: the response was previously a bare array. Behavior change 2026-09: `hits`, `truncated`, and `index` were added and the key order was fixed; see Compatibility.)

Each `searchResult` object has the following fields:

| Field                | Type             | Nullability             | Description                                                      |
| -------------------- | ---------------- | ----------------------- | ---------------------------------------------------------------- |
| `path`               | string           | always                  | Relative path beginning with `.archcore/`.                       |
| `title`              | string           | always                  | Frontmatter title (may be empty for malformed docs).             |
| `type`               | string           | always                  | Document type (e.g., `rule`, `adr`, `spec`).                     |
| `status`             | string           | omit if empty           | Frontmatter status.                                              |
| `mtime`              | string (RFC3339) | always                  | File modification time.                                          |
| `tags`               | string[]         | omit if empty           | Frontmatter tags.                                                |
| `source_id`          | string           | always                  | `local` for primary docs, the source id for a declared global, or `__global__` for undeclared reserved-tree content. |
| `source_kind`        | string           | always                  | `local` or `global`.                                            |
| `global`             | boolean          | omit if false           | `true` for mounted global sources.                              |
| `read_only`          | boolean          | omit if false           | `true` for mounted global sources.                              |
| `matches`            | Match[]          | always                  | Evidence array. Always present; empty for pure metadata queries. |
| `matches_total`      | integer          | omit unless capped      | Count of all match hits of the row, present only when §5.7 or §6.6 cut the `matches` array. |
| `body`               | string           | omit unless `mode=full` | Document body with frontmatter stripped. Populated only in `full` mode; omitted (never empty string) in `snippets` mode. The byte budget can shorten it (§12.4). |
| `body_truncated`     | boolean          | omit if false           | `true` when `body` is a shortened prefix of the document body. |
| `body_bytes`         | integer          | omit unless shortened   | Byte length of the whole body, present only beside `body_truncated`. |
| `incoming_relations` | Relation[]       | always (possibly empty) | Manifest edges where this doc is the target; at most 5 (§10).    |
| `incoming_relations_total` | integer    | omit unless capped      | Count of all incoming edges, present only when §10 cut the array. |
| `outgoing_relations` | Relation[]       | always (possibly empty) | Manifest edges where this doc is the source; at most 5 (§10).    |
| `outgoing_relations_total` | integer    | omit unless capped      | Count of all outgoing edges, present only when §10 cut the array. |

`Match` fields:

| Field         | Type    | Description                                                                                   |
| ------------- | ------- | --------------------------------------------------------------------------------------------- |
| `kind`        | string  | Enum: `path_ref_explicit`, `path_ref_mention`, `content`.                                     |
| `ref`         | string  | The raw matched token (e.g., `@src/payments/`), or one content word as matched — with its separators folded under `all`/`any`. |
| `specificity` | integer | Per-match specificity value (see Normative Behavior §6).                                      |
| `excerpt`     | string  | ≤ ~120-character window around the match, padded with `...` if truncated. Always valid UTF-8. |

`Relation` fields (reuses `DocumentRelation` from @cli/internal/mcp/tools/docs_bridge.go, aliased to `docs.DocumentRelation`): `path`, `type` (one of `related`, `implements`, `extends`, `depends_on`, `supports`, `contradicts`, `supersedes`).

## Normative Behavior

### §1 Filter and parameter validation

1. The handler MUST reject calls where all of `path_ref`, `content`, `types`, and `status` are empty or absent, returning the MCP error `specify at least one filter (path_ref, content, types, or status)`.
2. The handler MUST reject `limit < 0` with the error `limit must be non-negative`.
3. The handler MUST treat `limit == 0` as equivalent to omitted — both produce the mode default (snippets: 50, full: 3).
4. The handler MUST clamp `limit` above the mode cap (snippets: 200, full: 20) to that cap without emitting an error.
5. `mtime_after` MUST accept both RFC3339 timestamps and positive relative durations of the form `<N>h`, `<N>d`, `<N>w`, `<N>mo`, `<N>y`. Invalid input MUST return the error `invalid mtime_after: <reason>`.
6. The handler MUST normalize `mode` defensively: any value other than `full` (including omitted) maps to `snippets`. The framework enforces the enum, but the handler MUST NOT rely on that alone.
7. The handler MUST normalize `match` defensively: any value other than `exact` or `any` (including omitted) maps to `all`.
8. The handler MUST resolve `source` as: empty admits every document; `local` admits `source_kind == "local"`; `global` admits `source_kind == "global"`; `__global__` admits reserved-tree content; a declared global source id admits its documents.
9. WHEN `source` carries any other value, the handler MUST reject the call with `invalid source "<value>" (valid: "local", "global", or a declared global source id)`. The declared-id check reads `config.ReadGlobals` — an advisory, fail-open read on a read path.
10. WHEN `match` is `all` or `any` and the `content` filter tokenizes to zero words, the handler MUST reject the call with `content must contain at least one word`. `exact` keeps substring semantics for any non-empty `content`.

### §2 Document loading

1. When `path_ref` and `content` are both empty AND `mode` is `snippets`, the handler MUST call `scanDocuments(baseDir)`, which does not retain document bodies in memory beyond the frontmatter parse.
2. When either `path_ref` or `content` is non-empty, OR `mode` is `full`, the handler MUST call `scanDocumentsFull(baseDir)`, which retains bodies on each `LocalDocument`. (Full mode needs bodies to populate the `body` field even for pure-metadata filters.)
3. Scanner I/O cost MUST be identical in both cases; only heap retention differs.

### §3 Manifest loading

1. The handler MUST obtain relation data from the shared manifest store (`sharedManifestStore.load` in @cli/internal/mcp/tools/manifest_store.go), which caches the parsed manifest keyed on the file's (mtime, size) and delegates to `sync.LoadManifest`.
2. A **missing** `.sync-state.json` is NOT an error: it yields an empty manifest, and all results carry empty `incoming_relations` / `outgoing_relations` arrays.
3. A **present-but-invalid** manifest MUST fail the call with the MCP error `loading manifest: <reason>` — consistent with `get_document` and `list_relations`. Silently degrading to empty relations hides real graph state (the silent-incomplete-context failure class). The reason preserves validation detail (built from relative manifest keys); OS-level failures map to a fixed I/O class via `sanitizeError`, so the message never embeds an absolute filesystem path.

### §4 Source scope and metadata filters (AND semantics)

Filters are evaluated in the following order; a document MUST be excluded if any active filter fails:

1. `source`: per §1.8. A document admitted here MUST be counted in `coverage` under its `source_id` regardless of the later filters.
2. `types`: if non-empty, doc's `type` MUST be present in the list.
3. `status`: if non-empty, doc's `status` MUST equal the filter exactly.
4. `mtime_after`: if non-zero, doc's mtime MUST be strictly after the threshold.

### §5 Path-reference extraction

For each document's body (populated only when `path_ref` is set):

1. Apply regex `@[\w./\-_]+` to extract explicit references. Each match yields a candidate with `Kind: "explicit"`.
2. Apply regex `[\w\-_]+/[\w\-_./]+` to extract bare mention candidates. Each match yields a candidate with `Kind: "mention_candidate"`.
3. Remove overlap: if a bare candidate's byte range is contained within an explicit candidate's range, the bare candidate MUST be dropped. This prevents double-counting when an `@`-reference syntactically contains a bare path.
4. Filter bare candidates via `filterBareMentions`: a candidate is retained only if at least one of the following holds:
   - The candidate ends with `/`.
   - The candidate contains ≥ 2 `/` separators.
   - The final path segment's extension passes `templates.IsSourceExtension`.
5. For each surviving candidate, compute `computeSpecificity(candidate, pathRefFilter)` as the number of left-aligned `/`-segments shared between the normalized candidate (leading `@` and trailing `/` stripped) and the filter (same normalization). Specificity of `0` MUST NOT produce a match.
6. Each surviving candidate with specificity ≥ 1 is a path-reference hit. The handler MUST order the hits by specificity DESC, then explicit before mention, then byte offset ASC.
7. The handler MUST emit at most 5 hits (`searchMatchCap`) as `Match` objects, with `kind` = `path_ref_explicit` or `path_ref_mention` by origin regex.
8. The ranking score takes the **maximum** specificity across all of a document's path-ref hits, not their sum: repeating a path is not a relevance signal. The cap of §5.7 MUST NOT change the score.
9. WHEN §5.7 or §6.6 removes a hit from `matches`, the handler MUST set `matches_total` to the count of all hits of the row; otherwise the field MUST be omitted.

### §6 Content matching

When `content` is non-empty, for each document:

1. Lowercase the query once. Under `match=all` and `match=any`, split it on whitespace, replace each separator in each word with one space, trim the word, and drop a word left empty (§1.10 guarantees at least one remains); under `match=exact`, the whole query is one token and nothing folds.
2. Under `all` and `any`, the handler MUST compare each token against the title, the slug, and the body with their separators folded the same way. Folding replaces one byte with one byte, so an offset into the folded text is an offset into the source.
3. For each token, resolve its tier: a title hit is specificity 3; a slug hit is 3; a markdown heading-line hit (a body line starting with `#`) is 2; any other body hit is 1; absent is 0. The slug is a match field in every match mode.
4. Under `all` and `exact`, a token at tier 0 MUST exclude the document. Under `any`, a tier-0 token is skipped, and the document matches when at least one token hit.
5. Each hitting token MUST emit a `Match` with `kind: "content"`, `ref` = the token, `specificity` = its tier, and an `excerpt` built around the token's first occurrence (from the title for a title hit, the whole slug for a slug hit, from the body otherwise).
6. WHEN a document has more than 5 content matches (`searchMatchCap`), the handler MUST keep the 5 of highest specificity, in query-word order among equals; at or under the cap, `matches` keeps query-word order.
7. The handler MUST count all token occurrences across the lowercased title and body, capped at 20 (`contentFreqCap`), as the low-order component of the score.

### §7 Ranking (`sort` parameter)

1. `sort="relevance"` (default) MUST order results by four keys, in order:
   - Score DESC: `100 × (path-ref maximum specificity + Σ content-token specificities) + capped occurrence count` (documents with no matches score 0). Path refs contribute their best hit (§5.8); content tokens sum, with repetition bounded by `contentFreqCap` (§6.7).
   - Type priority ASC, per the map: `rule=1, adr=2, spec=3, cpat=4, guide=5, plan=6, idea=7, other=100`.
   - Effective mtime DESC, where a **global** document's effective mtime is the zero time: a vendored global's mtime is its clone date, not a relevance signal, and on a full tie a local document therefore ranks before a global (see @.archcore/globals/local-overrides-global.rule.md).
   - `path` ASC, as the final total-order key.
2. `sort="mtime"` MUST order purely by `mtime` DESC, ignoring score and type.
3. The sort MUST be stable: ties preserve input order (filesystem walk order), yielding deterministic output across runs.

### §8 Truncation and per-source representation

1. After sorting, the handler MUST cut the result slice to at most `limit` entries (the mode-resolved limit from §1.3–§1.4). The rows that remain are the admitted rows.
2. WHEN the cut removes every row of a source that has at least one match, the handler MUST swap that source's top row in over the lowest-ranked page row whose source keeps more than one row.
3. Sources claim their swap in rank order of their own top row; WHEN every page row is its source's last, the handler MUST stop swapping.
4. The handler MUST re-sort the page with the §7 keys after the swaps.
5. The handler MUST keep the serialized response at or under the byte budget.
6. WHEN the admitted rows do not fit the byte budget with each body at its floor (§12.5), the handler MUST remove rows from the tail and set `truncated` to `true`.
7. After each removal of §8.6, the handler MUST re-apply §8.2–§8.4 to the shorter page.
8. The handler MUST keep at least one row. WHEN one row remains and its `body` floor does not fit, the handler MUST shorten that `body` below the floor. A single row whose fields other than `body` exceed the byte budget is the only response that may exceed it.
9. `index` MUST list the admitted rows in §7 order, including a row that §8.6 removed from `results`.
10. WHEN the index entries exceed 20,000 bytes (`searchIndexByteBudget`), the handler MUST drop entries from the tail, keep at least one, and set `truncated` to `true`.

### §9 Excerpt construction

1. For path-ref matches, `pos` is the byte offset of the matched token in the source body.
2. For content matches, `pos` is the byte offset of the token's first occurrence in the source (title or body).
3. The window width is 120 characters (`excerptWindow`), split evenly around `pos`.
4. Byte offsets MUST be snapped to rune boundaries before slicing: `start` moves backward until `utf8.RuneStart(source[start])` is true; `end` moves forward until `utf8.RuneStart(source[end])` is true or EOF is reached. Resulting excerpts MUST be valid UTF-8.
5. Internal whitespace runs in the excerpt MUST be collapsed to a single space.
6. `...` MUST be prepended when the excerpt's start is > 0 and appended when the excerpt's end is < `len(source)`.

### §10 Relations enrichment

1. For every emitted result, the handler MUST populate `incoming_relations` and `outgoing_relations` from the manifest. Both arrays MUST be serialized as `[]` when empty, never `null`.
2. The handler MUST order each array by `path` ASC, then `type` ASC.
3. The handler MUST emit at most 5 relations per array (`searchRelationCap`).
4. WHEN §10.3 removes a relation, the handler MUST set that array's `_total` field to the count of all its edges; otherwise the field MUST be omitted. `get_document` returns the full graph of one document.

### §11 Response shape invariants

1. The response is a JSON object whose keys MUST serialize in the order `coverage`, `hits`, `truncated`, `index`, `results`. `results` and `index` MUST be empty arrays (not `null`) when no documents match; `coverage` MUST still carry the scanned counts.
2. `coverage` MUST hold one entry per source admitted by the `source` scope, keyed by `source_id`. `hits` MUST hold the same keys.
3. `matches` MUST be an empty array, never `null`, when a document passed metadata filters but has no per-match evidence (pure metadata query).
4. `mtime` MUST always be present in RFC3339 format. The `omitzero` JSON tag applies at the `LocalDocument` layer but the result-level struct serializes `mtime` unconditionally.
5. Source annotation MUST always be present: `source_id` and `source_kind` on every result; `global` and `read_only` omitted when false.

### §12 Full-mode body

1. When `mode="full"`, each emitted result MUST carry a `body` field containing the matched document's body with the YAML frontmatter block stripped (`stripFrontmatter(doc.Content)`), whole or shortened per §12.4.
2. When `mode="snippets"` (default), the `body` field MUST be omitted from the JSON entirely (`omitempty`), never emitted as an empty string.
3. Full mode does NOT change matching, ranking, excerpt, or representation behavior; `matches` excerpts are still emitted alongside the body. It only adds the inline body and applies the smaller mode-specific limit bounds (§1.3–§1.4). The §8 representation guarantee applies to the full-mode page as well, so one of the three default slots carries the top row of an otherwise-evicted matching source.
4. WHEN the bodies of the page do not fit the byte budget, the handler MUST shorten bodies by max-min fair share: a body smaller than an equal share arrives whole, and the unused bytes go to the larger bodies.
5. The handler MUST NOT remove a row to make room for a body while every body of the page can keep a share of 1,024 encoded bytes (`searchBodyFloorBytes`). The floor bounds the share; the line-break rule of §12.6 can deliver less than the share.
6. A shortened `body` MUST be a prefix of the document body, MUST end on a rune boundary, and SHOULD end at the last line break in the second half of its share.
7. A row with a shortened `body` MUST carry `body_truncated: true` and `body_bytes`. A caller MUST read the whole document through `get_document` before it writes through `update_document`.

## Constraints

| Constraint                          | Value                     | Rationale                                                                                |
| ----------------------------------- | ------------------------- | ---------------------------------------------------------------------------------------- |
| Default limit (snippets)            | 50                        | Balances coverage and payload size for typical LLM invocations.                          |
| Maximum limit (snippets)            | 200                       | Caps payload to bounded size even when caller requests more.                             |
| Default limit (full)                | 3                         | Each result carries a full body; a small default keeps a single response token-bounded.  |
| Maximum limit (full)                | 20                        | Hard cap on how many full bodies one response may inline.                                |
| Occurrence-count cap (`contentFreqCap`) | 20                    | Bounds the low-order score component so a term-stuffed body cannot outrank a structural hit. |
| Path-ref score contribution         | maximum specificity       | Path repetition is not a relevance signal; the same stuffing guard as `contentFreqCap` (§5.8). |
| Representation swap                 | one row per source        | The guarantee costs at most the page's tail; it never rewrites the top of the page.      |
| Response byte budget (`searchResponseByteBudget`) | 40,000 bytes | One host stored results from about 50,000 characters upward and refused larger ones by token count; bytes bound characters from above. |
| Index ceiling (`searchIndexByteBudget`) | 20,000 bytes            | 200 admitted rows with long titles otherwise fill the response by themselves (44,198 bytes measured). |
| Body floor (`searchBodyFloorBytes`) | 1,024 encoded bytes       | A row leaves the page before its body shrinks below a readable opening.                  |
| Match cap (`searchMatchCap`)        | 5 per filter              | One document cited one path 39 times; evidence was 60–70% of an overflowed response.     |
| Relation cap (`searchRelationCap`)  | 5 per direction           | A hub document's row otherwise grows with its degree.                                    |
| Excerpt window                      | 120 chars                 | Keeps per-match payload small while carrying enough context for user/LLM disambiguation. |
| Cold-scan target                    | ≤ 500 ms P95 for 200 docs | Informal target; not enforced by test harness yet.                                       |
| Heap growth for pure-metadata query | O(N × frontmatter_size)   | Bodies are not retained when `path_ref` and `content` are both empty AND `mode=snippets`. |
| Heap growth for content / full query | O(N × body_size)         | Bodies are retained; bounded by the `.archcore/` directory size.                         |

## Invariants

- The handler is read-only: no filesystem writes, no manifest mutations.
- Two identical calls against an unchanged `.archcore/` tree MUST produce byte-identical JSON output (excluding `mtime` serialization if clock-driven sources intrude — they do not in current implementation).
- All emitted `excerpt` strings MUST satisfy `utf8.ValidString(excerpt)`.
- The serialized response MUST NOT exceed the byte budget, except the single-row case of §8.8.
- A shortened `body` MUST be a valid-UTF-8 prefix of the document body.
- The first 2,048 bytes of a response with at most 6 admitted rows carry `coverage`, `hits`, and the whole `index`. [assumption: titles of ordinary length]
- Every returned result's `path` MUST begin with `.archcore/` for a primary document; a mounted external global renders with its declared relative prefix.
- Results MUST NOT contain duplicate documents. One document yields at most one result row; multiple match candidates for the same doc collapse into the `matches` array.
- WHEN a source has at least one matching document, the truncated page MUST carry at least that source's top match, bounded by the page size and the §8.3 slot rule.
- Each result MUST carry source annotation (`source_id`, `source_kind`, `global`, `read_only`) identical to `list_documents` / `get_document`; consumers MUST use these fields to tell local from global rather than inferring authority from the path (see @.archcore/globals/local-overrides-global.rule.md).

## Error Handling

| Condition             | Response                                                                   | Recovery                                                                                            |
| --------------------- | -------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| All filters empty     | MCP error: `specify at least one filter (...)`                             | Caller adds at least one filter.                                                                    |
| `limit < 0`           | MCP error: `limit must be non-negative`                                    | Caller supplies non-negative limit.                                                                 |
| Invalid `mtime_after` | MCP error: `invalid mtime_after: <reason>`                                 | Caller supplies valid RFC3339 or relative duration.                                                 |
| Unknown `source` value | MCP error: `invalid source "<value>" (valid: "local", "global", or a declared global source id)` | Caller corrects the source scope.                                              |
| `content` with zero words under `all`/`any` | MCP error: `content must contain at least one word`          | Caller supplies at least one word, or uses `match="exact"`.                                         |
| Manifest missing      | Empty relations arrays on every result                                     | Caller treats empty relations as absence of data.                                                   |
| Manifest present but invalid | MCP error: `loading manifest: <reason>` (validation detail preserved; never an absolute path) | Caller (or user) repairs `.sync-state.json`; relation data is authoritative, not silently droppable. |
| Malformed frontmatter | Silently skip frontmatter parse; doc still indexed with empty title/status | Caller cannot distinguish malformed frontmatter from absent frontmatter — acceptable for this spec. |
| I/O error during scan | MCP error: `scanning documents: <I/O class>` (sanitized, no absolute path) | Caller retries or falls back.                                                                       |

### Failure semantics

- Non-retriable: filter validation errors, `limit` validation, present-but-invalid manifest (until the file is repaired).
- Retriable: transient I/O errors during scan (sanitized and surfaced).
- The handler is idempotent: no observable state changes between calls.

## Conformance

An implementation conforms to this specification if it satisfies:

- All MUST and MUST NOT statements in §§1–12.
- All stated invariants.
- All applicable error-handling rows.
- The recorded unit tests in @cli/internal/mcp/tools/search_documents_test.go, @cli/internal/mcp/tools/search_recall_test.go, and @cli/internal/mcp/tools/response_budget_test.go (these tests are the executable acceptance harness for this spec).

## Examples

### Multi-word content query across sources

```txt
// Input
search_documents({
  content: "plugin compatibility"
})

// Output (abbreviated)
{
  "coverage": { "local": 102, "archcore": 42 },
  "hits": { "local": 1, "archcore": 1 },
  "truncated": false,
  "index": [
    { "path": ".archcore/integrations/plugin-cli-compatibility.rule.md", "title": "Compatibility Contract Between...", "source_id": "local" },
    { "path": "../global/.archcore/architecture/plugin-cli-compatibility.rule.md", "title": "Plugin / CLI Compatibility...", "source_id": "archcore" }
  ],
  "results": [
    {
      "path": ".archcore/integrations/plugin-cli-compatibility.rule.md",
      "type": "rule",
      "source_id": "local",
      "source_kind": "local",
      "matches": [
        { "kind": "content", "ref": "plugin", "specificity": 3, "excerpt": "...the Archcore Plugin..." },
        { "kind": "content", "ref": "compatibility", "specificity": 3, "excerpt": "Compatibility Contract Between..." }
      ]
    },
    {
      "path": "../global/.archcore/architecture/plugin-cli-compatibility.rule.md",
      "type": "rule",
      "source_id": "archcore",
      "source_kind": "global",
      "global": true,
      "read_only": true,
      "matches": [
        { "kind": "content", "ref": "plugin", "specificity": 3, "excerpt": "Plugin / CLI Compatibility..." },
        { "kind": "content", "ref": "compatibility", "specificity": 3, "excerpt": "Plugin / CLI Compatibility..." }
      ]
    }
  ]
}

// Notes
// Every word must occur, in any order and at any distance — the exact phrase
// "plugin compatibility" appears in neither title. match="exact" would return
// zero rows here.
```

### Pure metadata query (no body scan)

```txt
// Input
search_documents({
  types: ["plan", "idea"],
  status: "draft",
  limit: 10
})

// Output (per-doc matches array is empty)
{
  "coverage": { "local": 102 },
  "hits": { "local": 1 },
  "truncated": false,
  "index": [
    { "path": ".archcore/payments/stripe-3ds-rollout.plan.md", "title": "Stripe 3DS Rollout", "source_id": "local" }
  ],
  "results": [
    {
      "path": ".archcore/payments/stripe-3ds-rollout.plan.md",
      "type": "plan",
      "status": "draft",
      "mtime": "2026-04-20T16:00:00Z",
      "source_id": "local",
      "source_kind": "local",
      "matches": [],
      "incoming_relations": [],
      "outgoing_relations": []
    }
  ]
}

// Notes
// scanDocuments is used (not scanDocumentsFull) — bodies are not loaded.
```

### Verified absence

```txt
// Input
search_documents({ content: "blue-green deploys" })

// Output
{ "coverage": { "local": 102, "archcore": 42 }, "hits": { "local": 0, "archcore": 0 }, "truncated": false, "index": [], "results": [] }

// Notes
// The empty result names what was searched: 144 documents across both sources
// hold no document containing every query word. Broaden the words or try
// match="any" — do not conclude the corpus was skipped.
```

### Compound name and a shortened body

```txt
// Input
search_documents({ content: "acme-id-sdk", mode: "full", limit: 6 })

// Output (abbreviated)
{
  "coverage": { "local": 239, "org": 37 },
  "hits": { "local": 3, "org": 16 },
  "truncated": false,
  "index": [ /* six entries: path, title, source_id */ ],
  "results": [
    {
      "path": ".archcore/auth/acme-id-sdk.doc.md",
      "source_id": "local",
      "matches": [
        { "kind": "content", "ref": "acme id sdk", "specificity": 3, "excerpt": "acme-id-sdk" }
      ],
      "body": "## Overview\n...",
      "body_truncated": true,
      "body_bytes": 12403
    }
  ]
}

// Notes
// The query word folds to "acme id sdk", so it also matches a body that spells
// the name "@acme-id/sdk". The first row matched on its slug. Six bodies do not
// fit 40,000 bytes, so each long body is a prefix; get_document returns the rest.
```

### Invalid filter

```txt
// Input
search_documents({})

// Output
{ "error": "specify at least one filter (path_ref, content, types, or status)" }

// Input
search_documents({ content: "logging", source: "orgg" })

// Output
{ "error": "invalid source \"orgg\" (valid: \"local\", \"global\", or a declared global source id)" }
```

## Security Considerations

- The tool operates only under `baseDir/.archcore/` and declared global mounts, never elsewhere. No path traversal is possible via the filters (no filter accepts a filesystem path as input — `path_ref` is matched against document body content, not dereferenced).
- The tool is read-only; no content can be written, modified, or deleted through it.
- Every error message is sanitized: OS-level failures map to a fixed I/O class, so no absolute filesystem path ever reaches the MCP client (see @.archcore/mcp/no-absolute-paths-in-mcp-errors.rule.md). Nothing is written to stdout besides protocol frames.

## Compatibility

- **Behavior change (2026-08), breaking:** the response became the `{"results", "coverage"}` envelope (previously a bare array); content matching became tokenized all-words by default (`match="exact"` preserves the old substring semantics); ranking replaced max-specificity with the summed score, added the heading tier and occurrence count, stopped using a global document's mtime, and gained the `path` tiebreak; a truncated page now keeps every matching source represented; an unknown `source` and a zero-word `content` became validation errors. Decided in @.archcore/mcp/global-recall-guarantees.rfc.md; shipped together as one wire-shape major change.
- **Behavior change (2026-09):** the envelope gained `hits`, `truncated`, and `index`, and its key order became normative; `matches` and each relation array carry at most 5 entries, with a total beside a cut array; the response stays under a 40,000-byte budget, so `results` can hold fewer rows than `limit` and a full-mode `body` can be a shortened prefix; the slug became a match field and separators fold under `all`/`any`, so `ref` can differ from the typed word. Every added field is additive; the narrowed fields are `body`, `matches`, and the relation arrays. Decided in @.archcore/mcp/read-tool-responses-survive-host-truncation.adr.md and @.archcore/mcp/search-matches-the-slug-and-folds-separators.adr.md.
- The envelope is additive from here on: new fields MAY be added to the envelope, `searchResult`, or `Match` without breaking consumers that ignore unknown fields.
- Existing field names, types, and enum values MUST NOT be removed or repurposed in a minor CLI release. A breaking change requires a new major version.
- The `sort`, `mode`, and `match` enums MAY grow additional values in a minor release; existing `relevance`, `mtime`, `snippets`, `full`, `all`, `any`, and `exact` semantics MUST remain stable.
- Behavior change (2026-07): a present-but-invalid manifest was previously degraded to empty relations with a stderr log; it is now a tool error (§3.3), unified with `get_document` / `list_relations`.