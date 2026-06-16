---
title: "Do Not Auto-Link Nearby Documents on create_document"
status: accepted
tags:
  - "relations"
---

## Context

When `create_document` was first wired up to surface `nearby_documents` (other documents in the same subdirectory as the newly created file), it also auto-created `related` relations between the new document and every neighbor, persisting them to the sync manifest and reporting `auto_relations_added` in the response.

The intent was convenience: if you drop a new rule into `.archcore/auth/`, it is probably related to the other auth documents there. In practice the behavior was wrong more often than right:

- Co-location in a directory means shared topic, not semantic link. Two ADRs about unrelated auth decisions share a folder but not a real relation.
- `related` is the weakest relation type but is not semantically empty — creating it for every neighbor pollutes the graph with noise that dilutes the meaningful links.
- The correct relation between a new doc and an existing neighbor is often `implements`, `extends`, or `depends_on` — never discoverable from directory membership alone. Auto-creating `related` pre-empts that choice and hides the need to make it.
- Agents had no way to decline: the relation was persisted before the tool response was even seen.

## Decision

`create_document` no longer creates any relations. It returns `nearby_documents` as a hint only — a list of paths in the same directory that the agent should review. The agent decides, per neighbor, whether a semantic link exists and calls `add_relation` explicitly with the appropriate type.

The tool description was updated to make this explicit: "Treat nearby_documents as a hint only: review each candidate and call add_relation explicitly when a semantic link exists. Do not link every neighbor by default."

The helper was renamed from `addNearbyRelations` to `populateNearbyDocuments` to match its new single responsibility. The `auto_relations_added` and `auto_relations_error` response fields are gone.

## Alternatives Considered

### 1. Keep auto-linking but let agents opt out via a flag

Add a `skip_auto_relations: bool` parameter to `create_document`. Rejected: inverts the default in the wrong direction. The noisy behavior stays the baseline and every caller has to know to disable it. Agents that do not know the flag exists keep polluting the graph.

### 2. Auto-link only when the new document has a type that commonly relates to its neighbors

For example, auto-link a new `plan` to any `prd` in the same directory with `implements`. Rejected: the heuristics are fragile (directory layout is free-form), and guessing a relation type is a worse failure mode than not creating one — a wrong `implements` edge is harder to detect and clean up than a missing one.

### 3. Drop `nearby_documents` entirely

If we are not going to create relations, maybe the hint is not worth returning either. Rejected: the hint is cheap to produce and genuinely useful — it prompts the agent to consider relations at creation time, which is the right moment to do it. Without the hint, relations tend to be forgotten and documents stay orphaned.

## Consequences

**Positive:**

- Graph contains only intentional relations — `related` edges now carry real signal.
- Agent chooses the right relation type (`implements`, `extends`, `depends_on`, `related`) per pair instead of getting `related` by default.
- `create_document` is a single-responsibility tool again — it creates a document and nothing else.
- Aligns with how `add_relation` is designed: relations are a deliberate act, not a side effect.

**Negative:**

- Documents created without a follow-up `add_relation` call stay unlinked. `archcore doctor` and `/archcore:review` surface orphans, so this is detectable but no longer prevented at creation time.
- Agents that were relying on the old auto-link behavior need to be told to call `add_relation` explicitly. The updated tool description handles new sessions; existing agent prompts may carry stale expectations.

**Neutral:**

- The sync manifest schema and `add_relation` / `remove_relation` / `list_relations` tools are unchanged — only the `create_document` side effect is removed.
