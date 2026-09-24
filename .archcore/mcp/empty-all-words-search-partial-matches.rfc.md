---
title: "Near Misses When an All-Words Search Returns No Rows"
status: accepted
tags:
  - "component:cli"
  - "mcp"
---

## Summary

A `search_documents` call with `match: "all"`, the default, and two or more content words returns no rows when no document that passes the filters contains every word. The response calls this a verified absence. An agent that reads it as "no document on this topic" can stop searching and act without the governing decision.

This RFC proposes one bounded addition. When an all-words search with two or more distinct words returns no rows, the response adds `near_misses`: at most 3 documents that pass every filter and contain at least half of the words, each naming the words it lacks in `missing`. When no document qualifies, `near_misses` is `[]`. `results`, the default mode, and every response that returns rows stay unchanged. The absence claim narrows to "no document that passes the filters contains every word". The change amends @.archcore/mcp/search-documents.spec.md and adds an addendum to @.archcore/mcp/search-documents-matching-not-presentation.adr.md.

The maintainer settled five design choices on 2026-09-23, after an architecture review of the first draft: the field name `near_misses`, the half-of-the-words rule, the `[]` presence rule, the ADR addendum, and a trigger that starts at two words.

## Motivation

### Observed failure

On 2026-09-23, integration-bench experiment `20260923T153932Z-98ef5bbd20` ran the task `review-seam-buried` on Claude Code 2.1.280 with `claude-sonnet-5`, effort `medium`. The fixture holds an accepted ADR that requires `DeliveryPolicy`. The ADR carries no path reference and sits among 11 other accepted records.

In run 1 of the `ponytail-archcore-lean` recipe arm, the agent sent three searches: `content: "delivery policy price source"`, `path_ref: "shop/catalog.py"`, and `path_ref: "shop/delivery.py"`. All three returned no rows. The agent sent the content search in one tool batch with `git diff`, reported "Archcore records consulted: none matched", and recommended deleting the code the ADR requires.

Runs 2 and 3 of the same arm sent the same empty query, then searched `DeliveryPolicy`, found the ADR, and kept the code. The arm without a recipe made no `search_documents` call and deleted the code in 3 of 3 runs. This RFC does not address that failure.

A direct reproduction against the same fixture through `archcore mcp`, CLI v0.10.3:

| Query | Rows |
|---|---|
| `delivery policy price source`, default `all` | 0; with `any`, the ADR ranks first with 3 of 4 words |
| `delivery policy` | 1, the ADR |
| `DeliveryPolicy` | 1, the ADR |
| `delivery policy price source partner contract interface removal` | 0; with `any`, the ADR matches at least 5 of 8 words |
| `delivery policy доставка политика` | 0; with `any`, the ADR matches 2 of 4 words |
| `delivery policy` with `types: ["rule"]` | 0, although the ADR contains both words |

The ADR contains `delivery`, `policy`, and `price`, but not `source`. One word from a second topic removed the only relevant row. The last three rows set the requirements for longer queries, mixed-language queries, and filters.

### Frequency

- Bench, four experiments on 2026-09-23, 32 content queries: multi-word `all` queries returned no rows 3 of 4 times, and multi-word `any` queries 0 of 21 times. The 3 empty queries are one phrasing, once per repetition of one recipe arm, so the bench rate does not generalize.
- Real sessions, 649 Claude Code transcripts on one maintainer's machine, 2026-08-27 to 2026-09-23, 317 unique content queries including integration pilot sessions: `all` returned no rows for 1 of 69 one-word queries, 1 of 37 two-word queries, and 5 of 48 queries with three or more words (10%).
- After 4 of those 6 empty multi-word `all` queries, the agent searched again with other words and found rows. After the other 2, the agent did not search again. Whether those 2 were false absences is unknown, because the corpora changed since.

### Why the response invites the stop

