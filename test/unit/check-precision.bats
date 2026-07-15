#!/usr/bin/env bats
# Tests for bin/check-precision (PostToolUse soft warnings)

setup() {
  load '../helpers/common'
  common_setup
  WORK_DIR="$BATS_TEST_TMPDIR/workdir"
  mkdir -p "$WORK_DIR/.archcore"
}

# Helper: write doc content under .archcore/
make_doc() {
  local rel_path="$1"
  local content="$2"
  printf '%s' "$content" > "$WORK_DIR/.archcore/$rel_path"
}

# Helper: run check-precision with stdin in WORK_DIR
run_precision_stdin() {
  local stdin_data="$1"
  run sh -c "cd '$WORK_DIR' && printf '%s' '${stdin_data}' | '${PLUGIN_ROOT}/bin/check-precision'"
}

# A clean ADR with all required sections, valid frontmatter, body >200 chars,
# no forbidden lexicon hits.
CLEAN_ADR='---
title: My Decision
status: accepted
---

## Context

We chose this path because of explicit constraints documented in the team standard.

## Decision

We will adopt approach X with concrete versioning and ownership lines.

## Alternatives Considered

Approach Y was discussed but ruled out for specific compatibility reasons.

## Consequences

Future migrations will follow this exact pattern with clear hand-off rules.
'

# --- Silent paths ---

@test "empty stdin exits silently" {
  run_precision_stdin ''
  assert_success
  assert_output ""
}

@test "non-matching tool name exits silently" {
  run_precision_stdin '{"tool_name":"Write","tool_input":{"path":"my.adr.md"}}'
  assert_success
  assert_output ""
}

@test "missing tool_input.path exits silently" {
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{}}'
  assert_success
  assert_output ""
}

@test "doc file missing on disk exits silently" {
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"nope.adr.md"}}'
  assert_success
  assert_output ""
}

@test "clean ADR produces no findings" {
  make_doc "my.adr.md" "$CLEAN_ADR"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"my.adr.md"}}'
  assert_success
  assert_output ""
}

# --- Each check fires independently ---

@test "forbidden lexicon hit produces finding" {
  local doc='---
title: Robust Plan
status: draft
---

## Context

We need a robust approach for the migration plan with concrete steps documented.

## Decision

Use approach X with explicit versioning and clear ownership for downstream teams.

## Alternatives Considered

Approach Y was ruled out due to specific compatibility constraints last quarter.

## Consequences

Cleanup and migration tasks will follow this exact pattern with hand-off rules.
'
  make_doc "robust.adr.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"robust.adr.md"}}'
  assert_success
  assert_output --partial "forbidden words"
  assert_output --partial "robust"
}

@test "ADR missing mandatory section produces finding" {
  # Has Context+Decision but missing Alternatives Considered + Consequences.
  # Body padded so length warning does NOT fire (isolates section check).
  local doc='---
title: Partial ADR
status: draft
---

## Context

We chose this path with concrete constraints from the standard, documented inline so future readers can trace the reasoning end to end without external context.

## Decision

Adopt approach X with explicit versioning and ownership for downstream teams.
'
  make_doc "partial.adr.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"partial.adr.md"}}'
  assert_success
  assert_output --partial "missing section"
  assert_output --partial "Alternatives Considered"
  assert_output --partial "Consequences"
}

@test "spec with canonical headings produces no findings" {
  # Canonical six-section form per skills/_shared/spec-contract.md.
  local doc='---
title: Card Spec
status: accepted
---

## Purpose & Scope

Normative for the content card; depended on by the feed and search surfaces, which rely on which fields drive which blocks when items render.

## Surface

Blocks and field-drivers referenced at @ui/card/blocks with states listed inline.

## Normative Behavior

1. WHEN episodes is non-empty, the card MUST render the progress block.

## Constraints & Invariants

Invariant: exactly one primary action is visible in the ready state.

## Failure Behavior

1. IF the data source fails, THEN the card MUST enter the unavailable state.

## Conformance

An implementation conforms when it satisfies behavior 1 and the invariant above.
'
  make_doc "card.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"card.spec.md"}}'
  assert_success
  assert_output ""
}

