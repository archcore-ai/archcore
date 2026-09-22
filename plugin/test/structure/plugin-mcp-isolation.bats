#!/usr/bin/env bats
# Structure tests: the plugin's MCP declaration must reach Claude Code and
# ONLY Claude Code.
#
# The hazard is a name collision, not a missing file. `archcore init --agent
# copilot` writes a project-level server under the key "archcore"; a plugin
# that also ships a server under "archcore" replaces it, because Copilot
# merges user -> workspace -> plugins with last-wins. The surviving entry runs
# with cwd=${PLUGIN_ROOT}, which archcore >= v0.6.7 refuses by design
# (plugin-cache guard) — so the Copilot user ends up with no document tools at
# all, while bin/session-start's wiring advisory still finds "archcore" in
# .mcp.json and calls the project wired.
#
# Measured on Copilot CLI 1.0.76 and Claude Code 2.1.220 (2026-08-03), each
# arm a fresh COPILOT_HOME with the plugin registered in config.json and a
# workspace .mcp.json whose archcore entry used a sentinel command:
#
#   plugin layout                                        which server survived
#   ---------------------------------------------------  ---------------------
#   (no plugin installed)                                workspace   [control]
#   .mcp.json at plugin root                             plugin
#   renamed, no manifest key anywhere                    workspace
#   renamed + mcpServers in .claude-plugin/plugin.json   plugin
#   .mcp.json kept + empty mcpServers in .plugin/        plugin
#   renamed + BOTH manifest keys  (this layout)          workspace
#
# Two independent host behaviours produce that table, so the fix needs both
# halves and neither is redundant:
#
#   1. Copilot auto-discovers `.mcp.json` — and, where that is absent in the
#      same directory, `.github/mcp.json` — in the plugin root, regardless of
#      what any manifest says. No manifest key switches it off: declaring an
#      empty mcpServers while .mcp.json is still on disk leaves the plugin
#      server loaded. The filename is the only lever.
#   2. Copilot READS .claude-plugin/plugin.json when its own .plugin manifest
#      does not declare mcpServers. So the key that re-arms Claude Code after
#      the rename re-arms Copilot too, unless .plugin/plugin.json declares an
#      mcpServers of its own and shadows the fallback.
#
# Claude Code is unaffected by both: it never reads .plugin/plugin.json, and
# it loads the manifest path at runtime (verified with `claude mcp list`:
# `plugin:archcore:archcore ... Connected`). Note that `claude plugin details`
# under-reports the MCP count for manifest-declared paths — it is not a valid
# oracle for this.
#
# Each test below pins one part. Drop any one part and Copilot users silently
# lose every document tool.

setup() {
  load '../helpers/common'
  common_setup
}

CLAUDE_MCP_REL=".claude.mcp.json"

@test "plugin root ships no filename any host auto-discovers" {
  # Copilot's two: .mcp.json and .github/mcp.json. Cursor 2.5+ adds a
  # dotless mcp.json (see cursor-mcp-architecture.adr / json-configs.bats).
  # A file under any of these names re-creates the collision no matter what
  # the manifests declare.
  [ ! -e "$PLUGIN_ROOT/.mcp.json" ] \
    || fail ".mcp.json at the plugin root is auto-discovered by Copilot and shadows the project's archcore server"
  [ ! -e "$PLUGIN_ROOT/mcp.json" ] \
    || fail "mcp.json at the plugin root is auto-discovered by Cursor 2.5+"
  [ ! -e "$PLUGIN_ROOT/.github/mcp.json" ] \
    || fail ".github/mcp.json at the plugin root is Copilot's fallback discovery name"
}

@test "the Claude MCP config ships under a name no host auto-discovers" {
  [ -f "$PLUGIN_ROOT/$CLAUDE_MCP_REL" ]
  grep -q '"archcore"' "$PLUGIN_ROOT/$CLAUDE_MCP_REL"
}

@test "the Claude MCP config invokes the global CLI" {
  local file="$PLUGIN_ROOT/$CLAUDE_MCP_REL"
  [ "$(jq -r '.mcpServers.archcore.command' < "$file")" = "archcore" ]
  [ "$(jq -r '.mcpServers.archcore.args[0]' < "$file")" = "mcp" ]
  [ "$(jq -r '.mcpServers.archcore.args | length' < "$file")" = "1" ]
}

@test "the Claude manifest points at it — the only route left after the rename" {
  # Without .mcp.json in the root there is no convention left to fall back
  # on, so this key is the whole of Claude Code's MCP wiring.
  local manifest="$PLUGIN_ROOT/.claude-plugin/plugin.json"
  local declared
  declared=$(jq -r '.mcpServers // empty' "$manifest")
  [ "$declared" = "./$CLAUDE_MCP_REL" ] \
    || fail "Claude manifest must declare mcpServers = \"./$CLAUDE_MCP_REL\" (found: '${declared:-<absent>}')"
  [ -f "$PLUGIN_ROOT/${declared#./}" ] \
    || fail "Claude manifest points at a file that does not ship: $declared"
}

@test "the Copilot manifest declares an empty mcpServers to shadow the Claude key" {
  # Copilot falls back to .claude-plugin/plugin.json when this key is absent,
  # which would hand it the very server the rename just took away. An empty
  # object is a declaration — it stops the fallback while contributing
  # nothing. Deleting this key is what an unwitting manifest cleanup does.
  local manifest="$PLUGIN_ROOT/.plugin/plugin.json"
  jq -e 'has("mcpServers")' "$manifest" > /dev/null \
    || fail "Copilot manifest must declare mcpServers, or Copilot reads .claude-plugin/plugin.json and re-shadows the project server"
  [ "$(jq -r '.mcpServers | type' "$manifest")" = "object" ] \
    || fail "Copilot manifest mcpServers must be an object"
  [ "$(jq -r '.mcpServers | length' "$manifest")" = "0" ] \
    || fail "Copilot manifest must contribute no MCP servers — the project-level server is the only supported route on Copilot"
}

@test "no host manifest other than Claude's points at the Claude MCP config" {
  # Cursor and Codex have their own answers (user-level template and
  # .codex.mcp.json). If either starts pointing here, the same collision
  # reappears on that host.
  local manifest
  for manifest in .cursor-plugin/plugin.json .codex-plugin/plugin.json .plugin/plugin.json; do
    ! grep -q "$CLAUDE_MCP_REL" "$PLUGIN_ROOT/$manifest" \
      || fail "$manifest references $CLAUDE_MCP_REL — that file is Claude Code's alone"
  done
}
