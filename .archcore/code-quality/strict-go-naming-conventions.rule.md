---
title: "Strict Go Naming Conventions"
status: accepted
tags:
  - "code-quality"
  - "code-review"
  - "golang"
---

## Rule

This rule is **absolute** for new code. Every clause is MUST or MUST NOT unless marked SHOULD/MAY. Deviations require an inline comment naming the reason and referencing the clause.

### A. Visibility = case

- **MUST**: Exported identifiers (visible outside the package) start with an uppercase letter — `Settings`, `NewClient`, `URL`, `Manifest`.
- **MUST**: Unexported identifiers (package-internal) start with a lowercase letter — `syncFlags`, `hookEntry`, `parseURL`, `searchMatch`.
- **MUST NOT**: Use a leading or trailing underscore (`_Foo`, `Foo_`) to fake visibility. The blank import (`_ "embed"`) and blank interface assertion (`var _ Foo = (*Bar)(nil)`) are explicitly permitted as standard idioms.
- **SHOULD**: Default to unexported. Promote to exported only when (a) another package needs it now, or (b) it is part of a stable, documented API surface.

### B. Acronyms — all letters in the same case

- **MUST**: Acronyms in **Go identifiers** are entirely uppercase or entirely lowercase, never mixed: `URL` not `Url`, `ID` not `Id`, `HTTP` not `Http`, `JSON` not `Json`, `API` not `Api`, `MCP` not `Mcp`, `CLI` not `Cli`.
- **MUST NOT**: Apply this rule to **struct tags, JSON keys, YAML keys, environment variables, or any wire-format identifier**. Tags like `json:"id"`, `yaml:"url"`, `env:"http_proxy"` MUST stay as written by the protocol — changing tag casing breaks compatibility.
- **MUST**: When an acronym leads an unexported identifier, it is fully lowercase: `urlString`, `httpClient`, `jsonDecoder`.
- **MUST**: Apply to project-specific acronyms: `ADR`, `RFC`, `PRD`, `MRD`, `BRD`, `URD`, `BRS`, `SRS`.
- **MUST**: Adjacent acronyms keep both fully uppercase: `XMLHTTPRequest`, not `XMLHttpRequest`.
- **EXCEPTION**: ISO 29148 type names use their canonical mixed case as written in the standard: `StRS`, `SyRS`. These are the *only* mixed-case acronyms permitted. Add `//nolint:stylecheck // ST1003: ISO 29148 canonical name` at each declaration to silence the linter.

### C. Type names

- **MUST**: Type names are nouns or noun phrases — `Client`, `Manifest`, `Settings`, `DiffEntry`, `Frontmatter`. Never verbs.
- **MUST NOT**: Suffix with `T`, `Struct`, `Class`. Wrong: `UserT`, `UserStruct`. Right: `User`.
- **EXCEPTION**: Suffix `Type` is permitted only when the type itself classifies a *kind* of something — `DocumentType`, `RelationType`, `SyncType`. Avoid the `Type` suffix for ordinary types.
- **MUST NOT**: Prefix interfaces with `I`. Wrong: `IReader`. Right: `Reader`.
- **MUST**: No information already given by the package. In `agents/` the type is `Agent`, used as `agents.Agent` — never `agents.AgentInfo` or `agents.AgentEntry`. The `AgentID` typed-string is permitted because it disambiguates the agent's *string identifier* from the `Agent` struct.
- **MUST**: Generic type parameters use single uppercase letters in conventional roles — `T` (general element), `K` (key), `V` (value), `E` (element of a collection), `R` (return). When a single letter is ambiguous, use a descriptive single word with no prefix: `Item`, `Node`. Never `TElem` or `TKey`.

### D. Constructor functions

- **MUST**: Exported constructors that allocate and return a value are named `New<Type>` and return `*<Type>`, `<Type>`, or an interface satisfied by `<Type>`. Examples: `NewClient`, `NewManifest`, `NewUpdater`, `NewServer`.
- **MUST**: When a type has multiple constructors with distinct preset configurations, each variant is suffixed: `NewCloudSettings`, `NewOnPremSettings`, `NewNoneSettings`, `NewAuthenticatedClient`. The plain `New<Type>` form is reserved for the simplest default — or omitted if every variant has a meaningful name.
- **MAY**: Use the functional-options pattern (`NewSettings(WithCloud(), WithProject(42))`) as an alternative to variant-suffix constructors when the configuration space is large or open-ended. Pick one style per type and stay consistent.
- **MAY**: Use `Build<Type>` or `Make<Type>` for **multi-step assembly** that performs non-trivial work — reads files, parses input, traverses a graph — before producing the value. Example: `BuildPayload(baseDir, entries)` reads files and parses frontmatter; `BuildIndex` walks a directory. Keep `New<Type>` for lightweight constructors that only initialize fields.
- **MUST**: Cobra command factories (package `cmd`) are named `new<Name>Cmd` — lowercase `new`, suffix `Cmd`. They are unexported because cobra commands are wired only inside `cmd/`. (Project convention, not Go-wide practice.)
- **MUST NOT**: Use `Create<Type>` as a constructor verb. Reserve `Create*` for verbs that create something **outside the program** (`CreateFile`, `CreateUser` — talking to disk, DB, or API).

