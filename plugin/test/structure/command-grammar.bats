#!/usr/bin/env bats
# Structure test: the command entry grammar from
# context-driven-routing-beneath-command-grammar.adr and command-surface-v2.spec.
# plan, document, and review take one optional free-text argument and give a
# retired form no handling of its own; init keeps its closed mode list, which
# names no document type or route. No hint carries a flag (init settings are
# preview toggles per init-import-mode.adr), and no skill Result section tells the
# agent to print a gate address.

setup() {
  load '../helpers/common'
  common_setup
}

hint_of() {
  awk '/^---$/ { if (++d == 2) exit; next }
       d == 1 && /^argument-hint:/ { sub(/^argument-hint: */, ""); print; exit }' "$1"
}

@test "each argument hint pins its form" {
  local expected skill hint
  while IFS='|' read -r skill expected; do
    hint=$(hint_of "$PLUGIN_ROOT/skills/$skill/SKILL.md")
    [ "$hint" = "$expected" ] || { fail "$skill hint is $hint, expected $expected"; return 1; }
  done <<'HINTS'
init|"[import|refresh] [path or domain]"
plan|"[topic]"
document|"[subject]"
review|"[free text: a path, a topic, 'close out <plan>', 'are docs stale', 'audit']"
HINTS
}

@test "the init mode list names no document type or route" {
  local modes word
  modes=$(hint_of "$PLUGIN_ROOT/skills/init/SKILL.md" | sed -E 's/^"\[([^]]*)\].*/\1/')
  for word in $(printf '%s' "$modes" | tr '|' ' '); do
    case "$word" in
      adr|rfc|rule|guide|doc|spec|research|evidence|task-type|cpat|prd|idea|plan|rnd|mrd|brd|urd|brs|strs|syrs|srs|scenario|journey|null|decision|amendment|capability|umbrella)
        fail "init mode list names $word"; return 1 ;;
    esac
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
  hits=$(grep -rn -E '/archcore:(review --(drift|deep)|init (--(refresh|domain|depth|scale)|domain ))|document (adr|rfc|rule|evidence|scenario|journey)`|/archcore:(plan (sdd|sources|iso)|document (decision|code)|review (drift|deep|closeout|experience))\b' \
    "$PLUGIN_ROOT/skills" "$PLUGIN_ROOT/commands" "$PLUGIN_ROOT/agents" "$PLUGIN_ROOT/copilot-agents" "$REPO_ROOT/README.md" | grep -v 'README.md:[0-9]*:| `/archcore:' || true)
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

@test "document classification table maps each classification to its track entry" {
  local skill="$PLUGIN_ROOT/skills/document/SKILL.md"
  grep -F -q '| decision | `skills/_shared/tracks/decision.md`, `decision.classify` |' "$skill" || { fail "decision row"; return 1; }
  grep -F -q '| code-doc | `skills/_shared/tracks/describe.md`, `describe.read` |' "$skill" || { fail "code-doc row"; return 1; }
  grep -F -q '| research material | `skills/_shared/tracks/research.md`, `research.frame` |' "$skill" || { fail "research material row"; return 1; }
}

@test "review routes plain-language signals to each track" {
  local skill="$PLUGIN_ROOT/skills/review/SKILL.md"
  grep -F -q '| Staleness wording:' "$skill" || { fail "staleness row"; return 1; }
  grep -F -q '| Audit wording:' "$skill" || { fail "audit row"; return 1; }
  grep -F -q '| Completion signals:' "$skill" || { fail "completion row"; return 1; }
  grep -F -q '| Capture wording:' "$skill" || { fail "capture row"; return 1; }
  grep -F -q '| Git resolves no scope and the request names none | → ask one question for a path, topic, or ref; show no health report |' "$skill" || { fail "no-scope row"; return 1; }
}

@test "no skill gives a retired form handling of its own" {
  local hits
  hits=$(grep -n -i -E 'no longer a mode|retired (mode word|form)|name the current form' \
    "$PLUGIN_ROOT/skills/init/SKILL.md" "$PLUGIN_ROOT/skills/plan/SKILL.md" "$PLUGIN_ROOT/skills/document/SKILL.md" \
    "$PLUGIN_ROOT/skills/review/SKILL.md" "$PLUGIN_ROOT/skills/_shared/gate-contract.md" || true)
  [ -z "$hits" ] || fail "a former mode word still has its own handling:
$hits"
}

