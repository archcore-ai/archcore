#!/usr/bin/env bats
# The probe records table is complete and well-formed.
#
# host-adapter-contract.spec makes a dated probe record item 6 of what every
# adapter MUST provide, and repeats it in Conformance. Before
# host-probe-protocol.spec existed there was no protocol, no format and no
# records — the contract's most expensive requirement was also the only one
# nothing checked. This file makes it checkable.
#
# The host list is derived from host-coverage-matrix.bats rather than repeated
# here, so a fifth host cannot join the plugin without also owing a row.

setup() {
  load '../helpers/common'
  common_setup

  SPEC="$REPO_ROOT/.archcore/plugin/host-probe-protocol.spec.md"
  [ -f "$SPEC" ] || skip "host-probe-protocol.spec absent (stripped distribution)"
}

# Only the HTML-comment form delimits the table. The bare marker names also
# appear in the spec's prose, and matching those would start the range early
# and hand the parser a different table entirely.
records_block() {
  sed -n '/<!-- PROBE-RECORDS:BEGIN -->/,/<!-- PROBE-RECORDS:END -->/p' "$SPEC"
}

# Data rows only: pipe-delimited, not the header, not the separator.
records_rows() {
  records_block | awk -F'|' 'NF > 12 && $2 !~ /Date/ && $0 !~ /^\|-+/'
}

# Host ids as they appear in the table, derived from the hook config names the
# coverage matrix enrolls: hooks/copilot.hooks.json -> copilot,
# hooks/hooks.json -> claude-code.
enrolled_hosts() {
  local rel base
  while IFS='|' read -r rel _; do
    [ -n "$rel" ] || continue
    base=$(basename "$rel" .hooks.json)
    base=${base%.json}
    [ "$base" = "hooks" ] && base="claude-code"
    echo "$base"
  done < <(grep -A 6 '^matrix_rows()' "$REPO_ROOT/test/structure/host-coverage-matrix.bats" \
             | grep '^hooks/')
}

@test "the records section is delimited by both HTML markers" {
  grep -q '<!-- PROBE-RECORDS:BEGIN -->' "$SPEC" || fail "no PROBE-RECORDS:BEGIN marker"
  grep -q '<!-- PROBE-RECORDS:END -->' "$SPEC" || fail "no PROBE-RECORDS:END marker"
}

@test "the parser finds one row per enrolled host and nothing else" {
  # Guards the loop below against silently iterating an empty set — the way a
  # table-parsing test usually dies.
  local hosts rows
  hosts=$(enrolled_hosts | grep -c .)
  rows=$(records_rows | grep -c .)
  [ "$hosts" -ge 4 ] || fail "derived only $hosts hosts from the coverage matrix; expected >= 4"
  [ "$rows" = "$hosts" ] || fail "table has $rows data rows for $hosts enrolled hosts"
}

@test "every host enrolled in the coverage matrix has a probe row" {
  local host rows missing=""
  rows=$(records_rows | awk -F'|' '{ gsub(/^ +| +$/, "", $3); print $3 }')
  while IFS= read -r host; do
    [ -n "$host" ] || continue
    echo "$rows" | grep -qx "$host" || missing="$missing $host"
  done < <(enrolled_hosts)
  [ -z "$missing" ] || fail "hosts with no row in the probe records table:$missing"
}

@test "every probe outcome uses the documented vocabulary" {
  # Columns 7..12 are P0, A, A-d, B, C, D. An outcome outside the vocabulary
  # is how a record stops meaning anything: "ok", "works", "probably fine".
  local bad
  bad=$(records_rows | awk -F'|' '
    {
      for (i = 7; i <= 12; i++) {
        cell = $i
        gsub(/^ +| +$/, "", cell)
        if (cell == "pass" || cell == "fail" || cell == "—" || cell == "-" ||
            cell == "fail-open-confirmed" || cell == "fail-closed-observed" ||
            cell ~ /^n\/a:./ || cell ~ /^deferred:./) continue
        printf " [%s]", cell
      }
    }')
  [ -z "$bad" ] || fail "probe outcomes outside the documented vocabulary:$bad"
}

@test "no cell smuggles a pipe (it would silently shift every column after it)" {
  # The method column is the tempting place to write "install|log". A pipe in
  # a cell moves the outcome columns one to the right, and the vocabulary check
  # above would then be reading the wrong fields — passing on garbage.
  local wide
  wide=$(records_rows | awk -F'|' 'NF != 14 { print NR ": " NF " fields" }')
  [ -z "$wide" ] || fail "rows with an unexpected field count (escape pipes as '+'): $wide"
}
