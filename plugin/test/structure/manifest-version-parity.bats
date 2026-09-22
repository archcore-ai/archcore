#!/usr/bin/env bats
# Version parity across ALL FOUR host manifests.
#
# Partial pins existed before this file — copilot=claude
# (copilot-plugin.bats) and codex=claude (codex-plugin.bats) — but nothing
# compared the CURSOR manifest to anything: a version bump that missed
# .cursor-plugin/plugin.json shipped green. One release train, one version,
# four manifests (bump-plugin-version.cpat.md); any divergence is a botched
# bump by definition.

setup() {
  load '../helpers/common'
}

MANIFESTS=".claude-plugin .cursor-plugin .codex-plugin .plugin"

@test "all four host manifest versions are identical" {
  local ref dir v mismatched=""
  ref=$(jq -r '.version' "$PLUGIN_ROOT/.claude-plugin/plugin.json")
  [ -n "$ref" ] && [ "$ref" != "null" ] || fail "no version in .claude-plugin/plugin.json"

  for dir in $MANIFESTS; do
    v=$(jq -r '.version' "$PLUGIN_ROOT/$dir/plugin.json")
    [ "$v" = "$ref" ] || mismatched="$mismatched $dir=$v"
  done
  [ -z "$mismatched" ] \
    || fail "manifest versions diverged from .claude-plugin=$ref:$mismatched"
}

@test "the shared manifest version is plain semver" {
  local v
  v=$(jq -r '.version' "$PLUGIN_ROOT/.claude-plugin/plugin.json")
  printf '%s' "$v" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$' \
    || fail "manifest version '$v' is not plain x.y.z semver"
}
