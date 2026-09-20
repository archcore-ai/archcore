#!/usr/bin/env bats
# Structure test: the command entry grammar from command-entry-grammar.adr and
# command-surface-v2.spec. Every command reads `[mode] [subject]`: the first
# bracket of its argument hint is a closed list of mode words, no hint lists a
# document type name or a route name as a mode, no hint carries a flag (init
# settings are preview toggles per init-import-mode.adr), and no skill Result
# section tells the agent to print a gate address.

setup() {
  load '../helpers/common'
  common_setup
}

hint_of() {
  awk '/^---$/ { if (++d == 2) exit; next }
       d == 1 && /^argument-hint:/ { sub(/^argument-hint: */, ""); print; exit }' "$1"
}

@test "each argument hint pins its mode list" {
  local expected skill hint
  while IFS='|' read -r skill expected; do
    hint=$(hint_of "$PLUGIN_ROOT/skills/$skill/SKILL.md")
    [ "$hint" = "$expected" ] || { fail "$skill hint is $hint, expected $expected"; return 1; }
  done <<'HINTS'
init|"[import|refresh] [path or domain]"
plan|"[sdd|sources|iso|research] [topic]"
document|"[decision|code|research] [subject]"
review|"[drift|deep|closeout|experience] [path, tag, or scope]"
HINTS
}

@test "no mode list names a document type or a route" {
  local skill hint modes word
  for skill in init plan document review; do
    hint=$(hint_of "$PLUGIN_ROOT/skills/$skill/SKILL.md")
    modes=$(printf '%s' "$hint" | sed -E 's/^"\[([^]]*)\].*/\1/')
    for word in $(printf '%s' "$modes" | tr '|' ' '); do
      case "$word" in
        adr|rfc|rule|guide|doc|spec|research|evidence|task-type|cpat|prd|idea|plan|rnd|mrd|brd|urd|brs|strs|syrs|srs|scenario|journey)
          # `research` is the one mode that shares a type name: it names the research instrument, not the type.
          [ "$word" = research ] || { fail "$skill mode list names the type $word"; return 1; } ;;
        null|decision|amendment|capability|umbrella)
          # `decision` is a document mode, not the plan route of the same name.
          [ "$skill" = document ] && [ "$word" = decision ] || { fail "$skill mode list names the route $word"; return 1; } ;;
      esac
    done
  done
}

@test "no argument hint carries a flag" {
  local skill hint
  for skill in init plan document review; do
    hint=$(hint_of "$PLUGIN_ROOT/skills/$skill/SKILL.md")
    [[ "$hint" != *--* ]] || { fail "$skill hint carries a flag: $hint"; return 1; }
  done
}

@test "the init hint is byte-identical in the skill and the command wrapper" {
  [ "$(hint_of "$PLUGIN_ROOT/skills/init/SKILL.md")" = "$(hint_of "$PLUGIN_ROOT/commands/init.md")" ] \
    || fail "init argument-hint differs between skills/init/SKILL.md and commands/init.md"
}

@test "no shipped skill or command text uses a retired entry form" {
  local hits
  hits=$(grep -rn -E '/archcore:(review --(drift|deep)|init (--(refresh|domain|depth|scale)|domain ))|document (adr|rfc|rule|evidence|scenario|journey)`' \
    "$PLUGIN_ROOT/skills" "$PLUGIN_ROOT/commands" "$PLUGIN_ROOT/agents" "$PLUGIN_ROOT/copilot-agents" "$REPO_ROOT/README.md" || true)
  [ -z "$hits" ] || fail "retired entry forms remain:
$hits"
}

@test "every track-running skill Result section forbids printing a gate address" {
  local skill result
  for skill in plan document review; do
    result=$(sed -n '/^## Result/,$p' "$PLUGIN_ROOT/skills/$skill/SKILL.md")
    [[ "$result" == *'do not print a gate address of the form `<track>.<stage>`'* ]] \
      || { fail "$skill Result section lacks the gate-address rule"; return 1; }
    if printf '%s' "$result" | grep -v 'gate address' | grep -E -q '`(decision|describe|research|closeout|actualize|experience|sdd|requirements-cascade)\.[a-z]+`'; then
      fail "$skill Result section still names a gate address"; return 1
    fi
  done
}

