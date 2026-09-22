#!/usr/bin/env bats
# Latency budget for the mutation hook launchers.
#
# Restored after ca6dfb4 removed the original. That version measured
# bin/check-archcore-write and bin/check-code-alignment, which v0.7.0 deleted
# when cli-owns-layers-4-5.adr moved the guard policy into the archcore binary.
# The budget did not move with the policy: every host still runs PreToolUse on
# a 2-second timeout (`timeout: 2` in hooks.json / cursor.hooks.json /
# codex.hooks.json, `timeoutSec: 2` in copilot.hooks.json) and PostToolUse on 4.
#
# Why the budget still matters after the split:
#
#   pre-tool-use   — the deny path. On Copilot a preToolUse timeout fails OPEN
#                    (host-adapter-contract.spec, Failure Behavior 6), so a
#                    write into .archcore/ goes through. A slow deny guard is a
#                    deny guard that does not deny, and nothing in any log says
#                    so.
#
#   post-tool-use  — validation, cascade, precision. Additive and non-blocking,
#                    so a timeout raises nothing at all: the mutation lands and
#                    no finding is reported. That silence is why this file
#                    exists.
#
# What is measured here is the PLUGIN's share — normalize-stdin, the
# plugin-cache guard, and the cli-gte version probe — with the CLI mocked to
# return instantly. That share is what this repository controls and what a
# refactor can regress. One test at the end measures the real CLI end to end
# and skips when it is absent.
#
# The corpus is shaped for the bad case: every document matches the edited
# path, because the cost of the OLD implementation tracked matches rather than
# corpus size. A launcher that never reads .archcore/ must stay flat across it
# — that flatness is the invariant test 3 pins.
#
# Thresholds are deliberately wide. CI runners are slower and noisier than a
# laptop, and this file must fail on an algorithmic regression rather than on a
# busy runner. Best-of-N absorbs a transient stall; the regressions worth
# guarding are factors of ten and clear any of these bars regardless.

setup() {
  load '../helpers/common'
  common_setup

  WORK_DIR="$BATS_TEST_TMPDIR/project"
  mkdir -p "$WORK_DIR/.archcore/domain"
  printf '{ "version": 1 }\n' > "$WORK_DIR/.archcore/settings.json"

  command -v perl >/dev/null 2>&1 || skip "perl needed for millisecond timing"
}

