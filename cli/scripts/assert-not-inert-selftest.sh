#!/bin/sh
# Proves scripts/assert-not-inert.sh rejects the failure it exists to catch.
#
# That script is the release pipeline's only guard against shipping a binary
# that never self-updates and never reports. It had no test, and its first
# version was inert against the exact case its own header describes: `go build`
# records the whole `-ldflags` argument in the binary's build info, so grepping
# for the injected value matched the command line whether or not the `-X` symbol
# path named a real variable. A misspelled path passed the check.
#
# A guard with no test is a claim, not a guarantee. This builds a binary with a
# deliberately misspelled path and requires the guard to say no.
#
# Run from the repository root. Needs the Go toolchain; builds into a temp dir.

set -eu

cd "$(dirname "$0")/.."

guard="scripts/assert-not-inert.sh"
marker="archcore-official-build"
key="phc_SELFTESTKEY0000000000"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT INT TERM

failures=0

# check <name> <want-exit> <env-marker> <env-key> <binary>
check() {
    name=$1 want=$2 m=$3 k=$4 bin=$5
    set +e
    ARCHCORE_OFFICIAL_BUILD="$m" POSTHOG_KEY="$k" sh "$guard" "$bin" >"$work/out" 2>&1
    got=$?
    set -e
    if [ "$got" -eq "$want" ]; then
        echo "ok   $name (exit $got)"
    else
        echo "FAIL $name: want exit $want, got $got"
        sed 's/^/       /' "$work/out"
        failures=$((failures + 1))
    fi
}

build() {
    out=$1
    shift
    go build -ldflags "$*" -o "$out" .
}

echo "building probes..."
build "$work/good" \
    "-s -w -X archcore-cli/internal/telemetry.apiKey=$key -X archcore-cli/internal/update.officialBuild=$marker"
build "$work/bad-marker" \
    "-s -w -X archcore-cli/internal/telemetry.apiKey=$key -X archcore-cli/internal/update.officialBuiId=$marker"
build "$work/bad-key" \
    "-s -w -X archcore-cli/internal/telemetry.apiKeyy=$key -X archcore-cli/internal/update.officialBuild=$marker"
go build -o "$work/plain" .

# The one that matters: a real release build passes, and the same build with one
# symbol path misspelled does not. `go tool link` reports no error either way.
check "a correctly injected build passes"          0 "$marker" "$key" "$work/good"
check "a misspelled marker path is rejected"       1 "$marker" "$key" "$work/bad-marker"
check "a misspelled key path is rejected"          1 "$marker" "$key" "$work/bad-key"

# A build with no ldflags at all, which is what a fork's pipeline and a local
# `goreleaser build` produce. Inert is correct there and must not fail a release.
check "an unofficial build is skipped, not failed" 0 ""        ""    "$work/plain"

# The official pipeline exporting both values but losing the ldflags line.
check "a dropped ldflags line is rejected"         1 "$marker" "$key" "$work/plain"

# Half the wiring is never intentional.
check "a marker without a key is rejected"         1 "$marker" ""    "$work/good"
check "a key without a marker is rejected"         1 ""        "$key" "$work/good"

if [ "$failures" -ne 0 ]; then
    echo "assert-not-inert-selftest: $failures case(s) failed" >&2
    exit 1
fi
echo "assert-not-inert-selftest: all cases passed"
