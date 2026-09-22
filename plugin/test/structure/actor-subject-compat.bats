#!/usr/bin/env bats
# Structure tests: the actor-subject vocabulary gate (CLI 0.8.4) and its wiring.

setup() {
  load '../helpers/common'
  common_setup
  SHARED="$PLUGIN_ROOT/skills/_shared"
  COMPAT="$SHARED/actor-subject-compatibility.md"
}

@test "actor-subject compatibility file exists with the research-file structure" {
  [ -f "$COMPAT" ] || fail "missing skills/_shared/actor-subject-compatibility.md"
  local s
  for s in "Version probe" "Fallback" "Shared repositories"; do
    grep -q -x -F "## $s" "$COMPAT" || fail "missing section '## $s'"
  done
  grep -F -q '"$actor_subject_skill_dir/../../bin/cli-gte" 0.8.4' "$COMPAT" || fail "probe does not invoke cli-gte 0.8.4"
  grep -F -q 'The minimum is CLI `0.8.4`' "$COMPAT" || fail "minimum version line missing"
  grep -F -q 'Scenario and journey require Archcore CLI 0.8.4; skipping actor-subject documents.' "$COMPAT" \
    || fail "fallback report sentence missing"
  grep -F -q '`needs-vocabulary-probe`' "$COMPAT" || fail "no-shell sentinel missing"
  grep -F -q 'never converts' "$COMPAT" || fail "no-conversion rule missing"
}

@test "research compatibility file is unchanged by this release" {
  grep -F -q '"$research_skill_dir/../../bin/cli-gte" 0.8.3' "$SHARED/research-compatibility.md" \
    || fail "research-compatibility.md probe line changed"
  ! grep -F -q 'scenario' "$SHARED/research-compatibility.md" \
    || fail "research-compatibility.md mentions scenario; the vocabularies stay in separate files"
}

@test "every agent instruction file names both types and the compatibility file" {
  local f
  for f in "$PLUGIN_ROOT/agents/archcore-assistant.md" \
           "$PLUGIN_ROOT/agents/archcore-assistant.toml" \
           "$PLUGIN_ROOT/copilot-agents/archcore-assistant.agent.md"; do
    grep -F -q 'skills/_shared/actor-subject-compatibility.md' "$f" \
      || fail "no actor-subject-compatibility.md reference in ${f#"$PLUGIN_ROOT"/}"
    grep -F -q '`scenario` belongs to knowledge; `journey` belongs to vision' "$f" \
      || fail "no category sentence for scenario and journey in ${f#"$PLUGIN_ROOT"/}"
  done
}

@test "every skills/_shared/actor-subject-compatibility.md reference resolves" {
  local refs
  refs=$(grep -rlF 'skills/_shared/actor-subject-compatibility.md' "$PLUGIN_ROOT/skills" "$PLUGIN_ROOT/agents" "$PLUGIN_ROOT/copilot-agents" 2>/dev/null || true)
  [ -n "$refs" ] || fail "no file references skills/_shared/actor-subject-compatibility.md"
  [ -f "$COMPAT" ] || fail "referenced compatibility file does not exist"
}

@test "no argument-hint lists scenario or journey as a mode" {
  local hint
  hint=$(awk '/^---$/ { if (++d == 2) exit; next }
              d == 1 && /^argument-hint:/ { print; exit }' "$PLUGIN_ROOT/skills/document/SKILL.md")
  ! printf '%s' "$hint" | grep -E -q 'scenario|journey' \
    || fail "document/SKILL.md argument-hint exposes an actor-subject type: $hint"
  hint=$(awk '/^---$/ { if (++d == 2) exit; next }
              d == 1 && /^argument-hint:/ { print; exit }' "$PLUGIN_ROOT/skills/plan/SKILL.md")
  ! printf '%s' "$hint" | grep -E -q 'scenario|journey' \
    || fail "plan/SKILL.md argument-hint exposes an actor-subject type: $hint"
}

@test "document code reaches scenario through describe.draft; no document mode produces a journey" {
  local skill="$PLUGIN_ROOT/skills/document/SKILL.md"
  grep -F -q '`describe.draft` selects `spec`,' "$skill" \
    || fail "document/SKILL.md code mode does not name describe.draft as the type selector"
  grep -F -q 'No mode produces a `journey`' "$skill" \
    || fail "document/SKILL.md does not rule out journey production"
  grep -F -q 'This track never produces a `journey`' "$PLUGIN_ROOT/skills/_shared/tracks/describe.md" \
    || fail "describe.md does not rule out journey production"
  grep -F -q 'skills/_shared/actor-subject-compatibility.md' "$skill" \
    || fail "document/SKILL.md does not load the actor-subject compatibility file"
}