# A CLI that answers the version gate and then does nothing. Isolates the
# launcher's own cost from whatever the real binary spends.
make_instant_cli() {
  cat > "$MOCK_BIN/archcore" <<'MOCK'
#!/bin/sh
if [ "$1" = "--version" ]; then
  echo "archcore v0.7.0"
  exit 0
fi
exit 0
MOCK
  chmod +x "$MOCK_BIN/archcore"
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

# Best-of-$2 wall time in ms for launcher $1 given stdin $3. Honors PROBE_PATH
# when a test needs a restricted PATH.
best_ms() {
  local hook="$1" runs="$2" payload="$3"
  local best=999999 i start end elapsed
  for i in $(seq 1 "$runs"); do
    start=$(now_ms)
    ( cd "$WORK_DIR" && printf '%s' "$payload" \
        | env PATH="${PROBE_PATH:-$PATH}" "$PLUGIN_ROOT/bin/$hook" >/dev/null 2>&1 )
    end=$(now_ms)
    elapsed=$(( end - start ))
    [ "$elapsed" -lt "$best" ] && best=$elapsed
  done
  echo "$best"
}

SRC_WRITE='{"tool_name":"Write","tool_input":{"file_path":"src/api/handlers/users.ts"}}'
ARCHCORE_WRITE='{"tool_name":"Write","tool_input":{"file_path":".archcore/probe/x.adr.md"}}'
MCP_UPDATE='{"tool_name":"mcp__archcore__update_document","tool_input":{"path":".archcore/probe/x.adr.md"}}'

@test "pre-tool-use launcher overhead stays far inside the 2s PreToolUse budget" {
  make_instant_cli
  seed_corpus 200
  local ms
  ms=$(best_ms pre-tool-use 3 "$ARCHCORE_WRITE")
  [ "$ms" -lt 500 ] \
    || fail "pre-tool-use glue took ${ms}ms on 200 docs (budget 2000ms, bar 500ms) — on Copilot a timeout here fails open and the write lands"
}

@test "post-tool-use launcher overhead stays far inside the 4s PostToolUse budget" {
  make_instant_cli
  seed_corpus 200
  local ms
  ms=$(best_ms post-tool-use 3 "$MCP_UPDATE")
  [ "$ms" -lt 500 ] \
    || fail "post-tool-use glue took ${ms}ms on 200 docs (budget 4000ms, bar 500ms) — findings stop being reported, silently"
}

@test "launcher cost does not track knowledge-base size" {
  # The launcher must never read .archcore/ — the CLI does that. So an 8x
  # corpus must cost approximately nothing extra, and the assertion is on the
  # MARGINAL cost, not on a ratio.
  #
  # A ratio bar does not work here. The launcher carries a fixed cost of
  # roughly 50ms (two sourced libraries plus the cli-gte probe), and that floor
  # sits in the denominator: injecting a per-document grep measured 83ms at 25
  # docs and 309ms at 200 — a 226ms regression that shows up as only 3.7x and
  # slips under a 4x bar. Subtracting the floor makes the same defect a 226ms
  # growth against a 100ms bar, which it clears by more than 2x.
  make_instant_cli

  seed_corpus 25
  local small large growth
  small=$(best_ms pre-tool-use 3 "$SRC_WRITE")

  seed_corpus 200
  large=$(best_ms pre-tool-use 3 "$SRC_WRITE")

  growth=$(( large - small ))
  [ "$growth" -lt 100 ] \
    || fail "corpus size drives launcher cost: 25 docs = ${small}ms, 200 docs = ${large}ms (+${growth}ms, bar +100ms) — the glue is reading .archcore/ again"
}

@test "the no-CLI fail-open path is the cheapest path" {
  # No archcore on PATH at all. cli-gte answers __NO_CLI__ and both launchers
  # exit 0 without output. This is the path a machine without the CLI takes on
  # EVERY tool call, so it has to be close to free.
  seed_corpus 200
  PROBE_PATH="$MOCK_BIN:/usr/bin:/bin"
  local ms
  ms=$(best_ms pre-tool-use 3 "$ARCHCORE_WRITE")
  [ "$ms" -lt 300 ] \
    || fail "pre-tool-use took ${ms}ms with no CLI on PATH (bar 300ms) — the fail-open exit is doing work it should not"
}

@test "an empty knowledge base costs nothing" {
  make_instant_cli
  local ms
  ms=$(best_ms pre-tool-use 3 "$SRC_WRITE")
  [ "$ms" -lt 300 ] \
    || fail "pre-tool-use took ${ms}ms against an empty .archcore/ (bar 300ms) — the glue grew a startup cost"
}

@test "real CLI end to end stays inside Copilot's 2s fail-open budget" {
  # Everything above mocks the CLI, so none of it would catch a regression in
  # the binary itself. This one measures the whole path. It skips rather than
  # fails when the CLI is absent or too old, because that is a property of the
  # developer's machine, not of this repository.
  command -v archcore >/dev/null 2>&1 || skip "archcore CLI not on PATH"
  [ "$("$PLUGIN_ROOT/bin/cli-gte" 0.7.0)" = "yes" ] || skip "archcore CLI older than 0.7.0"

  seed_corpus 200
  ( cd "$WORK_DIR" && git init -q 2>/dev/null || true )
  local ms
  ms=$(best_ms pre-tool-use 3 "$ARCHCORE_WRITE")
  [ "$ms" -lt 1500 ] \
    || fail "pre-tool-use took ${ms}ms end to end on 200 docs (budget 2000ms, bar 1500ms) — Copilot's timeout fails open and the deny is lost"
}