### E. Function and method names

- **MUST**: Functions and methods are verbs or verb phrases — `Validate`, `AddRelation`, `CleanupRelations`, `CheckHealth`, `ListProjects`.
- **MUST NOT**: Prefix simple property-style getters with `Get`. Wrong: `GetSync`, `GetServerURL`. Right: `Sync`, `ServerURL`. (Effective Go.)
- **MUST**: Property-style methods (no parameters, return a derived value, idempotent) are nouns: `Client.ServerURL()`, `Settings.Sync()`.
- **MUST**: Setters use the `Set<Property>` form: `SetTimeout`, `SetVerbose`. Setters take exactly the value they set.
- **MUST**: A method that returns a single value of an obvious type does not repeat the type in its name. Wrong: `Settings.GetSyncString()`. Right: `s.Sync` (field) or `s.Sync()` (method).
- **EXCEPTION**: Generated code (gRPC stubs, mock factories) is exempt from the `Get` prohibition.

### F. Receiver names

- **MUST**: A single letter matching the type's initial — `c` for `Client`, `s` for `Settings`, `m` for `Manifest`, `u` for `Updater`, `a` for `Agent`.
- **MUST**: The same receiver name across every method on the same type. Mixing `c *Client` and `cl *Client` in one package is forbidden.
- **EDGE CASE**: When two types in the same package share an initial (e.g., `Settings` and `Server`), use the shortest unambiguous prefix and stay consistent — `s` for `Settings`, `srv` for `Server`. Document the choice with a one-line comment at the first method.
- **MUST NOT**: Use `this` or `self`.

### G. Constants and string enums

- **MUST**: Any string value drawn from a closed, finite set is declared as a typed alias. Pattern:

  ```go
  type Foo string
  const (
      FooA Foo = "a"
      FooB Foo = "b"
  )
  ```

- **MUST**: The constant name carries a type-derived prefix when bare names would collide or be ambiguous: `TypeADR` (DocumentType), `RelRelated` (RelationType), `StatusDraft` (DocStatus), `CategoryVision` (Category), `SyncTypeNone` (SyncType). The prefix may be omitted when the constants are themselves a singular vocabulary (e.g., `AgentID` constants `ClaudeCode`, `Cursor`).
- **MUST**: Validate enum values exhaustively. Either form is acceptable:
  - A `map[<Type>]bool` lookup (concise; preferred when the set is also iterated).
  - A `switch` listing every constant, with no `default: false` fallback (paired with the `exhaustive` linter).

  A `switch` with `default: false` and no other cases is **forbidden** — it silently accepts everything until linter catches up.

- **MUST**: Fields referencing the enum use the typed alias, not bare `string`. `Settings.Sync` MUST be `SyncType`, not `string`. `Frontmatter.Status` MUST be `DocStatus`, not `string`.
- **MUST NOT**: Define a closed-set string as untyped constants. The compiler must be able to enforce the domain.
- **NUMERIC ENUMS**: Use `iota` with a typed alias: `type Level int` + `const ( LevelDebug Level = iota; LevelInfo; ... )`.

### H. Package names

- **MUST**: Lowercase, single-word, ASCII letters only. Wrong: `helperUtils`, `helper_utils`, `myPkg`. Right: `agents`, `sync`, `update`, `templates`, `display`, `prompts`.
- **MUST**: Singular unless the package is intrinsically a registry of items: `agents`, `templates`, `prompts`, `tools` (registries — plural); `sync`, `update`, `display`, `config` (singular role/verb). (Project preference, not a Go-wide rule.)
- **MUST NOT**: Use generic names — `util`, `utils`, `helpers`, `common`, `misc`, `shared`. Pick a name that describes what is *inside*.
- **MUST NOT**: Stutter the package name in identifiers exposed by the package: `agents.AgentList`, `sync.SyncManifest`. Use `agents.List`, `sync.Manifest`.
- **NOTE**: `internal/sync` shadows stdlib `sync`. Internal use is fine; importers in `cmd/` alias it as `archsync`. Do not rename to avoid the shadow.

