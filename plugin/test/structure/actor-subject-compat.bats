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
  grep -F -q 'IF no document track produces it as its own product — `rule`, `journey`, `prd`, `idea`, `plan`, `cpat`, `task-type`' "$skill" \
    || fail "an explicitly named journey has no path through document"
  grep -F -q 'ask no question.' "$skill" \
    || fail "a failed routing condition on a named type costs a question"
  grep -F -q 'skills/_shared/actor-subject-compatibility.md' "$skill" \
    || fail "document/SKILL.md does not load the actor-subject compatibility file"
}

@test "an explicitly named type keeps its track, its contract, draft status, and the engine gate" {
  local skill flat_skill
  skill="$PLUGIN_ROOT/skills/document/SKILL.md"
  flat_skill=$(tr '\n' ' ' < "$skill" | sed 's/  */ /g')
  for phrase in \
    'Name any document type inside the subject' \
    '| The subject names a document type that no document track produces | → Named type, direct composition |' \
    '1. IF a document track produces the named type as its own product, THEN enter that track with the type settled' \
    '3. Apply the compatibility file for the type before the first MCP call that names it.' \
    'when one exists, offer to update it instead.' \
    '5. Load `skills/_shared/<type>-contract.md` when that file exists, and `skills/_shared/precision-rules.md`.' \
    '7. Create the document with `status: draft` through `create_document`.' \
    '- any category — a type composed on the Named type path'; do
    [[ "$flat_skill" == *"$phrase"* ]] || { fail "document/SKILL.md lost: $phrase"; return 1; }
  done
  grep -F -q 'explicitly takes the document skill'"'"'s Named type path instead of this track.' "$PLUGIN_ROOT/skills/_shared/tracks/describe.md" \
    || fail "describe.md does not hand a named journey to the document skill"
  [[ "$(tr '\n' ' ' < "$PLUGIN_ROOT/skills/_shared/journey-contract.md" | sed 's/  */ /g')" == *'and `document` when the subject names `journey` (its Named type path)'* ]] \
    || fail "journey-contract.md does not name document as a loader"
}

@test "a named type is a leading or backticked slug, applies before a mode entry, and every producer honors it" {
  local flat_skill
  flat_skill=$(tr '\n' ' ' < "$PLUGIN_ROOT/skills/document/SKILL.md" | sed 's/  */ /g')
  [[ "$flat_skill" == *'a mode word — is a kernel document type slug, or the subject writes a slug in backticks'* ]] \
    || fail "any slug in plain topic text hijacks routing"
  [[ "$flat_skill" == *'When two slugs qualify, the first one wins.'* ]] || fail "two named slugs have no tie rule"
  [[ "$flat_skill" == *'IF the subject names a type per Named type above, THEN apply Named type first.'* ]] \
    || fail "a mode word bypasses the named-type path"
  grep -F -q 'or names `rnd` or `research` per the document skill'"'"'s Named type rule, fix the artifact type' \
    "$PLUGIN_ROOT/skills/_shared/tracks/research.md" || fail "the closing test overrides a named rnd"
  [[ "$flat_skill" == *'— `rule`, `journey`, `prd`, `idea`, `plan`, `cpat`, `task-type`, `mrd`, `brd`, `urd`, `brs`, `strs`, `syrs`, `srs` — THEN compose it directly'* ]] \
    || fail "a named rule goes through the decision track and its adr questions"
  [[ "$flat_skill" == *'a leading type slug takes the Named type path.'* ]] || fail "a leading slug is still sent to classification"
  local f
  for f in agents/archcore-assistant.md agents/archcore-assistant.toml copilot-agents/archcore-assistant.agent.md; do
    grep -F -q 'or several for a cross-spec flow, with one' "$PLUGIN_ROOT/$f" \
      || { fail "$f still limits a scenario to one spec"; return 1; }
  done
}

