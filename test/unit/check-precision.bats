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
