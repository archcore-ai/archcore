---
name: bump-plugin-version
argument-hint: "[patch | minor | major | X.Y.Z]  (default: patch)"
description: Bump the archcore plugin version across all three host manifests (.claude-plugin, .cursor-plugin, .codex-plugin) in one step, keeping them byte-identical. Derives the next version from the latest git tag. Triggers on /bump-plugin-version. Edits files only — merging, tagging, and publishing stay manual per docs/release.md.
disable-model-invocation: true
model: haiku
---

# /bump-plugin-version

Bump the plugin `version` in all three per-host manifests at once, following the
canonical pattern in the `bump-plugin-version` cpat. The next version is derived
from the **latest git tag** (the release source of truth), not the manifest.

## Hard constraints (non-negotiable)

- Edit ONLY the three `plugin.json` files listed in Step 1. Nothing else in the
  repo carries the plugin version — not the marketplace catalogs, MCP configs, or
  hook JSON (`cursor.hooks.json`'s `"version": 1` is a schema version, not this).
- Use the `Edit` tool for a targeted `"version": "<OLD>"` → `"version": "<NEXT>"`
  swap, one file at a time. NEVER rewrite with `jq`, `Write`, `sed -i`, or any tool
  that could reorder keys or reformat — the files must stay byte-identical except
  the version substring.
- NEVER touch `name` or `description` — cross-host parity depends on them.
- NEVER run `git add`, `git commit`, `git tag`, or `git push`. This skill edits
  files only. Stop after verification and print the manual next steps.

## When to use

- Cutting a release: raise the plugin version before tagging.
- Invoked explicitly as `/bump-plugin-version [step]`. Not model-auto-invoked.

**Argument** (optional, default `patch`):

| Arg     | Effect                                                     |
| ------- | ---------------------------------------------------------- |
| `patch` | `x.y.z → x.y.(z+1)` — default                              |
| `minor` | `x.y.z → x.(y+1).0`                                        |
| `major` | `x.y.z → (x+1).0.0`                                        |
| `X.Y.Z` | explicit version, written verbatim (must exceed the base) |

## Execution

### Step 1 — Load the pattern (reuse the cpat)

Read the canonical pattern so this run stays aligned with it:

`mcp__archcore__get_document(path=".archcore/plugin/bump-plugin-version.cpat.md")`

The cpat is authoritative for **which files change** and **what must not**. The
three manifests (plugin root is `plugins/archcore/`):

- `plugins/archcore/.claude-plugin/plugin.json`
- `plugins/archcore/.cursor-plugin/plugin.json`
- `plugins/archcore/.codex-plugin/plugin.json`

### Step 2 — Resolve the version and compute the next one

Run this as **one** `Bash` call — it discovers, validates, and computes in a single
shell so no state leaks between calls. Replace the `ARG=` value with the user's
argument (or leave `patch`):

```bash
ARG="patch"   # ← replace with the text after /bump-plugin-version: patch | minor | major | X.Y.Z

extract_ver() { grep -oE '"version" *: *"[0-9]+\.[0-9]+\.[0-9]+"' "$1" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+'; }

LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null)
if [ -n "$LATEST_TAG" ]; then
  BASE="${LATEST_TAG#v}"
else
  LATEST_TAG="(no tags)"
  BASE=$(extract_ver plugins/archcore/.claude-plugin/plugin.json)
  BASE="${BASE:-0.0.0}"
  echo "NOTE: no git tags found; falling back to manifest version $BASE"
fi

case "$BASE" in
  *[!0-9.]* | . | *..* ) echo "ERROR: base '$BASE' is not a plain X.Y.Z version" >&2; exit 1 ;;
esac
printf '%s\n' "$BASE" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' || { echo "ERROR: base '$BASE' is not X.Y.Z" >&2; exit 1; }

MA=${BASE%%.*}; r=${BASE#*.}; MI=${r%%.*}; PA=${r##*.}

case "$ARG" in
  major) NEXT="$((10#$MA + 1)).0.0" ;;
  minor) NEXT="$((10#$MA)).$((10#$MI + 1)).0" ;;
  patch) NEXT="$((10#$MA)).$((10#$MI)).$((10#$PA + 1))" ;;
  *)
    printf '%s\n' "$ARG" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' \
      || { echo "ERROR: invalid argument '$ARG' — expected patch|minor|major|X.Y.Z" >&2; exit 1; }
    NEXT="$ARG"
    NMA=${ARG%%.*}; nr=${ARG#*.}; NMI=${nr%%.*}; NPA=${nr##*.}
    if [ "$((10#$NMA))" -lt "$((10#$MA))" ] \
       || { [ "$((10#$NMA))" -eq "$((10#$MA))" ] && [ "$((10#$NMI))" -lt "$((10#$MI))" ]; } \
       || { [ "$((10#$NMA))" -eq "$((10#$MA))" ] && [ "$((10#$NMI))" -eq "$((10#$MI))" ] && [ "$((10#$NPA))" -le "$((10#$PA))" ]; }; then
      echo "ERROR: explicit version $ARG is not greater than base $BASE" >&2; exit 1
    fi
    ;;
esac

echo "OLD versions:"; extract_ver plugins/archcore/.claude-plugin/plugin.json
extract_ver plugins/archcore/.cursor-plugin/plugin.json
extract_ver plugins/archcore/.codex-plugin/plugin.json
echo "$LATEST_TAG (base $BASE) -> NEXT $NEXT"
```

If the manifests already disagree (drift), note it — the bump reconciles them all
to `NEXT`. State `NEXT` explicitly before editing anything.

### Step 3 — Write NEXT to all three manifests

For **each** of the three files, use `Edit` to replace that file's current version
line — `"version": "<OLD>"` → `"version": "<NEXT>"`. Only the version substring
changes; formatting, key order, `name`, and `description` stay untouched. If a
file's `<OLD>` differs from the others (drift), still set it to `NEXT` — that is the
reconciliation.

### Step 4 — Verify parity

All three MUST now read the same `NEXT` — check by equality, not by eye:

```bash
extract_ver() { grep -oE '"version" *: *"[0-9]+\.[0-9]+\.[0-9]+"' "$1" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+'; }
V1=$(extract_ver plugins/archcore/.claude-plugin/plugin.json)
V2=$(extract_ver plugins/archcore/.cursor-plugin/plugin.json)
V3=$(extract_ver plugins/archcore/.codex-plugin/plugin.json)
if [ "$V1" = "$V2" ] && [ "$V2" = "$V3" ]; then echo "PARITY OK: $V1"
else echo "DRIFT: claude=$V1 cursor=$V2 codex=$V3" >&2; exit 1; fi
```

Then run the test-enforced invariant (this is the CI check the skill exists to keep
green; run it, don't skip it — only note "bats unavailable" if the binary is missing):

```bash
bats test/structure/json-configs.bats test/structure/codex-plugin.bats
```

## Result

Report in this shape:

```
## Version Bump

- Latest tag: <LATEST_TAG or "(no tags)">
- Step: <patch | minor | major | explicit X.Y.Z>
- Version: <BASE> -> <NEXT>
- Drift reconciled: <none | claude=<v> cursor=<v> codex=<v> → unified to <NEXT>>
- Parity: all three manifests now read "<NEXT>" (bats: <pass | not run>)

Manual next steps (not run):
1. Review & commit the bump on dev.
2. Merge the bump to dev.
3. git tag v<NEXT> && git push origin v<NEXT>
   (release.yml then synthesizes main and publishes the GitHub Release)
```

Stop there — tagging and publishing are the user's call (see `docs/release.md`).
