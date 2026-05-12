#!/usr/bin/env bats
# Structure tests: validate the cursor.mcp.json template shipped at the plugin
# root. This file is NOT auto-loaded by Cursor — it's a canonical snippet
# users copy into their own .cursor/mcp.json (or ~/.cursor/mcp.json) when
# wiring the archcore MCP server. The contract pinned here protects the two
# fields that prevent cross-project content leakage:
#   - cwd:                "${workspaceFolder}"
#   - env.ARCHCORE_CWD:   "${workspaceFolder}"
# See: .archcore/plugin/cwd-guard-for-cursor-and-claude.idea.md

setup() {
  load '../helpers/common'
  common_setup
}

@test "cursor.mcp.json exists at the plugin root" {
  [ -f "$PLUGIN_ROOT/cursor.mcp.json" ]
}

@test "cursor.mcp.json is valid JSON" {
  jq . < "$PLUGIN_ROOT/cursor.mcp.json" > /dev/null
}

@test "cursor.mcp.json declares an archcore server under mcpServers" {
  local file="$PLUGIN_ROOT/cursor.mcp.json"
  jq -e '.mcpServers.archcore' < "$file" > /dev/null
}

@test "cursor.mcp.json invokes archcore mcp" {
  local file="$PLUGIN_ROOT/cursor.mcp.json"
  [ "$(jq -r '.mcpServers.archcore.command' < "$file")" = "archcore" ] \
    || fail "command must be 'archcore' (resolved via PATH); set an absolute path in user config when not on PATH"
  [ "$(jq -r '.mcpServers.archcore.args[0]' < "$file")" = "mcp" ] \
    || fail "args[0] must be 'mcp'"
}

@test "cursor.mcp.json pins cwd to \"\${workspaceFolder}\"" {
  # Cursor expands ${workspaceFolder} per-server-spawn — pinning it here is
  # what stops a global ~/.cursor/mcp.json entry from leaking the first
  # workspace's .archcore/ into every subsequently opened project.
  local file="$PLUGIN_ROOT/cursor.mcp.json"
  local cwd
  cwd=$(jq -r '.mcpServers.archcore.cwd // empty' < "$file")
  [ "$cwd" = "\${workspaceFolder}" ] \
    || fail "cwd must be \"\${workspaceFolder}\", got: '$cwd'"
}

@test "cursor.mcp.json passes ARCHCORE_CWD env (belt-and-braces for hosts that ignore cwd)" {
  # Claude Code's `cwd` field in .mcp.json is silently ignored
  # (anthropics/claude-code#17565). The bundled bin/archcore launcher honors
  # ARCHCORE_CWD in Step 0 — so passing it alongside cwd is a free safety
  # net if a user copies this template to ~/.claude/.mcp.json or any future
  # host with similar behavior.
  local file="$PLUGIN_ROOT/cursor.mcp.json"
  local env_cwd
  env_cwd=$(jq -r '.mcpServers.archcore.env.ARCHCORE_CWD // empty' < "$file")
  [ "$env_cwd" = "\${workspaceFolder}" ] \
    || fail "env.ARCHCORE_CWD must be \"\${workspaceFolder}\", got: '$env_cwd'"
}

@test "cursor.mcp.json has only one server defined (no leftover entries)" {
  local file="$PLUGIN_ROOT/cursor.mcp.json"
  local count
  count=$(jq -r '.mcpServers | keys | length' < "$file")
  [ "$count" = "1" ] || fail "expected exactly 1 server (archcore), found $count"
}

@test "README references cursor.mcp.json so users find the template" {
  grep -q "cursor.mcp.json" "$PLUGIN_ROOT/README.md" \
    || fail "README.md must reference cursor.mcp.json so users discover the template"
}