@test "spec with legacy Contract Surface heading produces no section finding" {
  # Pre-canon corpora use Contract Surface / Error Handling; the Surface check
  # accepts the legacy heading so older specs do not produce advisory noise.
  local doc='---
title: Webhook Delivery Spec
status: accepted
---

## Purpose & Scope

Normative for webhook delivery; consumed by external subscriber endpoints that depend on delivery guarantees and the retry policy described in this contract.

## Contract Surface

Deliver entry point and payload schema referenced at @internal/webhooks with identifiers.

## Normative Behavior

1. The service MUST sign every payload with HMAC-SHA256 over the raw body.

## Error Handling

Retries exhausted leads to a failed mark and a delivery.failed event emission.

## Conformance

An implementation conforms when it satisfies behavior 1 and the retry rules above.
'
  make_doc "webhook.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"webhook.spec.md"}}'
  assert_success
  assert_output ""
}

@test "spec containing SHALL produces notation finding" {
  # SHALL is the pure-EARS keyword; the contract grades with BCP 14 modals.
  local doc='---
title: Shall Spec
status: draft
---

## Purpose & Scope

Normative for the loader; depended on by every command that reads the settings contract below.

## Surface

Load and Save referenced at @internal/config with identifiers listed inline.

## Normative Behavior

1. WHEN a file is loaded, the loader SHALL validate it before returning a value.

## Constraints & Invariants

Invariant: a loaded value always passes validation.

## Failure Behavior

1. IF the file is invalid, THEN the loader MUST reject it with a named field.

## Conformance

An implementation conforms when it satisfies behavior 1 and the invariant above.
'
  make_doc "shall.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"shall.spec.md"}}'
  assert_success
  assert_output --partial "SHALL found"
  assert_output --partial "MUST / SHOULD / MAY"
}

@test "lowercase shall does not trigger notation finding" {
  # RFC 8174: only uppercase keywords are normative; prose shall is not flagged.
  local doc='---
title: Prose Spec
status: draft
---

## Purpose & Scope

Normative for the exporter; depended on by downstream jobs that shall be documented in their own contract.

## Surface

Export entry point referenced at @internal/export with identifiers listed inline.

## Normative Behavior

1. WHEN an export is requested, the exporter MUST stream rows in key order.

## Constraints & Invariants

Invariant: output ordering is stable across runs.

## Failure Behavior

1. IF the sink is unreachable, THEN the exporter MUST return an error.

## Conformance

An implementation conforms when it satisfies behavior 1 and the invariant above.
'
  make_doc "prose.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"prose.spec.md"}}'
  assert_success
  assert_output ""
}

@test "spec body over 80 lines produces cap finding" {
  local doc='---
title: Long Spec
status: draft
---

## Purpose & Scope

Normative for the long subject; depended on by consumers that rely on the behavior below.

## Surface

Referenced at @internal/long with identifiers listed inline.

## Normative Behavior

1. The subject MUST respond to every request it accepts.

## Constraints & Invariants

Invariant: the subject holds its ordering guarantee.

## Failure Behavior

1. IF the input is malformed, THEN the subject MUST reject it.

## Conformance

An implementation conforms when it satisfies the behaviors above.
'
  # Pad past the 80-line body cap with clean numbered requirement lines.
  local i
  for i in $(seq 2 60); do
    doc="$doc
$i. WHEN trigger $i fires, the subject MUST respond to it."
  done
  make_doc "long.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__update_document","tool_input":{"path":"long.spec.md"}}'
  assert_success
  assert_output --partial "cap 80"
}

@test "spec compound requirement (two modals in one line) produces finding" {
  # Rule 2: one line, one modal. "MUST show ... and MUST NOT show ..." is two.
  local doc='---
title: Compound Spec
status: draft
---

## Purpose & Scope

Normative for the content card; depended on by the feed and search surfaces that render catalog items from the fields described below.

## Surface

Blocks and field-drivers referenced at @ui/card/blocks with states listed inline.

## Normative Behavior

1. WHEN episodes is non-empty, the card MUST render the progress block.
2. WHILE status is completed, the card MUST show the badge and MUST NOT show the action.

## Constraints & Invariants

Invariant: exactly one primary action is visible in the ready state.

## Failure Behavior

1. IF the data source fails, THEN the card MUST enter the unavailable state.

## Conformance

An implementation conforms when it satisfies the behaviors and the invariant above.
'
  make_doc "compound.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"compound.spec.md"}}'
  assert_success
  assert_output --partial "compound requirement"
}

