#!/usr/bin/env bats
# Structure tests: validate the Cursor MCP example template shipped at
# docs/cursor.mcp.example.json. This file is NOT auto-loaded by Cursor —
# it's a canonical snippet users copy into ~/.cursor/mcp.json (user-scoped)
# or .cursor/mcp.json (project-scoped) when wiring the archcore MCP server.
#
# The template lives under docs/ rather than the plugin root so it cannot
# accidentally trigger Cursor's plugin-MCP auto-detection (which would
# spawn the server from the plugin install directory, not the workspace).
# Workspace path is passed explicitly via `--project ${workspaceFolder}`
# in args, because Cursor's MCP stdio schema has no `cwd` field and
# `${workspaceFolder}` interpolation is supported in `args` (not in a
# non-existent `cwd`).

setup() {
  load '../helpers/common'
  common_setup
}

TEMPLATE_PATH="docs/cursor.mcp.example.json"

@test "docs/cursor.mcp.example.json exists" {
  [ -f "$PLUGIN_ROOT/$TEMPLATE_PATH" ]
}

@test "docs/cursor.mcp.example.json is valid JSON" {
  jq . < "$PLUGIN_ROOT/$TEMPLATE_PATH" > /dev/null
}

@test "template declares an archcore server under mcpServers" {
  local file="$PLUGIN_ROOT/$TEMPLATE_PATH"
  jq -e '.mcpServers.archcore' < "$file" > /dev/null
}

@test "template invokes 'archcore mcp'" {
  local file="$PLUGIN_ROOT/$TEMPLATE_PATH"
  [ "$(jq -r '.mcpServers.archcore.command' < "$file")" = "archcore" ] \
    || fail "command must be 'archcore' (resolved via PATH)"
  [ "$(jq -r '.mcpServers.archcore.args[0]' < "$file")" = "mcp" ] \
    || fail "args[0] must be 'mcp'"
}

@test "template passes --project \"\${workspaceFolder}\" in args" {
  # Cursor's MCP stdio schema has no `cwd` field; the only way to point
  # the server at the workspace is via interpolation in `args`. Without
  # this, the server inherits whatever cwd Cursor spawns it with — which
  # for plugin-shipped MCPs is the plugin install dir, not the workspace.
  local file="$PLUGIN_ROOT/$TEMPLATE_PATH"
  local project_arg
  project_arg=$(jq -r '.mcpServers.archcore.args[1] + " " + .mcpServers.archcore.args[2]' < "$file")
  [ "$project_arg" = "--project \${workspaceFolder}" ] \
    || fail "args must include --project \"\${workspaceFolder}\", got: '$project_arg'"
}

@test "template does NOT set a cwd field (Cursor schema does not support it)" {
  local file="$PLUGIN_ROOT/$TEMPLATE_PATH"
  local cwd
  cwd=$(jq -r '.mcpServers.archcore.cwd // empty' < "$file")
  [ -z "$cwd" ] \
    || fail "cwd must NOT be set — Cursor ignores it; use --project in args instead. Got: '$cwd'"
}

@test "template has only one server defined" {
  local file="$PLUGIN_ROOT/$TEMPLATE_PATH"
  local count
  count=$(jq -r '.mcpServers | keys | length' < "$file")
  [ "$count" = "1" ] || fail "expected exactly 1 server (archcore), found $count"
}

@test "README references docs/cursor.mcp.example.json so users find the template" {
  grep -q "docs/cursor.mcp.example.json" "$PLUGIN_ROOT/README.md" \
    || fail "README.md must reference docs/cursor.mcp.example.json so users discover the template"
}

@test "no legacy cursor.mcp.json at the plugin root" {
  # The legacy template at the plugin root could trigger Cursor's
  # plugin-MCP auto-detection — keep it out of root.
  [ ! -f "$PLUGIN_ROOT/cursor.mcp.json" ] \
    || fail "cursor.mcp.json must not exist at plugin root (use docs/cursor.mcp.example.json)"
}
