# conventions

Binding Go rules live in Archcore, tag `code-quality`. Before writing/editing any `.go` in `cmd/`, `internal/`, `templates/`, `main.go`: `list_documents(tags=["code-quality"])`, then `get_document` each, once per session. Table of what each rule binds: `CLAUDE.md` §Go Code Quality.

High-risk points often missed:
- Comments are the exception (`comments-are-the-exception.rule`); default is none. A comment on code existing because of a decision cites the document.
- Naming is strict and absolute for new code (`strict-go-naming-conventions.rule`); linter covers only a subset.
- Tests: co-located, table-driven; packages touching `$HOME`/XDG/git/host CLIs need an isolating `TestMain`.
- Commands use the constructor-command pattern with logic in testable functions.
- Domain packages return data; formatting happens at the boundary (`cmd/`, `internal/display/`).
- GOOS/GOARCH differences go in separate files.
- Deviation from a clause requires an inline comment naming the clause and reason.

Prose (docs, help, tool descriptions, Markdown): policy in `AGENTS.md`; per-type profile in global rule `concepts/document-prose-canon`. No claims of formal ASD-STE100 / ISO 24495-1 compliance.
New Archcore records are created with `status: draft`.
