# Capability Granularity — Sizing Contract for Δ Capability Lists

Plugin runtime asset. Loaded by the `plan` skill together with
`skills/_shared/delta-routing.md`. Companion to
`skills/_shared/spec-contract.md`.

## Anchor

A capability is one behavior an external consumer relies on — external code, a
team, users/UI, or a sibling module — recordable as one `spec` within the
80-line body cap of `skills/_shared/spec-contract.md`.

## Rules

1. Count one capability per consumer-relied behavior, not per file, module, or
   task.
2. WHEN one candidate capability cannot be specified within the spec body cap,
   split it by separable sub-surface and count each part.
3. WHEN two candidate capabilities always change together and share one
   consumer set, count them as one. [assumption — merge heuristic; revisit
   against recorded routing traces]
4. IF `creates` exceeds 5 capabilities, THEN re-scope the initiative before
   routing. [assumption — soft cap]
5. A behavior nobody outside the change relies on is not a capability; it
   needs no `spec`.

## Worked boundary

Non-normative examples.

- "CSV export for reports" — one capability: one consumer-relied behavior,
  regardless of file count.
- "Notifications: email, webhook, in-app digest" — three capabilities: three
  consumer surfaces with distinct failure modes.
- "Refactor the retry helper" — zero capabilities: consumers see unchanged
  behavior; the decision delta may still license an `adr` or `cpat`.