1. The tool description says "Empty results next to a populated coverage is a verified absence — broaden the words" (@cli/internal/mcp/tools/search_documents.go). The claim holds for the word set under the active filters, not for the topic.
2. The spec Outputs section repeats "a verified absence — the corpus was searched and holds no match".
3. The all-words rule reaches every project through the tool description and the `content` and `match` parameter texts. Only the sentence that links an empty result to that rule sits in the GLOBAL SOURCES paragraph, which `buildInstructions` emits only when `len(globals) > 0` (@cli/internal/mcp/server.go).
4. The failing agent received "broaden the words" and did not search again. A wording change alone does not close the gap.
5. The empty response carries no data to act on. The next query depends on the agent guessing which word to drop.

## Detailed Design

### 1. The `near_misses` field

1. Trigger: `match` resolves to `all`, `content` holds two or more distinct folded words, and every `hits` value is 0.
2. Candidates: documents that pass every filter — `source`, `types`, `status`, `mtime_after`, and `path_ref` — and contain at least ⌈n/2⌉ of the n distinct words. Word matching reuses the spec's folding rule and its title, slug, and body fields.
3. Row fields: `path`, `title`, `source_id`, and `missing`. `missing` is an array of the absent folded words in query order; a compound word keeps its parts joined by one space.
4. Order: matched-word count, highest first, then the §7 ranking keys, whatever `sort` says.
5. Cap: 3 rows, independent of `limit`. The §8 per-source representation applies to the 3 rows.
6. Key order: `coverage`, `hits`, `truncated`, `index`, `near_misses`, `results`. This amends §11.1.
7. Presence: the key is present, possibly as `[]`, exactly when the trigger holds, and absent otherwise. A response that returns rows stays byte-identical to today's.
8. Scope: the rows do not count toward `hits`, `index`, or `truncated`, and they carry no body, excerpt, or relations in any `mode`.

For two and three words, the half rule selects the same documents as a rule of "every word except one". For longer queries and for mixed-language queries, it keeps documents that the one-missing-word rule drops, such as the ADR in the 8-word and the two-language rows above.

Response for the observed query under this design:

```json
{"coverage": {"local": 12}, "hits": {"local": 0}, "truncated": false, "index": [],
 "near_misses": [{"path": ".archcore/shop/delivery-policy-seam.adr.md", "title": "Keep Delivery Fees Behind the DeliveryPolicy Interface", "source_id": "local", "missing": ["source"]}],
 "results": []}
```

### 2. Wording

1. The tool description sentence becomes: "Empty results next to a populated coverage mean that no document passing the filters contains every word; near_misses lists documents that contain most of the words, and missing names the words each lacks." The sentence names both new wire fields, which `TestNewSearchDocumentsTool_DescriptionNamesEveryWireField` (@cli/internal/mcp/tools/search_envelope_spec_test.go) checks.
2. The spec Outputs definition of a verified absence narrows the same way.
3. The server-instructions sentence that links an empty result to the all-words rule moves out of the GLOBAL SOURCES paragraph into the base instructions.

### 3. Addendum to the matching-primitive ADR

The ADR limits the tool to "ranked matches with evidence and manifest relations" and counts empty-state branching as presentation. `near_misses` returns documents that fail the caller's all-words filter, and it appears only on an empty result. The change therefore needs an addendum, not only an argument.

Proposed addendum text: "`near_misses` is matching data. It applies one deterministic rule, at least half of the distinct query words, to the documents that pass the filters, and it reports the words each row lacks. It carries no prose, retry instruction, grouping, or excerpt, and it never enters `results`. The threshold and the cap are contract values in the spec, not presentation settings."

### 4. Cost