@test "spec single MUST NOT line does not trigger compound finding" {
  # MUST NOT is one modal, not two — normalized before counting.
  local doc='---
title: Single Modal Spec
status: draft
---

## Purpose & Scope

Normative for the loader; depended on by every command that reads the settings contract described below.

## Surface

Load and Save referenced at @internal/config with identifiers listed inline.

## Normative Behavior

1. WHEN a retry budget is exhausted, the loader MUST NOT retry the request again.

## Constraints & Invariants

Invariant: a loaded value always passes validation.

## Failure Behavior

1. IF the file is invalid, THEN the loader MUST reject it with a named field.

## Conformance

An implementation conforms when it satisfies behavior 1 and the invariant above.
'
  make_doc "single-modal.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"single-modal.spec.md"}}'
  assert_success
  refute_output --partial "compound requirement"
}

@test "spec subjectless passive (EN) produces finding" {
  # Rule 1: name the obligated component. "Tokens MUST be rotated" names none.
  local doc='---
title: Passive Spec
status: draft
---

## Purpose & Scope

Normative for the token rotator; depended on by every session that reads a credential from the store described below.

## Surface

Rotate and Revoke referenced at @internal/auth/rotator with identifiers listed inline.

## Normative Behavior

1. Tokens MUST be rotated every 24 hours.

## Constraints & Invariants

Invariant: a live session always holds a non-expired token.

## Failure Behavior

1. IF rotation fails, THEN the rotator MUST alert the operator.

## Conformance

An implementation conforms when it satisfies behavior 1 and the invariant above.
'
  make_doc "passive.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"passive.spec.md"}}'
  assert_success
  assert_output --partial "subjectless passive"
}

@test "spec active-voice obligated subject does not trigger passive finding" {
  # Same requirement, active voice with the rotator as the obligated subject.
  local doc='---
title: Active Spec
status: draft
---

## Purpose & Scope

Normative for the token rotator; depended on by every session that reads a credential from the store described below.

## Surface

Rotate and Revoke referenced at @internal/auth/rotator with identifiers listed inline.

## Normative Behavior

1. The rotator MUST rotate the token every 24 hours.

## Constraints & Invariants

Invariant: a live session always holds a non-expired token.

## Failure Behavior

1. IF rotation fails, THEN the rotator MUST alert the operator.

## Conformance

An implementation conforms when it satisfies behavior 1 and the invariant above.
'
  make_doc "active.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"active.spec.md"}}'
  assert_success
  refute_output --partial "subjectless passive"
}

@test "spec subjectless passive (RU) produces finding" {
  # Bilingual coverage (mirrors the forbidden-lexicon RU branch): the reflexive
  # ending -ться/-тся on a numbered MUST line is a subjectless passive.
  local doc='---
title: Passive RU Spec
status: draft
---

## Purpose & Scope

Normative for the token rotator; depended on by every session that reads a credential from the store described below.

## Surface

Rotate and Revoke referenced at @internal/auth/rotator with identifiers listed inline.

## Normative Behavior

1. Токены MUST ротироваться каждые 24 часа.

## Constraints & Invariants

Invariant: a live session always holds a non-expired token.

## Failure Behavior

1. IF rotation fails, THEN the rotator MUST alert the operator.

## Conformance

An implementation conforms when it satisfies behavior 1 and the invariant above.
'
  make_doc "passive-ru.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"passive-ru.spec.md"}}'
  assert_success
  assert_output --partial "subjectless passive"
}

@test "spec missing Surface and Conformance produces findings, Subject not required" {
  local doc='---
title: Bare Spec
status: draft
---

## Purpose & Scope

Normative for the export pipeline; depended on by the reporting jobs that read its output files and rely on the ordering guarantees stated below in this document.

## Normative Behavior

1. The pipeline MUST write output files in deterministic order for consumers.
'
  make_doc "bare.spec.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"bare.spec.md"}}'
  assert_success
  assert_output --partial "missing section"
  assert_output --partial "Surface"
  assert_output --partial "Conformance"
  refute_output --partial "Subject"
}

