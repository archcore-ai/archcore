#!/usr/bin/env bats
# Structure tests: validate Codex plugin manifest and hooks config

setup() {
  load '../helpers/common'
  common_setup
}

@test ".codex-plugin/plugin.json exists" {
  [ -f "$PLUGIN_ROOT/.codex-plugin/plugin.json" ]
}

@test ".codex-plugin/plugin.json is valid JSON" {
  jq . < "$PLUGIN_ROOT/.codex-plugin/plugin.json" > /dev/null
}

@test ".codex-plugin/plugin.json has required fields" {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  jq -e '.name' < "$file" > /dev/null
  jq -e '.version' < "$file" > /dev/null
  jq -e '.description' < "$file" > /dev/null
}

@test ".codex-plugin/plugin.json has component pointers" {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  jq -e '.skills' < "$file" > /dev/null
  jq -e '.hooks' < "$file" > /dev/null
  jq -e '.mcpServers' < "$file" > /dev/null
}

@test ".codex-plugin/plugin.json uses Codex interface metadata block" {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  jq -e '.interface.displayName == "Archcore"' < "$file" > /dev/null
  jq -e '.interface.shortDescription' < "$file" > /dev/null
  jq -e '.interface.longDescription' < "$file" > /dev/null
  jq -e '.interface.developerName == "Archcore"' < "$file" > /dev/null
  jq -e '.interface.category == "Coding"' < "$file" > /dev/null
  jq -e '.interface.capabilities | index("Interactive")' < "$file" > /dev/null
  jq -e '.interface.capabilities | index("Read")' < "$file" > /dev/null
  jq -e '.interface.capabilities | index("Write")' < "$file" > /dev/null
}

@test ".codex-plugin/plugin.json has no legacy top-level UI metadata" {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  [ "$(jq 'has("displayName")' < "$file")" = "false" ]
  [ "$(jq 'has("category")' < "$file")" = "false" ]
  [ "$(jq 'has("tags")' < "$file")" = "false" ]
}

@test "codex hooks pointer references codex.hooks.json" {
  local hooks_path
  hooks_path=$(jq -r '.hooks' < "$PLUGIN_ROOT/.codex-plugin/plugin.json")
  [ "$hooks_path" = "./hooks/codex.hooks.json" ]
}

@test "codex mcp pointer references codex-specific plugin-root MCP config" {
  local mcp_path
  mcp_path=$(jq -r '.mcpServers' < "$PLUGIN_ROOT/.codex-plugin/plugin.json")
  [ "$mcp_path" = "./.codex.mcp.json" ]
}

@test ".codex.mcp.json invokes the global CLI" {
  local file="$PLUGIN_ROOT/.codex.mcp.json"
  jq . < "$file" > /dev/null
  [ "$(jq -r '.mcpServers.archcore.command' < "$file")" = "archcore" ] \
    || fail "command must be 'archcore' (resolved via PATH)"
  [ "$(jq -r '.mcpServers.archcore.args[0]' < "$file")" = "mcp" ] \
    || fail "args[0] must be 'mcp'"
}

@test "codex plugin metadata matches Claude Code plugin metadata" {
  local codex="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  local claude="$PLUGIN_ROOT/.claude-plugin/plugin.json"
  [ "$(jq -r '.name' < "$codex")" = "$(jq -r '.name' < "$claude")" ]
  [ "$(jq -r '.version' < "$codex")" = "$(jq -r '.version' < "$claude")" ]
  [ "$(jq -r '.description' < "$codex")" = "$(jq -r '.description' < "$claude")" ]
}

@test "codex slash command wrappers exist for every user-facing skill" {
  local commands_dir="$PLUGIN_ROOT/commands"
  [ -d "$commands_dir" ]

  local expected=(
    audit
    capture
    context
    decide
    help
    init
    plan
  )

  local name
  for name in "${expected[@]}"; do
    [ -f "$commands_dir/$name.md" ] || fail "missing Codex command wrapper: $name"
    [ -d "$PLUGIN_ROOT/skills/$name" ] || fail "command has no matching skill: $name"
    grep -q "skills/$name/SKILL.md" "$commands_dir/$name.md" || fail "command does not delegate to matching skill: $name"
  done

  local count
  count=$(find "$commands_dir" -maxdepth 1 -type f -name '*.md' | wc -l | tr -d ' ')
  [ "$count" = "${#expected[@]}" ] || fail "expected ${#expected[@]} Codex commands, found $count"
}

