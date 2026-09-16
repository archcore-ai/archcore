#!/usr/bin/env bats
# Structure test: the track-file line cap from plugin-architecture.spec.
#
# A track file hosts one gate record per instrument (sdd hosts six), and a gate
# record in the gate-contract template costs 30–45 lines, so the cap is 300 —
# recorded in track-file-line-cap-300.adr. A file past the cap is a signal to
# route prose out of the track file (Track notes → a shared contract) or to
# split the track, never to compress a gate record.

setup() {
  load '../helpers/common'
  common_setup
  TRACKS_DIR="$PLUGIN_ROOT/skills/_shared/tracks"
  CAP=300
}

@test "every track file stays at or under the 300-line cap" {
  local over="" f n
  for f in "$TRACKS_DIR"/*.md; do
    n=$(wc -l < "$f" | tr -d ' ')
    [ "$n" -le "$CAP" ] || over="$over $(basename "$f")=$n"
  done
  [ -z "$over" ] || fail "track files over the ${CAP}-line cap:$over"
}

@test "the cap in plugin-architecture.spec matches this test" {
  grep -F -q 'a track file MUST NOT exceed 300 lines' "$REPO_ROOT/.archcore/plugin/plugin-architecture.spec.md" \
    || fail ".archcore/plugin/plugin-architecture.spec.md does not state the 300-line track-file cap"
}
