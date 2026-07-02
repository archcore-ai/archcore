package tools

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"archcore-cli/internal/sync"
	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

// Ordered type priority for relevance sort (lower rank = higher priority).
// Anything not listed falls back to typePriorityDefault.
const typePriorityDefault = 100

var typePriority = map[string]int{
	"rule":      1,
	"adr":       2,
	"rfc":       3,
	"spec":      4,
	"cpat":      5,
	"guide":     6,
	"plan":      7,
	"idea":      8,
	"rnd":       9,
	"prd":       10,
	"brs":       11,
	"syrs":      12,
	"srs":       13,
	"strs":      14,
	"mrd":       15,
	"brd":       16,
	"urd":       17,
	"doc":       18,
	"task-type": 19,
}

const (
	searchDefaultLimit = 50
	searchMaxLimit     = 200
	// In full mode each result carries the document body, so the default and
	// max result counts are smaller to keep a single response token-bounded.
	searchFullDefaultLimit = 3
	searchFullMaxLimit     = 20
	excerptWindow          = 120
)

// search_documents output modes.
const (
	searchModeSnippets = "snippets" // excerpt windows only (default)
	searchModeFull     = "full"     // also include each matched doc's body inline
)

const searchDocumentsDescription = `Search .archcore/ documents by content or filters.

Use this when you need to find documents matching specific criteria — path references in the body, content substring, document types, status, or recency. Unlike list_documents (metadata-only), search_documents scans document bodies.

Returns: JSON array of matched documents with title, type, status, mtime, tags, match details (ref, kind, specificity, excerpt), and manifest relations. Results sorted by the ` + "`sort`" + ` parameter ("relevance" default: specificity → type priority → mtime; or "mtime").

Filters combine as AND. At least one filter must be provided. Use path_ref for path references (matches both @-notation and qualified bare paths). Use content for free-text substring search across title+body. Topic search is strict substring — singular/plural forms do not match.

Set ` + "`mode=full`" + ` when your goal is to read the matched document(s): each result then carries the full document body inline (frontmatter stripped), so you get answer-ready content in a single call and do NOT need a follow-up get_document. Full mode defaults to a small limit (3); raise ` + "`limit`" + ` if you need more candidates. Leave mode at the default "snippets" (excerpt windows only) when you just need to discover which docs match.

Prefer this tool over list_documents + get_document loops when you need "which docs match X" — and prefer search_documents(mode=full) over search + get_document when you then need to read those docs.`

// searchMatch is one piece of evidence tying a document to the query.
type searchMatch struct {
	Kind        string `json:"kind"`
	Ref         string `json:"ref"`
	Specificity int    `json:"specificity"`
	Excerpt     string `json:"excerpt"`
}

// searchResult is the per-document row returned by search_documents.
type searchResult struct {
	Path    string              `json:"path"`
	Title   string              `json:"title"`
	Type    string              `json:"type"`
	Status  templates.DocStatus `json:"status,omitempty"`
	ModTime time.Time           `json:"mtime"`
	Tags    []string            `json:"tags,omitempty"`
	// Source annotation, mirroring LocalDocument so all three read tools
	// (list_documents, get_document, search_documents) carry the same machine-
	// checkable local/global distinction. local-overrides-global.rule requires
	// agents to read these rather than guess authority from the path.
	SourceID   string        `json:"source_id"`
	SourceKind string        `json:"source_kind"`
	Global     bool          `json:"global,omitempty"`
	ReadOnly   bool          `json:"read_only,omitempty"`
	Matches    []searchMatch `json:"matches"`
	// Body is the full document body (frontmatter stripped), populated only in
	// mode=full so callers can read the matched doc without a get_document call.
	Body              string             `json:"body,omitempty"`
	IncomingRelations []DocumentRelation `json:"incoming_relations"`
	OutgoingRelations []DocumentRelation `json:"outgoing_relations"`

	// Private ranking keys, not serialized.
	maxSpecificity int
	typeRank       int
}

// pathRef is a single path-reference candidate extracted from a document body.
type pathRef struct {
	Raw   string // e.g. "@src/payments/" or "src/payments/stripe.ts"
	Kind  string // "explicit" or "mention_candidate"
	Start int    // byte offset of the first character in the source body
}

