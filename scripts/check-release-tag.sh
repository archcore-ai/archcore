#!/bin/sh
set -eu

# The release tag is the only version source, so a mistyped tag is the one
# version error left to catch. A release tag must be the direct successor of
# the highest other release tag: the next patch, minor, or major version.

tag=${1:?Usage: check-release-tag.sh TAG}
semver='^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'

if ! printf '%s\n' "$tag" | grep -Eq "$semver"; then
  echo "Release tag $tag is not vMAJOR.MINOR.PATCH" >&2
  exit 1
fi

# The tag under check is excluded, so a re-run of its own release still passes.
previous=$(git tag --list 'v*' --sort=-v:refname | grep -E "$semver" | grep -vxF "$tag" | head -n 1)
if [ -z "$previous" ]; then
  printf 'Release tag %s is the first release\n' "$tag"
  exit 0
fi

IFS=. read -r major minor patch <<EOF
${previous#v}
EOF
next_patch="$major.$minor.$((patch + 1))"
next_minor="$major.$((minor + 1)).0"
next_major="$((major + 1)).0.0"

case "${tag#v}" in
  "$next_patch" | "$next_minor" | "$next_major") ;;
  *)
    echo "Release tag $tag does not follow $previous; expected v$next_patch, v$next_minor, or v$next_major" >&2
    exit 1
    ;;
esac

printf 'Release tag %s follows %s\n' "$tag" "$previous"
