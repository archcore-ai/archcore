---
title: "document journey Enters sdd.require in Callable Mode and Produces Only the Journey"
status: rejected
tags:
  - "document-types"
  - "plugin"
  - "skills"
---

## Context

The global RFC `concepts/scenario-and-journey-types` [global · archcore · read-only] requires the `/archcore:document` argument hint to list `journey`, but names no track for that entry. A `journey` is a vision document: the intended path of one user type before a `spec` covers the interaction. On `/archcore:plan` it is produced beside the `prd` at `sdd.require` under the illustrate condition (@.archcore/plugin/illustrate-instrument.spec.md). The `document` command writes knowledge types, with one recorded exception: `document research` files a ready vision artifact through the research instrument (@.archcore/plugin/research-runtime-category-and-evidence-entry.adr.md). The compatibility spec carried the entry's track as `[assumption]` until the brainstorming session of 2026-09-16.

## Decision

`document journey <topic>` enters `sdd.require` in callable mode with the scope pre-filled from the request, produces only the `journey`, creates no `prd`, and exits to the command. The entry mirrors `document research`: a ready vision artifact filed through a plan-side instrument. `plan` exposes no `journey` entry; on `plan` the journey comes only from `sdd.require` under the illustrate condition.

## Alternatives Considered

- **No `document journey` entry; journey only on `plan`.** Rejected: diverges from the RFC's argument-hint clause and would require a global RFC revision at acceptance.
- **A journey gate on the describe track.** Rejected: describe documents existing code, a journey records intent before code; the type would sit in a track whose evidence base is source files.
- **A dedicated `journey` track file.** Rejected: one gate does not justify a track; the track-layer invariant charges every new track one file plus one routing row per calling skill.

## Consequences

### Enabled

- [expected] One production point for `journey` in `skills/_shared/tracks/sdd.md`; the document skill adds a routing row and no gate.
- [expected] The argument hint and the expert-invocation map name the same surface: `[adr|rfc|spec|doc|guide|rule|research|evidence|scenario|journey]`.
- [expected] The callable entry runs scope-question-free, per the instruments spec's callable-mode rule.

### Costs and limits

- [expected] `sdd.require` gains a second recorded exception to single-type production; the constraint is stated in the illustrate-instrument spec.
- [expected] A journey filed this way has no `prd` to relate to until a later `plan` run creates one; the `related` edge is added then.
- The entry is gated on CLI 0.8.4 through `skills/_shared/actor-subject-compatibility.md` [planned]; on an older engine the skill writes nothing and reports the version.

## Superseded when

- The global RFC assigns `document journey` a different track at acceptance.
- The engine moves `journey` out of vision.
