#!/bin/sh
set -eu

repo_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
source_root="$repo_root/plugin"
output=${1:?Usage: export-plugin.sh OUTPUT_DIRECTORY VERSION}
version=${2:?Usage: export-plugin.sh OUTPUT_DIRECTORY VERSION}

if ! printf '%s\n' "$version" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'; then
  echo "Version must be MAJOR.MINOR.PATCH: $version" >&2
  exit 1
fi

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

# The release tag is the only version source: dev manifests carry 0.0.0, and
# only the top-level version line changes, so the rest stays byte-identical.
for host in .claude-plugin .cursor-plugin .codex-plugin .plugin; do
  manifest="$output/plugins/archcore/$host/plugin.json"
  sed 's/^  "version": "[^"]*"/  "version": "'"$version"'"/' "$manifest" > "$manifest.tmp"
  mv "$manifest.tmp" "$manifest"
  if [ "$(jq -r .version "$manifest")" != "$version" ]; then
    echo "Version was not written to $manifest" >&2
    exit 1
  fi
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