@test "document mode table maps each mode to its track entry" {
  local skill="$PLUGIN_ROOT/skills/document/SKILL.md"
  grep -F -q '| `decision` | `skills/_shared/tracks/decision.md`, `decision.classify` |' "$skill" || { fail "decision mode row"; return 1; }
  grep -F -q '| `code` | `skills/_shared/tracks/describe.md`, `describe.read` |' "$skill" || { fail "code mode row"; return 1; }
  grep -F -q '| `research` | `skills/_shared/tracks/research.md`, `research.frame` |' "$skill" || { fail "research mode row"; return 1; }
  grep -F -q -- '- `decision` → decision track.' "$skill" || { fail "decision mode entry step"; return 1; }
  grep -F -q -- '- `code` → describe track at `describe.read`.' "$skill" || { fail "code mode entry step"; return 1; }
  grep -F -q -- '- `research` → research track at `research.frame`' "$skill" || { fail "research mode entry step"; return 1; }
}

@test "review mode rows route each first word to its track" {
  local skill="$PLUGIN_ROOT/skills/review/SKILL.md"
  grep -F -q '| First word `drift` | → actualize track' "$skill" || { fail "drift row"; return 1; }
  grep -F -q '| First word `deep` | → actualize track over all documents' "$skill" || { fail "deep row"; return 1; }
  grep -F -q '| First word `closeout` | → closeout track' "$skill" || { fail "closeout row"; return 1; }
  grep -F -q '| First word `experience` | → experience track' "$skill" || { fail "experience row"; return 1; }
}

@test "decision.classify sends a standard over one existing adr to the cascade, and a standard with no adr to decision.adr" {
  local track="$PLUGIN_ROOT/skills/_shared/tracks/decision.md"
  grep -F -q 'Next: `decision.cascade` with the standard cascade selected when the request carries standard signals and one local `adr` on the topic exists' "$track" \
    || { fail "standard over an existing adr does not reach decision.cascade"; return 1; }
  grep -F -q 'or when standard signals appear and no local `adr` exists; `decision.rfc`' "$track" \
    || { fail "standard with no adr does not go to decision.adr first"; return 1; }
}

@test "every command description and its skill description name each mode of the hint" {
  local skill hint modes word desc
  for skill in init plan document review; do
    hint=$(hint_of "$PLUGIN_ROOT/skills/$skill/SKILL.md")
    modes=$(printf '%s' "$hint" | sed -E 's/^"\[([^]]*)\].*/\1/' | tr '|' '\n' | sed 's/ .*//')
    for f in "$PLUGIN_ROOT/commands/$skill.md" "$PLUGIN_ROOT/skills/$skill/SKILL.md"; do
      desc=$(awk '/^---$/ { if (++d == 2) exit; next } d == 1 && /^description:/ { print; exit }' "$f")
      for word in $modes; do
        printf '%s' "$desc" | grep -E -q "(^|[^a-z-])${word}([^a-z-]|$)" \
          || { fail "${f#"$PLUGIN_ROOT"/} description does not name the $skill mode '$word'"; return 1; }
      done
    done
  done
}

@test "every command defines what runs with no arguments" {
  grep -F -q '| No arguments | Ground per step 1;' "$PLUGIN_ROOT/skills/plan/SKILL.md" || { fail "plan has no no-arguments row"; return 1; }
  grep -F -q '| No arguments | → git investigation of the branch changes, then one classifying question' "$PLUGIN_ROOT/skills/document/SKILL.md" || { fail "document has no no-arguments row"; return 1; }
  grep -F -q '| No arguments, branch with changes | → branch review' "$PLUGIN_ROOT/skills/review/SKILL.md" || { fail "review has no no-arguments row"; return 1; }
  grep -F -q 'No arguments, or any other first word' "$PLUGIN_ROOT/skills/init/SKILL.md" || { fail "init does not state the plain run"; return 1; }
}
