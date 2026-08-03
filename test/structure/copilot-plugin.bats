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
# this plugin. Same class of defect as cursor-mcp-architecture.adr; the
# reasoning lives in copilot-mcp-architecture.adr.
#
# Note what this test does NOT assert. Omitting the key was the original
# defense and it does not work: with the key absent Copilot falls back to
# reading .claude-plugin/plugin.json, and picks up the Claude-only server from
# there. An empty declaration is the thing that stops the fallback. The full
# three-part contract and the measurements behind it live in
# test/structure/plugin-mcp-isolation.bats.
@test "Copilot manifest ships no MCP servers of its own" {
  local manifest="$PLUGIN_ROOT/$MANIFEST_REL"
  jq -e 'has("mcpServers")' "$manifest" > /dev/null \
    || fail "mcpServers is absent from the Copilot manifest — Copilot then reads .claude-plugin/plugin.json and adopts the Claude server"
  [ "$(jq -r '.mcpServers | length' "$manifest")" = "0" ] \
    || fail "the Copilot manifest contributes MCP servers — see github/copilot-cli#4234"
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
    ([.hooks[][]] | length) == 5 and
    all(.hooks[][]; .type == "command" and .env.ARCHCORE_HOST == "copilot")
  ' "$hooks" > /dev/null
}

@test "every Copilot hook runs from the user project root" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  jq -e '
    all(.hooks[][]; .cwd == ".")
  ' "$hooks" > /dev/null
}

# Every hook command must reach its script through a chain of plugin-root
# candidates rather than one variable.
#
# COPILOT_PLUGIN_ROOT appears in no GitHub documentation — checked 2026-07-27
# against the CLI plugin reference and the hooks reference, neither of which
# contains the string. The only documented spelling is ${PLUGIN_ROOT} ("Use
# ${PLUGIN_ROOT} to reference paths within the plugin directory"), which is also
# what codex.hooks.json uses.
#
# Naming a single variable is what shipped the defect: unset, it left the
# literal path /bin/<script>, and Copilot classifies a failed exec as "other
# non-zero exit", i.e. a DENY of every matched tool call. See
# copilot-adapter-design.adr.
hook_commands() {
  jq -r '.hooks[][] | .bash' "$PLUGIN_ROOT/$HOOKS_REL"
}

# The one bin script a command routes to, from every mention it makes of one.
command_script() {
  printf '%s' "$1" | grep -o 'bin/[a-z0-9_-]*' | sort -u
}

@test "every Copilot hook command considers all three plugin-root candidates" {
  local cmd var
  while IFS= read -r cmd; do
    [ -z "$cmd" ] && continue
    for var in COPILOT_PLUGIN_ROOT PLUGIN_ROOT CLAUDE_PLUGIN_ROOT; do
      case "$cmd" in
        *"\$$var"*) ;;
        *) fail "hook command never considers \$$var: $cmd" ;;
      esac
    done
  done < <(hook_commands)
}

@test "every Copilot hook command routes to exactly one shared bin script" {
  local cmd scripts
  while IFS= read -r cmd; do
    [ -z "$cmd" ] && continue
    scripts=$(command_script "$cmd")
    [ "$(printf '%s\n' "$scripts" | wc -l | tr -d ' ')" = "1" ] \
      || fail "hook command mixes several scripts ($scripts): $cmd"
  done < <(hook_commands)
}

# check-code-alignment is deliberately absent: its only output channel on this
# host (preToolUse additionalContext) is not a field Copilot reads. See the
# preToolUse tests below.
@test "Copilot hook commands cover the shared bin scripts and nothing else" {
  local actual expected
  actual=$(hook_commands | grep -o 'bin/[a-z0-9_-]*' | sort -u)
  expected=$(printf '%s\n' \
    'bin/check-archcore-write' \
    'bin/check-cascade' \
    'bin/check-precision' \
    'bin/session-start' \
    'bin/validate-archcore' | sort)
  [ "$actual" = "$expected" ] || {
    echo "expected: $expected"
    echo "actual: $actual"
    fail "Copilot hook commands must route to the shared bin scripts"
  }
}

# --- Behaviour of the resolution chain -------------------------------------
#
# The assertions above cannot tell a working chain from a plausible-looking one,
# and that is exactly the defect that shipped: the config read correctly and
# denied every edit. These three run the commands.

@test "an unresolved plugin root exits 0 and says so" {
  local cmd status out
  while IFS= read -r cmd; do
    [ -z "$cmd" ] && continue
    out=$(env -u COPILOT_PLUGIN_ROOT -u PLUGIN_ROOT -u CLAUDE_PLUGIN_ROOT \
            sh -c "$cmd" 2>&1 </dev/null) && status=0 || status=$?
    # The exit status matters more than the message. Any non-zero exit other
    # than 2 denies the tool call on Copilot, so a plugin we cannot locate must
    # fail open rather than block the session out of its own repo.
    [ "$status" = "0" ] \
      || fail "unresolved root exits $status — Copilot reads that as a deny: $cmd"
    case "$out" in
      *"plugin root unresolved"*) ;;
      *) fail "unresolved root fails silently, disabling the guard unnoticed: $cmd" ;;
    esac
  done < <(hook_commands)
}

