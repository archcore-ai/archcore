#!/bin/sh
# Behavioral sdd.design gate bench. Runs model calls only when explicitly requested.
# GATE_BENCH_HOST selects claude (default), codex, or copilot (binary: GATE_BENCH_COPILOT, default copilot).
# GATE_BENCH_MODEL selects the model; GATE_BENCH_LIMIT limits fixture count; GATE_BENCH_REPS repeats each fixture.
# GATE_BENCH_REF reads the contracts from a git ref (for example a release tag) instead of the working tree.
# GATE_BENCH_FIXTURES overrides input; GATE_BENCH_OUTPUT_DIR retains raw replies.
# Exit 1: verdict/format mismatch. Exit 2: invalid inputs or CLI failure.
set -eu

REPO_ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
FIXTURES=${GATE_BENCH_FIXTURES:-$REPO_ROOT/test/behavioral/fixtures/gate-bench.tsv}
SHARED="$REPO_ROOT/plugins/archcore/skills/_shared"
CONTRACTS="tracks/sdd.md gate-contract.md elicitation-contract.md delta-routing.md"
KEYS="skip resume questions gate assumptions persist"
HOST=${GATE_BENCH_HOST:-claude}
COPILOT=${GATE_BENCH_COPILOT:-copilot}
REF=${GATE_BENCH_REF:-}
TAB=$(printf '\t')

case "$HOST" in
  claude) host_bin=claude ;;
  codex) host_bin=codex ;;
  copilot) host_bin=$COPILOT ;;
  *) echo "GATE_BENCH_HOST must be claude, codex, or copilot" >&2; exit 2 ;;
esac
[ -f "$FIXTURES" ] || { echo "missing fixtures: $FIXTURES" >&2; exit 2; }
command -v "$host_bin" >/dev/null 2>&1 || { echo "$HOST CLI not found: $host_bin" >&2; exit 2; }
command -v jq >/dev/null 2>&1 || { echo "jq not found on PATH" >&2; exit 2; }
LIMIT=${GATE_BENCH_LIMIT:-0}
case "$LIMIT" in ''|*[!0-9]*) echo "GATE_BENCH_LIMIT must be a nonnegative integer" >&2; exit 2 ;; esac
REPS=${GATE_BENCH_REPS:-1}
case "$REPS" in ''|*[!0-9]*|0) echo "GATE_BENCH_REPS must be a positive integer" >&2; exit 2 ;; esac
# expected: key=value pairs joined by ';' — a fixture pins only the keys it names.
if ! awk -F '\t' '
  /^#/ || /^[[:space:]]*$/ {next}
  NF != 3 || $1 == "" || $2 == "" || seen[$1]++ {bad=1}
  { pair = "((skip|persist)=(yes|no)|questions=(>0|[0-9]+)|gate=(open|advance|skipped)|assumptions=[0-9]+|resume=(none|sdd\\.[a-z]+))" }
  $3 !~ ("^" pair "(;" pair ")*$") {bad=1}
  {n++}
  END {exit (bad || !n)}
' "$FIXTURES"; then
  echo "fixtures must contain unique IDs, a situation, and valid key=value expectations" >&2
  exit 2
fi

contracts_text=""
for c in $CONTRACTS; do
  if [ -n "$REF" ]; then
    body=$(git -C "$SHARED" show "$REF:./$c") || { echo "cannot read $c at $REF" >&2; exit 2; }
  else
    body=$(cat "$SHARED/$c")
  fi
  contracts_text="$contracts_text
--- CONTRACT: skills/_shared/$c ---
$body"
done
if [ -n "${GATE_BENCH_OUTPUT_DIR:-}" ]; then
  results_dir=$GATE_BENCH_OUTPUT_DIR
  mkdir -p "$results_dir"
else
  results_dir=$(mktemp -d "${TMPDIR:-/tmp}/archcore-gate-bench.XXXXXX") || exit 2
  trap 'rm -rf "$results_dir"' EXIT
fi
results_dir=$(CDPATH='' cd -- "$results_dir" && pwd)
: > "$results_dir/pass"
: > "$results_dir/fail"
: > "$results_dir/error"

