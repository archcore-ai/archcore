---
title: "Track-File Line Cap Raised to 300 and Pinned by a Structure Test"
status: draft
tags:
  - "architecture"
  - "component:plugin"
  - "plugin"
  - "skills"
  - "testing"
---

## Context

@.archcore/plugin/plugin-architecture.spec.md carried the constraint "a track file MUST NOT exceed 200 lines" with no test behind it. On 2026-09-16 three of the eight track files under @plugin/plugins/archcore/skills/_shared/tracks/ exceeded it: `research.md` at 209 lines, `closeout.md` at 235, and `sdd.md` at 277 after the `sdd.illustrate` gate joined it. Two of the three were over the cap before the actor-subject vocabulary work (`closeout.md` 226, `sdd.md` 214). The excess is structural: a track file hosts one gate record per instrument, the gate record template in @plugin/plugins/archcore/skills/_shared/gate-contract.md costs 30 to 45 lines per gate, and `sdd.md` hosts six instruments by the conductor's design (@.archcore/plugin/delta-routing-instruments.spec.md). Six gates alone occupy about 230 lines before the Track notes.

## Decision

Raise the constraint to 300 lines and pin it with @plugin/test/structure/track-file-cap.bats, which fails on any track file over the cap and on a spec that states a different number. The cap keeps its purpose — a file past it is a signal to route prose out of the file (Track notes into a shared contract) or to split the track — and the number now admits a six-instrument track written to the gate template. No track file is edited by this decision.

## Alternatives Considered

- **Decompose `sdd.md` into one file per instrument.** Rejected for now: it breaks the track-layer invariant that a track is one file plus one routing row per calling skill, and it touches the goldens, the instrument registry paths, the routing fixtures, and four accepted specs; `closeout.md` and `research.md` would stay over 200 regardless.
- **Trim the Track notes of `sdd.md` to fit 200.** Rejected: the notes are 47 lines, so the file would still sit near 230; the gate records are the mass, and compressing a gate record removes the fields the gate contract requires.
- **Delete the constraint.** Rejected: an unbounded track file is the failure mode the cap exists to catch; a number with a test is better than no number.

## Consequences

### Enabled

- [expected] The constraint and the repository agree, and `make test-structure` reports the next breach at the commit that causes it.
- [expected] A six-instrument track fits the cap with about 20 lines of headroom; a seventh instrument on `sdd` forces the decomposition question rather than a silent overrun.

### Costs and limits

- [expected] A 300-line track file is 100 lines longer for an agent to hold in context than the earlier number; the routing bench has not measured that cost.
- The split-by-instrument refactor stays a candidate for a later `/archcore:plan`; this decision defers it, it does not reject it.

## Superseded when

- A track is split by instrument and the largest remaining track file fits a lower number.
- The gate contract template shrinks or grows so that a six-gate file no longer fits 300 lines.
