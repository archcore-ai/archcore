#!/bin/sh
set -eu

repo_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
source_root="$repo_root/plugin"
output=${1:?Usage: export-plugin.sh OUTPUT_DIRECTORY}

if [ -L "$output" ]; then
  echo "Output directory must not be a symlink: $output" >&2
  exit 1
fi
mkdir -p "$output"
if [ -n "$(ls -A "$output")" ]; then
  echo "Output directory must be empty: $output" >&2
  exit 1
fi

for directory in .agents .claude-plugin .cursor-plugin plugins; do
  cp -R "$source_root/$directory" "$output/$directory"
done
mkdir -p "$output/docs"
for file in TERMS.md cursor.mcp.example.json; do
  cp "$source_root/docs/$file" "$output/docs/$file"
done
cp "$repo_root/README.md" "$output/README.md"
for file in LICENSE NOTICE; do
  cp "$repo_root/$file" "$output/$file"
done

forbidden=$(find "$output" \( -type l -o -name .archcore -o -name AGENTS.md -o -name CLAUDE.md \
  -o -name .mcp.json -o -name .git -o -name .gitmodules -o -name .github \
  -o -name .claude -o -name .codex -o -name reference-materials -o -name test \) -print)
if [ -n "$forbidden" ]; then
  printf 'Development files found in plugin distribution:\n%s\n' "$forbidden" >&2
  exit 1
fi

pattern='\.archcore/(plugin|knowledge|vision|experience)/[a-zA-Z0-9_-]+\.(adr|spec|prd|plan|idea|rule|guide|doc|task-type|cpat|rfc|rnd|research|evidence|mrd|brd|urd|brs|strs|syrs|srs)\.md'
if grep -rEn "$pattern" "$output/plugins/archcore" "$output/README.md" \
    | grep -v -E '\.archcore/auth/jwt-strategy\.adr\.md' | grep .; then
  echo "Plugin distribution references internal project documents" >&2
  exit 1
fi

printf 'Plugin distribution: %s\n' "$output"
