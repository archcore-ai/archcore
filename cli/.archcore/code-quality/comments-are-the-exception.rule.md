---
title: "Code Carries No Comment Unless the Code Cannot Speak"
status: accepted
tags:
  - "code-quality"
  - "code-review"
  - "golang"
---

## Rule

The default is no comment. A comment is an exception that earns its place.

1. This rule binds every `.go` file under `cmd/`, `internal/`, and `templates/`, and `main.go`.
2. The developer MUST NOT write a comment that restates what the code already says.
3. The developer MUST reach for a name, a smaller function, or a type before reaching for a comment.
4. The developer MAY comment a solution that is non-obvious, counter-intuitive, or a deliberate
   workaround.
5. The developer MAY comment a block whose logic a competent Go reader cannot follow from the code.
6. A comment permitted by clause 4 or clause 5 MUST hold three sentences or fewer.
7. A comment MUST state why the code is as it is, never what it does.
8. The developer MUST NOT delete an existing comment only to satisfy this rule.

### Comments other rules still require

Clause 2 does not override an obligation stated elsewhere. These comments stay mandatory.

| Comment | Required by |
|---|---|
| doc comment on an exported identifier | Go convention, enforced by `revive` |
| the reason a change deviates from a convention | `go-code-quality.rule`, `strict-go-naming-conventions.rule` |
| the budget a timeout or ceiling constant protects | `go-code-quality.rule`, `bounded-and-deterministic-output.rule` §2 |
| guard or advisory, at the branch handling the error | `fail-open-or-fail-closed-reads.rule` §3 |
| the governing document, as `<slug>.<type>` | `cite-the-governing-document-from-code.rule` |
| a test seam, saying production never reassigns it | `registry-agreement-and-test-seams.guide` §6 |
| the residual invariant of a weaker platform variant | `platform-splits-are-files.rule` §5 |
| the analyzer directive and its reason | `go-code-quality.rule` §Enforcement |

## Rationale

A comment that repeats the code is a second copy of the truth, and the copy is the one that goes
stale. Clauses 4 and 5 keep the cases where the code genuinely cannot carry the meaning — a
workaround, a platform quirk, an ordering that looks arbitrary and is not. Clause 8 exists because
this repository's existing comments record real incidents, and a sweep that removed them would delete
evidence no test holds.

## Examples

**Bad** — the comment restates the line.

> ```go
> // increment the counter
> count++
> ```

**Good** — no comment. The names carry it.

> ```go
> globals, err := config.LoadGlobals(baseDir)
> ```

**Good** — clause 4. The choice looks wrong until the sentence explains it.
@internal/plugin/exec.go:

> ```go
> // Testing the buffer length marked a stream of exactly maxCommandOutput bytes
> // truncated with nothing discarded, and readListing throws a truncated answer
> // away — so a listing that fit the cap precisely was refused intact.
> ```

**Bad** — clause 6. A correct observation stretched past three sentences belongs in an
`.archcore/` document with a citation from the code, not in the source file.

## Enforcement

- No analyzer measures clauses 2 to 7. Review holds them.
- `revive`'s `exported` and `package-comments` rules hold the doc-comment carve-out.
- WHEN a review finds a comment that only restates the code, the reviewer MUST ask for its removal
  rather than its rewording.
