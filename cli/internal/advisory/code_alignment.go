package advisory

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/internal/docs"
	"archcore-cli/templates"
)

// Code-alignment injection and the file-context result behind it
// (file-context-resolution.spec).
//
// An agent about to edit a file has no reason to know a rule constrains it. This
// finds the documents that name the file or its directories, adds the rules
// that name no path at all, and puts them in front of the edit. The same result
// serves search_documents for_path, so the hook and the agent's call agree.
//
// Cost is bounded by the accept-list, not by corpus size: only the ranked types
// are ever opened, so the walk rejects most of the corpus before reading
// anything. Removing that filter puts the whole corpus back on a path that
// blocks the user's edit.

// DefaultSourceRoots are the directories treated as source code when
// settings.json declares none.
var DefaultSourceRoots = []string{
	"src", "lib", "app", "pkg", "cmd", "internal", "apps", "packages", "modules", "components",
}

// ContextReason says why a document applies to a file. It reaches the MCP
// wire, so it carries a type (§G).
type ContextReason string

// File-context reasons (file-context-resolution.spec, Surface).
const (
	ReasonFile      ContextReason = "file"
	ReasonKind      ContextReason = "kind"
	ReasonDirectory ContextReason = "directory"
	ReasonGeneral   ContextReason = "general"
)

// reasonRank orders matched rows by how directly the reason ties a document to
// the file.
var reasonRank = map[ContextReason]int{ReasonFile: 0, ReasonKind: 1, ReasonDirectory: 2, ReasonGeneral: 3}

const (
	// MaxContextMatched caps the matched rows: five documents with their titles
	// fit the hook budget beside the general rows.
	MaxContextMatched = 5
	// MaxContextGeneral caps the general rows. The Litres monorepo holds 15
	// pathless rules, so the remainder line is a real case, not a theory.
	MaxContextGeneral = 10
	// maxAlignmentTokens bounds how far up the directory chain to look.
	maxAlignmentTokens = 5
	// maxAlignmentRunes caps the hook message inside the per-handler hook
	// context limits (Codex: 2 500 tokens). Counted in runes, not bytes: a byte
	// cap can split a multi-byte character and leave the payload invalid.
	maxAlignmentRunes = 2048
)

// Errors the file-context result reports for a path it cannot serve.
var (
	ErrOutsideProject  = errors.New("path is outside the project")
	ErrArchcoreDocPath = errors.New("path is an .archcore document, not a source file")
)

