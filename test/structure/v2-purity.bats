#!/usr/bin/env bats
# Structure tests: v2 command-surface purity (plan task 27, Stage 5).
#
# Acceptance criterion: "CI grep finds no /archcore:{context,capture,decide,
# audit,help} string in shipped files."
#
# Until the Stage 4 cutover this suite was a RATCHET around a hardcoded LEGACY
# allowlist of files still legitimately carrying stale strings. Stage 4 deleted
# the old skills/wrappers (task 21) and swept the survivors (task 22), so the
# allowlist is retired and the acceptance criterion holds directly:
#   - Test 1 greps the ENTIRE shipped plugin tree and requires zero hits.
#   - Test 2 guards the v2 core set file-by-file, so a renamed or deleted core
#     file is reported as MISSING instead of silently dropping out of Test 1's
#     coverage.
#   - Test 3 guards the shipped repo-root adjacents (.claude-plugin/,
#     .cursor-plugin/, .agents/, docs/).
#
# Pattern choice: the tree was checked on 2026-08-05 for slash-less forms
# (`archcore:capture` in backticks etc.) — zero occurrences; every stale
# string carried the leading slash. STALE_RE nevertheless omits the slash so
# a future backticked or plugin-qualified form (e.g. `archcore:help`) is
# caught too. Portable grep only: -r/-l/-n/-I/-E (no -P, works on GNU + BSD).
#
# Scope: shipped locations only — the plugin tree plus the four repo-root
# adjacents above. test/, Makefile, and .github/ are not shipped and may
# mention old commands (fixtures, this file's own pattern).

setup() {
  load '../helpers/common'
  common_setup
}

STALE_RE='archcore:(capture|decide|audit|context|help)'

# CLEAN: the v2 core set, relative to PLUGIN_ROOT. Guarded file-by-file so a
# rename or deletion surfaces as MISSING rather than escaping Test 1 silently.
CLEAN=(
  "skills/plan/SKILL.md"
  "skills/document/SKILL.md"
  "skills/review/SKILL.md"
  "skills/init/SKILL.md"
  "skills/init/lib/compose-overview.md"
  "commands/init.md"
  "commands/plan.md"
  "commands/document.md"
  "commands/review.md"
  "skills/_shared/adr-contract.md"
  "skills/_shared/branch-state.md"
  "skills/_shared/coverage-taxonomy.md"
  "skills/_shared/elicitation-contract.md"
  "skills/_shared/gate-contract.md"
  "skills/_shared/globals.md"
  "skills/_shared/precision-rules.md"
  "skills/_shared/rule-contract.md"
  "skills/_shared/spec-contract.md"
  "skills/_shared/tracks/actualize.md"
  "skills/_shared/tracks/decision.md"
  "skills/_shared/tracks/describe.md"
  "skills/_shared/tracks/experience.md"
  "skills/_shared/tracks/requirements-cascade.md"
  "skills/_shared/tracks/sdd.md"
  "skills/_shared/grounding/detect-config.md"
  "skills/_shared/grounding/detect-cross-cutting.md"
  "skills/_shared/grounding/detect-data-model.md"
  "skills/_shared/grounding/detect-domains.md"
  "skills/_shared/grounding/detect-entry-points.md"
  "skills/_shared/grounding/detect-hotspots.md"
  "skills/_shared/grounding/detect-integrations.md"
  "skills/_shared/grounding/detect-modules.md"
  "skills/_shared/grounding/detect-scale.md"
  "skills/_shared/grounding/detect-stack.md"
  "skills/_shared/grounding/detect-surface.md"
  "skills/_shared/grounding/extract-routing.md"
  "skills/_shared/grounding/extract-run-instructions.md"
)

@test "no stale v1 command strings anywhere in the shipped plugin tree" {
  local viol="" f rel
  while IFS= read -r f; do
    rel="${f#"$PLUGIN_ROOT/"}"
    viol="$viol"$'\n'"$(grep -nE "$STALE_RE" "$f" | sed "s|^|${rel}:|")"
  done < <(grep -rIlE "$STALE_RE" "$PLUGIN_ROOT")
  [ -z "$viol" ] || fail "Stale v1 command strings in the plugin tree (v2 surface is init/plan/document/review):$viol"
}

@test "v2 core set (skills, wrappers, contracts, tracks, grounding) present and free of stale strings" {
  local viol="" entry hits
  for entry in "${CLEAN[@]}"; do
    if [ ! -f "$PLUGIN_ROOT/$entry" ]; then
      viol="$viol"$'\n'"$entry: MISSING (update CLEAN in test/structure/v2-purity.bats if renamed)"
    else
      hits=$(grep -nE "$STALE_RE" "$PLUGIN_ROOT/$entry" || true)
      [ -z "$hits" ] || viol="$viol"$'\n'"$(printf '%s\n' "$hits" | sed "s|^|${entry}:|")"
    fi
  done
  [ -z "$viol" ] || fail "Stale v1 command strings in the v2 core set (must reference only init/plan/document/review):$viol"
}

@test "repo-root shipped adjacents have zero stale command strings" {
  # .claude-plugin/ (marketplace manifest), .cursor-plugin/ and .agents/
  # (host adapters), docs/ (published docs) ship alongside the plugin and
  # were verified already clean — guard them with no allowlist. A missing
  # directory is skipped: these locations may be restructured independently
  # of the command surface.
  local viol="" d hits
  for d in .claude-plugin .cursor-plugin .agents docs; do
    [ -d "$REPO_ROOT/$d" ] || continue
    hits=$(grep -rnIE "$STALE_RE" "$REPO_ROOT/$d" || true)
    [ -z "$hits" ] || viol="$viol"$'\n'"$hits"
  done
  [ -z "$viol" ] || fail "Stale v1 command strings in repo-root shipped locations:$viol"
}
