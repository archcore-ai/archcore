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
  run jq . "$REPO_ROOT/.claude-plugin/marketplace.json"
  assert_success
}

@test "cursor-plugin/plugin.json is valid JSON" {
  run jq . "$PLUGIN_ROOT/.cursor-plugin/plugin.json"
  assert_success
}

@test "cursor-plugin/marketplace.json is valid JSON" {
  run jq . "$REPO_ROOT/.cursor-plugin/marketplace.json"
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
  # NOTE: Cursor plugin manifests intentionally do NOT carry an `mcpServers`
  # field, and we do not ship an `mcp.json` (or `cursor.mcp.json`) at the
  # plugin root. Cursor 2.5+ auto-detects `mcp.json` at the plugin root and
  # registers it under "Plugin MCP Servers", but it spawns that MCP from
  # the plugin install directory (not the workspace), and its MCP stdio
  # schema has no `cwd` field — so a plugin-shipped MCP would serve the
  # plugin install dir's `.archcore/` instead of the user's. To avoid this,
  # we publish `docs/cursor.mcp.example.json` as a copy-into-user-config
  # template that passes `--project ${workspaceFolder}` in `args`, and
  # never expose Cursor's plugin-MCP surface. See
  # cursor-mcp-architecture.adr.md for the full rationale.
  run jq -e '.skills and .agents and .hooks' "$PLUGIN_ROOT/.cursor-plugin/plugin.json"
  assert_success
}

@test "marketplace.json has plugins array" {
  run jq -e '.plugins | length > 0' "$REPO_ROOT/.claude-plugin/marketplace.json"
  assert_success
}

@test "hooks.json has PascalCase event keys" {
  run jq -e '.hooks | keys | map(select(. == "SessionStart" or . == "PreToolUse" or . == "PostToolUse")) | length == 3' "$PLUGIN_ROOT/hooks/hooks.json"
  assert_success
}

@test "cursor.hooks.json has camelCase event keys" {
  run jq -e '.hooks | has("sessionStart") and has("preToolUse")' "$PLUGIN_ROOT/hooks/cursor.hooks.json"
  assert_success
}

# --- Cross-reference consistency ---

# Every manifest the plugin ships. Claude Code is the reference: it is the
# host bump-plugin-version and the marketplace catalogs are keyed to.
host_manifests() {
  cat <<'EOF'
cursor|.cursor-plugin/plugin.json
codex|.codex-plugin/plugin.json
copilot|.plugin/plugin.json
EOF
}

@test "plugin.json name/description/version match across ALL four hosts" {
  # Was Claude vs Cursor only, which left Codex and Copilot free to drift —
  # and drift here is not cosmetic: these four manifests are one plugin seen
  # from four hosts, so a stale version ships an install that reports the
  # wrong release, and a stale description ships a different pitch to those
  # users. verify-plugin-integrity §4 compares all four; this makes it a gate
  # rather than a checklist item.
  local ref="$PLUGIN_ROOT/.claude-plugin/plugin.json"
  local host manifest field ref_val val
  while IFS='|' read -r host manifest; do
    [ -n "$host" ] || continue
    for field in name description version; do
      ref_val=$(jq -r ".$field" "$ref")
      val=$(jq -r ".$field" "$PLUGIN_ROOT/$manifest")
      [ "$val" = "$ref_val" ] \
        || fail "$manifest: $field drift — claude='$ref_val' $host='$val'"
    done
  done < <(host_manifests)
}

@test "every shipped plugin.json is enrolled in the metadata parity table" {
  # A fifth host's manifest must not be able to ship unchecked.
  local dir base missing=""
  for dir in "$PLUGIN_ROOT"/.*-plugin "$PLUGIN_ROOT"/.plugin; do
    [ -f "$dir/plugin.json" ] || continue
    base="$(basename "$dir")/plugin.json"
    [ "$base" = ".claude-plugin/plugin.json" ] && continue  # the reference
    host_manifests | grep -q "|$base$" || missing="$missing $base"
  done
  [ -z "$missing" ] || fail "manifests missing from host_manifests():$missing"
}

@test "marketplace.json plugin metadata matches across Claude and Cursor" {
  local cc="$REPO_ROOT/.claude-plugin/marketplace.json"
  local cursor="$REPO_ROOT/.cursor-plugin/marketplace.json"
  for field in name version description; do
    local cc_val cursor_val
    cc_val=$(jq -r ".plugins[0].$field" "$cc")
    cursor_val=$(jq -r ".plugins[0].$field" "$cursor")
    [ "$cc_val" = "$cursor_val" ] || fail "marketplace.json $field drift: claude='$cc_val' cursor='$cursor_val'"
  done
}

