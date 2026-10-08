#!/bin/sh
# Behavioral review durable-context bench: what a branch review writes, omits, and asks. Runs model calls only when explicitly requested.
# REVIEW_BENCH_HOST selects claude (default), codex, or copilot (binary: REVIEW_BENCH_COPILOT, default copilot).
# REVIEW_BENCH_MODEL selects the model; REVIEW_BENCH_LIMIT limits fixture count; REVIEW_BENCH_REPS repeats each fixture.
# REVIEW_BENCH_REF reads the contracts from a git ref (for example a release tag) instead of the working tree.
# REVIEW_BENCH_FIXTURES overrides input; REVIEW_BENCH_OUTPUT_DIR retains raw replies.
# Exit 1: verdict/format mismatch. Exit 2: invalid inputs or CLI failure.
set -eu

REPO_ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd)
FIXTURES=${REVIEW_BENCH_FIXTURES:-$REPO_ROOT/test/behavioral/fixtures/review-bench.tsv}
SHARED="$REPO_ROOT/plugins/archcore/skills"
CONTRACTS="review/SKILL.md _shared/durable-context-selection.md _shared/tracks/closeout.md _shared/tracks/experience.md _shared/elicitation-contract.md _shared/verdict-contract.md"
KEYS="created updated omitted questions plan"
HOST=${REVIEW_BENCH_HOST:-claude}
COPILOT=${REVIEW_BENCH_COPILOT:-copilot}
REF=${REVIEW_BENCH_REF:-}
TAB=$(printf '\t')

case "$HOST" in
  claude) host_bin=claude ;;
  codex) host_bin=codex ;;
  copilot) host_bin=$COPILOT ;;
  *) echo "REVIEW_BENCH_HOST must be claude, codex, or copilot" >&2; exit 2 ;;
esac
[ -f "$FIXTURES" ] || { echo "missing fixtures: $FIXTURES" >&2; exit 2; }
command -v "$host_bin" >/dev/null 2>&1 || { echo "$HOST CLI not found: $host_bin" >&2; exit 2; }
command -v jq >/dev/null 2>&1 || { echo "jq not found on PATH" >&2; exit 2; }
LIMIT=${REVIEW_BENCH_LIMIT:-0}
case "$LIMIT" in ''|*[!0-9]*) echo "REVIEW_BENCH_LIMIT must be a nonnegative integer" >&2; exit 2 ;; esac
REPS=${REVIEW_BENCH_REPS:-1}
case "$REPS" in ''|*[!0-9]*|0) echo "REVIEW_BENCH_REPS must be a positive integer" >&2; exit 2 ;; esac
# expected: key=value pairs joined by ';' — a fixture pins only the keys it names.
if ! awk -F '\t' '
  /^#/ || /^[[:space:]]*$/ {next}
  NF != 3 || $1 == "" || $2 == "" || seen[$1]++ {bad=1}
  { pair = "(created=(none|[a-z-]+(,[a-z-]+)*)|(updated|omitted|questions)=(>0|[0-9]+)|plan=(offer|retain|none))" }
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
--- CONTRACT: skills/$c ---
$body"
done
if [ -n "${REVIEW_BENCH_OUTPUT_DIR:-}" ]; then
  results_dir=$REVIEW_BENCH_OUTPUT_DIR
  mkdir -p "$results_dir"
else
  results_dir=$(mktemp -d "${TMPDIR:-/tmp}/archcore-review-bench.XXXXXX") || exit 2
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
      if [ -n "${REVIEW_BENCH_MODEL:-}" ]; then set -- "$@" --model "$REVIEW_BENCH_MODEL"; fi
      (cd "$results_dir" && claude "$@" < "$b.prompt") > "$b.json" 2> "$b.stderr" || return 1
      jq -e '.is_error == false and (.result | type == "string")' "$b.json" >/dev/null 2>&1 || return 1
      jq -r '.result' "$b.json" > "$b.out" ;;
    codex)
      set -- exec --skip-git-repo-check --ephemeral -s read-only -o "$b.out"
      if [ -n "${REVIEW_BENCH_MODEL:-}" ]; then set -- "$@" -m "$REVIEW_BENCH_MODEL"; fi
      (cd "$results_dir" && codex "$@" - < "$b.prompt") > "$b.log" 2> "$b.stderr" || return 1
      [ -s "$b.out" ] ;;
    copilot)
      set -- -s --no-custom-instructions --available-tools= --disable-builtin-mcps --no-ask-user
      if [ -n "${REVIEW_BENCH_MODEL:-}" ]; then set -- "$@" --model "$REVIEW_BENCH_MODEL"; fi
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
      "You are the review skill running a plain branch review (/archcore:review, no arguments) under the contracts below. Apply them literally." \
      "The situation is already established by reading the branch. The user answers every question with the recommended option and authorizes every proposed write." \
      "Output EXACTLY one line in this format and nothing else:" \
      "review-verdict: created=<types or none>; updated=<integer>; omitted=<integer>; questions=<integer>; plan=<offer|retain|none>" \
      "created: the document types this invocation creates, comma-separated in alphabetical order without spaces, a type repeated once per document, or none. updated: how many existing documents this invocation updates. omitted: how many candidate units the selection omits. questions: how many questions this invocation asks the user, confirmations included. plan: offer when a matched plan reaches its removal confirmation, retain when a matched plan stays, none when no plan matched." \
      "Write each count as digits (0 when none). Do not write the documents or explain." \
      "$contracts_text" \
      "--- SITUATION ---" "$situation" \
      > "$base.prompt"
    if ! ask "$base"; then
      printf '%s\t%s\tERROR\t%s CLI failed; see %s.stderr\n' "$run_id" "$expected" "$HOST" "$base"
      echo "$run_id" >> "$results_dir/error"
      continue
    fi
    out=$(sed '/^[[:space:]]*$/d' "$base.out")
    case "$out" in '`review-verdict:'*'`') out=${out#'`'}; out=${out%'`'} ;; esac
    verdict=pass
    case "$out" in review-verdict:*) ;; *) verdict=FAIL ;; esac
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
echo "# review-bench ($HOST${REF:+ @ $REF}): $passed pass, $failed fail, $errors errors; artifacts: $results_dir"
[ "$errors" -eq 0 ] || exit 2
[ "$failed" -eq 0 ]