### I. File names

- **MUST**: snake_case, ASCII letters and digits only — `hooks_claude_code.go`, `gemini_cli.go`, `mcp_helpers.go`.
- **MUST**: When a file is keyed to a slug-valued enum (e.g., `AgentID = "claude-code"`), the file basename mirrors the slug with `-` replaced by `_`. The triple `AgentID = "claude-code"` ↔ file `claude_code.go` ↔ identifier `ClaudeCode` MUST align.
- **MUST**: Tests live alongside source as `<basename>_test.go`. No separate `tests/` directories inside Go packages.

### J. Error names and messages

- **MUST**: Sentinel errors are package-level `var` declared as `Err<Cause>`, exported when the caller is expected to compare with `errors.Is`: `ErrServerUnreachable`. Internal sentinels stay unexported (`errInvalidPath`).
- **MUST**: Custom error types are named `<cause>Error` — `serverUnreachableError`, `validationError`, `pathError`. Export them when external callers must `errors.As` to read structured fields (cf. `*os.PathError`); keep unexported otherwise.
- **MUST**: Error messages start with a lowercase letter and do not end with punctuation: `"sync mode requires project_id"`, not `"Sync mode requires project_id."`.
- **EXCEPTION**: Proper nouns retain their canonical case, even at the start of a message: `"GitHub API returned status %d"`, `"S3 upload failed"`, `"Postgres connection refused"` are all valid.
- **MUST**: Wrap errors with `%w` when the caller may want to unwrap or `errors.Is`/`errors.As`: `fmt.Errorf("loading config: %w", err)`. Use `%v` only when you intentionally do not want the error to be unwrappable.

### K. Variables

- **MUST**: Local variable names are short and contextual: `i`, `j`, `k` for indices; `err` for errors; `ctx` for context; `cmd` for cobra command; `req` / `resp` for HTTP request/response; `t` for `*testing.T`, `b` for `*testing.B`, `tb` for `testing.TB`; `tt` for the current case in a table-driven test.
- **MUST**: Package-level vars holding compiled regex use the suffix `Re`: `SlugRe`, `TagRe`. The suffix may be omitted when the variable name itself reads as a pattern (`pseudoVersionSuffix`), but `Re` is preferred for new code.
- **MUST**: Discarded values are assigned to `_` — never named `unused`, `ignored`, or `tmp`.
- **MUST NOT**: Use Hungarian notation (`strName`, `iCount`, `bDone`).

### L. Interfaces

- **MUST**: Single-method interfaces use the `<Method>er` form: `Reader` (`Read`), `Writer` (`Write`), `Closer` (`Close`).
- **MUST**: Multi-method interfaces are named for the role, not the methods: `Storage`, `Repository`, `Picker`.
- **MUST NOT**: Suffix struct types with `-er` if they are not interfaces. The existing `internal/update.Updater` struct is grandfathered to avoid in-flight churn; **new** struct types must not use the `-er` suffix. Renaming `Updater` is a fine follow-up but not required by this rule.

## Rationale

The codebase already follows ~95% of these rules organically. Codifying them prevents three failure modes:

1. **Drift from AI agents.** Agents pattern-match against recent code; without an explicit retrievable rule, the first non-conforming example becomes the next agent's template, and drift compounds.
2. **Bikeshedding in review.** A clear MUST resolves every "should this be `Url` or `URL`?" question instantly. Reviewers cite a clause; authors fix and move on.
3. **Type-system erosion.** Three of our four string-enum domains are typed; the other three (`SyncType`, `Status`, `Category`) are bare strings. Without a mandate, the typed pattern slowly atrophies.

The strictness is deliberate: every clause maps to a decision the codebase has already made organically. There is no aspirational rule.

## Examples

### Good

```go
// §B Acronyms uppercase in identifiers; lowercase in struct tags
type AgentID string
type Settings struct {
    ProjectID *int `json:"project_id,omitempty"` // tag stays lowercase per protocol
}
func (c *Client) ServerURL() string { ... }
const HTTPTimeout = 30 * time.Second

// §D Constructors
func NewClient(serverURL string) *Client { ... }
func NewCloudSettings() *Settings { ... }
func newSyncCmd() *cobra.Command { ... }
func BuildPayload(baseDir string, entries []DiffEntry) (*Payload, error) { ... } // multi-step OK

// §G Typed enum with prefixed constants and exhaustive validation map
type SyncType string
const (
    SyncTypeNone   SyncType = "none"
    SyncTypeCloud  SyncType = "cloud"
    SyncTypeOnPrem SyncType = "on-prem"
)
type Settings struct {
    Sync SyncType `json:"sync"`
}
var validSyncTypes = map[SyncType]bool{
    SyncTypeNone: true, SyncTypeCloud: true, SyncTypeOnPrem: true,
}

// §F Receiver — single letter, uniform
func (m *Manifest) AddRelation(...)    { ... }
func (m *Manifest) RemoveRelation(...) { ... }
func (m *Manifest) RelationsFor(...)   { ... }

// §J Errors — sentinel + structured type, %w wrap, lowercase + proper noun
var ErrServerUnreachable = errors.New("server unreachable")
type serverUnreachableError struct { url string }
func (e *serverUnreachableError) Error() string { ... }
return fmt.Errorf("loading config: %w", err)
return fmt.Errorf("GitHub API returned status %d", resp.StatusCode) // proper noun OK

// §I File ↔ identifier ↔ slug
// AgentID = "claude-code" → file claude_code.go → const ClaudeCode
```

