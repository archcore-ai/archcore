---
title: "Add a Document Type to the Engine"
status: draft
tags:
  - "document-types"
  - "golang"
  - "mcp"
---

## What

Add one or more document types to the CLI so that `create_document` generates them, the scan and the categories recognize them, the post-write precision check measures them, the advisory engines rank them, and every public inventory counts them. The result is a green suite, a clean linter, and a working tree the release process can tag.

Applied twice: the research vocabulary (`research`, `evidence`, v0.8.3) and the actor-subject vocabulary (`scenario`, `journey`, unreleased). Both runs followed the same shape; the pitfalls below are the places the second run had to discover again.

## When to Use

Use when:

- An accepted or draft RFC in the `archcore` global source adds a type to the shared vocabulary
- A local plan under `document-types/` names the type as a `creates` capability

Do NOT use when:

- The change alters a template of an existing type only: edit the generator and its test directly
- The change adds a relation value: follow the research-vocabulary plan's phase 2 instead

## Steps

1. Read the procedure in the building doc (How to Add a New Document Type) and the two prior plans under `document-types/`.
2. Load every document tagged `code-quality` before the first Go edit.
3. Add the `TypeXxx` constant, the `categoryMap` row, and the `ValidTypes` entry in @templates/templates.go.
4. Write `generateXxxTemplate` with visible placeholders and wire it into `GenerateTemplate`.
5. Add `RequiredSections`, `ProseProfiles`, and any `StepSections` or `ForeignSections` rows in @templates/precision.go.
6. Add a body cap row in `MaxBodyLines` when the type carries one.
7. Add a per-type finding function in @internal/advisory/precision.go only for checks the shared tables cannot express.
8. Decide the injection rank in @internal/advisory/code_alignment.go; leave vision types out.
9. Extend the type lines in @internal/mcp/tools/create_document.go and @internal/mcp/tools/list_documents.go.
10. Extend the type list, WHEN TO CREATE, TYPE SELECTION RULES, relations, and status text in @internal/mcp/server.go.
11. Add the type to `corpusTypes` in @internal/testsupport/corpus.go on the correct side of the allowlist.
12. Add template, precision, alignment, search-ranking, cascade, and restatement tests beside each change.
13. Add a lifecycle test in @internal/mcp/integration/ on the pattern of @internal/mcp/integration/research_spec_test.go.
14. Update the counts and tables in @README.md, the categories doc, the building doc, the free-form rule, and the advisory doc.
15. Search `.archcore/` for the old type count and for the old injection list; fix every hit.
16. Run `go test ./...`, `golangci-lint run ./...`, and `go build -o archcore .`.
17. Remove one new registry entry, confirm the tests fail, and restore it by editing, never by `git checkout`.

## Example

The actor-subject run: @templates/templates.go (`TypeScenario`, `TypeJourney`), @templates/precision.go (`SectionActors`, `ObservationSections`, `ActorStepSections`, `MaxBodyLines`), @internal/advisory/precision.go (`actorSubjectFindings`), @internal/advisory/precision_actor_subject_test.go, @internal/mcp/integration/actor_subject_spec_test.go.

## Things to Watch Out For

- `TestValidTypes_Completeness` and `TestProseProfiles_Completeness` fail when one table gains the type and another does not; add all three rows together.
- `TestPrecisionFindings_GeneratedTemplatesAreClean` runs the checker over the new template; a modal in a step placeholder, a vagueness word, or a missing Anchors line fails it.
- `TestSectionTables_MatchTemplates` needs at least one numbered item under every `StepSections` heading; an unnumbered form such as Given/When/Then needs its own collector.
- Section matching is a prefix match on the heading text, so `Examples` in a new type collides with the `Examples` heading of `rule` and `doc`; scope a `ForeignSections` row to the types whose templates never emit that heading.
- A type absent from `alignmentTypePriority` is never injected; the corpus helper and the advisory doc both state the allowlist and go stale silently.
- The type count appears in the README three times, in the free-form rule, in the categories doc, and in older ideas; a content search for the old number finds them.
- An older binary skips files of an unknown type in the scan and reports them in `status`; state that in the spec's Failure Behavior and in the release notes.
- The plugin gates the new names on a CLI version probe; record the shipped version for the plugin repository.
- `git checkout <file>` discards uncommitted edits to that file; restore an injected fault by editing.
