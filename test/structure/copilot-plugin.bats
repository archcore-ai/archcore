#!/usr/bin/env bats
# Structure tests: validate the native GitHub Copilot CLI adapter.

setup() {
  load '../helpers/common'
  common_setup
}

MANIFEST_REL=".plugin/plugin.json"
HOOKS_REL="hooks/copilot.hooks.json"

@test "Copilot manifest exists and is valid JSON" {
  local manifest="$PLUGIN_ROOT/$MANIFEST_REL"
  [ -f "$manifest" ]
  jq . < "$manifest" > /dev/null
}

@test "Copilot manifest points explicitly at every shared component" {
  local manifest="$PLUGIN_ROOT/$MANIFEST_REL"
  jq -e '
    .name == "archcore" and
    .hooks == "./hooks/copilot.hooks.json" and
    .skills == "./skills/" and
    .agents == "./copilot-agents/" and
    .commands == "./commands/"
  ' "$manifest" > /dev/null
}

# Copilot's plugin reference gives a default only to agents/ and skills/;
# commands has no default at all. Every other host picks the wrappers up
# implicitly, so on Copilot this pointer is the only thing that makes
# /archcore:* exist — asserted separately from the block above with its own
# reason so a future "cleanup" of the manifest cannot quietly delete it.
@test "Copilot manifest declares commands explicitly (no default covers it)" {
  local manifest="$PLUGIN_ROOT/$MANIFEST_REL"
  [ "$(jq -r '.commands' "$manifest")" = "./commands/" ]
  [ -d "$PLUGIN_ROOT/commands" ]
}

# Copilot's agent loader keys on the *.agent.md extension (plugin reference:
# "Path(s) to agent directories (.agent.md files)"). The shared agents/ dir
# holds plain NAME.md for Claude Code and Cursor plus NAME.toml for Codex, so
# Copilot's copies need their own directory: .agent.md still ends in .md, and
# a copy sitting next to the original would give Claude and Cursor two files
# declaring the same frontmatter `name:`.
@test "Copilot agents live in their own directory with the .agent.md extension" {
  local manifest="$PLUGIN_ROOT/$MANIFEST_REL"
  [ "$(jq -r '.agents' "$manifest")" = "./copilot-agents/" ]
  [ -d "$PLUGIN_ROOT/copilot-agents" ]

  local plain
  plain=$(find "$PLUGIN_ROOT/copilot-agents" -name '*.md' ! -name '*.agent.md' -print)
  [ -z "$plain" ] || fail "copilot-agents holds files Copilot will not load: $plain"
}

# github/copilot-cli#4234: Copilot launches a plugin's MCP children with cwd set
# to the plugin install directory and passes them no project path, so tools
# report success while documents land in ~/.copilot/installed-plugins/. MCP for
# Copilot therefore comes from the project's own .mcp.json (written by
# `archcore init --agent copilot`) or ~/.copilot/mcp-config.json — never from
# this manifest. Same class of defect as cursor-mcp-architecture.adr; the
# reasoning lives in copilot-mcp-architecture.adr.
@test "Copilot manifest does NOT ship plugin MCP" {
  local manifest="$PLUGIN_ROOT/$MANIFEST_REL"
  jq -e 'has("mcpServers") | not' "$manifest" > /dev/null \
    || fail "mcpServers is back in the Copilot manifest — see github/copilot-cli#4234"

  # The file itself must stay: Claude Code discovers plugin-root .mcp.json with
  # no manifest key, and it is the same file archcore init writes into a repo.
  [ -f "$PLUGIN_ROOT/.mcp.json" ]
}

@test "every './'-relative path in the Copilot manifest resolves" {
  # Generic guard: any string value beginning with "./" must point at something
  # that exists under PLUGIN_ROOT. The pointer assertions above compare strings
  # only, so without this a rename leaves the manifest green and the host with
  # nothing to load.
  local manifest="$PLUGIN_ROOT/$MANIFEST_REL"
  local missing="" rel
  while IFS= read -r rel; do
    [ -z "$rel" ] && continue
    if [ ! -e "$PLUGIN_ROOT/${rel#./}" ]; then
      missing="$missing $rel"
    fi
  done < <(jq -r '.. | strings | select(startswith("./"))' < "$manifest")
  [ -z "$missing" ] || fail "unresolved './' paths in .plugin/plugin.json:$missing"
}