- Time: the handler records matched-word counts inside the existing matching loop, so no body is lowercased or folded twice. `BenchmarkReadToolsScaling` (@cli/internal/mcp/tools/scaling_bench_test.go) runs only `content: "lorem"` today. It gains an empty two-word case, `lorem zephyrite`, where every document is a near miss, and the result is recorded before merge. [assumption] The overhead stays below the cost of the scan.
- Bytes: a row costs about 160 bytes for an ASCII path and title, and about 340 bytes for a long global path with a Cyrillic title. Three rows stay near 1 KB, inside the 40,000-byte budget, and `results` is empty whenever the field appears.

### 5. Version skew

A current CLI always emits the key on a triggering response, so the key's absence there identifies an older CLI. A plugin skill that teaches the field reads it when present, as @.archcore/mcp/read-tool-responses-survive-host-truncation.adr.md arranges for additive fields. No `cli-gte` gate is needed.

## Drawbacks

- The response grows by one key on empty all-words results. A client that validates responses strictly sees the new key.
- For two words, the half rule equals "either word". On the fixture, `a zephyrite` lists 3 arbitrary documents, because every document contains `a`. The trigger still starts at two words: two-word queries came back empty 1 of 37 times in real sessions, and `missing` names the absent word on every row.
- A local near miss stays hidden when a global document matches every word, because the trigger requires 0 hits in every source. This touches the local-overrides-global rule.
- The field does not help when every word misses, such as `fee` for `price`. Vocabulary gaps stay with semantic retrieval.
- In the bench, the change addresses 1 of the 4 runs that deleted decided code. The other 3 never searched; that failure belongs to recipes and instructions.
- `near_misses` lowers the cost of a vague query, so agents can send multi-topic queries more often. [assumption]

## Alternatives

- Every word except one: the first draft of this RFC. It misses the 8-word and the two-language queries in the reproduction table.
- Best-k, the documents with the highest matched-word count, whatever it is: it never comes back empty, and on long queries it lists documents that match 1 word of 6.
- A caller parameter `min_words`, in the manner of `minimum_should_match`: a clean filter, but it helps only when the agent sets it, and the failing agent did not broaden.
- Per-word document counts such as `token_hits: {"source": 0}`, without rows: they fit the ADR without an addendum and cost about 15 bytes per word, but the agent needs another call to reach the document, and the failing agent made none.
- `any` rows in `results` with a `fallback` flag: `results` then changes meaning by outcome, and an agent that skips the flag reads `any` rows as all-words matches.
- `any` as the default: the recall RFC rejected it for noisy pages.
- Wording and instructions only: this RFC keeps them in §2, but the failing agent already had "broaden the words".
- Recipe guidance only: the `ponytail-minimal` recipe kept the decided code in 3 of 3 runs, but only projects that load such a recipe benefit.
- `any` ranked by distinct-word count first: it helps callers that already use `any`, and it belongs in a separate RFC.

## Verification

1. Unit tests next to @cli/internal/mcp/tools/search_documents.go cover the trigger for one, two, and three or more distinct words, repeated words, the half rule on 8 words and on a two-language query, `missing` for a compound word and for Cyrillic, `[]` presence, omission on non-empty results, order under `sort: "mtime"`, per-source representation, the wording under filters, and the byte budget.
2. `TestSearchDocuments_EnvelopeKeyOrder` and the description test move to the new key order and wording.
3. A replay of the failing run-1 conversation with the new tool result, at least 10 samples, measures whether the agent reads the ADR. Three bench repetitions cannot tell 1 failure in 3 from none.
4. integration-bench reruns `review-seam-buried` with the `ponytail-archcore-lean` arm.
5. A transcript scan compares, before and after the release, the share of empty multi-word `all` queries that end without another search.

## Unresolved Questions

1. Whether the trigger evaluates per source, so that a local near miss appears next to a global full match.
2. Whether the cap follows `limit`, as the smaller of 3 and `limit`.
3. Whether the half rule holds on real queries. Replaying the 5 empty real queries with three or more words would measure it, but their corpora changed since.
4. Whether a `path_ref` search with no rows needs a similar signal. The observed `path_ref` misses were true absences, because the ADR carries no path.