@test "each plugin-root candidate on its own reaches the script" {
  local cmd script stub var out
  stub="$BATS_TEST_TMPDIR/stub-root"
  while IFS= read -r cmd; do
    [ -z "$cmd" ] && continue
    script=$(command_script "$cmd")
    rm -rf "$stub"
    mkdir -p "$stub/bin"
    printf '#!/bin/sh\necho REACHED\n' > "$stub/$script"
    chmod +x "$stub/$script"
    for var in COPILOT_PLUGIN_ROOT PLUGIN_ROOT CLAUDE_PLUGIN_ROOT; do
      out=$(env -u COPILOT_PLUGIN_ROOT -u PLUGIN_ROOT -u CLAUDE_PLUGIN_ROOT \
              "$var=$stub" sh -c "$cmd" 2>&1 </dev/null)
      [ "$out" = "REACHED" ] \
        || fail "\$$var alone does not reach $script (got: $out)"
    done
  done < <(hook_commands)
}

@test "a candidate that does not hold the script is skipped, not fatal" {
  # A PLUGIN_ROOT set for an unrelated tool must not shadow the real plugin:
  # each candidate is probed with -x, not merely tested for emptiness.
  local cmd script stub out
  stub="$BATS_TEST_TMPDIR/skip-root"
  while IFS= read -r cmd; do
    [ -z "$cmd" ] && continue
    script=$(command_script "$cmd")
    rm -rf "$stub"
    mkdir -p "$stub/bin"
    printf '#!/bin/sh\necho REACHED\n' > "$stub/$script"
    chmod +x "$stub/$script"
    out=$(env -u CLAUDE_PLUGIN_ROOT \
            COPILOT_PLUGIN_ROOT="$BATS_TEST_TMPDIR/absent" \
            PLUGIN_ROOT="$stub" sh -c "$cmd" 2>&1 </dev/null)
    [ "$out" = "REACHED" ] \
      || fail "a dead first candidate blocks the rest of the chain: $out"
  done < <(hook_commands)
}

# Copilot's preToolUse accepts exactly three output fields — permissionDecision,
# permissionDecisionReason and modifiedArgs (docs.github.com/en/copilot/
# reference/hooks-reference). additionalContext is NOT among them, so a
# preToolUse hook on this host can decide, but it cannot inform.
#
# check-archcore-write decides, and belongs here. check-code-alignment only
# ever calls archcore_hook_pretool_info and returns 0 — on Copilot that output
# is discarded by the host, so registering it would fork a process on every
# single edit to produce nothing at all. It stays registered on hosts whose
# preToolUse does carry context (see hooks/hooks.json, cursor, codex).
#
# If Copilot ever adds additionalContext to preToolUse, re-registering it here
# is the whole change.
@test "Copilot preToolUse registers only the hook that can act on this host" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  local matcher="create|edit|str_replace_editor|apply_patch"
  jq -e --arg matcher "$matcher" '
    (.hooks.preToolUse | length) == 1 and
    all(.hooks.preToolUse[];
      .matcher == $matcher and
      .timeoutSec == 1 and
      (.bash | test("bin/check-archcore-write"))
    )
  ' "$hooks" > /dev/null
}

@test "Copilot does not register the context-only preToolUse hook" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  ! jq -e '.hooks.preToolUse[]? | select(.bash | test("bin/check-code-alignment"))' "$hooks" > /dev/null \
    || fail "check-code-alignment is registered on Copilot preToolUse, where its additionalContext output is discarded by the host"
}

@test "the context-only preToolUse hook stays registered on hosts that carry context" {
  # Negative control for the test above: the removal must be Copilot-specific,
  # not a quiet deletion of the feature everywhere.
  local f
  for f in hooks.json cursor.hooks.json codex.hooks.json; do
    jq -e '[.. | strings | select(test("check-code-alignment"))] | length > 0' \
      "$PLUGIN_ROOT/hooks/$f" > /dev/null \
      || fail "check-code-alignment vanished from hooks/$f"
  done
}

@test "Copilot postToolUse self-filters through all shared validation scripts" {
  local hooks="$PLUGIN_ROOT/$HOOKS_REL"
  jq -e '
    (.hooks.postToolUse | length) == 3 and
    all(.hooks.postToolUse[];
      (has("matcher") | not) and
      .timeoutSec == 3 and
      (.bash | test("bin/(validate-archcore|check-cascade|check-precision)"))
    )
  ' "$hooks" > /dev/null
}
