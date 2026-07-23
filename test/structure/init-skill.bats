#!/usr/bin/env bats
# Structure tests: guard the /archcore:init contract (magic-first-day-init).
#
# Init is an LLM-executed prompt, so its runtime output cannot be asserted here.
# These tests guard the *structural* invariants that the adversarial review of
# the SKILL.md rewrite identified as the real failure modes: dangling catalog
# references, the detect-config security boundary, idempotency-tag drift between
# SKILL.md and the catalogs, and stale pre-rewrite step numbers.

setup() {
  load '../helpers/common'
  common_setup
  SKILL="$PLUGIN_ROOT/skills/init/SKILL.md"
  LIB="$PLUGIN_ROOT/skills/init/lib"
  SHARED="$PLUGIN_ROOT/skills/_shared"
}

@test "init SKILL.md references resolve to existing catalog files" {
  local missing="" ref base
  for ref in $(grep -oE '(skills/)?(lib|_shared)/[a-z0-9-]+\.md' "$SKILL" | sort -u); do
    base=$(basename "$ref")
    case "$ref" in
      *_shared/*) [ -f "$SHARED/$base" ] || missing="$missing $ref" ;;
      *lib/*)     [ -f "$LIB/$base" ]    || missing="$missing $ref" ;;
    esac
  done
  [ -z "$missing" ] || fail "init SKILL.md references missing files:$missing"
}

@test "init foundation catalogs and rule-contract exist" {
  local f
  for f in detect-data-model detect-integrations detect-config detect-surface compose-overview; do
    [ -f "$LIB/$f.md" ] || fail "missing lib/$f.md"
  done
  [ -f "$SHARED/rule-contract.md" ] || fail "missing _shared/rule-contract.md"
}

@test "detect-config enforces the never-emit-values security boundary" {
  grep -qi "NEVER VALUES" "$LIB/detect-config.md" \
    || fail "detect-config.md must carry the bolded 'never values' security rule"
}

@test "init Tier-1 catalogs emit the tags init idempotency keys on" {
  grep -q "data-model"            "$LIB/detect-data-model.md"   || fail "detect-data-model missing 'data-model' tag"
  grep -q "integrations"          "$LIB/detect-integrations.md" || fail "detect-integrations missing 'integrations' tag"
  grep -q "'config'"              "$LIB/detect-config.md"       || fail "detect-config missing 'config' tag"
  grep -q "entry-points"          "$LIB/detect-entry-points.md" || fail "detect-entry-points missing 'entry-points' tag"
  grep -q "'surface'"             "$LIB/detect-surface.md"      || fail "detect-surface missing 'surface' tag"
  grep -q "top-level-map"         "$LIB/detect-domains.md"      || fail "detect-domains missing 'top-level-map' tag"
  grep -q "architecture-overview" "$LIB/compose-overview.md"    || fail "compose-overview missing 'architecture-overview' tag"
}

@test "every Tier-1 detection catalog defines an ## Output section" {
  local f
  for f in detect-stack extract-run-instructions detect-entry-points detect-domains \
           detect-data-model detect-integrations detect-config detect-surface; do
    grep -q '^## Output' "$LIB/$f.md" || fail "lib/$f.md missing an ## Output section"
  done
}

@test "init has no stale pre-rewrite step references" {
  run grep -rn "Step 8" "$PLUGIN_ROOT/skills/init/"
  [ "$status" -ne 0 ] || fail "init still references removed 'Step 8': $output"
}

@test "init SKILL.md documents the phased single-confirm flow" {
  grep -q "Phase A — DETECT"    "$SKILL" || fail "missing Phase A (DETECT)"
  grep -q "Phase E — CREATE"    "$SKILL" || fail "missing Phase E (CREATE)"
  grep -qi "confirm / cancel\|confirm / edit / cancel\|confirm" "$SKILL" || fail "missing confirm gate"
}

@test "init SKILL.md states the pre-confirm write-gate invariant" {
  grep -q "Nothing is written before" "$SKILL" \
    || fail "init must state the no-writes-before-confirm gate"
}

@test "init SKILL.md still calls init_project as pre-gate infrastructure" {
  grep -q "mcp__archcore__init_project" "$SKILL" \
    || fail "init must call init_project"
}

@test "rule-contract defines the mandatory rule body sections" {
  local f="$SHARED/rule-contract.md"
  grep -qi "RFC 2119"   "$f" || fail "rule-contract must require RFC 2119 statements"
  grep -qi "Enforcement" "$f" || fail "rule-contract must require an Enforcement section"
}

# Universality guard: each generalized detection catalog must LEAD with a
# high-level concept + a universal evidence method, and explicitly mark its
# concrete lists as non-exhaustive with the positive-evidence guardrail — so a
# future edit can't silently revert it to a closed allowlist that fails on
# unfamiliar/niche stacks (per magic-first-day-init universality redesign).
@test "generalized detect-* catalogs lead high-level and mark lists non-exhaustive" {
  local missing="" f base
  for base in detect-stack detect-modules detect-domains detect-scale \
              detect-entry-points detect-data-model detect-integrations \
              detect-config detect-surface detect-hotspots detect-cross-cutting \
              extract-run-instructions; do
    f="$LIB/$base.md"
    grep -q "How to find it"   "$f" || missing="$missing $base:no-method"
    grep -q "non-exhaustive"   "$f" || missing="$missing $base:no-nonexhaustive"
    grep -q "positive evidence" "$f" || missing="$missing $base:no-guardrail"
  done
  [ -z "$missing" ] || fail "generalization regressed:$missing"
}

# Import-rebalance guards: agent-instruction files split into aggregate (link) and
# modular-rule (extract-to-rule) classes, and SKILL.md must name the modular Cursor
# directory so detection does not stop at the three root filenames (the emphasis-bias
# failure that left .cursor/rules/*.mdc unimported).
@test "agent-files.md defines aggregate and modular-rule import classes" {
  grep -q "aggregate"    "$LIB/agent-files.md" \
    || fail "agent-files.md must define the 'aggregate' file class"
  grep -q "modular-rule" "$LIB/agent-files.md" \
    || fail "agent-files.md must define the 'modular-rule' file class"
}

@test "init SKILL.md elevates modular .cursor/rules in agent-file detection" {
  grep -q "cursor/rules" "$SKILL" \
    || fail "SKILL.md must name .cursor/rules in agent-file detection, not only CLAUDE.md/AGENTS.md/.cursorrules"
}

# Depth-axis guards: init exposes a synthesis-budget axis (light opt-down / standard
# default / deep opt-up) orthogonal to scale, and the universality invariant that a
# depth is a budget ceiling — never a quota the model pads to hit — must survive edits.
@test "init SKILL.md documents the --depth axis with standard as the default" {
  grep -q -- "--depth" "$SKILL" \
    || fail "SKILL.md must document the --depth axis"
  grep -qF 'default `standard`' "$SKILL" \
    || fail "SKILL.md must state standard as the default depth (light is the opt-down)"
  grep -qi "opt-down" "$SKILL" \
    || fail "SKILL.md must frame light as the explicit opt-down tier"
}

@test "init SKILL.md states the depth ceiling-not-quota universality invariant" {
  grep -qiE "ceiling, never a quota|ceiling, not.*quota|never pad" "$SKILL" \
    || fail "SKILL.md must state that a depth is a budget ceiling, never a quota to pad"
}

# Import-rebalance renamed the extract-mode target dirs (Route 1 → conventions/,
# Routes 2/3 → imported/). Guard against a stale reference to the old imports/* scheme
# creeping back — the taxonomy has no imports/rules|docs|decisions bucket anymore.
@test "init lib/ has no stale pre-rebalance import directories" {
  run grep -rn "imports/rules\|imports/docs\|imports/decisions" "$LIB"
  [ "$status" -ne 0 ] || fail "init lib/ still references pre-rebalance imports/* directories: $output"
}

# Cross-file cap guard: SKILL.md's Depth axis table and detect-hotspots.md's "Top-N by
# mode" table must agree on the large-mode per-selected-domain spec caps. These verbatim
# "min N, cap M" tokens are the highest drift-risk surface after the flat-cap → per-domain
# rebalance (the fix for the 24-domain-repo-gets-3-specs thinness — magic-first-day-init.adr).
@test "SKILL.md and detect-hotspots.md agree on the large-mode per-domain caps" {
  local f
  for f in "$SKILL" "$LIB/detect-hotspots.md"; do
    grep -qF "min 6, cap 12"  "$f" || fail "$(basename "$f") missing large/light cap (min 6, cap 12)"
    grep -qF "min 10, cap 24" "$f" || fail "$(basename "$f") missing large/standard cap (min 10, cap 24)"
    grep -qF "min 14, cap 40" "$f" || fail "$(basename "$f") missing large/deep cap (min 14, cap 40)"
  done
}

@test "init Phase A collects hotspots up to the deep ceiling, not the standard baseline" {
  grep -q "deep.-depth ceiling" "$SKILL" \
    || fail "SKILL.md Step A.3 must collect candidates up to the deep-depth ceiling, not just the standard baseline"
}

# Depth/breadth re-balance guards (foundation-thinness fix): cross-cutting runs at EVERY
# depth (not standard/deep only); the large-mode spec budget scales per selected domain;
# the preview surfaces coverage; flagship hotspots get a raised cap or decomposition; and
# init-synthesized specs are drafts.
@test "cross-cutting synthesis runs at every depth, not standard/deep only" {
  grep -qF 'at **every**' "$LIB/detect-cross-cutting.md" \
    || fail "detect-cross-cutting.md must state the scan runs at every --depth"
  grep -qi "every depth" "$SKILL" \
    || fail "SKILL.md must state cross-cutting runs at every depth"
}

@test "large-mode hotspot budget scales per selected domain" {
  grep -q "per selected domain" "$LIB/detect-hotspots.md" \
    || fail "detect-hotspots.md must scale the large-mode cap per selected domain"
  grep -qiE "per.selected.domain|per-domain" "$SKILL" \
    || fail "SKILL.md must document per-domain spec scaling in large mode"
}

@test "init preview surfaces a coverage line" {
  grep -q "Coverage:" "$SKILL" \
    || fail "SKILL.md Phase C must include a Coverage line"
  grep -q "load-bearing modules" "$SKILL" \
    || fail "SKILL.md Coverage line must report load-bearing module count"
}

@test "flagship hotspots get a raised cap or decomposition" {
  grep -q "Flagship specs" "$LIB/detect-hotspots.md" \
    || fail "detect-hotspots.md must define the flagship (size/churn-gated) treatment"
  grep -qi "flagship" "$SHARED/spec-contract.md" \
    || fail "spec-contract.md must define the flagship body cap"
}

@test "init-synthesized hotspot specs are created as drafts" {
  grep -qiE "status.{0,3}draft" "$SHARED/spec-contract.md" \
    || fail "spec-contract.md must state init-synthesized specs are status: draft"
  grep -q "status='draft'" "$SKILL" \
    || fail "SKILL.md Phase E must create hotspot specs with status='draft'"
}
