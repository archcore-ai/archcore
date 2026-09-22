#!/usr/bin/env bats
# Structure tests: validate hook configurations

setup() {
  load '../helpers/common'
  common_setup
}

# --- Script resolution, all hosts ------------------------------------------
#
# One table instead of a pair of near-identical tests per host. The pair-per-
# host shape is what let Copilot ship without either check: adding a host meant
# copy-pasting two more tests, and nobody did.
#
# Two host-specific details make a naive copy worse than useless here:
#
#   * Copilot names the field "bash", not "command". `jq '.. | .command?'`
#     returns NOTHING for copilot.hooks.json, so a copied test iterates an
#     empty set and reports ok — coverage that exists only in the test name.
#   * Each host substitutes its own plugin-root variable, and a config that
#     borrows another host's resolves to nothing at runtime. That is now checked
#     by its own test below rather than by substitution, because Copilot's
#     commands name three candidate variables (see copilot-plugin.bats) and a
#     one-variable sed would leave the other two unresolved.
#
# host|config|plugin-root variables the config may name
hook_configs() {
  cat <<'EOF'
claude|hooks/hooks.json|CLAUDE_PLUGIN_ROOT
cursor|hooks/cursor.hooks.json|CURSOR_PLUGIN_ROOT
codex|hooks/codex.hooks.json|PLUGIN_ROOT
copilot|hooks/copilot.hooks.json|COPILOT_PLUGIN_ROOT PLUGIN_ROOT CLAUDE_PLUGIN_ROOT
EOF
}

# Emits every script a hook config invokes, as an absolute path.
#
# Extraction is by script basename, not by substituting the host's plugin-root
# variable: Copilot's commands name their script once per candidate plus once in
# the fallback warning, so a substituting sed would emit the surrounding shell
# as though it were a path.
hook_scripts() {
  local config="$1"
  jq -r '.. | (.command? // .bash?) // empty' "$PLUGIN_ROOT/$config" \
    | grep -o 'bin/[a-z0-9_-]*' \
    | sed "s|^|$PLUGIN_ROOT/|" \
    | sort -u
}

# Emits every plugin-root variable a hook config names, one per line.
hook_root_vars() {
  local config="$1"
  jq -r '.. | (.command? // .bash?) // empty' "$PLUGIN_ROOT/$config" \
    | grep -o '\$[{]\?[A-Z_]*PLUGIN_ROOT' \
    | tr -d '${' \
    | sort -u
}

@test "every host hook config invokes scripts that exist" {
  local host config vars script missing=""
  while IFS='|' read -r host config vars; do
    [ -n "$host" ] || continue
    local found=0
    while IFS= read -r script; do
      [ -z "$script" ] && continue
      found=1
      [ -f "$script" ] || missing="$missing $host:$script"
    done < <(hook_scripts "$config")
    # An empty extraction means the accessor stopped matching this host's
    # schema — the exact failure this table exists to prevent.
    [ "$found" = "1" ] || fail "$config: no script paths extracted at all"
  done < <(hook_configs)
  [ -z "$missing" ] || fail "hook scripts do not exist:$missing"
}

@test "every host hook config invokes scripts that are executable" {
  local host config vars script not_exec=""
  while IFS='|' read -r host config vars; do
    [ -n "$host" ] || continue
    while IFS= read -r script; do
      [ -z "$script" ] && continue
      if [ -f "$script" ] && [ ! -x "$script" ]; then
        not_exec="$not_exec $host:$script"
      fi
    done < <(hook_scripts "$config")
  done < <(hook_configs)
  [ -z "$not_exec" ] || fail "hook scripts not executable:$not_exec"
}