@test "plain branch review checks matching plans without a completion phrase" {
  local skill="$PLUGIN_ROOT/skills/review/SKILL.md"
  local closeout="$PLUGIN_ROOT/skills/_shared/tracks/closeout.md"
  grep -F -q 'Match local `plan` documents to the branch work:' "$skill" \
    || { fail "plain branch review does not search for matching plans"; return 1; }
  grep -F -q 'Run `closeout.verify` even when the request contains no completion wording.' "$skill" \
    || { fail "matching plan does not enter closeout on plain review"; return 1; }
  grep -F -q 'IF step 2 matched no `plan`, THEN skip closeout and run the durable-context selection.' "$skill" \
    || { fail "plain review without a plan skips durable-context selection"; return 1; }
  grep -F -q 'IF matched plans exist but none reached `closeout.capture`, THEN run the durable-context selection' "$skill" \
    || { fail "a review whose plans stop at verify selects nothing from the branch"; return 1; }
  grep -F -q 'reports the capture line once' "$PLUGIN_ROOT/skills/_shared/tracks/closeout.md" \
    || { fail "capture line is not bounded to once per review"; return 1; }
  grep -F -q 'IF a candidate matches ambiguously, THEN report it as a candidate and skip closeout for it.' "$skill" \
    || { fail "ambiguous plan candidates are not bounded"; return 1; }
  grep -F -q 'a task or' "$closeout" && grep -F -q 'acceptance criterion of that plan carries a verdict other than fulfilled.' "$closeout" \
    || { fail "closeout no longer guards unfinished plans"; return 1; }
  grep -F -q 'IF any task or acceptance criterion of the plan' "$closeout" \
    || { fail "plain-review entry runs gates past verify on unfinished work"; return 1; }
  grep -F -q 'run the full gate sequence once per plan' "$closeout" \
    || { fail "closeout does not loop per matched plan"; return 1; }
  grep -F -q 'confirmation naming that specific plan' "$closeout" \
    || { fail "closeout no longer confirms plan removal"; return 1; }
}

@test "decision.classify sends a standard over one existing adr to the cascade, and a standard with no adr to decision.adr" {
  local track="$PLUGIN_ROOT/skills/_shared/tracks/decision.md"
  grep -F -q '`decision.cascade` with the standard cascade selected when the request carries standard signals and one local `adr` on the topic exists' "$track" \
    || { fail "standard over an existing adr does not reach decision.cascade"; return 1; }
  grep -F -q 'or when standard signals appear and no local `adr` exists; `decision.rfc`' "$track" \
    || { fail "standard with no adr does not go to decision.adr first"; return 1; }
}

@test "no command or skill description lists a mode" {
  local skill f desc
  for skill in plan document review; do
    for f in "$PLUGIN_ROOT/commands/$skill.md" "$PLUGIN_ROOT/skills/$skill/SKILL.md"; do
      desc=$(awk '/^---$/ { if (++d == 2) exit; next } d == 1 && /^description:/ { print; exit }' "$f")
      ! printf '%s' "$desc" | grep -E -q 'Modes?[,:]|named as the first word|by mode' \
        || { fail "${f#"$PLUGIN_ROOT"/} description still lists modes"; return 1; }
    done
  done
}

@test "every command defines what runs with no arguments" {
  grep -F -q '| No arguments, one draft with a state block this command resumes | Resume it' "$PLUGIN_ROOT/skills/plan/SKILL.md" || { fail "plan has no resume row"; return 1; }
  grep -F -q 'Offer a start menu of at most four items' "$PLUGIN_ROOT/skills/plan/SKILL.md" || { fail "plan has no start menu"; return 1; }
  grep -F -q '| No arguments and no reliable signal | Ask one question — what outcome to plan; a dirty branch alone is not a target |' "$PLUGIN_ROOT/skills/plan/SKILL.md" || { fail "plan has no one-question row"; return 1; }
  grep -F -q '| No arguments, one draft with a state block this command resumes | → resume it |' "$PLUGIN_ROOT/skills/document/SKILL.md" || { fail "document has no resume row"; return 1; }
  grep -F -q 'a start menu of at most four owners' "$PLUGIN_ROOT/skills/document/SKILL.md" || { fail "document has no start menu"; return 1; }
  grep -F -q '| No arguments, branch with changes | → branch review' "$PLUGIN_ROOT/skills/review/SKILL.md" || { fail "review has no no-arguments row"; return 1; }
  grep -F -q 'No arguments, or any other first word' "$PLUGIN_ROOT/skills/init/SKILL.md" || { fail "init does not state the plain run"; return 1; }
}
