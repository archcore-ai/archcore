#!/usr/bin/env bats

setup() {
  load '../helpers/common'
  common_setup
  EXPORT="$WORKSPACE_ROOT/scripts/export-plugin.sh"
  OUTPUT="$BATS_TEST_TMPDIR/published"
}

@test "export preserves every marketplace's runtime subdirectory" {
  run "$EXPORT" "$OUTPUT"
  assert_success
  local catalog source manifest
  for catalog in .agents/plugins/marketplace.json .claude-plugin/marketplace.json .cursor-plugin/marketplace.json; do
    source=$(jq -r '.plugins[0].source | if type == "object" then .path else . end' "$OUTPUT/$catalog")
    [ "$source" = './plugins/archcore' ] || fail "$catalog changed its public runtime path"
  done
  for manifest in .claude-plugin .cursor-plugin .codex-plugin .plugin; do
    [ -f "$OUTPUT/plugins/archcore/$manifest/plugin.json" ] || fail "missing $manifest"
  done
}

@test "export preserves runtime executables, branding, MCP configs and public docs" {
  run "$EXPORT" "$OUTPUT"
  assert_success
  [ -x "$OUTPUT/plugins/archcore/bin/session-start" ]
  [ -x "$OUTPUT/plugins/archcore/bin/cli-gte" ]
  local path
  for path in assets/icon.png assets/logo.png .claude.mcp.json .codex.mcp.json hooks/copilot.hooks.json; do
    [ -f "$OUTPUT/plugins/archcore/$path" ] || fail "missing $path"
  done
  for path in docs/TERMS.md docs/cursor.mcp.example.json README.md LICENSE NOTICE; do
    [ -f "$OUTPUT/$path" ] || fail "missing $path"
  done
}

@test "export excludes both component source trees and development context" {
  run "$EXPORT" "$OUTPUT"
  assert_success
  [ ! -e "$OUTPUT/cli" ]
  [ ! -e "$OUTPUT/plugin" ]
  [ ! -e "$OUTPUT/.github" ]
  [ ! -e "$OUTPUT/.gitmodules" ]
  local forbidden
  forbidden=$(find "$OUTPUT" \( -name .archcore -o -name AGENTS.md -o -name CLAUDE.md -o -name .mcp.json -o -name test \) -print)
  [ -z "$forbidden" ] || fail "development files leaked: $forbidden"
}

@test "export refuses to overwrite an existing destination" {
  mkdir -p "$OUTPUT"
  printf 'keep me\n' > "$OUTPUT/keep"
  run "$EXPORT" "$OUTPUT"
  assert_failure
  assert_output --partial 'must be empty'
  [ "$(cat "$OUTPUT/keep")" = 'keep me' ]
}

@test "export rejects nested Archcore context inside otherwise shipped runtime" {
  local fixture="$BATS_TEST_TMPDIR/fixture"
  mkdir -p "$fixture/scripts" "$fixture/plugin"
  cp "$EXPORT" "$fixture/scripts/export-plugin.sh"
  cp "$WORKSPACE_ROOT/LICENSE" "$WORKSPACE_ROOT/NOTICE" "$fixture/"
  local path
  for path in .agents .claude-plugin .cursor-plugin plugins docs README.md demo.gif 3-commands.png; do
    cp -R "$REPO_ROOT/$path" "$fixture/plugin/"
  done
  mkdir -p "$fixture/plugin/plugins/archcore/nested/.archcore"
  printf 'internal context\n' > "$fixture/plugin/plugins/archcore/nested/.archcore/leak.md"
  run "$fixture/scripts/export-plugin.sh" "$OUTPUT"
  assert_failure
  assert_output --partial 'Development files found'
  assert_output --partial 'nested/.archcore'
}

@test "export refuses a destination symlink without writing through it" {
  local target="$BATS_TEST_TMPDIR/target"
  mkdir -p "$target"
  ln -s "$target" "$OUTPUT"
  run "$EXPORT" "$OUTPUT"
  assert_failure
  assert_output --partial 'must not be a symlink'
  [ -z "$(ls -A "$target")" ]
}
