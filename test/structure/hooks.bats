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

# --- codex.hooks.json ---

@test "codex.hooks.json: all commands reference existing files" {
  local missing=""
  while IFS= read -r cmd; do
    local resolved
    resolved="$PLUGIN_ROOT/${cmd#./}"
    if [ ! -f "$resolved" ]; then
      missing="$missing $cmd"
    fi
  done < <(jq -r '.. | .command? // empty' "$PLUGIN_ROOT/hooks/codex.hooks.json")
  [ -z "$missing" ] || fail "Missing files: $missing"
}

@test "codex.hooks.json: all referenced scripts are executable" {
  local not_exec=""
  while IFS= read -r cmd; do
    local resolved
    resolved="$PLUGIN_ROOT/${cmd#./}"
    if [ -f "$resolved" ] && [ ! -x "$resolved" ]; then
      not_exec="$not_exec $cmd"
    fi
  done < <(jq -r '.. | .command? // empty' "$PLUGIN_ROOT/hooks/codex.hooks.json")
  [ -z "$not_exec" ] || fail "Not executable: $not_exec"
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

# --- Phase 2.1 anti-regression invariants ---

@test "hooks.json: PostToolUse has no Write|Edit matcher (dead hook removed)" {
  local matchers
  matchers=$(jq -r '.hooks.PostToolUse[].matcher' "$PLUGIN_ROOT/hooks/hooks.json")
  if echo "$matchers" | grep -qE '(^|\|)Write(\||$)'; then
    fail "PostToolUse Write|Edit matcher was re-introduced — it is dead (PreToolUse already blocks direct writes to .archcore/)."
  fi
}

@test "hooks.json: PostToolUse matchers all target mcp__archcore__*" {
  local matchers
  matchers=$(jq -r '.hooks.PostToolUse[].matcher' "$PLUGIN_ROOT/hooks/hooks.json")
  while IFS= read -r m; do
    [ -z "$m" ] && continue
    echo "$m" | grep -qE '^mcp__archcore__' || fail "Unexpected PostToolUse matcher: $m"
  done <<<"$matchers"
}

@test "cursor.hooks.json: no postToolUse event (cleaned in Phase 2.1)" {
  local has
  has=$(jq 'has("postToolUse")' "$PLUGIN_ROOT/hooks/cursor.hooks.json" 2>/dev/null || echo "err")
  # The hooks are actually under .hooks (v1 format). Re-query:
  has=$(jq '.hooks | has("postToolUse")' "$PLUGIN_ROOT/hooks/cursor.hooks.json")
  [ "$has" = "false" ] || fail "cursor.hooks.json grew a postToolUse event — Phase 2.1 removed it because PreToolUse + afterMCPExecution cover every case."
}

@test "cursor.hooks.json: event set is exactly sessionStart/preToolUse/afterMCPExecution" {
  local events
  events=$(jq -r '.hooks | keys[]' "$PLUGIN_ROOT/hooks/cursor.hooks.json" | sort | tr '\n' ',')
  [ "$events" = "afterMCPExecution,preToolUse,sessionStart," ] || {
    echo "Actual events: $events"
    fail "cursor.hooks.json event set drifted from the expected {sessionStart, preToolUse, afterMCPExecution}."
  }
}

@test "hooks.json: event set is exactly SessionStart/PreToolUse/PostToolUse" {
  local events
  events=$(jq -r '.hooks | keys[]' "$PLUGIN_ROOT/hooks/hooks.json" | sort | tr '\n' ',')
  [ "$events" = "PostToolUse,PreToolUse,SessionStart," ] || {
    echo "Actual events: $events"
    fail "hooks.json event set drifted from the expected {SessionStart, PreToolUse, PostToolUse}."
  }
}

@test "codex.hooks.json: event set is exactly SessionStart/PreToolUse/PostToolUse" {
  local events
  events=$(jq -r '.hooks | keys[]' "$PLUGIN_ROOT/hooks/codex.hooks.json" | sort | tr '\n' ',')
  [ "$events" = "PostToolUse,PreToolUse,SessionStart," ] || {
    echo "Actual events: $events"
    fail "codex.hooks.json event set drifted from the expected {SessionStart, PreToolUse, PostToolUse}."
  }
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

@test "codex hook config references the same script set as Claude Code" {
  local cc_scripts codex_scripts
  cc_scripts=$(jq -r '.. | .command? // empty' "$PLUGIN_ROOT/hooks/hooks.json" | sed 's|${CLAUDE_PLUGIN_ROOT}/||' | sort -u)
  codex_scripts=$(jq -r '.. | .command? // empty' "$PLUGIN_ROOT/hooks/codex.hooks.json" | sed 's|^\./||' | sort -u)
  [ "$cc_scripts" = "$codex_scripts" ] || {
    echo "Claude Code scripts: $cc_scripts"
    echo "Codex scripts: $codex_scripts"
    fail "Script sets differ between Claude Code and Codex"
  }
}
