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

@test "session-start uses launcher" {
  grep -q '"\$LAUNCHER"' "$PLUGIN_ROOT/bin/session-start"
}

@test "validate-archcore uses launcher" {
  grep -q '"\$LAUNCHER"' "$PLUGIN_ROOT/bin/validate-archcore"
}

@test ".mcp.json intentionally has no cwd field (Claude Code ignores it)" {
  # Absence is the contract. Claude Code silently ignores cwd field in .mcp.json
  # (anthropics/claude-code#17565). A cwd field would be misleading — it would
  # have no effect. Absence documents this and prevents future confusion.
  run jq -e '.mcpServers.archcore | has("cwd")' < "$PLUGIN_ROOT/.mcp.json"
  assert_failure
}

@test ".mcp.json intentionally has no env.ARCHCORE_CWD (Claude Code cannot pass it)" {
  # ARCHCORE_CWD cannot be injected via plugin manifest for Claude Code — the
  # launcher's Step 0 would have nothing to cd to. The documented fix is to
  # launch `claude` from the project directory. Absence here is intentional.
  run jq -e '.mcpServers.archcore.env.ARCHCORE_CWD' < "$PLUGIN_ROOT/.mcp.json"
  assert_failure
}
