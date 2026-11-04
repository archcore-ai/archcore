---
title: "Strict Go Naming Conventions"
status: accepted
tags:
  - "code-quality"
  - "code-review"
  - "component:cli"
  - "golang"
---

## Rule

This rule is **absolute** for new code. Every clause is MUST or MUST NOT unless marked SHOULD/MAY. Deviations require an inline comment naming the reason and referencing the clause (`// §L: <reason>.`). gofmt, revive, staticcheck, and errname (`.golangci.yml`) enforce the standard Go naming rules, so this document states only what they do not.

### A. Visibility

- **MUST NOT**: Use a leading or trailing underscore to fake visibility. A named result such as `context_` is covered too.
- **SHOULD**: Default to unexported. Promote only when another package needs it now or it is part of a stable, documented API.

### B. Acronyms

- **MUST**: Project-specific acronyms follow the Go acronym rule: `ADR`, `RFC`, `PRD`, `MRD`, `BRD`, `URD`, `BRS`, `SRS`.
- **MUST NOT**: Apply identifier casing to struct tags, JSON keys, YAML keys, or environment variables. A wire format stays as the protocol writes it.
- **EXCEPTION**: ISO 29148 type names `StRS` and `SyRS` keep their canonical mixed case, with `//nolint:staticcheck // ST1003: ISO 29148 canonical name` at each declaration.

### C. Type names

- **MUST**: Carry no information the package already gives. In `agents/` the type is `Agent`, never `agents.AgentInfo`. The `AgentID` typed string is permitted because it separates the identifier from the struct.
- **EXCEPTION**: Suffix `Type` is permitted only when the type classifies a kind: `DocumentType`, `RelationType`, `SyncType`.

### D. Constructors

- **MUST**: Cobra command factories in package `cmd` are named `new<Name>Cmd` — lowercase `new`, suffix `Cmd`. They are unexported because commands are wired only inside `cmd/`. This is a project convention, not Go-wide practice.
- **MUST**: A type with several preset constructors suffixes each variant: `NewCloudSettings`, `NewOnPremSettings`. Plain `New<Type>` is reserved for the simplest default.
- **MAY**: Use `Build<Type>` for multi-step assembly that reads files or parses input.
- **MUST NOT**: Use `Create<Type>` as a constructor verb. `Create*` is reserved for verbs that create something outside the program.

### G. Constants and string enums

- **MUST**: Any string value drawn from a closed set is a typed alias with a `const` block. The constant name carries a type-derived prefix when bare names would collide: `TypeADR`, `RelRelated`, `StatusDraft`, `SyncTypeNone`.
- **MAY**: A closed set that never crosses a process boundary stays untyped when a comment at the declaration says so and names why. Anything that reaches a wire, a file, or an MCP result is typed with no exception. Example: `refKindExplicit` and `refKindMention` in `@cli/internal/mcp/tools/search_documents.go`.
- **MUST**: Validate enum values exhaustively, through a `map[<Type>]bool` lookup or a `switch` that lists every constant. A `switch` with only `default: false` is forbidden.
- **MUST**: Fields referencing the enum use the typed alias. `Settings.Sync` is `SyncType`, not `string`.
- **MUST**: Key a map on the typed enum with the constants, never with string literals. `map[DocumentType]int{"rul": 1}` compiles and silently yields the zero value.
- **MUST**: A `switch` that deliberately handles a subset carries `//exhaustive:ignore` with the reason on the same comment.

### I. File names

- **MUST**: A file keyed to a slug-valued enum mirrors the slug with `-` replaced by `_`. The triple `AgentID = "claude-code"`, file `claude_code.go`, and identifier `ClaudeCode` MUST align.
- **MUST**: A file with a `//go:build` platform constraint is named `<subject>_<constraint>.go` — `platform-splits-are-files.rule`.
- **MAY**: A test file that pins normative properties rather than coverage is named `<subject>_spec_test.go` and carries a doc comment listing them before the first declaration.

### H. Packages

- **NOTE**: `internal/sync` shadows stdlib `sync`. Importers alias it as `archsync` whenever the file also needs stdlib `sync`. Do not rename to avoid the shadow.

### L. Interfaces

- **MUST NOT**: Suffix a struct with `-er` when the struct is a data record rather than a behavior. `hookMatcher` in `@cli/internal/wiring/hooks_agents.go` was the shape this forbids — a `{Matcher, Hooks}` JSON record that matched nothing — and became `hookMatcherGroup`.
- **MAY**: Name a struct after the interface it implements. `pluginReporter` (`plugin.Reporter`), `capWriter` (`io.Writer`), and `sessionRootProvider` (`tools.RootProvider`) are correct, and so is the stdlib practice they follow (`bufio.Writer`).

## Rationale

Codifying the conventions prevents three failure modes:

1. **Drift from AI agents.** Agents pattern-match against recent code; without an explicit retrievable rule, the first non-conforming example becomes the next agent's template.
2. **Bikeshedding in review.** A clear MUST ends a question like "`Url` or `URL`?" instantly.
3. **Type-system erosion.** Typed string enums atrophy without a mandate.

Every clause maps to a decision the codebase already made. §L was narrowed because the blanket `-er` prohibition contradicted four conforming call sites and ordinary Go practice.

## Enforcement

Linters run in CI after `go vet`; the config is `.golangci.yml`. `exhaustive` sets `default-signifies-exhaustive: true`, so a switch ending in `default` is not reported. That is deliberate: the switches it would flag use `default` as meaningful behavior, and `GlobalState.Fatal` at `@cli/internal/docs/inspect.go` depends on it to make a new state fatal by default. The consequence is that §G's exhaustiveness is linted only for default-less switches. A defaulted switch is a review item, and the `//exhaustive:ignore` reasons in the tree must be true.

Code review (human or the `review-go` skill) covers what linters cannot: constructor variant naming, file-to-slug correspondence, typed enums in field types, and whether a `default` over a typed enum is meaningful behavior.
