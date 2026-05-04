#!/usr/bin/env bats
# Integration smoke checks for Codex plugin packaging.

setup() {
  load '../helpers/common'
  common_setup

  command -v codex >/dev/null 2>&1 || skip "codex CLI not installed"

  TEST_HOME="$BATS_TEST_TMPDIR/home"
  mkdir -p "$TEST_HOME"
  export TEST_HOME
}

@test "codex marketplace add accepts repo root marketplace" {
  run env HOME="$TEST_HOME" codex plugin marketplace add "$PLUGIN_ROOT"
  assert_success
  assert_output --partial 'Added marketplace `archcore-plugins`'

  [ -f "$TEST_HOME/.codex/config.toml" ]
  grep -q '^\[marketplaces.archcore-plugins\]' "$TEST_HOME/.codex/config.toml"
  grep -q "source = \"$PLUGIN_ROOT\"" "$TEST_HOME/.codex/config.toml"
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
    printf 'source = "%s"\n' "$PLUGIN_ROOT"
    printf '\n[plugins."archcore@archcore-plugins"]\n'
    printf 'enabled = true\n'
  } >> "$TEST_HOME/.codex/config.toml"

  run env HOME="$TEST_HOME" codex debug prompt-input "use archcore review"
  assert_success
  assert_output --partial 'archcore:review'
  assert_output --partial 'skills/review/SKILL.md'
}

@test "codex mcp list includes plugin-managed Archcore MCP when plugin is enabled" {
  local installed_root="$TEST_HOME/.codex/plugins/cache/archcore-plugins/archcore/LOCAL"
  mkdir -p "$installed_root"
  ln -s "$PLUGIN_ROOT/.codex-plugin" "$installed_root/.codex-plugin"
  ln -s "$PLUGIN_ROOT/.codex.mcp.json" "$installed_root/.codex.mcp.json"
  ln -s "$PLUGIN_ROOT/bin" "$installed_root/bin"

  mkdir -p "$TEST_HOME/.codex"
  {
    printf '[marketplaces.archcore-plugins]\n'
    printf 'last_updated = "2026-05-04T00:00:00Z"\n'
    printf 'source_type = "local"\n'
    printf 'source = "%s"\n' "$PLUGIN_ROOT"
    printf '\n[plugins."archcore@archcore-plugins"]\n'
    printf 'enabled = true\n'
  } >> "$TEST_HOME/.codex/config.toml"

  run env HOME="$TEST_HOME" codex mcp list --json
  assert_success
  assert_output --partial '"archcore"'
  assert_output --partial '"command": "./bin/archcore"'
}
