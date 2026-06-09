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
  jq -e '.interface.category == "Productivity"' < "$file" > /dev/null
  jq -e '.interface.capabilities | index("Read")' < "$file" > /dev/null
  jq -e '.interface.capabilities | index("Write")' < "$file" > /dev/null
  # Codex docs (developers.openai.com/codex/plugins/build) document only
  # "Read" and "Write" as capability values; "Interactive" is not documented
  # and was removed to avoid marketplace validation risk.
  [ "$(jq '.interface.capabilities | index("Interactive")' < "$file")" = "null" ]
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
  [ "$(jq -r '.archcore.command' < "$file")" = "archcore" ] \
    || fail "command must be 'archcore' (resolved via PATH)"
  [ "$(jq -r '.archcore.args[0]' < "$file")" = "mcp" ] \
    || fail "args[0] must be 'mcp'"
  [ "$(jq -r 'has("mcpServers") or has("mcp_servers")' < "$file")" = "false" ] \
    || fail ".codex.mcp.json should use the Codex-documented direct server map"
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
  jq -e '.plugins[0].category == "Productivity"' < "$file" > /dev/null
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

# --- Codex marketplace-surface invariants ----------------------------------
# The fields below populate the install card / composer surfaces that Codex
# renders for the plugin. A broken field doesn't fail at runtime — it fails
# silently in the marketplace UI, which is hard to notice from `codex` CLI.
# The tests below pin the contract so accidental edits regress loudly.

png_magic_ok() {
  # PNG magic bytes: 89 50 4E 47 0D 0A 1A 0A
  [ "$(head -c 8 "$1" | od -An -tx1 | tr -d ' \n')" = "89504e470d0a1a0a" ]
}

@test "assets/icon.png exists and is a valid PNG" {
  local file="$PLUGIN_ROOT/assets/icon.png"
  [ -f "$file" ] || fail "missing icon: $file"
  png_magic_ok "$file" || fail "icon.png is not a valid PNG"
  # Sanity: not a stub / empty placeholder
  [ "$(wc -c < "$file" | tr -d ' ')" -gt 1000 ] || fail "icon.png suspiciously small"
}

@test "assets/logo.png exists and is a valid PNG" {
  local file="$PLUGIN_ROOT/assets/logo.png"
  [ -f "$file" ] || fail "missing logo: $file"
  png_magic_ok "$file" || fail "logo.png is not a valid PNG"
  [ "$(wc -c < "$file" | tr -d ' ')" -gt 1000 ] || fail "logo.png suspiciously small"
}

@test "interface.composerIcon points to an existing file" {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  local rel
  rel=$(jq -r '.interface.composerIcon // empty' < "$file")
  [ -n "$rel" ] || fail "interface.composerIcon is missing or empty"
  [[ "$rel" == ./* ]] || fail "composerIcon must start with './': $rel"
  [ -f "$PLUGIN_ROOT/${rel#./}" ] || fail "composerIcon does not resolve: $rel"
}

@test "interface.logo points to an existing file" {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  local rel
  rel=$(jq -r '.interface.logo // empty' < "$file")
  [ -n "$rel" ] || fail "interface.logo is missing or empty"
  [[ "$rel" == ./* ]] || fail "logo must start with './': $rel"
  [ -f "$PLUGIN_ROOT/${rel#./}" ] || fail "logo does not resolve: $rel"
}

@test "interface.screenshots is an array; every entry resolves" {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  [ "$(jq -r '.interface.screenshots | type' < "$file")" = "array" ]
  local missing=""
  local rel
  while IFS= read -r rel; do
    [ -z "$rel" ] && continue
    [[ "$rel" == ./* ]] || { missing="$missing $rel(not-relative)"; continue; }
    [ -f "$PLUGIN_ROOT/${rel#./}" ] || missing="$missing $rel"
  done < <(jq -r '.interface.screenshots[]?' < "$file")
  [ -z "$missing" ] || fail "broken screenshot paths:$missing"
}

@test "interface.brandColor is a valid 6-digit hex color" {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  local color
  color=$(jq -r '.interface.brandColor // empty' < "$file")
  [[ "$color" =~ ^#[0-9A-Fa-f]{6}$ ]] || fail "brandColor must match ^#[0-9A-Fa-f]{6}$, got: $color"
}

@test "interface.defaultPrompt is a non-empty array of strings" {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  [ "$(jq -r '.interface.defaultPrompt | type' < "$file")" = "array" ]
  local count
  count=$(jq -r '.interface.defaultPrompt | length' < "$file")
  [ "$count" -gt 0 ] || fail "defaultPrompt must contain at least one starter"
  # All elements must be strings
  local types
  types=$(jq -r '.interface.defaultPrompt | map(type) | unique | join(",")' < "$file")
  [ "$types" = "string" ] || fail "defaultPrompt entries must all be strings, found types: $types"
  # No empty strings
  local empties
  empties=$(jq -r '.interface.defaultPrompt | map(select(. == "")) | length' < "$file")
  [ "$empties" = "0" ] || fail "defaultPrompt contains empty string(s)"
}

@test "interface external URLs are non-empty https://..." {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  local key url
  for key in websiteURL privacyPolicyURL termsOfServiceURL; do
    url=$(jq -r ".interface.\"$key\" // empty" < "$file")
    [ -n "$url" ] || fail "interface.$key is missing or empty"
    [[ "$url" == https://* ]] || fail "interface.$key must be https://..., got: $url"
  done
}

@test "author.url and homepage are non-empty https URLs" {
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  local author_url homepage
  author_url=$(jq -r '.author.url // empty' < "$file")
  [[ "$author_url" == https://* ]] || fail "author.url must be https://..., got: $author_url"
  homepage=$(jq -r '.homepage // empty' < "$file")
  [[ "$homepage" == https://* ]] || fail "homepage must be https://..., got: $homepage"
}

@test "docs/TERMS.md exists (referenced by interface.termsOfServiceURL)" {
  # termsOfServiceURL points to the raw GitHub blob of this file; if the
  # file is deleted, the marketplace card link 404s.
  [ -f "$PLUGIN_ROOT/docs/TERMS.md" ] || fail "docs/TERMS.md missing — termsOfServiceURL will 404"
}

@test "category in .codex-plugin/plugin.json matches .agents/plugins/marketplace.json" {
  # Drift guard: editing one without the other yields inconsistent marketplace
  # presentation (composer says one category, registry says another).
  local plugin_cat market_cat
  plugin_cat=$(jq -r '.interface.category' < "$PLUGIN_ROOT/.codex-plugin/plugin.json")
  market_cat=$(jq -r '.plugins[0].category' < "$PLUGIN_ROOT/.agents/plugins/marketplace.json")
  [ "$plugin_cat" = "$market_cat" ] \
    || fail "category drift: plugin.json='$plugin_cat' but marketplace='$market_cat'"
}

@test "every './'-relative path in .codex-plugin/plugin.json resolves" {
  # Generic guard: any string value that begins with "./" must point to an
  # existing file or directory under PLUGIN_ROOT. Catches typos, renames,
  # and accidentally deleted assets across all manifest fields at once.
  local file="$PLUGIN_ROOT/.codex-plugin/plugin.json"
  local missing="" rel
  while IFS= read -r rel; do
    [ -z "$rel" ] && continue
    if [ ! -e "$PLUGIN_ROOT/${rel#./}" ]; then
      missing="$missing $rel"
    fi
  done < <(jq -r '.. | strings | select(startswith("./"))' < "$file")
  [ -z "$missing" ] || fail "unresolved './' paths in plugin.json:$missing"
}
