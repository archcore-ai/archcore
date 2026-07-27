#!/usr/bin/env bats
# Latency budget for the PreToolUse hooks.
#
# Every host runs these two on a 1-second timeout (`timeout: 1` in hooks.json /
# cursor.hooks.json / codex.hooks.json, `timeoutSec: 1` in copilot.hooks.json),
# and the two run as SEPARATE entries, so each gets its own budget.
#
# The two have very different failure modes when they blow it:
#
#   check-archcore-write  — the deny guard. A timeout here is a correctness
#                           problem: on Copilot a preToolUse timeout fails OPEN
#                           (host-adapter-contract.spec), so a write into
#                           .archcore/ would go through. It does almost no work,
#                           and this test keeps it that way.
#
#   check-code-alignment  — context injection. Additive and non-blocking, so a
#                           timeout raises nothing at all: the write proceeds and
#                           no context is injected. That silence is exactly why
#                           this test exists. The hook once cost ~6 ms per
#                           MATCHING document (two process spawns each, per
#                           token), so a knowledge base where ~170 documents
#                           mentioned a common source root blew the whole budget
#                           — and push-mode simply stopped working, on precisely
#                           the repositories with the most context to give, with
#                           nothing in any log.
#
# The corpus below is shaped for the bad case: every document matches, because
# cost tracked matches rather than corpus size. Thresholds are deliberately
# generous — CI runners are slower and noisier than a laptop, and this test must
# fail on an algorithmic regression, not on a busy runner. Best-of-N absorbs a
# transient stall; a real regression is a factor of ten and clears any of these
# bars regardless.

setup() {
  load '../helpers/common'
  common_setup

  WORK_DIR="$BATS_TEST_TMPDIR/project"
  mkdir -p "$WORK_DIR/.archcore/domain"

  command -v perl >/dev/null 2>&1 || skip "perl needed for millisecond timing"
}

# Build a corpus of $1 documents, every one of them matching src/api/handlers/.
seed_corpus() {
  local n="$1" i type
  for i in $(seq 1 "$n"); do
    case $(( i % 5 )) in
      0) type=rule ;; 1) type=adr ;; 2) type=spec ;; 3) type=guide ;; *) type=cpat ;;
    esac
    cat > "$WORK_DIR/.archcore/domain/doc-$i.$type.md" <<EOF
---
title: Document $i
status: accepted
---

Applies to src/api/handlers/ and more broadly to src/api/ and src/.
EOF
  done
}

now_ms() { perl -MTime::HiRes=time -e 'printf "%.0f", time*1000'; }

# Best-of-$2 wall time in ms for hook $1 given stdin $3.
best_ms() {
  local hook="$1" runs="$2" payload="$3"
  local best=999999 i start end elapsed
  for i in $(seq 1 "$runs"); do
    start=$(now_ms)
    ( cd "$WORK_DIR" && printf '%s' "$payload" | "$PLUGIN_ROOT/bin/$hook" >/dev/null 2>&1 )
    end=$(now_ms)
    elapsed=$(( end - start ))
    [ "$elapsed" -lt "$best" ] && best=$elapsed
  done
  echo "$best"
}

SRC_WRITE='{"tool_name":"Write","tool_input":{"file_path":"src/api/handlers/users.ts"}}'
ARCHCORE_WRITE='{"tool_name":"Write","tool_input":{"file_path":".archcore/probe/x.adr.md"}}'

@test "check-code-alignment stays well inside its 1s budget on 100 matching docs" {
  seed_corpus 100
  local ms
  ms=$(best_ms check-code-alignment 3 "$SRC_WRITE")
  [ "$ms" -lt 500 ] \
    || fail "check-code-alignment took ${ms}ms on 100 matching docs (budget 1000ms, bar 500ms) — injection will silently stop on large knowledge bases"
}

@test "check-code-alignment cost does not track the number of matches" {
  # The regression this guards is algorithmic: per-match process spawns. With
  # them, going 25 -> 200 matching documents was an ~8x jump; without them the
  # work is a fixed number of greps and the curve is nearly flat. A 4x bar is
  # far looser than the regression and far tighter than the fix.
  seed_corpus 25
  local small large
  small=$(best_ms check-code-alignment 3 "$SRC_WRITE")

  seed_corpus 200
  large=$(best_ms check-code-alignment 3 "$SRC_WRITE")

  # Guard against a near-zero denominator on a very fast machine.
  [ "$small" -lt 5 ] && small=5
  [ "$large" -lt $(( small * 4 )) ] \
    || fail "matches drive cost: 25 docs = ${small}ms, 200 docs = ${large}ms (>4x) — per-match process spawns are back"
}

@test "check-archcore-write answers fast enough that Copilot's fail-open stays unreachable" {
  # This one actually blocks, and on Copilot a preToolUse timeout fails OPEN —
  # a slow deny guard is a deny guard that does not deny.
  seed_corpus 200
  local ms
  ms=$(best_ms check-archcore-write 3 "$ARCHCORE_WRITE")
  [ "$ms" -lt 300 ] \
    || fail "check-archcore-write took ${ms}ms (budget 1000ms, bar 300ms) — on Copilot a timeout here lets the write through"
}

@test "an empty knowledge base costs nothing" {
  # No corpus at all: the hook must bail on the .archcore/ check, not scan.
  local ms
  ms=$(best_ms check-code-alignment 3 "$SRC_WRITE")
  [ "$ms" -lt 300 ] \
    || fail "check-code-alignment took ${ms}ms with an empty .archcore/ — the early exit is gone"
}
