#!/usr/bin/env bats
# Structure tests: validate all JSON config files

setup() {
  load '../helpers/common'
  common_setup
}

# --- JSON validity ---

@test "claude-plugin/plugin.json is valid JSON" {
  run jq . "$PLUGIN_ROOT/.claude-plugin/plugin.json"
  assert_success
}

@test "claude-plugin/marketplace.json is valid JSON" {
  run jq . "$PLUGIN_ROOT/.claude-plugin/marketplace.json"
  assert_success
}

@test "cursor-plugin/plugin.json is valid JSON" {
  run jq . "$PLUGIN_ROOT/.cursor-plugin/plugin.json"
  assert_success
}

@test "cursor-plugin/marketplace.json is valid JSON" {
  run jq . "$PLUGIN_ROOT/.cursor-plugin/marketplace.json"
  assert_success
}

@test "hooks.json is valid JSON" {
  run jq . "$PLUGIN_ROOT/hooks/hooks.json"
  assert_success
}

@test "cursor.hooks.json is valid JSON" {
  run jq . "$PLUGIN_ROOT/hooks/cursor.hooks.json"
  assert_success
}

# --- Required fields ---

@test "claude plugin.json has name and version" {
  run jq -e '.name and .version' "$PLUGIN_ROOT/.claude-plugin/plugin.json"
  assert_success
}

@test "cursor plugin.json has name and version" {
  run jq -e '.name and .version' "$PLUGIN_ROOT/.cursor-plugin/plugin.json"
  assert_success
}

@test "cursor plugin.json has skills, agents, hooks paths" {
  run jq -e '.skills and .agents and .hooks' "$PLUGIN_ROOT/.cursor-plugin/plugin.json"
  assert_success
}

@test "marketplace.json has plugins array" {
  run jq -e '.plugins | length > 0' "$PLUGIN_ROOT/.claude-plugin/marketplace.json"
  assert_success
}

@test "hooks.json has PascalCase event keys" {
  run jq -e '.hooks | keys | map(select(. == "SessionStart" or . == "PreToolUse" or . == "PostToolUse")) | length == 3' "$PLUGIN_ROOT/hooks/hooks.json"
  assert_success
}

@test "cursor.hooks.json has camelCase event keys" {
  run jq -e '.hooks | has("sessionStart", "preToolUse")' "$PLUGIN_ROOT/hooks/cursor.hooks.json"
  assert_success
}

# --- Cross-reference consistency ---

@test "plugin.json name matches across hosts" {
  local cc_name cursor_name
  cc_name=$(jq -r '.name' "$PLUGIN_ROOT/.claude-plugin/plugin.json")
  cursor_name=$(jq -r '.name' "$PLUGIN_ROOT/.cursor-plugin/plugin.json")
  [ "$cc_name" = "$cursor_name" ]
}

@test "plugin.json version matches across hosts" {
  local cc_ver cursor_ver
  cc_ver=$(jq -r '.version' "$PLUGIN_ROOT/.claude-plugin/plugin.json")
  cursor_ver=$(jq -r '.version' "$PLUGIN_ROOT/.cursor-plugin/plugin.json")
  [ "$cc_ver" = "$cursor_ver" ]
}

@test "marketplace.json plugin metadata matches across Claude and Cursor" {
  local cc="$PLUGIN_ROOT/.claude-plugin/marketplace.json"
  local cursor="$PLUGIN_ROOT/.cursor-plugin/marketplace.json"
  for field in name version description; do
    local cc_val cursor_val
    cc_val=$(jq -r ".plugins[0].$field" "$cc")
    cursor_val=$(jq -r ".plugins[0].$field" "$cursor")
    [ "$cc_val" = "$cursor_val" ] || fail "marketplace.json $field drift: claude='$cc_val' cursor='$cursor_val'"
  done
}

