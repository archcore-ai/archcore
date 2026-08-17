---
title: "Decompose over Truncate — Over-Cap Documents Split, Trimmed Lists State Their Remainder"
status: accepted
tags:
  - "architecture"
  - "plugin"
  - "precision"
  - "skills"
---

## Context

An audit of every line and count cap in the plugin found four places where content was reduced with no prescribed remedy and no visible marker. `skills/_shared/spec-contract.md`'s 80-line body cap had an alternative only inside `/archcore:init`'s flagship gate, so `describe.draft` — the gate `/archcore:document` runs on a module — composed one spec, and an over-cap subject was left to the composing model while the CLI reported `spec body is N lines (cap 80)` after the write. `skills/_shared/grounding/detect-integrations.md` and `detect-surface.md` dropped detected units past their 15- and 25-line caps with no count of what went missing, while `detect-config.md` and `detect-data-model.md` already printed `+N more`, so a reader could not tell an index from an inventory. `skills/_shared/capability-granularity.md` directed the conductor to re-scope an initiative above five capabilities, and `maxReportFindings` in the Archcore CLI (`@internal/advisory/precision.go`, `archcore-ai/cli` repository) cut every advisory report at five findings, which hid whole check kinds on the documents that tripped the most.

## Decision

Adopted a decompose-over-truncate policy across the content contracts and the advisory report: `spec-contract.md` gains an "Over the cap" procedure — reference instead of reproduce, route foreign content to its owning type, split by separable sub-surface, keep one spec and report the excess when no boundary is unambiguous, and never delete normative content to fit — wired into `describe.draft` as a blocking exit check; `detect-integrations.md` and `detect-surface.md` state their remainder as `+N more`; `capability-granularity.md` routes any capability count to an umbrella instead of a re-scope; and `maxReportFindings` rises from 5 to 12.

The policy rests on one distinction. A cap that bounds **how much one document says about one subject** is a quality instrument, and it stays. A cap that bounds **how much of the repository is described at all** is a coverage instrument, and it silently loses knowledge. Every remedy above converts the second kind into either a split, a route, or a stated remainder.

Two boundaries stay unchanged. The hook reports and never blocks, so an over-cap document is still written. The flagship gate in `skills/_shared/grounding/detect-hotspots.md` keeps its ceiling of 3 sub-specs, because a decomposed flagship shares **one** slot of init's spec budget: an unbounded split would let a single budgeted slot expand without bound, and the budget is what the user priced when confirming. The user does see the treatment before it runs — `SKILL.md` Phase C prints `[flagship: raised cap]` or `[flagship: split → 2 sub-specs]` on the stub — so the ceiling bounds cost, not review exposure.

## Alternatives Considered

1. **Raise the line caps instead of prescribing decomposition** — rejected because a larger number moves the truncation point without giving the composing skill a remedy; a 120-line cap fails the same way at 121 lines. This does not preempt a cap revision: `spec-single-narrative-ears-bcp14.adr` names "a revised cap rather than decomposition" as its own supersession trigger, on measurement that the 80-line cap is unreachable for a majority of specs. That measurement changes the number; the procedure decided here governs what happens above whatever the number becomes, so the two decisions compose rather than compete.
2. **Make the PostToolUse hook block an over-cap write** — rejected because the report-never-block boundary is load-bearing for the whole advisory layer, and a line count cannot distinguish a padded spec from a legitimate flagship one.
3. **Group advisory findings by check kind rather than raising the cap** — deferred because findings are unlabeled strings and grouping needs a typed refactor of every check in the CLI's `@internal/advisory/precision.go`; the raise delivers the coverage now and leaves that refactor free to happen later.
4. **Keep the five-capability soft cap and let the user re-scope** — rejected because `skills/_shared/delta-routing.md` already routes `creates ≥ 2` to an umbrella `prd` plus one `spec` per capability at any count, so refusing to route an eight-capability initiative declined to record real work rather than simplifying it.
5. **Add remainder markers only where a user reported a loss** — rejected because a silent trim is by construction unreported: the reader who needs the marker is the one who cannot know it is missing.

## Consequences

- A module too large for one spec now yields one spec per separable sub-surface from `/archcore:document`, where it previously yielded one over-cap document and a post-write advisory finding.
- Every capped list in the init seed states its remainder, so `detect-integrations.md` and `detect-surface.md` match the convention `detect-config.md` and `detect-data-model.md` already followed.
- An advisory report now carries up to 12 findings instead of 5, so a backlogged document surfaces the kinds that the cut previously hid. The report length is bounded by the number of distinct checks a type owns, not by the number of occurrences, because each check already caps its own examples at 5.
- Tradeoff: a decomposed subject produces more documents and more relation edges, which raises the count `/archcore:review` reports and the graph a reader traverses.
- Tradeoff: [expected] an advisory report at the new cap injects roughly 2.4 times the tokens of the old one on the worst documents, bounded at about 12 lines per write.
- Tradeoff: rule 5 of the "Over the cap" procedure leaves a judgment call — whether a sub-surface boundary is unambiguous — with the composing model, and a wrong call splits one contract into fragments. "Prefer omission over a guess" governs it, the same way it governs hotspot selection.

## Superseded when

- Corpus measurement shows split sub-specs being read or referenced less often than the single over-cap documents they replaced, over a 90-day window.
- Advisory reports routinely reach the 12-finding cap on documents authored after this decision, which would show the cap still cutting rather than covering.
- A typed finding model lands in the CLI's `@internal/advisory/precision.go`, making per-kind grouping cheaper than a flat cap.
