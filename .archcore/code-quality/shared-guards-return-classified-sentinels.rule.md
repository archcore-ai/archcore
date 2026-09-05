---
title: "A Shared Guard Returns Classified Sentinels, Not Messages"
status: accepted
tags:
  - "code-quality"
  - "golang"
  - "mcp"
---

## Rule

One verdict, one function. Each surface renders that verdict in its own words.

1. This rule binds a predicate in `internal/docs/`, `internal/config/`, or `internal/git/` that more
   than one surface consults.
2. WHEN two or more surfaces must reach the same verdict about a path or a resource, exactly one
   predicate function MUST compute it.
3. The predicate MUST return a package-level sentinel error per refusal reason.
4. The predicate MUST NOT return a rendered user message.
5. A sentinel MUST NOT embed a filesystem path, a secret, or a host name.
6. The predicate's doc comment MUST list its validation layers in the order they run.
7. A surface mapping the sentinels MUST enumerate the cases it treats as permissive.
8. A surface MUST NOT default to permissive for an unrecognized sentinel.
9. A sentinel that separates two causes a caller acts on differently MUST stay separate.

## Rationale

A guard that returns a message forces every surface to parse prose, and the surface that renders a
filesystem path into an MCP error leaks it. Clause 8 is the one with a defect behind it: a
default-allow permitted every refusal class the caller had not enumerated, which is how a document
reachable through a symlink out of the store stayed writable while MCP refused it.

## Examples

**Good** — the sentinel set with the constraint stated on it.

> `ErrPathReadOnlyGlobal`, `ErrPathNotDocument`, `ErrPathEscapes` at @internal/docs/guard.go, whose
> comment says callers map them via `errors.Is` and that none of them embed filesystem paths.

**Good** — clause 6. `GuardWritablePath` lists five numbered layers, and clause 2 is why: both the MCP
write tools and the pre-write hook call it, so a direct editor write and an MCP mutation are judged by
exactly the same rules.

**Good** — clause 9. `git.ErrGitAbsent` at @internal/git/git.go is distinct from "not a repository",
so a caller can tell "no git installed" from "not a repo" instead of collapsing both into an empty
result.

**Bad** — the surface allows what it does not recognize.

> ```go
> switch {
> case errors.Is(err, docs.ErrPathNotDocument):
>     return allowHook()
> default:
>     return allowHook()
> }
> ```
>
> `GuardWritablePath` reports four classes of failure and only two carry a comparable sentinel.

## Enforcement

- `sanitizeError` at @internal/mcp/tools/common.go is the rendering half at the MCP boundary and holds
  clause 5 for paths that reach a tool result.
- The write-guard tests in @cmd/hook_write_guard_test.go pin clause 8 for the one surface where a
  default-allow was observed.
- Clauses 2 to 4, 6, 7, and 9 carry no automated check. Review holds them.