var (
	// Explicit @-prefixed references.
	pathRefExplicitRe = regexp.MustCompile(`@[\w./-]+`)
	// Bare mention candidates: identifier / path.
	pathRefBareRe = regexp.MustCompile(`[\w-]+/[\w./-]+`)
)

// NewSearchDocumentsTool returns the tool definition for search_documents.
func NewSearchDocumentsTool() mcp.Tool {
	return mcp.NewTool("search_documents",
		mcp.WithDescription(searchDocumentsDescription),
		mcp.WithString("path_ref",
			mcp.Description("Match documents that reference this path (either @-notation or qualified bare paths) in their body."),
		),
		mcp.WithString("content",
			mcp.Description("Case-insensitive substring matched against title + body. Strict substring — no stemming or fuzzy matching."),
		),
		mcp.WithArray("types",
			mcp.Description("Filter by one or more document types (e.g. [\"adr\", \"rule\"])."),
			mcp.WithStringItems(),
		),
		mcp.WithString("status",
			mcp.Description("Filter by frontmatter status."),
			mcp.Enum(templates.ValidStatusStrings()...),
		),
		mcp.WithString("mtime_after",
			mcp.Description("Only include documents modified after this time. Accepts RFC3339 (ISO-8601) or a relative value like \"24h\", \"30d\", \"90d\"."),
		),
		mcp.WithString("sort",
			mcp.Description("Result ordering. \"relevance\" (default) = max specificity DESC → type priority → mtime DESC. \"mtime\" = pure mtime DESC."),
			mcp.Enum("relevance", "mtime"),
		),
		mcp.WithString("mode",
			mcp.Description("Output detail. \"snippets\" (default) returns only excerpt windows around matches. \"full\" additionally returns each matched document's full body inline (frontmatter stripped), so you can read the doc without a follow-up get_document. Full mode defaults to limit=3 (max 20)."),
			mcp.Enum(searchModeSnippets, searchModeFull),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return. Defaults and caps are mode-dependent: snippets=50 default/200 max, full=3 default/20 max. Values above the cap are clamped; 0 or omitted maps to the mode default."),
		),
		mcp.WithTitleAnnotation("Search Documents"),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}

// HandleSearchDocuments handles the search_documents tool call.
func HandleSearchDocuments(baseDir string) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pathRefFilter := strings.TrimSpace(request.GetString("path_ref", ""))
		contentFilter := request.GetString("content", "")
		types := request.GetStringSlice("types", nil)
		status := templates.DocStatus(request.GetString("status", ""))
		mtimeAfterRaw := strings.TrimSpace(request.GetString("mtime_after", ""))
		sortMode := request.GetString("sort", "relevance")
		mode := request.GetString("mode", searchModeSnippets)
		// Normalize defensively (the framework enforces the enum, but any value
		// other than "full" maps to the cheaper snippets output).
		if mode != searchModeFull {
			mode = searchModeSnippets
		}
		// 0 / omitted => mode-dependent default (full mode is smaller because each
		// result carries a body).
		limitFloat := request.GetFloat("limit", 0)

		// Validate at least one filter.
		if pathRefFilter == "" && contentFilter == "" && len(types) == 0 && status == "" {
			return errorResult("specify at least one filter (path_ref, content, types, or status)"), nil
		}

		// Validate and clamp limit (mode-aware bounds).
		if limitFloat < 0 {
			return errorResult("limit must be non-negative"), nil
		}
		defaultLimit, maxLimit := searchDefaultLimit, searchMaxLimit
		if mode == searchModeFull {
			defaultLimit, maxLimit = searchFullDefaultLimit, searchFullMaxLimit
		}
		limit := int(limitFloat)
		if limit == 0 {
			limit = defaultLimit
		}
		if limit > maxLimit {
			limit = maxLimit
		}

		// Parse mtime_after.
		mtimeAfter, err := parseMtimeAfter(mtimeAfterRaw)
		if err != nil {
			return errorResult("invalid mtime_after: " + err.Error()), nil
		}

		// Normalize sort mode (framework enforces enum, but be defensive).
		if sortMode == "" {
			sortMode = "relevance"
		}

		// Only load bodies when we actually need them. Pure metadata queries
		// (types / status / mtime_after) never inspect body content, so we
		// avoid holding up to N×body_size on the heap for the request.
		needsBody := pathRefFilter != "" || contentFilter != "" || mode == searchModeFull
		var docs []LocalDocument
		if needsBody {
			docs, err = ScanDocumentsFull(baseDir)
		} else {
			docs, err = ScanDocuments(baseDir)
		}
		if err != nil {
			return errorResult(sanitizeError("scanning documents", err)), nil
		}

		// Load manifest for relations. If it fails, continue without relations.
		var manifest *sync.Manifest
		if m, mErr := sync.LoadManifest(baseDir); mErr == nil {
			manifest = m
		} else if !errors.Is(mErr, fs.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "search_documents: failed to load manifest: %v\n", mErr)
		}

		lowerContent := strings.ToLower(contentFilter)

		results := make([]searchResult, 0, len(docs))
		for _, doc := range docs {
			// Metadata filters (AND).
			if len(types) > 0 && !slices.Contains(types, doc.Type) {
				continue
			}
			if status != "" && doc.Status != status {
				continue
			}
			if !mtimeAfter.IsZero() && !doc.ModTime.After(mtimeAfter) {
				continue
			}

			// Match evidence accumulators.
			matches := make([]searchMatch, 0)
			maxSpec := 0

			// path_ref filter.
			if pathRefFilter != "" {
				refs := extractPathRefs(doc.Content)
				refs = filterBareMentions(refs)
				normalizedTarget := strings.TrimPrefix(pathRefFilter, "@")
				for _, r := range refs {
					refPath := strings.TrimPrefix(r.Raw, "@")
					spec := computeSpecificity(refPath, normalizedTarget)
					if spec == 0 {
						continue
					}
					kind := "path_ref_mention"
					if r.Kind == "explicit" {
						kind = "path_ref_explicit"
					}
					excerpt := buildExcerpt(doc.Content, r.Start, len(r.Raw))
					matches = append(matches, searchMatch{
						Kind:        kind,
						Ref:         r.Raw,
						Specificity: spec,
						Excerpt:     excerpt,
					})
					if spec > maxSpec {
						maxSpec = spec
					}
				}
				if len(matches) == 0 {
					continue
				}
			}

			// content filter.
			if contentFilter != "" {
				spec, excerpt, found := extractContentMatch(doc.Title, doc.Content, lowerContent)
				if !found {
					// Pure content filter: no content hit means we drop the doc.
					// But if path_ref also active, we already required a path_ref match,
					// so the AND semantics say: content must also match.
					continue
				}
				matches = append(matches, searchMatch{
					Kind:        "content",
					Ref:         contentFilter,
					Specificity: spec,
					Excerpt:     excerpt,
				})
				if spec > maxSpec {
					maxSpec = spec
				}
			}

			// When no content/path_ref filter was provided, matches stays empty
			// — that's the pure-metadata case: we still return the doc.

			rank, ok := typePriority[doc.Type]
			if !ok {
				rank = typePriorityDefault
			}

			result := searchResult{
				Path:              doc.Path,
				Title:             doc.Title,
				Type:              doc.Type,
				Status:            doc.Status,
				ModTime:           doc.ModTime,
				Tags:              doc.Tags,
				SourceID:          doc.SourceID,
				SourceKind:        doc.SourceKind,
				Global:            doc.Global,
				ReadOnly:          doc.ReadOnly,
				Matches:           matches,
				IncomingRelations: []DocumentRelation{},
				OutgoingRelations: []DocumentRelation{},
				maxSpecificity:    maxSpec,
				typeRank:          rank,
			}

			// In full mode, attach the body so the caller can read the doc
			// without a separate get_document round-trip. Content is present
			// because needsBody forced ScanDocumentsFull above.
			if mode == searchModeFull {
				result.Body = stripFrontmatter(doc.Content)
			}

			if manifest != nil {
				relPath := normalizeRelPath(doc.Path)
				outgoing, incoming := manifest.RelationsFor(relPath)
				for _, r := range outgoing {
					result.OutgoingRelations = append(result.OutgoingRelations, DocumentRelation{
						Path: ".archcore/" + r.Target,
						Type: string(r.Type),
					})
				}
				for _, r := range incoming {
					result.IncomingRelations = append(result.IncomingRelations, DocumentRelation{
						Path: ".archcore/" + r.Source,
						Type: string(r.Type),
					})
				}
			}

			results = append(results, result)
		}

		sortResults(results, sortMode)

		if limit > 0 && len(results) > limit {
			results = results[:limit]
		}

		data, err := json.Marshal(results)
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

