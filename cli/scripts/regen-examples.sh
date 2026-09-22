#!/usr/bin/env bash
# regen-examples.sh — regenerate the instruction-nudge fixtures in examples/.
#
# Every AGENTS.md / CLAUDE.md under examples/ is a pure archcore managed-block
# file (no user content), shipped as a reference of what `archcore init` writes.
# They must track the REAL writer output byte-for-byte: the fixture invariant
# test (internal/agents/instructions_fixtures_test.go) fails whenever the nudge
# body changes until this script is re-run.
#
# Regeneration deletes the fixtures first: upsertFencedBlock deliberately never
# consumes a malformed (end-less) orphan block, so writing over a corrupted
# fixture would preserve the corruption instead of healing it.
#
# Usage: ./scripts/regen-examples.sh   (from the repo root)

set -euo pipefail

cd "$(dirname "$0")/.."

# Every example project root = the parent of a .archcore/settings.json.
# examples/_global_/* are shared global-SOURCE repos (mounted read-only by the
# other examples), not agent-wired projects — they carry no nudge files.
find examples -type f -path '*/.archcore/settings.json' -not -path 'examples/_global_/*' | sort | while read -r settings; do
  project="$(dirname "$(dirname "$settings")")"
  echo "==> $project"
  rm -f "$project/AGENTS.md" "$project/CLAUDE.md"
  # claude-code is the multi-target agent: one install writes both files.
  go run . instructions install --agent claude-code --project "$project" >/dev/null
done

echo "Done. Verify with: go test ./internal/agents/ -run TestShippedInstructionFixtures"
