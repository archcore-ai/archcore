#!/usr/bin/env bats
# Structure tests: validate bin scripts

setup() {
  load '../helpers/common'
  common_setup
}

@test "all bin scripts are executable" {
  local not_exec=""
  for f in "$PLUGIN_ROOT"/bin/check-* "$PLUGIN_ROOT"/bin/validate-* "$PLUGIN_ROOT"/bin/session-start "$PLUGIN_ROOT"/bin/archcore; do
    [ -f "$f" ] || continue
    if [ ! -x "$f" ]; then
      not_exec="$not_exec $(basename "$f")"
    fi
  done
  [ -z "$not_exec" ] || fail "Not executable: $not_exec"
}

@test "all bin scripts have #!/bin/sh shebang" {
  local bad_shebang=""
  for f in "$PLUGIN_ROOT"/bin/check-* "$PLUGIN_ROOT"/bin/validate-* "$PLUGIN_ROOT"/bin/session-start "$PLUGIN_ROOT"/bin/archcore; do
    [ -f "$f" ] || continue
    local first_line
    first_line=$(head -1 "$f")
    if [ "$first_line" != "#!/bin/sh" ]; then
      bad_shebang="$bad_shebang $(basename "$f")"
    fi
  done
  [ -z "$bad_shebang" ] || fail "Bad shebang: $bad_shebang"
}

@test "normalize-stdin.sh exists and has shebang" {
  [ -f "$PLUGIN_ROOT/bin/lib/normalize-stdin.sh" ]
  local first_line
  first_line=$(head -1 "$PLUGIN_ROOT/bin/lib/normalize-stdin.sh")
  [ "$first_line" = "#!/bin/sh" ]
}

@test "check-archcore-write sources normalize-stdin.sh" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/check-archcore-write"
}

@test "validate-archcore sources normalize-stdin.sh" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/validate-archcore"
}

@test "check-cascade sources normalize-stdin.sh" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/check-cascade"
}

@test "session-start sources normalize-stdin.sh" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/session-start"
}

@test "check-staleness does NOT source normalize-stdin.sh" {
  ! grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/check-staleness"
}

@test "Windows launcher files exist" {
  [ -f "$PLUGIN_ROOT/bin/archcore.ps1" ]
  [ -f "$PLUGIN_ROOT/bin/archcore.cmd" ]
}

@test "archcore.ps1 is ASCII-only (Windows PowerShell 5.1 reads without BOM as ANSI)" {
  # Non-ASCII chars (em-dash, smart quotes) get mis-decoded by PS 5.1 as
  # Windows-1252, and can terminate string literals early (U+201D from the
  # em-dash trailing byte 0x94), producing parser errors at MCP startup.
  # Portable across BSD/GNU grep: match any byte outside printable ASCII + tab.
  run env LC_ALL=C grep -n '[^ -~	]' "$PLUGIN_ROOT/bin/archcore.ps1"
  [ "$status" -ne 0 ] || fail "archcore.ps1 contains non-ASCII bytes:
$output"
}

@test "CLI_VERSION file exists and matches semver" {
  [ -f "$PLUGIN_ROOT/bin/CLI_VERSION" ]
  local version
  version=$(tr -d '[:space:]' < "$PLUGIN_ROOT/bin/CLI_VERSION")
  [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "CLI_VERSION '$version' is not semver"
}

@test ".mcp.json ships at plugin root" {
  [ -f "$PLUGIN_ROOT/.mcp.json" ]
  grep -q '"archcore"' "$PLUGIN_ROOT/.mcp.json"
  grep -q 'CLAUDE_PLUGIN_ROOT' "$PLUGIN_ROOT/.mcp.json"
}

@test ".mcp.json command and args are correct" {
  local file="$PLUGIN_ROOT/.mcp.json"
  [ "$(jq -r '.mcpServers.archcore.command' < "$file")" = "\${CLAUDE_PLUGIN_ROOT}/bin/archcore" ]
  [ "$(jq -r '.mcpServers.archcore.args[0]' < "$file")" = "mcp" ]
  [ "$(jq -r '.mcpServers.archcore.args | length' < "$file")" = "1" ]
}

# Regression guard for commit caaa725 (".mcp.json hotfix"):
# someone changed "${CLAUDE_PLUGIN_ROOT}/bin/archcore" to "./bin/archcore",
# thinking Claude Code resolves the relative path against the .mcp.json file
# location. It does not — Claude Code resolves MCP commands against the host
# process CWD (the user's project), so the relative path failed in production
# with: posix_spawn './bin/archcore': ENOENT.
#
# Mirrors the hooks.bats resolve+verify pattern: substitute the env var, then
# check the binary exists and is executable. Any future hotfix that breaks the
# substitution chain (relative path, wrong env var, typo, missing binary) trips
# this test before shipping.
@test ".mcp.json: resolved command path exists and is executable" {
  local file="$PLUGIN_ROOT/.mcp.json"
  local cmd
  cmd=$(jq -r '.mcpServers.archcore.command' < "$file")
  local resolved
  resolved=$(echo "$cmd" | sed "s|\${CLAUDE_PLUGIN_ROOT}|${PLUGIN_ROOT}|g")
  # The resolved path MUST be absolute. Claude Code spawns MCP servers from the
  # user's project CWD, not the plugin dir, so any relative path here is broken
  # in production even if it accidentally works during local bats runs.
  case "$resolved" in
    /*) ;;
    *) fail ".mcp.json command does not resolve to an absolute path (got: '$resolved' from raw: '$cmd'). Use \${CLAUDE_PLUGIN_ROOT}/bin/<binary>." ;;
  esac
  [ -f "$resolved" ] || fail ".mcp.json command resolves to non-existent file: $resolved (raw: $cmd)"
  [ -x "$resolved" ] || fail ".mcp.json command resolves to non-executable file: $resolved"
}

@test ".mcp.json: command MUST NOT use a relative path (regression guard)" {
  local file="$PLUGIN_ROOT/.mcp.json"
  local cmd
  cmd=$(jq -r '.mcpServers.archcore.command' < "$file")
  case "$cmd" in
    ./*|../*)
      fail ".mcp.json command must not be relative ('$cmd'). Claude Code resolves MCP commands against process CWD, not the .mcp.json file location — relative paths break in production. Use \${CLAUDE_PLUGIN_ROOT}/bin/<binary>. See commit caaa725 for the regression this guards against."
      ;;
  esac
  if ! echo "$cmd" | grep -q '\${CLAUDE_PLUGIN_ROOT}'; then
    fail ".mcp.json command must reference \${CLAUDE_PLUGIN_ROOT} (got: '$cmd'). Claude Code substitutes this env var at plugin load time; without it the launcher cannot be found from outside the plugin directory."
  fi
}

@test "session-start uses launcher" {
  grep -q '"\$LAUNCHER"' "$PLUGIN_ROOT/bin/session-start"
}

@test "validate-archcore uses launcher" {
  grep -q '"\$LAUNCHER"' "$PLUGIN_ROOT/bin/validate-archcore"
}