// parseMtimeAfter accepts an empty string, an RFC3339 timestamp, or a relative
// value like "30d", "24h", "90d". Returns the zero time for an empty input.
func parseMtimeAfter(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	// Try RFC3339 first.
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Relative form: <number><unit> where unit is 'd' (days) or 'h' (hours).
	if len(s) < 2 {
		return time.Time{}, fmt.Errorf("expected RFC3339 timestamp or relative duration like \"30d\" or \"24h\", got %q", s)
	}
	unit := s[len(s)-1]
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n < 0 {
		return time.Time{}, fmt.Errorf("expected RFC3339 timestamp or relative duration like \"30d\" or \"24h\", got %q", s)
	}
	// Guard against values that would overflow time.Duration
	// (Duration is int64 nanoseconds → about 290 years total).
	const maxRelativeDays = 36500
	switch unit {
	case 'd':
		if n > maxRelativeDays {
			return time.Time{}, fmt.Errorf("day count too large: %d (max %d)", n, maxRelativeDays)
		}
		return time.Now().Add(-time.Duration(n) * 24 * time.Hour), nil
	case 'h':
		if n > maxRelativeDays*24 {
			return time.Time{}, fmt.Errorf("hour count too large: %d (max %d)", n, maxRelativeDays*24)
		}
		return time.Now().Add(-time.Duration(n) * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("unknown duration unit %q (expected 'd' or 'h')", string(unit))
	}
}

// extractPathRefs runs the two path-ref regexes over body and returns all
// matches as pathRef values. Explicit matches are tagged "explicit"; bare
// candidates are tagged "mention_candidate" and need filterBareMentions.
func extractPathRefs(body string) []pathRef {
	var refs []pathRef
	// Explicit first — we record their offsets so we can skip bare hits that
	// overlap (the bare regex would otherwise re-match the same token minus
	// the leading "@").
	explicitSpans := pathRefExplicitRe.FindAllStringIndex(body, -1)
	for _, m := range explicitSpans {
		refs = append(refs, pathRef{
			Raw:   body[m[0]:m[1]],
			Kind:  "explicit",
			Start: m[0],
		})
	}
	for _, m := range pathRefBareRe.FindAllStringIndex(body, -1) {
		start, end := m[0], m[1]
		// Skip bare matches covered by an explicit match (explicit spans include
		// the leading '@', so the bare match starts at s[0]+1).
		overlap := false
		for _, s := range explicitSpans {
			if start >= s[0] && end <= s[1] {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}
		refs = append(refs, pathRef{
			Raw:   body[start:end],
			Kind:  "mention_candidate",
			Start: start,
		})
	}
	return refs
}

// filterBareMentions keeps explicit refs unchanged and drops bare
// mention-candidates unless one of these heuristics holds:
//   - the candidate ends with '/'
//   - the candidate has ≥2 '/' separators
//   - the final segment's extension is a known source extension
func filterBareMentions(candidates []pathRef) []pathRef {
	out := make([]pathRef, 0, len(candidates))
	for _, r := range candidates {
		if r.Kind == "explicit" {
			out = append(out, r)
			continue
		}
		raw := r.Raw
		if strings.HasSuffix(raw, "/") {
			out = append(out, r)
			continue
		}
		if strings.Count(raw, "/") >= 2 {
			out = append(out, r)
			continue
		}
		// Exactly one '/': require a known source extension on the final segment.
		lastSlash := strings.LastIndex(raw, "/")
		final := raw[lastSlash+1:]
		ext := filepath.Ext(final)
		if ext != "" && templates.IsSourceExtension(ext) {
			out = append(out, r)
			continue
		}
		// Drop.
	}
	return out
}

// computeSpecificity returns the number of '/'-separated segments shared as a
// left-prefix between ref and target. Both are compared after trimming a
// leading '@' from either side.
//
// Examples:
//
//	ref="src/payments/stripe.ts" target="src/payments/"  → 2
//	ref="src/payments/"          target="src/payments/"  → 2
//	ref="src/auth/"              target="src/payments/"  → 1
//	ref="docs/guide.md"          target="src/payments/"  → 0
func computeSpecificity(ref, target string) int {
	ref = strings.TrimPrefix(ref, "@")
	target = strings.TrimPrefix(target, "@")
	ref = strings.TrimSuffix(ref, "/")
	target = strings.TrimSuffix(target, "/")
	if ref == "" || target == "" {
		return 0
	}
	refSegs := strings.Split(ref, "/")
	targetSegs := strings.Split(target, "/")
	n := min(len(refSegs), len(targetSegs))
	shared := 0
	for i := range n {
		if refSegs[i] != targetSegs[i] {
			break
		}
		shared++
	}
	return shared
}

// extractContentMatch searches title then body for a case-insensitive match
// of lowerQuery (which must already be lower-cased). Specificity is 3 for a
// title hit, 1 for a body-only hit.
func extractContentMatch(title, body, lowerQuery string) (specificity int, excerpt string, found bool) {
	if lowerQuery == "" {
		return 0, "", false
	}
	// Title first.
	lowerTitle := strings.ToLower(title)
	if idx := strings.Index(lowerTitle, lowerQuery); idx >= 0 {
		return 3, buildExcerpt(title, idx, len(lowerQuery)), true
	}
	lowerBody := strings.ToLower(body)
	if idx := strings.Index(lowerBody, lowerQuery); idx >= 0 {
		return 1, buildExcerpt(body, idx, len(lowerQuery)), true
	}
	return 0, "", false
}

// buildExcerpt returns up to excerptWindow characters of source centered on
// pos, with leading/trailing "..." added when the excerpt is truncated on
// either side. refLen bounds the minimum portion kept intact.
func buildExcerpt(source string, pos int, refLen int) string {
	if pos < 0 {
		pos = 0
	}
	if refLen < 0 {
		refLen = 0
	}
	// Keep the match and some context around it.
	half := max((excerptWindow-refLen)/2, 0)
	start := pos - half
	end := pos + refLen + half
	if start < 0 {
		start = 0
	}
	if end > len(source) {
		end = len(source)
	}
	if start > len(source) {
		start = len(source)
	}
	// Snap byte offsets to rune boundaries so we never slice a multibyte
	// UTF-8 rune in half. Move start backward into the rune it belongs to,
	// move end forward past the rune it belongs to.
	for start > 0 && !utf8.RuneStart(source[start]) {
		start--
	}
	for end < len(source) && !utf8.RuneStart(source[end]) {
		end++
	}
	excerpt := source[start:end]
	// Collapse whitespace runs to a single space for readability.
	excerpt = collapseWhitespace(excerpt)
	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "..."
	}
	if end < len(source) {
		suffix = "..."
	}
	return prefix + excerpt + suffix
}

// collapseWhitespace replaces runs of whitespace (spaces, tabs, newlines)
// with a single space, so excerpts stay on one line.
func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
			continue
		}
		inSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// sortResults orders results per the mode. Stable so equal keys preserve input
// order for deterministic output.
func sortResults(results []searchResult, mode string) {
	slices.SortStableFunc(results, func(a, b searchResult) int {
		if mode == "mtime" {
			return b.ModTime.Compare(a.ModTime)
		}
		if c := cmp.Compare(b.maxSpecificity, a.maxSpecificity); c != 0 {
			return c
		}
		if c := cmp.Compare(a.typeRank, b.typeRank); c != 0 {
			return c
		}
		return b.ModTime.Compare(a.ModTime)
	})
}
