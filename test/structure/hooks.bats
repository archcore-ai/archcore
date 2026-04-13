#!/usr/bin/env bats
# Structure tests: validate hook configurations

setup() {
  load '../helpers/common'
  common_setup
}

# --- hooks.json (Claude Code) ---

@test "hooks.json: all commands reference existing files" {
  local missing=""
  while IFS= read -r cmd; do
    local resolved
    resolved=$(echo "$cmd" | sed "s|\${CLAUDE_PLUGIN_ROOT}|${PLUGIN_ROOT}|g")
    if [ ! -f "$resolved" ]; then
      missing="$missing $cmd"
    fi
  done < <(jq -r '.. | .command? // empty' "$PLUGIN_ROOT/hooks/hooks.json")
  [ -z "$missing" ] || fail "Missing files: $missing"
}

@test "hooks.json: all referenced scripts are executable" {
  local not_exec=""
  while IFS= read -r cmd; do
    local resolved
    resolved=$(echo "$cmd" | sed "s|\${CLAUDE_PLUGIN_ROOT}|${PLUGIN_ROOT}|g")
    if [ -f "$resolved" ] && [ ! -x "$resolved" ]; then
      not_exec="$not_exec $cmd"
    fi
  done < <(jq -r '.. | .command? // empty' "$PLUGIN_ROOT/hooks/hooks.json")
  [ -z "$not_exec" ] || fail "Not executable: $not_exec"
}

# --- cursor.hooks.json ---

@test "cursor.hooks.json: all commands reference existing files" {
  local missing=""
  while IFS= read -r cmd; do
    local resolved
    resolved=$(echo "$cmd" | sed "s|\${CURSOR_PLUGIN_ROOT}|${PLUGIN_ROOT}|g")
    if [ ! -f "$resolved" ]; then
      missing="$missing $cmd"
    fi
  done < <(jq -r '.. | .command? // empty' "$PLUGIN_ROOT/hooks/cursor.hooks.json")
  [ -z "$missing" ] || fail "Missing files: $missing"
}

@test "cursor.hooks.json: all referenced scripts are executable" {
  local not_exec=""
  while IFS= read -r cmd; do
    local resolved
    resolved=$(echo "$cmd" | sed "s|\${CURSOR_PLUGIN_ROOT}|${PLUGIN_ROOT}|g")
    if [ -f "$resolved" ] && [ ! -x "$resolved" ]; then
      not_exec="$not_exec $cmd"
    fi
  done < <(jq -r '.. | .command? // empty' "$PLUGIN_ROOT/hooks/cursor.hooks.json")
  [ -z "$not_exec" ] || fail "Not executable: $not_exec"
}

# --- Consistency ---

@test "both hook configs reference the same set of scripts" {
  local cc_scripts cursor_scripts
  cc_scripts=$(jq -r '.. | .command? // empty' "$PLUGIN_ROOT/hooks/hooks.json" | sed 's|${CLAUDE_PLUGIN_ROOT}||' | sort -u)
  cursor_scripts=$(jq -r '.. | .command? // empty' "$PLUGIN_ROOT/hooks/cursor.hooks.json" | sed 's|${CURSOR_PLUGIN_ROOT}||' | sort -u)
  [ "$cc_scripts" = "$cursor_scripts" ] || {
    echo "Claude Code scripts: $cc_scripts"
    echo "Cursor scripts: $cursor_scripts"
    fail "Script sets differ between hosts"
  }
}
