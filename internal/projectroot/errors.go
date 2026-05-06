package projectroot

import (
	"errors"
	"fmt"
	"strings"
)

// Project markers in priority order: .archcore wins over .git, which wins
// over generic build files. The two slices are kept separate so the priority
// boundary is explicit (walkUp iterates genericMarkers directly instead of
// slicing allMarkers).
//
// Append-only contract: never reorder, never remove. Downstream tooling
// (the archcore plugin, agents) parses these names.
var (
	genericMarkers = []string{
		"go.mod",
		"package.json",
		"pyproject.toml",
		"Cargo.toml",
		"pom.xml",
		"composer.json",
		"Gemfile",
	}
	allMarkers = append([]string{".archcore", ".git"}, genericMarkers...)
)

// Markers returns a copy of the canonical project marker list, in priority
// order. .archcore is highest, then .git, then generic build files.
func Markers() []string {
	out := make([]string, len(allMarkers))
	copy(out, allMarkers)
	return out
}

// Stable error codes. Part of the CLI's public contract: the archcore
// plugin and downstream tooling parse these. Never change a code once
// shipped — only add new ones.
const (
	CodeNotProject  = "ERR_NOT_PROJECT"
	CodeHomeRefused = "ERR_HOME_REFUSED"
	CodeNoProject   = "ERR_NO_PROJECT"
	CodePathInvalid = "ERR_PATH_INVALID"
	CodeTildeExpand = "ERR_TILDE_EXPAND"
)

// Sentinel errors. Use errors.Is to match.
var (
	ErrNoProject        = errors.New("no project root found")
	ErrBaseDirHome      = errors.New("base dir is $HOME")
	ErrBaseDirNotExist  = errors.New("base dir does not exist")
	ErrBaseDirSystem    = errors.New("base dir is a system path")
	ErrBaseDirNoMarkers = errors.New("base dir has no project markers")
	ErrTildeExpand      = errors.New("tilde expansion failed")
)

// ResolveError wraps a sentinel with structured context for the unified
// error block. Consumers should access Code (stable) and may render
// Message/Reason/Fix (human-readable, may change).
type ResolveError struct {
	Sentinel error
	Code     string
	Path     string
	Source   Source
	Reason   string
	Fix      string
}

func (e *ResolveError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Sentinel, e.Reason)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Sentinel)
}

func (e *ResolveError) Unwrap() error {
	return e.Sentinel
}

// FormatError renders the unified stderr block as plain text. Stable
// across versions in its `code:` line. Returns "" if err is nil or not
// a *ResolveError.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	var re *ResolveError
	if !errors.As(err, &re) {
		return ""
	}
	var b strings.Builder
	b.WriteString("--- archcore error ---\n")
	fmt.Fprintf(&b, "code    : %s\n", re.Code)
	fmt.Fprintf(&b, "message : %s\n", re.Sentinel.Error())
	if re.Path != "" {
		fmt.Fprintf(&b, "path    : %s\n", re.Path)
	}
	if re.Source != "" {
		fmt.Fprintf(&b, "source  : %s\n", re.Source)
	}
	if re.Reason != "" {
		fmt.Fprintf(&b, "detail  : %s\n", re.Reason)
	}
	if re.Fix != "" {
		lines := strings.Split(re.Fix, "\n")
		for i, line := range lines {
			label := "hint    : "
			if i > 0 {
				label = "          "
			}
			b.WriteString(label)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("---------------------\n")
	return b.String()
}
