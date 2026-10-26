#!/bin/sh
# Builds the publishable tree of @archcore/opencode-plugin (opencode-adapter.spec):
# bin/ and agents/ copied from plugins/archcore/, skills/ copied with each SKILL.md
# name: line prefixed archcore- (opencode-archcore-prefixed-skill-names.adr), and
# dist/index.js bundled by bun build (opencode-plugin-ships-built-javascript.adr).
set -eu

pkg=$(cd "$(dirname "$0")/.." && pwd)
src="$pkg/../archcore"

rm -rf "${pkg:?}/bin" "${pkg:?}/skills" "${pkg:?}/agents" "${pkg:?}/dist"

cp -R "$src/bin" "$pkg/bin"
cp -R "$src/skills" "$pkg/skills"
mkdir "$pkg/agents"
cp "$src"/agents/*.md "$pkg/agents/"

for skill in "$pkg"/skills/*/SKILL.md; do
  awk 'NR == 1 && $0 == "---" { fm = 1; print; next }
       fm && $0 == "---" { fm = 0 }
       fm && /^name: / { sub(/^name: /, "name: archcore-") }
       { print }' "$skill" > "$skill.tmp"
  mv "$skill.tmp" "$skill"
done

bun build "$pkg/src/index.ts" --outdir "$pkg/dist" --target node --format esm
