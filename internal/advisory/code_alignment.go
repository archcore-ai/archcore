package advisory

import (
	"fmt"
	"os"
	"path"
	"slices"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/internal/docs"
	"archcore-cli/templates"
)

// Code-alignment injection.
//
// An agent about to edit a file has no reason to know a rule constrains it. This
// finds the documents that mention the file's directory and puts the most
// specific ones in front of the edit.
//
// Cost is bounded by the accept-list, not by corpus size: only the five ranked
// types are ever opened, so the walk rejects roughly three quarters of the
// corpus before reading anything. Removing that filter puts the whole corpus
// back on a path that blocks the user's edit.

// DefaultSourceRoots are the directories treated as source code when
// settings.json declares none.
var DefaultSourceRoots = []string{
	"src", "lib", "app", "pkg", "cmd", "internal", "apps", "packages", "modules", "components",
}

const (
	// maxAlignmentDocs is how many documents reach the agent. Beyond three the
	// injection stops being a pointer and becomes reading homework.
	maxAlignmentDocs = 3
	// maxAlignmentTokens bounds how far up the directory chain to look.
	maxAlignmentTokens = 5
	// maxAlignmentRunes caps the message. Counted in runes, not bytes: a byte cap
	// can split a multi-byte character and leave the payload invalid.
	maxAlignmentRunes = 2048
)

// alignmentTypePriority ranks document types by how much they constrain an edit.
// A type absent from this map is not injected at all — a plan or an idea is
// context for a discussion, not a constraint on a line of code.
var alignmentTypePriority = map[templates.DocumentType]int{
	templates.TypeRule:  5,
	templates.TypeCPAT:  4,
	templates.TypeADR:   3,
	templates.TypeSpec:  2,
	templates.TypeGuide: 1,
}

// alignmentTypes is the accept-set the scan filters on, derived from the ranking
// so the allowlist has exactly one definition.
var alignmentTypes = func() map[templates.DocumentType]bool {
	set := make(map[templates.DocumentType]bool, len(alignmentTypePriority))
	for t := range alignmentTypePriority {
		set[t] = true
	}
	return set
}()

// CodeAlignment returns the context to inject before a source edit, or
// an empty string when there is nothing useful to say.
func CodeAlignment(baseDir, filePath string) string {
	if os.Getenv("ARCHCORE_DISABLE_INJECTION") == "1" {
		return ""
	}
	if filePath == "" || !config.DirExists(baseDir) {
		return ""
	}
	rel, ok := docs.RelativeToBase(baseDir, filePath)
	if !ok {
		return ""
	}
	// Editing a document is not a source edit; the guard handles that path.
	if rel == ".archcore" || strings.HasPrefix(rel, ".archcore/") {
		return ""
	}
	if !underSourceRoot(baseDir, rel) {
		return ""
	}

	tokens := derivePathTokens(rel)
	if len(tokens) == 0 {
		return ""
	}

	matches := rankAlignmentMatches(baseDir, tokens)
	if len(matches) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[Archcore Context] Before editing %s:\n", rel)
	for _, m := range matches {
		suffix := ""
		if m.global {
			// Org-wide rules constrain the edit as much as local ones, but they
			// live in a read-only mount — marking them stops the reader from
			// trying to update what they cannot write.
			suffix = " [global]"
		}
		fmt.Fprintf(&b, "- %s: %s [%s]%s\n", m.docType, m.title, m.shortPath, suffix)
	}
	return truncateRunes(b.String(), maxAlignmentRunes)
}

// underSourceRoot reports whether rel sits inside a configured source root. A
// root must be followed by a separator: a file literally named "src" is not
// source code inside "src/".
//
// A plain prefix test, because both sides are already in one coordinate space:
// rel is slash-separated and baseDir-relative, and config normalizes every
// declared root on load. Matching raw configured values here is what let "./src"
// and a Windows-separated "src\api" validate cleanly and then match nothing.
func underSourceRoot(baseDir, rel string) bool {
	for _, root := range resolveSourceRoots(baseDir) {
		if strings.HasPrefix(rel, root+"/") {
			return true
		}
	}
	return false
}

// resolveSourceRoots returns the configured roots, or the defaults. A settings file
// that cannot be read falls back to the defaults rather than disabling the
// feature: an advisory must not go silent because of a config typo.
func resolveSourceRoots(baseDir string) []string {
	settings, err := config.Load(baseDir)
	if err != nil || settings.CodeAlignment == nil || len(settings.CodeAlignment.SourceRoots) == 0 {
		return DefaultSourceRoots
	}
	return settings.CodeAlignment.SourceRoots
}

// derivePathTokens returns the directory chain of rel, longest first: for
// "src/api/handlers/users.ts" that is "src/api/handlers/", "src/api/", "src/".
// Longest first is what makes a document about the exact package outrank one
// about the whole tree.
func derivePathTokens(rel string) []string {
	var tokens []string
	dir := path.Dir(rel)
	for dir != "." && dir != "/" && dir != "" && len(tokens) < maxAlignmentTokens {
		tokens = append(tokens, dir+"/")
		parent := path.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return tokens
}

// alignmentMatch is one ranked document.
type alignmentMatch struct {
	docType   templates.DocumentType
	title     string
	shortPath string
	global    bool
	tokenLen  int
	priority  int
	sortPath  string
}

// rankAlignmentMatches finds and ranks the documents that mention any token.
func rankAlignmentMatches(baseDir string, tokens []string) []alignmentMatch {
	// Only five of the ~18 document types are ever injected, and the type comes
	// from the filename — so the walk rejects roughly three quarters of the
	// corpus before opening anything. This runs before every source edit, inside
	// a one-second host budget, so the documents that cannot matter must not be
	// read at all.
	corpus, err := docs.ScanTypes(baseDir, alignmentTypes)
	if err != nil {
		// A broken global must not blank the advisory — degrade to local only,
		// the same trade the session recap makes.
		corpus, err = docs.ScanLocalTypes(baseDir, alignmentTypes)
		if err != nil {
			return nil
		}
	}

	var out []alignmentMatch
	for _, doc := range corpus {
		priority, ranked := alignmentTypePriority[doc.Type]
		if !ranked || !doc.InAgentContext() {
			continue
		}
		// The longest matching token wins: a document found by "src/api/handlers/"
		// is more specific than the same document found by "src/".
		best := -1
		for i, token := range tokens {
			if strings.Contains(doc.Content, token) {
				best = i
				break
			}
		}
		if best < 0 {
			continue
		}
		out = append(out, alignmentMatch{
			docType:   doc.Type,
			title:     resolveAlignmentTitle(doc),
			shortPath: docs.NormalizeRelPath(doc.Path),
			global:    doc.Global,
			tokenLen:  len(tokens[best]),
			priority:  priority,
			sortPath:  doc.Path,
		})
	}

	slices.SortStableFunc(out, func(a, b alignmentMatch) int {
		if a.tokenLen != b.tokenLen {
			return b.tokenLen - a.tokenLen
		}
		if a.priority != b.priority {
			return b.priority - a.priority
		}
		// A stable, readable tie-break.
		return strings.Compare(a.sortPath, b.sortPath)
	})

	return out[:min(len(out), maxAlignmentDocs)]
}

// resolveAlignmentTitle returns the document title, falling back to a readable form of
// its slug so a document missing frontmatter still names itself.
func resolveAlignmentTitle(doc docs.Document) string {
	if doc.Title != "" {
		return doc.Title
	}
	return strings.ReplaceAll(doc.Slug, "-", " ")
}

// truncateRunes caps s at limit runes without splitting a character.
func truncateRunes(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit])
}