@test "frontmatter missing title produces finding" {
  # doc.md type — no section checks fire, isolates the frontmatter check.
  # Body padded to skip length warning.
  local doc='---
status: draft
---

This document is a placeholder reference covering the core idea with enough text to clear the placeholder threshold so the only finding emitted is the missing frontmatter title.
'
  make_doc "no-title.doc.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"no-title.doc.md"}}'
  assert_success
  assert_output --partial "frontmatter"
  assert_output --partial "title"
}

@test "body shorter than 200 chars produces placeholder finding" {
  # doc.md type avoids section checks, frontmatter complete, body deliberately tiny.
  local doc='---
title: Tiny
status: draft
---

# Hi
'
  make_doc "tiny.doc.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"tiny.doc.md"}}'
  assert_success
  assert_output --partial "<200"
  assert_output --partial "placeholder"
}

@test "body referencing other .archcore/ documents produces finding" {
  # Body padded so length warning does NOT fire (isolates cross-doc check).
  local doc='---
title: Auth Frame Extraction
status: draft
---

## Context

We chose this path with concrete constraints from the team standard, documented inline so future readers can trace the reasoning end to end without external context.

## Decision

Adopt approach X with explicit versioning and ownership for downstream teams.

## Alternatives Considered

Approach Y was ruled out due to specific compatibility constraints last quarter.

## Consequences

Migration tasks will follow this exact pattern with explicit hand-off rules and clear ownership boundaries between modules.

## Related Documents

- `.archcore/auth/popup/architecture.doc.md`
- `.archcore/auth/popup/component-interaction.rule.md`
'
  make_doc "auth-frame.adr.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"auth-frame.adr.md"}}'
  assert_success
  assert_output --partial "references other .archcore/ documents"
  assert_output --partial "architecture.doc.md"
  assert_output --partial "relation graph"
}

@test "body without .archcore/ paths does not trigger cross-doc finding" {
  # Body cites code via @path notation and external sources — both allowed.
  local doc='---
title: Use Postgres
status: accepted
---

## Context

Latency spikes in @internal/scheduler/dispatcher.go forced a database review tied to Grafana dashboard #42 from the recent oncall incident notes.

## Decision

Adopt PostgreSQL 16.2 on RDS db.r7g.xlarge with explicit ownership and runbook coverage.

## Alternatives Considered

MySQL 8 was ruled out because the scheduler module needs pg_advisory_lock semantics not portable across engines.

## Consequences

Teams owning @internal/scheduler/ inherit migration responsibility per the runbook hand-off pattern documented in oncall notes.
'
  make_doc "use-postgres.adr.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"use-postgres.adr.md"}}'
  assert_success
  refute_output --partial "references other .archcore/ documents"
}

@test "multiple findings concatenated with separator" {
  # ADR triggering: forbidden word, missing sections, missing title, short body.
  local doc='---
status: draft
---

## Context
robust approach
'
  make_doc "messy.adr.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"messy.adr.md"}}'
  assert_success
  assert_output --partial "forbidden words"
  assert_output --partial "missing section"
  assert_output --partial "title"
  assert_output --partial "; "
}

# ---------------------------------------------------------------------------
# Check 6: multi-line code blocks in architect-voice types
# ---------------------------------------------------------------------------

@test "check6: long code block (>=5 lines) in ADR produces finding" {
  local doc='---
title: Cache Strategy
status: accepted
---

## Context

Cache invalidation was causing stale reads under high write throughput, measured at p99 > 400 ms.

## Decision

Adopt write-through caching on the user profile service with a 60 s TTL.

## Alternatives Considered

Read-through with lazy invalidation was ruled out because it amplified cache stampede risk during deploys.

## Consequences

All writes to user profile now incur a synchronous cache update; the trade-off is bounded latency at the cost of slightly higher write latency (~5 ms measured on staging).

## Implementation

```go
func writeThrough(ctx context.Context, key string, val []byte) error {
    if err := db.Set(ctx, key, val); err != nil {
        return err
    }
    return cache.Set(ctx, key, val, 60*time.Second)
}
```
'
  make_doc "cache-strategy.adr.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"cache-strategy.adr.md"}}'
  assert_success
  assert_output --partial "code block >=5 lines"
  assert_output --partial "@path/to/file"
}

