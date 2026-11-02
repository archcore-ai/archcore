---
title: "Delta-Routing Bench — 42 Task Traces Through ΔΠMR"
status: accepted
tags:
  - "architecture"
  - "component:plugin"
  - "plugin"
  - "testing"
---

## Overview

Reference notes from the benches that scored the delta-routing model and its context-driven rollout (2026-08-15 to 2026-10-16). The route-name bench (`route-bench.sh`, `routing-bench.tsv`) was removed on 2026-10-16, when the matrix bench replaced route-name scoring as the routing guard. The surviving harnesses are `plugin/test/behavioral/matrix-bench.py` (`make test-matrix-bench`) and `delivery-bench.py` (`make test-delivery-bench`, no model calls). Per-run score tables are not kept. This document keeps what the next reader needs to trust or discount a bench result.

The original 42 task traces were the model's predicted routes, not executed runs [assumption]. A route-name score saturated at 44 of 44 and could not separate the old package table from a claim-level selection. That saturation is why the matrix bench scores the created types, the questions, the entry, the updated count, and the plan disposition instead.

## Content

### Validity limits

Found in a pre-commit review of the 2026-10-16 runs.

1. The held-out set was tuned. Contract fixes answered held-out cases H03, H11, and H15, so a high held-out score measured fitted cases. A fresh set written by a second author, without reading the runtime contracts or the earlier fixtures, is the only held-out measure (`--set fresh`, `matrix-holdout-fresh.tsv`).
2. The bench gave `plan` and `review` no `skills/_shared/gate-contract.md`, the file that defines the context record and the read rules. The harness loads it for every command since 2026-10-16, so earlier runs are not comparable.
3. The harness exited 0 on every result and passed a verdict that left out a field. It now exits 1 on a failed call, 2 on an error, and fails a verdict with a missing field.
4. Tokens and latency went unrecorded after Phase 0. The report prints both per call.
5. A Haiku run varies by up to 6 held-out calls with no contract change. Read a change only against a same-model baseline, with three repetitions, and compare means.
6. Sonnet failing a case in every repetition points at the contract or the fixture. Three such failures were contract gaps. Three were fixture errors, such as a case that expected no work after a start menu although the harness answers "you decide".

### Retro replay and the recall bound

The replay showed 15 past commits whose author later recorded a `cpat`, `task-type`, or `rule`, and 8 commits with no such record. Each appeared to the review as a one-commit branch without the `.archcore/` changes, three repetitions.

- Recall on the current contracts was 1 of 45 positive calls. False captures were 0 of 24 negative calls.
- The ceiling explains the low recall. A judge reading only the commit message and diff found the reference lesson derivable in full for 1 of 19 reference documents, in part for 12, and not at all for 6. Authors' lessons hold knowledge beyond the diff. The one fully derivable case is a commit whose new skill already cites that `cpat` (`plugin-version-written-from-the-tag.adr`).
- A retro replay therefore measures false captures and bounds recall at about 1 of 19. It does not measure the effect of contract changes.
- With the author's reason supplied as the MR description, an upper bound because the text contains the lesson, the current contracts captured a lesson in 18 of 45 positive calls and 7 of 15 commits by majority. Contracts with every experience sentence removed captured 0 of 45.
- Of 8 commits without a capture, 6 were retro artifacts: the reference document recorded a policy, a standard, or a procedure that the author wrote in the same commit and that the diff does not exhibit. One was a real boundary: a dependency bump proposed as a `task-type` or `guide` in 3 of 3 calls with two earlier bumps in history and 0 of 3 without, so the contract follows its repetition rule.
- A stricter rule that demands evidence of a second run lowered the retro from 18 to 12 of 45 positive calls, because authors recorded procedures after a single run. The rule was reverted.
- The experience set separates "experience text present" from "absent". It does not separate the current contracts from partial removals. Not measured: the effect on later edits, and the acceptance rate in live use.

### F12 and its resolution

Case F12 asks `should we pre-generate image thumbnail variants instead of resizing on the fly?` with no material. The document routing table read a "should we switch to Y" proposal as a decision and created an `rfc`. The rollout plan's verification case 12 wrote nothing for a request without material. The two intent sources disagreed.

`open-choice-offers-four-exits.adr` resolves it. An open choice asks one question with three exits (`adr`, `rfc`, or investigation). A lone "should we" without facts or a reviewer recommends investigation. A fourth exit, an accepted `rfc` and its `adr` in one call, was tried and removed: the settled case D05 then created an `rfc` beside the `adr` in 2 to 3 of 6 Haiku calls, against 6 of 6 correct before the change.

Case F09 (`document journey guest checkout`) created nothing on Haiku in 3 of 3 calls. The situation names no CLI version and `journey` needs CLI 0.8.4, so the case may be underspecified.

### Delivery bench

The file-context change recorded in `file-context-resolution.spec` (local pathless rules before global ones, bare-name collisions) cut files at the cap of 5 matched rows from 100 to 76 over 406 files, cut average candidates per file from 6.01 to 3.73, and cut shown rows on shared base names from 140 to 0. Until 2026-10-16 the bench counted bare-name rows of local documents only. It counts global rows too, per source kind.

## Examples

Legend for route traces: Δ is the canon delta (`∅` none, `mod` modifies accepted spec, `new×N` creates N capabilities, `dec` decision delta, `+int` intent gap). Π is the dominant information source (`machine`, `user`, `world`, `undecided`, `empirical`). M is touched-zone maturity (`pencil`, `stone`). R is the risk flag.

Non-normative worked trace, OpenCode adapter: Δ = one new capability. Π = machine plus world (OpenCode host facts unverified), so a research instrument comes before contract production. M = stone. R = multi-host (`external-contract`). Route: rnd, then spec, then plan. The Codex adapter followed the same route in recorded history.

Non-normative worked trace, Safari button fix: negative canon search, no risk word, no decision. Tier-1 grounding short-circuits to null. No document is created and no closeout obligation arises.