### Bad

```go
// §B Mixed-case acronyms in identifiers
type AgentId string                       // MUST be AgentID
func (c *Client) ServerUrl() string { }   // MUST be ServerURL
const HttpTimeout = 30 * time.Second      // MUST be HTTPTimeout

// §B Wrong: editing struct tags to match identifier casing
type Settings struct {
    ProjectID *int `json:"projectID"`     // MUST stay "project_id" — wire format
}

// §D Wrong constructor verb
func MakeClient(...) *Client { }          // MUST be NewClient
func CreateCloudSettings() *Settings { }  // MUST be NewCloudSettings (Create reserved for I/O)

// §E Get-prefixed getter
func (s *Settings) GetSync() SyncType {}  // MUST be Sync()

// §G Untyped enum + bare string field
const (
    SyncTypeNone   = "none"               // MUST be SyncType = "none"
    SyncTypeCloud  = "cloud"
)
type Settings struct {
    Sync string `json:"sync"`             // MUST be SyncType
}

// §G Switch with default:false (silently accepts new constants)
func valid(s SyncType) bool {
    switch s {
    case SyncTypeNone: return true
    default: return false                 // MUST list every constant; let exhaustive linter enforce
    }
}

// §F Receiver mismatch
func (c *Manifest) AddRelation(...)      // MUST match other Manifest methods (m)
func (mfst *Manifest) RemoveRelation(...) // MUST be single letter `m`

// §C Type stuttering / wrong suffix / interface prefix
type AgentInfo struct { ... }             // package agents — MUST be Agent
type UserStruct struct { ... }            // MUST be User
type IReader interface { ... }            // MUST be Reader

// §K Hungarian notation
var strName string                        // MUST be name
var iCount int                            // MUST be count

// §J Error string formatting
return errors.New("Invalid path.")        // MUST be "invalid path"
return fmt.Errorf("loading config: %v", err) // MUST be %w if caller may unwrap
```

## Enforcement

**Linter coverage** (catches a subset, run as part of CI). Configure in `.golangci.yml`:

- `gofmt` / `gofumpt` — formatting.
- `go vet` — receiver mismatches, common errors.
- `revive` — `var-naming`, `package-comments`, `exported`, `error-strings`.
- `stylecheck` — `ST1003` (acronym style). Note: `stylecheck` is a separate analyzer from `staticcheck` proper; both ship together but `ST*` checks must be enabled explicitly.
- `errname` — enforces §J `Err*` and `*Error` patterns automatically.
- `errcheck` — flags ignored errors.
- `exhaustive` — required for §G typed enums; flags non-exhaustive switches.
- `unconvert` — flags redundant type conversions (catches over-casting at typed-enum boundaries).

**Code review** (human or AI via `review-go` skill) covers what linters cannot:

- Constructor variant naming (`NewCloudSettings` vs ad-hoc `MakeCloud`).
- File ↔ identifier slug correspondence.
- Receiver-name uniformity per type across files.
- Field types using the typed enum, not bare `string`.
- Stuttering with package name in exposed identifiers.
- `%w` vs `%v` choice in error wrapping.
- Default-to-unexported judgment (§A).

**Known existing deviations** (new code MUST conform; existing violations are migrated opportunistically when the surrounding code is touched):

- `SyncType*` constants in `internal/config/` must become `type SyncType string`; `Settings.Sync` must use the typed alias.
- `Status*` constants in `templates/` must become `type DocStatus string`; `Frontmatter.Status` must use the typed alias.
- `Category*` constants in `templates/` must become `type Category string`; `categoryMap` and `CategoryForType` must use the typed alias.

**New code** MUST be conforming. A deviation requires a one-line comment naming the reason and the clause: `// §L: grandfathered; in-flight churn risk.`. Silent deviations are review-blocking.