# ask <base>: send <base>.prompt to the host with no tools and no MCP; write the reply text to <base>.out.
ask() {
  b=$1
  case "$HOST" in
    claude)
      set -- -p --tools '' --strict-mcp-config --mcp-config '{"mcpServers":{}}' \
        --setting-sources user --no-session-persistence --output-format json
      if [ -n "${GATE_BENCH_MODEL:-}" ]; then set -- "$@" --model "$GATE_BENCH_MODEL"; fi
      (cd "$results_dir" && claude "$@" < "$b.prompt") > "$b.json" 2> "$b.stderr" || return 1
      jq -e '.is_error == false and (.result | type == "string")' "$b.json" >/dev/null 2>&1 || return 1
      jq -r '.result' "$b.json" > "$b.out" ;;
    codex)
      set -- exec --skip-git-repo-check --ephemeral -s read-only -o "$b.out"
      if [ -n "${GATE_BENCH_MODEL:-}" ]; then set -- "$@" -m "$GATE_BENCH_MODEL"; fi
      (cd "$results_dir" && codex "$@" - < "$b.prompt") > "$b.log" 2> "$b.stderr" || return 1
      [ -s "$b.out" ] ;;
    copilot)
      set -- -s --no-custom-instructions --available-tools= --disable-builtin-mcps --no-ask-user
      if [ -n "${GATE_BENCH_MODEL:-}" ]; then set -- "$@" --model "$GATE_BENCH_MODEL"; fi
      (cd "$results_dir" && "$COPILOT" "$@" -p "$(cat "$b.prompt")" < /dev/null) > "$b.out" 2> "$b.stderr" || return 1
      [ -s "$b.out" ] ;;
  esac
}

n=0
printf 'id\texpected\tverdict\treply\n'
while IFS="$TAB" read -r id situation expected || [ -n "$id" ]; do
  case "$id" in ''|\#*) continue ;; esac
  case "$id$situation$expected" in *[![:space:]]*) ;; *) continue ;; esac
  n=$((n + 1))
  if [ "$LIMIT" -gt 0 ] && [ "$n" -gt "$LIMIT" ]; then break; fi
  r=0
  while [ "$r" -lt "$REPS" ]; do
    r=$((r + 1))
    run_id=$id; base="$results_dir/$n"
    if [ "$REPS" -gt 1 ]; then run_id="$id#$r"; base="$results_dir/$n.$r"; fi
    printf '%s\n' \
      "You are the plan skill executing gate sdd.design under the contracts below. Apply them literally." \
      "Judge this one step of the gate: the situation is already established; nobody will answer further questions in this step." \
      "Output EXACTLY one line in this format and nothing else:" \
      "gate-verdict: skip=<yes|no>; resume=<gate id where work resumes, or none>; questions=<integer>; gate=<open|advance|skipped>; assumptions=<integer>; persist=<yes|no>" \
      "skip: whether sdd.design's skip_when holds. resume: the gate the resume rules send the work to, or none when nothing is resumed. questions: how many new questions you ask the user in this step. gate: the state of sdd.design after this step. assumptions: how many choices the draft marks [assumption] after this step. persist: whether the state block is saved after this step." \
      "Write each count as digits (0 when none). Do not write the questions or name the choices." \
      "$contracts_text" \
      "--- SITUATION ---" "$situation" \
      > "$base.prompt"
    if ! ask "$base"; then
      printf '%s\t%s\tERROR\t%s CLI failed; see %s.stderr\n' "$run_id" "$expected" "$HOST" "$base"
      echo "$run_id" >> "$results_dir/error"
      continue
    fi
    out=$(sed '/^[[:space:]]*$/d' "$base.out")
    case "$out" in '`gate-verdict:'*'`') out=${out#'`'}; out=${out%'`'} ;; esac
    verdict=pass
    case "$out" in gate-verdict:*) ;; *) verdict=FAIL ;; esac
    [ "$(printf '%s\n' "$out" | wc -l | tr -d ' ')" -eq 1 ] || verdict=FAIL
    for k in $KEYS; do
      printf '%s' "$out" | grep -Eq "[:;] *$k=[^;]+" || verdict=FAIL
    done
    for pair in $(printf '%s' "$expected" | tr ';' ' '); do
      k=${pair%%=*}; want=${pair#*=}
      got=$(printf '%s' "$out" | sed -n "s/.*[:;] *$k=\([^;]*\).*/\1/p" | tr -d ' ')
      case "$want" in
        '>0') case "$got" in ''|*[!0-9]*|0) verdict=FAIL ;; esac ;;
        *) [ "$got" = "$want" ] || verdict=FAIL ;;
      esac
    done
    echo "$run_id" >> "$results_dir/$(if [ "$verdict" = pass ]; then echo pass; else echo fail; fi)"
    printf '%s\t%s\t%s\t%s\n' "$run_id" "$expected" "$verdict" "$(printf '%s' "$out" | tr '\t\n' '  ')"
  done
done < "$FIXTURES"

passed=$(wc -l < "$results_dir/pass" | tr -d ' ')
failed=$(wc -l < "$results_dir/fail" | tr -d ' ')
errors=$(wc -l < "$results_dir/error" | tr -d ' ')
echo "# gate-bench ($HOST${REF:+ @ $REF}): $passed pass, $failed fail, $errors errors; artifacts: $results_dir"
[ "$errors" -eq 0 ] || exit 2
[ "$failed" -eq 0 ]
