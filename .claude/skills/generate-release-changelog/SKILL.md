---
name: generate-release-changelog
description: Generate a polished changelog for the latest Git tag. Classifies commits by impact, adds product and technical perspective, and prints the result directly in chat. Triggers on /generate-release-changelog or requests to generate/draft a changelog for the latest release.
---

# Generate Release Changelog

Produce a tight, human-readable changelog for the latest published Git tag and print it as the final assistant message. This skill is read-only — do not write, edit, or create any files, directories, or git/GitHub objects at any point.

The product owner reads this to understand what changed for users, not what was implemented. Optimize for **impact density**: every bullet must answer "what is different now from a user's seat?". Cut anything that does not.

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

Drop everything else. Do not include sections for documentation, refactors, tests, CI, dependency bumps, or internal cleanup. The only exception: if a commit changes something downstream users will feel — minimum Go version bump, removed dependency, security CVE — surface it under Breaking changes or Improvements with a product-facing sentence.

Skip purely cosmetic commits (whitespace, typo).

Merge near-duplicates into one bullet. Two commits that fix the same bug get one line. Three commits that build one feature get one line.

## Step 4 — Write each bullet

**Strict rules — apply to every bullet:**

- One sentence. **Maximum 15 words.** Aim for 8–12.
- Lead with the user-visible state, not the work performed. Describe the new behavior as it now is.
- Name the concrete surface (flag, command, config field, file) only when it is the thing the user touches. Drop it if the impact is clearer without it.
- No filler: cut "now", "you can now", "users can", "we have", "this release", "added support for", "improved the way", "made it possible to".
- No commit-message echoes. Rewrite for a reader who has not seen the code.
- Active voice. Present tense.
- No commit SHAs, no author names, no PR/issue numbers unless they carry meaning the reader needs.

**Examples (calibrate against these):**

| Avoid                                                                                                             | Prefer                                                      |
| ----------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| `feat: improve working with cwd`                                                                                  | `Commands run from any subdirectory inside the repo.`       |
| `Added support for storing local document relations in .sync-state.json so they survive across sync runs.`        | `Local relations persist across syncs.`                     |
| `Fixed a bug where the init command would fail if the .archcore directory already existed with a partial config.` | ``init` no longer fails on partial existing configs.`       |
| `Improved the validation logic for sync mode fields to provide clearer error messages.`                           | `Sync-mode validation errors point at the offending field.` |
| `BREAKING: The project field has been removed from the configuration schema.`                                     | ``project` field removed from `settings.json`.`             |

If you cannot write a bullet that meets these rules, the change probably does not belong in the changelog — drop it.

## Step 5 — Output format

Emit exactly this structure (omit any section that would be empty):

```markdown
# <latest-tag>

_<YYYY-MM-DD> · <N> commits_

<One sentence, max 20 words. The headline of the release in user-impact terms. Omit the line entirely if no single change dominates — do not pad.>

## Breaking changes

- ...

## Features

- ...

## Improvements

- ...

## Fixes

- ...

---

`<previous-tag>...<latest-tag>`
```

- Release date: `git log -1 --format=%ai <latest-tag>` (date portion only).
- Do not invent items not backed by a commit in the range.
- The headline is optional. Skip it rather than write a generic one ("This release brings improvements and fixes.").
- A release with three good bullets is better than one with twelve mediocre ones.