@test "only plugin.json lives under .plugin" {
  local extra_files
  extra_files=$(find "$PLUGIN_ROOT/.plugin" -type f ! -name plugin.json -print)
  [ -z "$extra_files" ] || fail ".plugin contains non-manifest files: $extra_files"
}

@test "Copilot exposes the same command wrappers as the other hosts" {
  local expected actual
  expected=$(find "$PLUGIN_ROOT/commands" -name '*.md' -exec basename {} .md \; | sort)
  actual=$(find "$PLUGIN_ROOT/skills" -mindepth 1 -maxdepth 1 -type d \
    ! -name '_*' -exec basename {} \; | sort)
  [ -n "$expected" ] || fail "no command wrappers found"
  # Every wrapper must name a real skill; a wrapper pointing at a removed skill
  # would surface a /archcore:<name> entry that dead-ends on Copilot.
  local cmd
  for cmd in $expected; do
    echo "$actual" | grep -qx "$cmd" \
      || fail "commands/$cmd.md has no matching skills/$cmd/"
  done
}

@test "Copilot manifest metadata matches the Claude manifest" {
  # name/description/version, not version alone: the four manifests are the
  # same plugin seen from four hosts, and a description that drifts on one of
  # them ships a different pitch to those users. verify-plugin-integrity §4
  # compares all four.
  local copilot="$PLUGIN_ROOT/$MANIFEST_REL"
  local claude="$PLUGIN_ROOT/.claude-plugin/plugin.json"
  local field
  for field in name description version; do
    [ "$(jq -r ".$field" "$copilot")" = "$(jq -r ".$field" "$claude")" ] \
      || fail "$field differs: copilot=$(jq -r ".$field" "$copilot") claude=$(jq -r ".$field" "$claude")"
  done
}

@test "Copilot hooks config exists and is valid version 1 JSON" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  [ -f "$hooks" ]
  jq -e '.version == 1 and (.hooks | type == "object")' "$hooks" > /dev/null
}

@test "Copilot hooks use only native camelCase lifecycle events" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  local events
  events=$(jq -r '.hooks | keys[]' "$hooks" | sort | tr '\n' ',')
  [ "$events" = "postToolUse,preToolUse,sessionStart," ] \
    || fail "unexpected Copilot hook event set: $events"
}

@test "every Copilot hook entry sets deterministic host detection" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  jq -e '
    ([.hooks[][]] | length) == 6 and
    all(.hooks[][]; .type == "command" and .env.ARCHCORE_HOST == "copilot")
  ' "$hooks" > /dev/null
}

@test "every Copilot hook runs from the user project root" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  jq -e '
    all(.hooks[][]; .cwd == ".")
  ' "$hooks" > /dev/null
}

@test "Copilot hook commands use COPILOT_PLUGIN_ROOT and the shared scripts" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  local actual expected
  actual=$(jq -r '.hooks[][] | .bash' "$hooks" | sort)
  expected=$(printf '%s\n' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/check-archcore-write' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/check-cascade' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/check-code-alignment' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/check-precision' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/session-start' \
    '"${COPILOT_PLUGIN_ROOT}"/bin/validate-archcore' | sort)
  [ "$actual" = "$expected" ] || {
    echo "expected: $expected"
    echo "actual: $actual"
    fail "Copilot hook commands must route to the shared bin scripts"
  }
}

@test "Copilot preToolUse covers every native mutation tool" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  local matcher="create|edit|str_replace_editor|apply_patch"
  jq -e --arg matcher "$matcher" '
    (.hooks.preToolUse | length) == 2 and
    all(.hooks.preToolUse[];
      .matcher == $matcher and
      .timeoutSec == 1 and
      (.bash | test("/bin/check-(archcore-write|code-alignment)$"))
    )
  ' "$hooks" > /dev/null
}

@test "Copilot postToolUse self-filters through all shared validation scripts" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  jq -e '
    (.hooks.postToolUse | length) == 3 and
    all(.hooks.postToolUse[];
      (has("matcher") | not) and
      .timeoutSec == 3 and
      (.bash | test("/bin/(validate-archcore|check-cascade|check-precision)$"))
    )
  ' "$hooks" > /dev/null
}
