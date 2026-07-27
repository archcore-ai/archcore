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

# Host-wiring parity guards (CLI >= v0.6.1 moved the claude-code nudge from the owned
# .claude/rules/archcore.md to CLAUDE.md + AGENTS.md managed blocks; the CLI deletes the
# legacy file on rewire). The preview/closing file lists and the version gate must move
# in lockstep — a stale path or gate value re-opens the preview/reality mismatch where
# the skill promises one layout and the CLI writes another.
@test "init host wiring names CLAUDE.md + AGENTS.md for claude-code, never the legacy rules file" {
  run grep -rn "\.claude/rules/archcore\.md" "$PLUGIN_ROOT/skills/init/"
  [ "$status" -ne 0 ] || fail "init still references the legacy .claude/rules/archcore.md: $output"
  grep -qE 'claude-code → .*CLAUDE\.md.*AGENTS\.md' "$SKILL" \
    || fail "SKILL.md per-host list must name CLAUDE.md + AGENTS.md for claude-code"
}

# Copilot host parity (copilot-adapter-design.adr). bin/detect-host cannot detect
# Copilot — the CLI exports no marker into agent shell commands — so a Copilot
# session ALWAYS falls through to the AskUserQuestion branch. If that question
# offers only the three detectable hosts, a Copilot user is stuck with no correct
# answer and init picks the wrong agent id. The ask-fallback is therefore the only
# thing that makes init usable on this host, and it must not silently regress.
@test "init host wiring names all three files copilot needs, and calls them non-optional" {
  # The claude-code row above is pinned by name; this is its copilot twin, and
  # without it the row can be trimmed to any subset and nothing notices.
  #
  # The three are not interchangeable. The plugin ships no MCP for Copilot
  # (copilot-mcp-architecture.adr), so .mcp.json is the ONLY thing that gives
  # that host document tools — a session missing it has skills and hooks and no
  # way to read or write a document, which looks like a broken plugin rather
  # than missing wiring. The `archcore init --agent copilot` writes exactly
  # these three (internal/agents/copilot.go, internal/wiring/hooks_agents.go),
  # so a drift here is the skill promising a layout the CLI does not produce.
  grep -qE 'copilot → .*\.mcp\.json.*\.github/hooks/archcore\.json.*AGENTS\.md' "$SKILL" \
    || fail "SKILL.md per-host list must name .mcp.json + .github/hooks/archcore.json + AGENTS.md for copilot"
  grep -q 'Host wiring line is never optional' "$SKILL" \
    || fail "SKILL.md must state that host wiring is not optional on copilot — no wiring means no document tools at all"
}

@test "init host question offers Copilot, which detect-host can never return" {
  grep -qi "GitHub Copilot CLI" "$SKILL" \
    || fail "SKILL.md Step -1 must offer GitHub Copilot CLI in the host AskUserQuestion — detect-host cannot return it"
  grep -qF '`copilot`' "$SKILL" \
    || fail "SKILL.md must map the Copilot answer to the 'copilot' agent id"
  grep -q '__UNKNOWN__' "$SKILL" \
    || fail "SKILL.md must keep the __UNKNOWN__ fallback the Copilot path depends on"
}

# The four hosts detect-host CAN return must stay in lockstep between the script's
# contract and the skill's prose — a token added to one and not the other silently
# routes a real session into the ask-fallback (or worse, an unmapped id).
@test "init SKILL.md and bin/detect-host agree on the emitted token set" {
  local tok
  for tok in claude-code cursor codex-cli __UNKNOWN__; do
    grep -qF "$tok" "$PLUGIN_ROOT/bin/detect-host" \
      || fail "bin/detect-host no longer emits '$tok' — SKILL.md Step -1 still documents it"
    grep -qF "$tok" "$SKILL" \
      || fail "SKILL.md Step -1 must document the '$tok' token bin/detect-host emits"
  done
  grep -q 'echo "copilot"' "$PLUGIN_ROOT/bin/detect-host" \
    && fail "bin/detect-host now emits 'copilot' — update SKILL.md Step -1 and drop the ask-fallback note"
  return 0
}

@test "init version gate pins CLI >= 0.6.4 with no stale gate references" {
  # v0.6.4 is the release where the Copilot writer stopped targeting
  # .vscode/mcp.json — a surface Copilot CLI dropped in v1.0.37 — and started
  # writing the workspace-root .mcp.json it actually reads. Since Copilot has
  # no plugin-shipped MCP (copilot-mcp-architecture.adr), an older CLI leaves
  # that host with no document tools at all, so the gate has to move with it.
  grep -qF 'cli-gte" 0.6.4' "$SKILL" \
    || fail "SKILL.md must gate host wiring on cli-gte 0.6.4"
  run grep -n "cli-gte\" 0\.6\.[0-3]\b\|cli-gte 0\.6\.[0-3]\b" "$SKILL"
  [ "$status" -ne 0 ] || fail "SKILL.md still calls the gate with a stale version: $output"
  run grep -n "CLI < v0\.6\.[0-3]\b\|older than v0\.6\.[0-3]\b" "$SKILL"
  [ "$status" -ne 0 ] || fail "SKILL.md still names a stale gate version to the user: $output"
}

@test "import flow strips the archcore managed block, never re-importing its own nudge" {
  grep -qF "archcore:start" "$LIB/extract-routing.md" \
    || fail "extract-routing.md Block splitting must ignore the archcore managed block"
  grep -qF "archcore:start" "$LIB/agent-files.md" \
    || fail "agent-files.md must exclude managed-block-only files from import candidacy"
}

@test "SKILL.md and skills-system.spec.md agree on the CLI wiring gate version" {
  # Derived from the skill rather than hardcoded twice: the next bump then
  # touches one literal, and this test still catches the spec falling behind.
  local gate
  gate=$(grep -o 'cli-gte" [0-9]\+\.[0-9]\+\.[0-9]\+' "$SKILL" | head -1 | awk '{print $2}')
  [ -n "$gate" ] || fail "SKILL.md has no cli-gte call to read the gate version from"
  grep -qF "v$gate" "$REPO_ROOT/.archcore/plugin/skills-system.spec.md" \
    || fail "skills-system.spec.md must pin the same v$gate wiring gate (SKILL.md says $gate)"
}

@test "init Step A.4 sizes CLAUDE.md/AGENTS.md only after stripping the managed block" {
  grep -qF "after stripping any archcore managed block" "$SKILL" \
    || fail "SKILL.md Step A.4 must require stripping the managed block before sizing CLAUDE.md/AGENTS.md — a raw byte-size probe re-admits managed-block-only files as import candidates"
}

@test "init description names the managed block and no lib file drifts to a synonym" {
  grep -qE '^description: .*CLAUDE\.md/AGENTS\.md managed block' "$SKILL" \
    || fail "SKILL.md frontmatter description must name the CLAUDE.md/AGENTS.md managed block as a host-wiring output"
  run grep -rn "usage nudge" "$PLUGIN_ROOT/skills/init/"
  [ "$status" -ne 0 ] || fail "init skill re-introduced the 'usage nudge' synonym (canonical terms: 'managed block' / 'usage hint'): $output"
}
