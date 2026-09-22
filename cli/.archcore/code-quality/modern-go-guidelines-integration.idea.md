---
title: "Adopting JetBrains Modern Go Guidelines: Linter First, Plugin Second"
status: draft
tags:
  - "code-quality"
  - "golang"
---

## Idea

`JetBrains/go-modern-guidelines` is a catalog of roughly 70 modern Go idioms grouped by language version, distributed as a plugin for Claude Code, Cursor, and Junie CLI (repository README, 2026-09-05). Adopt the deterministic half of that catalog as golangci-lint analyzers in `.golangci.yml`, and leave the agent-facing plugin as an individual developer's local install. Record the version floor and the precedence of `.archcore/code-quality/` over the external catalog in `go-code-quality.rule`.

Nothing here is implemented. This document records the option and its shape only.

### Problem / Opportunity

- A model writes older Go than the toolchain supports, because recent constructs are rare in training data — this is the stated motivation of the upstream repository.
- The repository targets `go 1.25.11` (@go.mod), so idioms introduced in Go 1.26 and Go 1.27 do not apply and would be a defect if written today.
- `go-code-quality.rule` §Standard-library idioms already covers part of the same ground: `slices` and `cmp` instead of `sort`, and `errors.Is(err, fs.ErrNotExist)` instead of `os.IsNotExist` (@.archcore/code-quality/go-code-quality.rule.md).
- The overlap is partial, so the catalog carries idioms no local document names, and the gap reaches the agent only through review.

## Value

### For Users

- No user-visible change. The idea touches source style and the local development loop only.

### For Business

- No revenue or adoption effect. [assumption]

### For Team

- A linter check runs in CI on every commit, so a missed idiom is caught without a reviewer reading for it.
- `.golangci.yml` already carries the analyzers named in `strict-go-naming-conventions.rule` §Enforcement, so the additions extend an existing mechanism rather than introducing one.
- A recorded version floor stops an agent from writing a Go 1.27 construct into a Go 1.25 module.

## Possible Implementation

### Technical Approach

Three layers exist, and they carry different cost and different reliability.

| Layer | Mechanism | Determinism | Committed to the repository |
|---|---|---|---|
| Analyzers | `.golangci.yml` linters | Deterministic, runs in CI | Yes |
| Agent plugin | Claude Code marketplace plugin | Model-dependent, per session | No |
| Local canon | A clause in `go-code-quality.rule` | Review-time | Yes |

Layer 1 — the analyzers. golangci-lint v2 lists `modernize`, `usestdlibvars`, `intrange`, `copyloopvar`, `exptostd`, and `perfsprint` as separate linters (golangci-lint.run/docs/linters, read 2026-09-05). `modernize` is the analyzer family the upstream repository names as the aligned Go team initiative. The installed golangci-lint version must be checked before `modernize` is enabled, because the entry postdates the v2 linters this repository already runs. [assumption]

Layer 2 — the plugin. Installation is `/plugin marketplace add JetBrains/go-modern-guidelines` followed by `/plugin install modern-go-guidelines@goland-claude-marketplace`, and the CLI reads `go.mod` to filter the catalog by version. The plugin requires a Go 1.25+ toolchain on `PATH`, which `go 1.25.11` satisfies (@go.mod).

Layer 3 — the canon clause. Two to three lines in `go-code-quality.rule` §Standard-library idioms: the version floor is `go.mod`, and a `.archcore/` rule wins over the external catalog on conflict.

### Integrations

- `.golangci.yml` — the enable list and a comment per addition, in the style the file already uses.
- `.archcore/code-quality/go-code-quality.rule.md` — the version floor and the precedence clause.
- `.claude/settings.json` — `enabledPlugins` would carry the plugin if it were committed. This idea does not commit it.

## Risks and Constraints

### Potential Risks

- A second authority contradicts the local canon. `comments-are-the-exception.rule` and `strict-go-naming-conventions.rule` are stricter than common Go practice, and the catalog does not know them.
- The upstream marketplace auto-updates, so a committed plugin would change agent behavior in this repository without a commit.
- Copying the catalog into `.archcore/` would add roughly 70 entries to the `code-quality` tag set, which `review-go` loads in full into the review context (@.claude/skills/review-go/SKILL.md, Step 2). The context cost is paid on every review.
- The catalog goes stale at each Go release, and a copy inside `.archcore/` would need a maintainer.

### Known Constraints

- The module targets `go 1.25.11` (@go.mod), so Go 1.26 and Go 1.27 entries are out of scope until the module version moves.
- `.golangci.yml` documents the reason for every exclusion it carries. An addition follows the same form.
- The finding volume of `modernize` on the current tree is unmeasured. [METRIC REQUIRED]

## Next Steps

- [ ] Check the installed golangci-lint version against the version that introduced `modernize`.
- [ ] Run the candidate linters on the current tree and count the findings per linter.
- [ ] Decide which of the six linters to enable, based on that count.
- [ ] Add the chosen linters to `.golangci.yml` with a comment per entry.
- [ ] Add the version floor and the precedence clause to `go-code-quality.rule`.
