#!/usr/bin/env bats
# Integration smoke checks for Codex plugin packaging + discovery.
#
# Regression coverage for issue #2 ("Codex marketplace install does not
# discover Archcore"). The marketplace catalog lives at the repo root and
# points `source.path` at `./plugins/archcore`. The first three tests exercise
# the REAL discovery path — marketplace add -> plugin list -> plugin add —
# rather than a symlinked fake. The symlink shortcut (used by the last two
# tests, which probe skill/MCP loading) is exactly what let issue #2 ship
# green: it bypasses marketplace resolution entirely.

setup() {
  load '../helpers/common'
  common_setup

  command -v codex >/dev/null 2>&1 || skip "codex CLI not installed"

  TEST_HOME="$BATS_TEST_TMPDIR/home"
  mkdir -p "$TEST_HOME"
  export TEST_HOME
}

@test "codex marketplace add accepts the repo-root marketplace" {
  run env HOME="$TEST_HOME" codex plugin marketplace add "$REPO_ROOT"
  assert_success
  assert_output --partial 'Added marketplace `archcore-plugins`'

  [ -f "$TEST_HOME/.codex/config.toml" ]
  grep -q '^\[marketplaces.archcore-plugins\]' "$TEST_HOME/.codex/config.toml"
}

@test "codex plugin list discovers archcore from the subdirectory (issue #2 regression)" {
  run env HOME="$TEST_HOME" codex plugin marketplace add "$REPO_ROOT"
  assert_success

  run env HOME="$TEST_HOME" codex plugin list
  assert_success
  # Pre-fix: archcore was absent from `plugin list` because source.path was
  # the marketplace root ("./"), which Codex does not scan for plugins.
  assert_output --partial 'archcore@archcore-plugins'
  # And it must resolve to the dedicated subdirectory, never the repo root.
  assert_output --partial 'plugins/archcore'
}

@test "codex plugin add archcore succeeds (issue #2 regression)" {
  run env HOME="$TEST_HOME" codex plugin marketplace add "$REPO_ROOT"
  assert_success

  run env HOME="$TEST_HOME" codex plugin add archcore@archcore-plugins
  # Pre-fix this failed with: plugin `archcore` was not found in marketplace.
  assert_success
  refute_output --partial 'was not found'
  assert_output --partial 'Added plugin `archcore`'
}

@test "codex debug prompt-input loads Archcore skills when plugin is enabled" {
  local installed_root="$TEST_HOME/.codex/plugins/cache/archcore-plugins/archcore/LOCAL"
  mkdir -p "$installed_root"
  ln -s "$PLUGIN_ROOT/.codex-plugin" "$installed_root/.codex-plugin"
  ln -s "$PLUGIN_ROOT/skills" "$installed_root/skills"
  ln -s "$PLUGIN_ROOT/agents" "$installed_root/agents"
  ln -s "$PLUGIN_ROOT/hooks" "$installed_root/hooks"
  ln -s "$PLUGIN_ROOT/bin" "$installed_root/bin"
  ln -s "$PLUGIN_ROOT/.codex.mcp.json" "$installed_root/.codex.mcp.json"

  mkdir -p "$TEST_HOME/.codex"
  {
    printf '[marketplaces.archcore-plugins]\n'
    printf 'last_updated = "2026-05-04T00:00:00Z"\n'
    printf 'source_type = "local"\n'
    printf 'source = "%s"\n' "$REPO_ROOT"
    printf '\n[plugins."archcore@archcore-plugins"]\n'
    printf 'enabled = true\n'
  } >> "$TEST_HOME/.codex/config.toml"

  run env HOME="$TEST_HOME" codex debug prompt-input "use archcore audit"
  assert_success
  assert_output --partial 'archcore:audit'
  assert_output --partial 'skills/audit/SKILL.md'
}

@test "codex mcp list includes plugin-managed Archcore MCP when plugin is enabled" {
  local installed_root="$TEST_HOME/.codex/plugins/cache/archcore-plugins/archcore/LOCAL"
  mkdir -p "$installed_root"
  ln -s "$PLUGIN_ROOT/.codex-plugin" "$installed_root/.codex-plugin"
  ln -s "$PLUGIN_ROOT/.codex.mcp.json" "$installed_root/.codex.mcp.json"

  mkdir -p "$TEST_HOME/.codex"
  {
    printf '[marketplaces.archcore-plugins]\n'
    printf 'last_updated = "2026-05-04T00:00:00Z"\n'
    printf 'source_type = "local"\n'
    printf 'source = "%s"\n' "$REPO_ROOT"
    printf '\n[plugins."archcore@archcore-plugins"]\n'
    printf 'enabled = true\n'
  } >> "$TEST_HOME/.codex/config.toml"

  run env HOME="$TEST_HOME" codex mcp list --json
  assert_success
  assert_output --partial '"archcore"'
  assert_output --partial '"command": "archcore"'
}