@test "codex slash command wrappers have descriptions" {
  local file
  for file in "$PLUGIN_ROOT"/commands/*.md; do
    grep -q '^description:' "$file" || fail "missing description in $file"
  done
}

@test ".agents/plugins/marketplace.json exists and uses Codex marketplace schema" {
  local file="$PLUGIN_ROOT/.agents/plugins/marketplace.json"
  [ -f "$file" ]
  jq . < "$file" > /dev/null
  jq -e '.name == "archcore-plugins"' < "$file" > /dev/null
  jq -e '.interface.displayName == "Archcore"' < "$file" > /dev/null
  jq -e '.plugins[0].name == "archcore"' < "$file" > /dev/null
  jq -e '.plugins[0].source.source == "local"' < "$file" > /dev/null
  jq -e '.plugins[0].source.path == "./"' < "$file" > /dev/null
  jq -e '.plugins[0].policy.installation == "INSTALLED_BY_DEFAULT"' < "$file" > /dev/null
  jq -e '.plugins[0].policy.authentication == "ON_INSTALL"' < "$file" > /dev/null
  jq -e '.plugins[0].category == "Coding"' < "$file" > /dev/null
}

@test "legacy .codex-plugin/marketplace.json is absent" {
  [ ! -e "$PLUGIN_ROOT/.codex-plugin/marketplace.json" ]
}

@test "only plugin.json lives under .codex-plugin" {
  local extra_files
  extra_files=$(find "$PLUGIN_ROOT/.codex-plugin" -type f ! -name plugin.json -print)
  [ -z "$extra_files" ] || fail ".codex-plugin contains non-manifest files: $extra_files"
}

@test "hooks/codex.hooks.json exists and is valid JSON" {
  [ -f "$PLUGIN_ROOT/hooks/codex.hooks.json" ]
  jq . < "$PLUGIN_ROOT/hooks/codex.hooks.json" > /dev/null
}

@test "codex.hooks.json uses \${PLUGIN_ROOT}/bin/... substitution (host-neutral canonical)" {
  # Codex's hooks engine injects two env vars (codex-rs/hooks/src/engine/discovery.rs):
  #   env.insert("PLUGIN_ROOT", ...);                   // canonical, host-neutral
  #   env.insert("CLAUDE_PLUGIN_ROOT", ...);            // OOTB compat shim only
  # The substitution loop folds ${KEY} over the command string at spawn.
  # We use the canonical PLUGIN_ROOT in this Codex-specific hook file rather
  # than borrowing Claude's name. CLAUDE_PLUGIN_ROOT is intentionally NOT used
  # here (it is a compat alias for porting old Claude plugins, not the right
  # name for a Codex-native hook config).
  # Plugin hooks require `codex features enable plugin_hooks` (currently
  # `under development, false` in Codex 0.130.0).
  local file="$PLUGIN_ROOT/hooks/codex.hooks.json"
  while IFS= read -r command; do
    [[ "$command" == \$\{PLUGIN_ROOT\}/bin/* ]] \
      || fail "Codex hook command must use \${PLUGIN_ROOT}/bin/... (host-neutral canonical), got: $command"
  done < <(jq -r '.. | .command? // empty' "$file")
  if grep -q '\${CLAUDE_PLUGIN_ROOT}\|\${CODEX_PLUGIN_ROOT}\|\${CURSOR_PLUGIN_ROOT}' "$file"; then
    fail "codex.hooks.json must not borrow other hosts' env-var names; use \${PLUGIN_ROOT}"
  fi
}

@test "codex.hooks.json registers SessionStart, PreToolUse, PostToolUse" {
  local file="$PLUGIN_ROOT/hooks/codex.hooks.json"
  jq -e '.hooks.SessionStart' < "$file" > /dev/null
  jq -e '.hooks.PreToolUse' < "$file" > /dev/null
  jq -e '.hooks.PostToolUse' < "$file" > /dev/null
}

@test "codex PreToolUse matcher includes Write, Edit, apply_patch" {
  local file="$PLUGIN_ROOT/hooks/codex.hooks.json"
  local matcher
  matcher=$(jq -r '.hooks.PreToolUse[0].matcher' < "$file")
  [[ "$matcher" == *"Write"* ]] || fail "matcher missing Write: $matcher"
  [[ "$matcher" == *"Edit"* ]] || fail "matcher missing Edit: $matcher"
  [[ "$matcher" == *"apply_patch"* ]] || fail "matcher missing apply_patch: $matcher"
}

@test "codex PostToolUse does NOT register validate-archcore on Write/Edit path" {
  # Compatibility Layer invariant: validate-archcore runs only on the MCP path,
  # never on Write/Edit PostToolUse (would fork a shell repo-wide for no benefit).
  local file="$PLUGIN_ROOT/hooks/codex.hooks.json"
  local has_write_path
  has_write_path=$(jq -r '.hooks.PostToolUse[]? | select(.matcher | test("^Write|Edit$")) | .matcher' < "$file")
  [ -z "$has_write_path" ]
}
