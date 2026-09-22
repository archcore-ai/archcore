#!/usr/bin/env bats
# Structure tests: validate bin scripts

setup() {
  load '../helpers/common'
  common_setup
}

@test "all bin scripts are executable" {
  local not_exec=""
  for f in "$PLUGIN_ROOT"/bin/session-start "$PLUGIN_ROOT"/bin/pre-tool-use "$PLUGIN_ROOT"/bin/post-tool-use "$PLUGIN_ROOT"/bin/detect-host "$PLUGIN_ROOT"/bin/cli-gte; do
    [ -f "$f" ] || fail "Missing bin script: $f"
    if [ ! -x "$f" ]; then
      not_exec="$not_exec $(basename "$f")"
    fi
  done
  [ -z "$not_exec" ] || fail "Not executable: $not_exec"
}

@test "all bin scripts have #!/bin/sh shebang" {
  local bad_shebang=""
  for f in "$PLUGIN_ROOT"/bin/session-start "$PLUGIN_ROOT"/bin/pre-tool-use "$PLUGIN_ROOT"/bin/post-tool-use "$PLUGIN_ROOT"/bin/detect-host "$PLUGIN_ROOT"/bin/cli-gte; do
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

@test "pre-tool-use sources normalize-stdin.sh and the plugin-cache guard" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/pre-tool-use"
  grep -q 'plugin-cache-guard.sh' "$PLUGIN_ROOT/bin/pre-tool-use"
}

@test "post-tool-use sources normalize-stdin.sh and the plugin-cache guard" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/post-tool-use"
  grep -q 'plugin-cache-guard.sh' "$PLUGIN_ROOT/bin/post-tool-use"
}

@test "session-start sources normalize-stdin.sh" {
  grep -q 'normalize-stdin.sh' "$PLUGIN_ROOT/bin/session-start"
}

@test "launchers gate on cli-gte 0.7.0 — the release that added the pre/post-tool-use leaves" {
  grep -qF 'cli-gte" 0.7.0' "$PLUGIN_ROOT/bin/pre-tool-use"
  grep -qF 'cli-gte" 0.7.0' "$PLUGIN_ROOT/bin/post-tool-use"
}


# The Claude MCP config (.claude.mcp.json) and the manifest keys that keep it
# reachable from Claude Code without leaking into Copilot are asserted in
# test/structure/plugin-mcp-isolation.bats, which owns the whole three-part
# contract and the host measurements behind it.

@test ".codex.mcp.json command and args use direct server map" {
  local file="$PLUGIN_ROOT/.codex.mcp.json"
  [ "$(jq -r '.archcore.command' < "$file")" = "archcore" ]
  [ "$(jq -r '.archcore.args[0]' < "$file")" = "mcp" ]
  [ "$(jq -r '.archcore.args | length' < "$file")" = "1" ]
  [ "$(jq -r 'has("mcpServers") or has("mcp_servers")' < "$file")" = "false" ]
}
