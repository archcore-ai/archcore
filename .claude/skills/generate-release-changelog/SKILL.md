---
name: generate-release-changelog
description: Generate a polished changelog for the latest Git tag. Classifies commits by impact, adds product and technical perspective, and prints the result directly in chat. Triggers on /generate-release-changelog or requests to generate/draft a changelog for the latest release.
---

# Generate Release Changelog

Produce a clean, human-readable changelog for the latest published Git tag and print it as the final assistant message. This skill is read-only — do not write, edit, or create any files, directories, or git/GitHub objects at any point.

## Hard constraints (non-negotiable)

- No file writes: do not use Write, Edit, Bash mkdir/touch, or any equivalent.
- No remote writes: do not run git commit, git tag, git push, or gh release create/edit.
- Output is a single chat message in Markdown. No preamble ("Here's your changelog:"), no follow-up questions, no commentary after the changelog ends.
- If the commit range is empty, print a single line: `No commits found between <previous> and <latest>.` and stop.

## Step 1 — Resolve the tag range

Run in parallel:

```bash
git tag --sort=-creatordate | head -5
git describe --tags --abbrev=0
```

- `latest` = output of `git describe --tags --abbrev=0`.
- `previous` = the next tag in the sorted list. If no previous tag exists, use `git rev-list --max-parents=0 HEAD | head -1` and note "initial release" in the changelog.
- If the user provides a tag argument, treat that as `latest`; derive `previous` from the sorted list.

## Step 2 — Collect commits

```bash
git log <previous>..<latest> --no-merges --pretty=format:'%H%x09%s%x09%an'
```

For each commit, capture changed files:

```bash
git show --stat --pretty=format:'' <sha> | head -40
```

For commits with ambiguous subjects (e.g., `chore: misc`, `wip`, `update`), inspect the diff only for paths that look load-bearing — skip obvious `docs:` / `chore:` commits:

```bash
git show <sha> -- <relevant-paths>
```

## Step 3 — Classify commits

Map every commit to exactly one bucket. Conventional Commit prefixes are a hint; the diff is authoritative — a `chore:` commit that ships user-visible behavior belongs in Features.

Buckets (use this order in the output):

1. **Breaking changes** — backward-incompatible changes to config schema, flags, output format, file layout, or API contracts.
2. **Features** — new user-visible capability.
3. **Improvements** — enhancements to existing behavior (UX, performance, validation, compatibility).
4. **Fixes** — corrections to incorrect behavior.
5. **Documentation** — README, docs, in-repo `.archcore/` docs.
6. **Internal** — refactors, tests, build, CI, deps. Keep terse; omit entirely if nothing notable.

Skip purely cosmetic commits (whitespace, typo) unless they are the only changes in the release.

## Step 4 — Write each bullet

For **Breaking changes, Features, Improvements, and Fixes**: write one sentence that leads with the user-visible outcome and, when it adds clarity, names the concrete mechanism (flag, command, config field, file).

Avoid echoing the commit subject. Rewrite for a reader who has not seen the code.

Bad: `feat: improve working with cwd`
Good: `archcore commands now resolve .archcore/ from the nearest parent directory, so you can run them from anywhere inside the repo.`

For **Internal** entries, one terse line is fine. No product framing required.

## Step 5 — Output format

Emit exactly this structure (omit any section that would be empty):

```markdown
# <latest-tag>

_Released YYYY-MM-DD · <N> commits since <previous-tag>_

<2–4 sentence product summary. Lead with the most impactful change. No bullets.>

## Breaking changes
- ...

## Features
- ...

## Improvements
- ...

## Fixes
- ...

## Documentation
- ...

## Internal
- ...

---
**Full diff:** `<previous-tag>...<latest-tag>`
```

- Release date: `git log -1 --format=%ai <latest-tag>` (date portion only).
- No commit SHAs in bullets.
- No author names.
- Do not invent items not backed by a commit in the range.
- The summary paragraph is mandatory.
