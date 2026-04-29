# Precision Rules — Forbidden Lexicon and Authoring Conventions

Plugin runtime asset. Loaded by skills (`decide`, `standard`, `capture`) before
composing documents of type `adr`, `spec`, `rule`, `guide`. See also
`skills/_shared/adr-contract.md`.

## Rules

1. **Forbidden vagueness lexicon.** New documents and updates MUST NOT introduce
   these words: `appropriate`, `robust`, `scalable`, `modern`, `best practices`,
   `various`, `as needed`, `optimal`, `efficient`, `flexible`, `convenient`,
   `seamless`, `streamlined`, `world-class`, `cutting-edge`, `оптимальный`,
   `удобный`, `правильный`, `надёжный`, `гибкий`, `современный`, `передовой`,
   `эффективный`, `масштабируемый`. Replace with a concrete fact, version,
   threshold, or measured outcome. Existing occurrences in pre-existing
   documents are not flagged.

2. **Imperative phrasing in normative sections.** Documents of type `rule`,
   `spec`, and any contract document MUST use `MUST` / `MUST NOT` / `MAY` for
   prescriptive statements. Narrative phrasing ("we should", "it is recommended",
   "следует") is forbidden in those sections. Other sections (Rationale,
   Examples, Context) MAY use narrative voice.

3. **`[assumption]` marker.** When a technical claim cannot be grounded in
   existing code, prior measurement, or external authority, it MUST be marked
   `[assumption]` inline at the start of the claim or sentence. Vision-stage
   documents (idea, prd, plan, mrd, brd, urd) MAY contain many such markers;
   decision-stage documents (adr, spec, rule) SHOULD contain few.

4. **Falsifiable claims.** Performance, scale, reliability, and behavior claims
   MUST include a measured value with units and a measurement context
   (`p99 < 200ms at 1000 rps, load profile L2`) OR be marked `[assumption]`.
   Adjective-only claims (`fast`, `scalable`, `low-latency`) without a
   measurement are forbidden.

## Examples

### Good

- "PostgreSQL 16.2 on RDS db.r7g.xlarge — chosen because the team needs
  `pg_advisory_lock` (used in the scheduler module's distributed-lock helper)."
- "Authentication MUST verify JWT signature using ES256. [assumption] Token
  rotation will be 24h pending security review."
- "Reduces p99 latency from 4.2s to <80ms under load profile L2 (Grafana
  dashboard #42, 2024-03-15)."
- "MUST NOT call this function from within a transaction — it acquires its
  own connection."

### Bad

- "Chose a robust, scalable database appropriate for our needs."
- "Authentication should be reliable and modern."
- "Significantly improves performance under load."
- "It is recommended to use the helper for convenience."

## Enforcement

- The plugin's `bin/check-precision` PostToolUse hook detects forbidden lexicon
  words in newly created documents and (in later phases) in additions during
  `update_document`. Findings are emitted as `additionalContext`. The hook never
  blocks (always exits 0).
- Skills load this asset and the relevant contract before composition.
- `/archcore:verify` scans existing documents for forbidden words and missing
  `[assumption]` markers in vision-stage documents; reports findings as
  recommendations rather than failures for pre-existing documents.
