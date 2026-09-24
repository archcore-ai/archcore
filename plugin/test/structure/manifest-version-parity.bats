#!/usr/bin/env bats
# The release tag is the only plugin version source. scripts/export-plugin.sh
# writes the tag's version into the four published manifests; on dev all four
# carry the placeholder 0.0.0, so no source manifest can drift from a tag or be
# mistaken for a release. A hand bump here is a regression, not a release step.

setup() {
  load '../helpers/common'
}

MANIFESTS=".claude-plugin .cursor-plugin .codex-plugin .plugin"

@test "all four source manifests carry the unreleased placeholder 0.0.0" {
  local dir v wrong=""
  for dir in $MANIFESTS; do
    v=$(jq -r '.version' "$PLUGIN_ROOT/$dir/plugin.json")
    [ "$v" = "0.0.0" ] || wrong="$wrong $dir=$v"
  done
  [ -z "$wrong" ] \
    || fail "source manifests must stay at 0.0.0; the release tag sets the version:$wrong"
}
