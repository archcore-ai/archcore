#!/usr/bin/env bats
# Contract specs for bin/check-references — the mechanical half of the
# actualize track's reference checks.
#
# Contract:
#   - always exit 0; outside a git work tree prints __NO_GIT__;
#   - reports a cited path that resolves under no root, with the commit that
#     last touched it (or never-tracked) and the new path when it was renamed;
#   - resolves against the project root and each manifest directory;
#   - skips fragments whose first segment exists under no root, elided,
#     globbed, URL, document-relative, and Archcore document tokens;
#   - skips documents with status rejected;
#   - reports one line-anchor line per document, with the count;
#   - prints unresolved lines before line-anchor lines.

setup() {
  load '../helpers/common'
  common_setup
  CHECK="$PLUGIN_ROOT/bin/check-references"
  [ -x "$CHECK" ] || fail "bin/check-references missing or not executable"
  PROJ="$BATS_TEST_TMPDIR/proj"
  mkdir -p "$PROJ/.archcore/area" "$PROJ/src/lib" "$PROJ/apps/web/src"
  cd "$PROJ" || fail "cd"
  git init -q .
  git config user.email t@example.com
  git config user.name t
  echo 'export const kept = 1;' > src/lib/kept.ts
  printf 'export const parser = 2;\nexport const lexer = 3;\n' > src/lib/old-name.ts
  echo 'export const gone = 4;' > src/lib/deleted.ts
  echo '{}' > apps/web/package.json
  echo 'export default 5;' > apps/web/src/page.tsx
  git add -A && git commit -qm init
  git mv src/lib/old-name.ts src/lib/new-name.ts
  git commit -qm move
  git rm -q src/lib/deleted.ts
  git commit -qm delete
}

write_doc() {
  # $1 = path under .archcore, $2 = status, $3 = body
  printf -- '---\ntitle: "T"\nstatus: %s\n---\n\n%s\n' "$2" "$3" > "$PROJ/.archcore/$1"
  git -C "$PROJ" add -A && git -C "$PROJ" commit -qm doc
}

@test "check-references: outside git prints __NO_GIT__" {
  run "$CHECK" "$BATS_TEST_TMPDIR"
  assert_success
  assert_output "__NO_GIT__"
}

@test "check-references: an existing path and a workspace-relative path are silent" {
  write_doc area/a.doc.md accepted 'See @src/lib/kept.ts and `src/page.tsx` and `apps/web/src/`.'
  run "$CHECK" "$PROJ"
  assert_success
  assert_output ""
}

@test "check-references: a renamed path names its new path" {
  write_doc area/a.doc.md accepted 'See @src/lib/old-name.ts for the parser.'
  run "$CHECK" "$PROJ"
  assert_success
  assert_output --regexp $'^unresolved\t.archcore/area/a.doc.md\tsrc/lib/old-name.ts\t[0-9a-f]+\tsrc/lib/new-name.ts$'
}

@test "check-references: a deleted path names the deleting commit" {
  write_doc area/a.doc.md accepted 'See `src/lib/deleted.ts`.'
  run "$CHECK" "$PROJ"
  assert_success
  assert_output --regexp $'^unresolved\t.archcore/area/a.doc.md\tsrc/lib/deleted.ts\t[0-9a-f]+\t-$'
}

@test "check-references: a path that never existed is never-tracked" {
  write_doc area/a.doc.md accepted 'Planned: `src/lib/future.ts`.'
  run "$CHECK" "$PROJ"
  assert_success
  assert_output $'unresolved\t.archcore/area/a.doc.md\tsrc/lib/future.ts\tnever-tracked\t-'
}

@test "check-references: fragments, elisions, globs, URLs, and document tokens are skipped" {
  write_doc area/a.doc.md accepted 'In `components/Button.tsx`, `src/.../x.ts`, `src/**/*.ts`, https://example.com/a/b.html, `../other/x.doc.md`, `area/b.adr`, @litres/errors.'
  run "$CHECK" "$PROJ"
  assert_success
  assert_output ""
}

@test "check-references: a rejected document is skipped" {
  write_doc area/a.adr.md rejected 'See `src/lib/deleted.ts`.'
  run "$CHECK" "$PROJ"
  assert_success
  assert_output ""
}

@test "check-references: a scope argument limits the documents checked" {
  mkdir -p "$PROJ/.archcore/other"
  write_doc area/a.doc.md accepted 'See `src/lib/deleted.ts`.'
  write_doc other/b.doc.md accepted 'See `src/lib/future.ts`.'
  run "$CHECK" "$PROJ" other
  assert_success
  assert_output $'unresolved\t.archcore/other/b.doc.md\tsrc/lib/future.ts\tnever-tracked\t-'
}

@test "check-references: line anchors are counted per document, after unresolved lines" {
  write_doc area/a.doc.md accepted 'See `src/lib/kept.ts:12`, `src/lib/kept.ts:20-30`, and @src/lib/deleted.ts#L4.'
  run "$CHECK" "$PROJ"
  assert_success
  assert_line --index 0 --regexp $'^unresolved\t.archcore/area/a.doc.md\tsrc/lib/deleted.ts\t'
  assert_line --index 1 $'line-anchor\t.archcore/area/a.doc.md\t3\tsrc/lib/kept.ts:12'
}