// alignmentTypePriority ranks document types by how much they constrain an edit.
// A type absent from this map is not injected at all — a plan or an idea is
// context for a discussion, not a constraint on a line of code. A doc ranks
// last: it explains, it does not oblige, but it is often the one document that
// names the file.
var alignmentTypePriority = map[templates.DocumentType]int{
	templates.TypeRule:     7,
	templates.TypeCPAT:     6,
	templates.TypeADR:      5,
	templates.TypeSpec:     4,
	templates.TypeScenario: 3,
	templates.TypeGuide:    2,
	templates.TypeDoc:      1,
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

// genericFileStems are base names too common to identify one file: a document
// naming "page.tsx" is not about the page being edited.
// ponytail: fixed list; a per-project setting if a corpus needs another stem.
var genericFileStems = map[string]bool{
	"index": true, "page": true, "layout": true, "route": true, "loading": true, "error": true,
	"not-found": true, "template": true, "default": true, "main": true, "mod": true, "types": true,
	"utils": true, "helpers": true, "constants": true, "config": true, "styles": true, "test": true,
	"setup": true, "readme": true, "init": true, "__init__": true,
}

// ContextRow is one document that constrains an edit to a file.
type ContextRow struct {
	Doc    docs.Document
	Reason ContextReason
	// depth orders rows of one reason: the matched directory length, or 2 for a
	// full-path file match and 1 for a file-name match.
	depth int
	hits  int
}

// FileContext is the ordered, uncapped file-context result. Callers cut it with
// MaxContextMatched and MaxContextGeneral after their own filters.
type FileContext struct {
	Rel     string
	Matched []ContextRow
	General []ContextRow
	// Scanned counts the ranked-type documents read per source id.
	Scanned map[string]int
}

// ResolveFileContext returns the documents that constrain an edit to filePath.
// It returns ErrOutsideProject or ErrArchcoreDocPath for a path it cannot serve.
func ResolveFileContext(baseDir, filePath string) (FileContext, error) {
	rel, ok := docs.RelativeToBase(baseDir, filePath)
	if !ok || rel == ".." || strings.HasPrefix(rel, "../") || path.IsAbs(rel) {
		return FileContext{}, ErrOutsideProject
	}
	if rel == ".archcore" || strings.HasPrefix(rel, ".archcore/") {
		return FileContext{}, ErrArchcoreDocPath
	}
	fc := FileContext{Rel: rel, Scanned: make(map[string]int)}
	if !config.DirExists(baseDir) {
		return fc, nil
	}

	var tokens []string
	if underSourceRoot(baseDir, rel) {
		tokens = derivePathTokens(rel)
	}
	base := path.Base(rel)
	nameIsDistinct := !genericFileStems[strings.ToLower(strings.TrimSuffix(base, path.Ext(base)))]
	kind := compoundExtension(base)

	for _, doc := range scanAlignmentCorpus(baseDir) {
		fc.Scanned[doc.SourceID]++
		if !doc.InAgentContext() {
			continue
		}
		switch {
		case strings.Contains(doc.Content, rel):
			fc.Matched = append(fc.Matched, ContextRow{Doc: doc, Reason: ReasonFile, depth: 2, hits: strings.Count(doc.Content, rel)})
		case nameIsDistinct && strings.Contains(doc.Content, base):
			fc.Matched = append(fc.Matched, ContextRow{Doc: doc, Reason: ReasonFile, depth: 1, hits: strings.Count(doc.Content, base)})
		case kind != "" && isAcceptedRule(doc) && countStandalone(doc.Content, kind) > 0:
			fc.Matched = append(fc.Matched, ContextRow{Doc: doc, Reason: ReasonKind, depth: 1, hits: countStandalone(doc.Content, kind)})
		default:
			if token := firstMatchingToken(doc.Content, tokens); token != "" {
				fc.Matched = append(fc.Matched, ContextRow{Doc: doc, Reason: ReasonDirectory, depth: len(token), hits: strings.Count(doc.Content, token)})
			} else if isGeneralRule(baseDir, doc) {
				fc.General = append(fc.General, ContextRow{Doc: doc, Reason: ReasonGeneral})
			}
		}
	}

	slices.SortStableFunc(fc.Matched, compareMatched)
	slices.SortStableFunc(fc.General, func(a, b ContextRow) int {
		if c := cmp.Compare(alignmentTypePriority[b.Doc.Type], alignmentTypePriority[a.Doc.Type]); c != 0 {
			return c
		}
		return strings.Compare(a.Doc.Path, b.Doc.Path)
	})
	return fc, nil
}

// compareMatched orders matched rows: file before directory, deeper first,
// accepted before draft, stronger type, more mentions, and the path last.
func compareMatched(a, b ContextRow) int {
	if c := cmp.Compare(reasonRank[a.Reason], reasonRank[b.Reason]); c != 0 {
		return c
	}
	if c := cmp.Compare(b.depth, a.depth); c != 0 {
		return c
	}
	if aAcc, bAcc := a.Doc.Status == templates.StatusAccepted, b.Doc.Status == templates.StatusAccepted; aAcc != bAcc {
		if aAcc {
			return -1
		}
		return 1
	}
	if c := cmp.Compare(alignmentTypePriority[b.Doc.Type], alignmentTypePriority[a.Doc.Type]); c != 0 {
		return c
	}
	if c := cmp.Compare(b.hits, a.hits); c != 0 {
		return c
	}
	return strings.Compare(a.Doc.Path, b.Doc.Path)
}

// scanAlignmentCorpus reads the ranked types. A broken global must not blank
// the advisory — it degrades to local only, the same trade the session recap
// makes; a failing local scan yields nothing, because an advisory never blocks.
func scanAlignmentCorpus(baseDir string) []docs.Document {
	corpus, err := docs.ScanTypes(baseDir, alignmentTypes)
	if err != nil {
		corpus, _ = docs.ScanLocalTypes(baseDir, alignmentTypes)
	}
	return corpus
}

// firstMatchingToken returns the longest directory token the content names.
func firstMatchingToken(content string, tokens []string) string {
	for _, token := range tokens {
		if strings.Contains(content, token) {
			return token
		}
	}
	return ""
}

// isGeneralRule reports whether doc is an accepted rule or cpat that names no
// path of this project (pathless-rules-are-general.adr). A reference such as
// `@testing-library/react` is a package, not a path here, so it does not scope
// the rule.
func isGeneralRule(baseDir string, doc docs.Document) bool {
	if !isAcceptedRule(doc) {
		return false
	}
	for _, ref := range docs.FilterBareMentions(docs.ExtractPathRefs(doc.Content)) {
		p := strings.TrimSuffix(strings.TrimPrefix(ref.Raw, "@"), ".")
		if p == "" || p == ".archcore" || strings.HasPrefix(p, ".archcore/") {
			continue
		}
		// Advisory read: a stat failure means "not a path here", never a refusal.
		if _, err := os.Stat(filepath.Join(baseDir, filepath.FromSlash(p))); err == nil {
			return false
		}
	}
	return true
}

func isAcceptedRule(doc docs.Document) bool {
	return doc.Status == templates.StatusAccepted && (doc.Type == templates.TypeRule || doc.Type == templates.TypeCPAT)
}

// compoundExtension returns everything from the first dot of a base name with at
// least two dots — ".test.tsx" for "Foo.test.tsx" — and "" otherwise. A rule
// naming it governs every file of that kind.
func compoundExtension(base string) string {
	if strings.Count(base, ".") < 2 {
		return ""
	}
	return base[strings.Index(base, "."):]
}

// countStandalone counts occurrences of ext that no file-name character
// precedes. `*.test.tsx` names the kind; `Form.test.tsx` names one file, and a
// rule that lists its own test files is not a convention for every test.
func countStandalone(content, ext string) int {
	n := 0
	for i := 0; ; {
		rel := strings.Index(content[i:], ext)
		if rel == -1 {
			return n
		}
		at := i + rel
		if at == 0 || !isFileNameByte(content[at-1]) {
			n++
		}
		i = at + len(ext)
	}
}

func isFileNameByte(b byte) bool {
	return b == '_' || b == '-' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// CodeAlignment returns the context to inject before a source edit, or an empty
// string when there is nothing useful to say.
func CodeAlignment(baseDir, filePath string) string {
	if os.Getenv("ARCHCORE_DISABLE_INJECTION") == "1" || filePath == "" {
		return ""
	}
	fc, err := ResolveFileContext(baseDir, filePath)
	if err != nil {
		return ""
	}
	return renderFileContext(fc)
}

// renderFileContext writes the hook text. It drops rows from the tail — general
// rows first — until the text fits the rune budget, and states every remainder.
func renderFileContext(fc FileContext) string {
	matched := min(len(fc.Matched), MaxContextMatched)
	general := min(len(fc.General), MaxContextGeneral)
	if matched+general == 0 {
		return ""
	}
	for {
		text := writeFileContext(fc, matched, general)
		if len([]rune(text)) <= maxAlignmentRunes || matched+general <= 1 {
			return truncateRunes(text, maxAlignmentRunes)
		}
		if general > 0 {
			general--
		} else {
			matched--
		}
	}
}

func writeFileContext(fc FileContext, matched, general int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[Archcore Context] Read these before editing %s:\n", fc.Rel)
	for _, row := range fc.Matched[:matched] {
		writeContextRow(&b, row)
	}
	if rest := len(fc.Matched) - matched; rest > 0 {
		fmt.Fprintf(&b, "  … and %d more — search_documents(path_ref=%q)\n", rest, fc.Rel)
	}
	if len(fc.General) > 0 {
		b.WriteString("Rules that name no path apply to every file:\n")
	}
	for _, row := range fc.General[:general] {
		writeContextRow(&b, row)
	}
	if rest := len(fc.General) - general; rest > 0 {
		fmt.Fprintf(&b, "  … and %d more — list_documents(types=[\"rule\",\"cpat\"], status=\"accepted\")\n", rest)
	}
	return b.String()
}

func writeContextRow(b *strings.Builder, row ContextRow) {
	fmt.Fprintf(b, "- %s: %s [%s]", row.Doc.Type, resolveAlignmentTitle(row.Doc), row.Doc.Path)
	if row.Doc.Status == templates.StatusDraft {
		b.WriteString(" [draft]")
	}
	if row.Doc.Global {
		// Org-wide rules constrain the edit as much as local ones, but they
		// live in a read-only mount — marking them stops the reader from
		// trying to update what they cannot write.
		b.WriteString(" [global]")
	}
	b.WriteByte('\n')
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
