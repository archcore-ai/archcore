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