@test "check6: short inline code in ADR does not trigger finding" {
  local doc='---
title: Use Postgres
status: accepted
---

## Context

Latency spikes in `pkg/scheduler/dispatcher.go` forced a database review. p99 was 420 ms.

## Decision

Adopt PostgreSQL 16.2 on RDS `db.r7g.xlarge` with explicit ownership and runbook coverage.

## Alternatives Considered

MySQL 8 ruled out: `pg_advisory_lock` semantics are not portable across engines.

## Consequences

Teams owning `pkg/scheduler/` inherit migration responsibility per the runbook hand-off pattern.
'
  make_doc "use-postgres.adr.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"use-postgres.adr.md"}}'
  assert_success
  refute_output --partial "code block >=5 lines"
}

@test "check6: short code block (4 lines) in ADR does not trigger finding" {
  # Threshold is 5 content lines; 4 should be silent.
  local doc='---
title: Error Codes
status: accepted
---

## Context

Inconsistent error shapes across services caused integration overhead measured across three teams last quarter.

## Decision

Standardise on a two-field envelope: `code`, `message`.

## Alternatives Considered

Using HTTP status only was ruled out as insufficient for machine-readable retry logic.

## Consequences

All service boundaries must validate against the shared schema; client libraries updated in v2.4.

## Shape

```json
{
  "code": "ERR_NOT_FOUND",
  "message": "resource not found"
}
```
'
  make_doc "error-codes.adr.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"error-codes.adr.md"}}'
  assert_success
  refute_output --partial "code block >=5 lines"
}

@test "check6: long code block in rule type does not trigger finding" {
  # rule is explicitly exempt from Check 6 (Good/Bad examples are normative).
  local doc='---
title: Error Envelope Standard
status: accepted
---

## Rule

All service boundaries MUST return a three-field error envelope.

## Rationale

Consistent shapes reduce client-side branching and enable shared retry middleware.

## Enforcement

PostToolUse hook validates envelope shape in generated code.

## Good

```go
return &Error{
    Code:    "ERR_AUTH",
    Message: "token expired",
    Details: map[string]any{"exp": exp},
}
```

## Bad

```go
return fmt.Errorf("token expired at %v", exp)
```
'
  make_doc "error-envelope.rule.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"error-envelope.rule.md"}}'
  assert_success
  refute_output --partial "code block >=5 lines"
}

@test "check6: long code block in guide type does not trigger finding" {
  # guide is exempt: terminal steps require verbatim commands.
  local doc='---
title: Bootstrap Local DB
status: draft
---

## Steps

Run the initialisation sequence:

```sh
createdb myapp_dev
psql myapp_dev < schema/migrations/001_initial.sql
psql myapp_dev < schema/migrations/002_seed.sql
psql myapp_dev -c "\dt"
\q
```

## Verification

Connect with `psql myapp_dev` and confirm tables are listed.
'
  make_doc "bootstrap-db.guide.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"bootstrap-db.guide.md"}}'
  assert_success
  refute_output --partial "code block >=5 lines"
}

@test "check6: long code block in rfc type produces finding" {
  # rfc is in the architect-voice set; check must fire.
  local doc='---
title: Streaming Ingest RFC
status: draft
---

## Summary

Replace batch ingestion with a streaming pipeline to cut P99 ingest latency from 8 s to under 500 ms.

## Motivation

Batch jobs miss SLO on peak days (measured 14 % breach rate last quarter).

## Detailed Design

Event schema defined inline:

```protobuf
message IngestEvent {
  string  event_id   = 1;
  string  source     = 2;
  bytes   payload    = 3;
  int64   timestamp  = 4;
  string  tenant_id  = 5;
}
```
'
  make_doc "streaming-ingest.rfc.md" "$doc"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"streaming-ingest.rfc.md"}}'
  assert_success
  assert_output --partial "code block >=5 lines"
}

@test "check6: clean ADR (CLEAN_ADR fixture) still produces no findings" {
  # Regression: the canonical clean fixture must remain silent after adding Check 6.
  make_doc "regression-clean.adr.md" "$CLEAN_ADR"
  run_precision_stdin '{"tool_name":"mcp__archcore__create_document","tool_input":{"path":"regression-clean.adr.md"}}'
  assert_success
  assert_output ""
}