@test "every host hook config names only its own plugin-root variables" {
  # Replaces what the old per-row sed enforced implicitly: a config naming
  # another host's variable resolves to nothing at runtime. Copilot is the one
  # host with a chain of candidates, because COPILOT_PLUGIN_ROOT appears in no
  # GitHub documentation while ${PLUGIN_ROOT} is the documented spelling —
  # copilot-plugin.bats carries the detail.
  local host config vars var allowed found
  while IFS='|' read -r host config vars; do
    [ -n "$host" ] || continue
    local seen=0
    while IFS= read -r var; do
      [ -z "$var" ] && continue
      seen=1
      found=0
      for allowed in $vars; do
        [ "$var" = "$allowed" ] && found=1
      done
      [ "$found" = "1" ] \
        || fail "$config names \$$var, which $host does not provide (allowed: $vars)"
    done < <(hook_root_vars "$config")
    [ "$seen" = "1" ] || fail "$config names no plugin-root variable at all"
  done < <(hook_configs)
}

@test "every hooks/*.json is enrolled in the resolution table" {
  # Without this, a fifth host's config ships unchecked exactly the way
  # copilot.hooks.json did.
  local file base missing=""
  for file in "$PLUGIN_ROOT"/hooks/*.json; do
    base="hooks/$(basename "$file")"
    hook_configs | grep -q "|$base|" || missing="$missing $base"
  done
  [ -z "$missing" ] || fail "hook configs missing from hook_configs():$missing"
}

# --- Phase 2.1 anti-regression invariants ---

@test "hooks.json: PreToolUse matcher includes Write and Edit" {
  local matcher
  matcher=$(jq -r '.hooks.PreToolUse[0].matcher' "$PLUGIN_ROOT/hooks/hooks.json")
  [[ "$matcher" == *"Write"* ]] || fail "Claude matcher missing Write: $matcher"
  [[ "$matcher" == *"Edit"* ]] || fail "Claude matcher missing Edit: $matcher"
}

@test "cursor.hooks.json: preToolUse matcher is exactly 'Write' (Cursor has no Edit tool)" {
  # Invariant from .archcore/plugin/multi-host-compatibility-layer.spec.md:
  # Cursor's API exposes only a Write tool; Edit/apply_patch don't exist there.
  local matcher
  matcher=$(jq -r '.hooks.preToolUse[0].matcher' "$PLUGIN_ROOT/hooks/cursor.hooks.json")
  [ "$matcher" = "Write" ] || fail "Cursor preToolUse matcher drifted from 'Write': $matcher"
}

@test "hooks.json: PostToolUse has no Write|Edit matcher (dead hook removed)" {
  local matchers
  matchers=$(jq -r '.hooks.PostToolUse[].matcher' "$PLUGIN_ROOT/hooks/hooks.json")
  if echo "$matchers" | grep -qE '(^|\|)Write(\||$)'; then
    fail "PostToolUse Write|Edit matcher was re-introduced — it is dead (PreToolUse already blocks direct writes to .archcore/)."
  fi
}

@test "hooks.json: PostToolUse matchers all target mcp__archcore__*" {
  local matchers
  matchers=$(jq -r '.hooks.PostToolUse[].matcher' "$PLUGIN_ROOT/hooks/hooks.json")
  while IFS= read -r m; do
    [ -z "$m" ] && continue
    echo "$m" | grep -qE '^mcp__archcore__' || fail "Unexpected PostToolUse matcher: $m"
  done <<<"$matchers"
}

@test "cursor.hooks.json: no postToolUse event (cleaned in Phase 2.1)" {
  local has
  has=$(jq 'has("postToolUse")' "$PLUGIN_ROOT/hooks/cursor.hooks.json" 2>/dev/null || echo "err")
  # The hooks are actually under .hooks (v1 format). Re-query:
  has=$(jq '.hooks | has("postToolUse")' "$PLUGIN_ROOT/hooks/cursor.hooks.json")
  [ "$has" = "false" ] || fail "cursor.hooks.json grew a postToolUse event — Phase 2.1 removed it because PreToolUse + afterMCPExecution cover every case."
}

@test "cursor.hooks.json: event set is exactly sessionStart/preToolUse/afterMCPExecution" {
  local events
  events=$(jq -r '.hooks | keys[]' "$PLUGIN_ROOT/hooks/cursor.hooks.json" | sort | tr '\n' ',')
  [ "$events" = "afterMCPExecution,preToolUse,sessionStart," ] || {
    echo "Actual events: $events"
    fail "cursor.hooks.json event set drifted from the expected {sessionStart, preToolUse, afterMCPExecution}."
  }
}

@test "hooks.json: event set is exactly SessionStart/PreToolUse/PostToolUse" {
  local events
  events=$(jq -r '.hooks | keys[]' "$PLUGIN_ROOT/hooks/hooks.json" | sort | tr '\n' ',')
  [ "$events" = "PostToolUse,PreToolUse,SessionStart," ] || {
    echo "Actual events: $events"
    fail "hooks.json event set drifted from the expected {SessionStart, PreToolUse, PostToolUse}."
  }
}

@test "codex.hooks.json: event set is exactly SessionStart/PreToolUse/PostToolUse" {
  local events
  events=$(jq -r '.hooks | keys[]' "$PLUGIN_ROOT/hooks/codex.hooks.json" | sort | tr '\n' ',')
  [ "$events" = "PostToolUse,PreToolUse,SessionStart," ] || {
    echo "Actual events: $events"
    fail "codex.hooks.json event set drifted from the expected {SessionStart, PreToolUse, PostToolUse}."
  }
}

# Codex has no `env` field on a hook handler — the documented handler keys are
# type, command, commandWindows, timeout, statusMessage, additionalContextLimit
# and async (developers.openai.com/codex/hooks) — so the host id travels as an
# assignment prefix on the command itself, which Codex runs through a shell.
#
# Without it the scripts fall back to the stdin heuristic, and a Codex
# SessionStart payload carries no turn_id, so the session was read as
# claude-code: the CLI leaf was called with the wrong dialect and the
# single-document stdout branch never fired (codex-adapter-conformance.adr).
@test "codex.hooks.json: every command pins ARCHCORE_HOST=codex" {
  local cmds cmd
  cmds=$(jq -r '.hooks | to_entries[] | .value[].hooks[].command' \
    "$PLUGIN_ROOT/hooks/codex.hooks.json")
  [ -n "$cmds" ] || fail "codex.hooks.json yielded no commands at all"
  while IFS= read -r cmd; do
    case "$cmd" in
      "ARCHCORE_HOST=codex "*) ;;
      *) fail "codex hook command must start with ARCHCORE_HOST=codex, got: $cmd" ;;
    esac
  done <<< "$cmds"
}

@test "codex.hooks.json: no other host config pins ARCHCORE_HOST=codex" {
  local host config vars
  while IFS='|' read -r host config vars; do
    [ "$config" = "hooks/codex.hooks.json" ] && continue
    jq -r '.. | (.command? // .bash?) // empty' "$PLUGIN_ROOT/$config" \
      | grep -q 'ARCHCORE_HOST=codex' \
      && fail "$config must not pin the codex host id"
  done < <(hook_configs)
  return 0
}

# --- Consistency ---

@test "every host hook config references the same set of scripts" {
  # The portable core is one set of scripts; a host that wires up a subset is a
  # host where some guard silently does not run. Paths are compared with the
  # plugin root resolved, so all four configs are directly comparable.
  #
  # ONE exception is allowed, and only this one: Copilot omits
  # bin/check-code-alignment. That script's entire job is to inject context from
  # a preToolUse hook, and Copilot's preToolUse reads only permissionDecision,
  # permissionDecisionReason and modifiedArgs — additionalContext is not a field
  # it accepts, so the script could only ever fork and produce nothing. It is
  # spelled out here rather than loosening the comparison, so any OTHER host
  # dropping any OTHER script still fails loudly.
  local host config vars scripts reference="" ref_host="" expected
  while IFS='|' read -r host config vars; do
    [ -n "$host" ] || continue
    scripts=$(hook_scripts "$config" | sort -u)
    if [ -z "$reference" ]; then
      reference="$scripts"
      ref_host="$host"
      continue
    fi
    expected="$reference"
    if [ "$host" = copilot ]; then
      expected=$(printf '%s\n' "$reference" | grep -v '/bin/check-code-alignment$')
    fi
    if [ "$scripts" != "$expected" ]; then
      echo "expected for $host:"; echo "$expected"
      echo "$host scripts:"; echo "$scripts"
      fail "script sets differ between $ref_host and $host"
    fi
  done < <(hook_configs)
}

@test "every host hook config registers bin/session-start on its session-start event" {
  # bin/session-start emits the init_project nudge on first-time / empty-state
  # sessions. If any host loses this wiring, onboarding silently breaks on that
  # host. Two things differ per host: the event key casing — SessionStart
  # (Claude/Codex) vs sessionStart (Cursor/Copilot) — and the entry shape,
  # since Copilot's entries are flat objects carrying "bash" where the others
  # nest a .hooks[] array of objects carrying "command". The union accessor
  # covers both; an empty extraction fails loudly rather than passing on an
  # empty set.
  local entries=(
    "hooks/hooks.json:SessionStart"
    "hooks/codex.hooks.json:SessionStart"
    "hooks/cursor.hooks.json:sessionStart"
    "hooks/copilot.hooks.json:sessionStart"
  )
  local entry file event cmds
  for entry in "${entries[@]}"; do
    file="${entry%:*}"
    event="${entry#*:}"
    cmds=$(jq -r --arg e "$event" \
      '.hooks[$e][]? | (.hooks[]?.command // .command? // .bash?) // empty' \
      "$PLUGIN_ROOT/$file")
    [ -n "$cmds" ] || fail "$file: '$event' event yielded no commands at all"
    # Not anchored at end of line: Copilot's command continues past the script
    # path with the rest of its candidate chain and a fallback warning.
    echo "$cmds" | grep -q 'bin/session-start' \
      || fail "$file: '$event' event must invoke bin/session-start; got: $cmds"
  done
}

@test "PostToolUse matchers cover BOTH project and plugin MCP tool naming" {
  # Claude Code names plugin-bundled MCP tools mcp__plugin_<plugin>_<server>__*
  # while a project-level .mcp.json yields mcp__archcore__*. Hook matchers
  # without regex metacharacters are EXACT matches, so every archcore tool in
  # a PostToolUse matcher must appear under both prefixes — otherwise
  # validation hooks silently never fire in one of the two setups.
  #
  # Cursor and Copilot are absent by design, not by omission: Cursor has no
  # postToolUse event at all (it uses afterMCPExecution), and Copilot's
  # postToolUse entries carry no matcher — the scripts self-filter there, which
  # copilot-plugin.bats asserts directly. Where there is no matcher there is no
  # matcher to get wrong.
  local file matcher tool
  for file in "hooks/hooks.json" "hooks/codex.hooks.json"; do
    while IFS= read -r matcher; do
      [[ "$matcher" == *"mcp__archcore__"* ]] || continue
      for tool in ${matcher//|/ }; do
        case "$tool" in
          mcp__archcore__*)
            [[ "$matcher" == *"mcp__plugin_archcore_archcore__${tool#mcp__archcore__}"* ]] \
              || fail "$file: matcher covers $tool but not its plugin-naming twin: $matcher"
            ;;
          mcp__plugin_archcore_archcore__*)
            [[ "$matcher" == *"mcp__archcore__${tool#mcp__plugin_archcore_archcore__}"* ]] \
              || fail "$file: matcher covers $tool but not its project-naming twin: $matcher"
            ;;
        esac
      done
    done < <(jq -r '.hooks.PostToolUse[]?.matcher // empty' "$PLUGIN_ROOT/$file")
  done
}
