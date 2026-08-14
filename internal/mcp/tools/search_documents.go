package tools

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"archcore-cli/internal/config"
	"archcore-cli/internal/docs"
	"archcore-cli/internal/sync"
	"archcore-cli/templates"

	"github.com/mark3labs/mcp-go/mcp"
)

// Ordered type priority for relevance sort (lower rank = higher priority).
// Anything not listed falls back to typePriorityDefault.
const typePriorityDefault = 100

var typePriority = map[templates.DocumentType]int{
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

// searchMode is the output-detail vocabulary of the mode parameter (§G typed
// enum; the wire values are unchanged).
type searchMode string

const (
	searchModeSnippets searchMode = "snippets" // excerpt windows only (default)
	searchModeFull     searchMode = "full"     // also include each matched doc's body inline
)

// contentMatchMode is how content tokens must match against a document
// (global-recall-guarantees.rfc).
type contentMatchMode string

const (
	matchModeExact contentMatchMode = "exact" // the whole query as one substring (pre-tokenization behavior)
	matchModeAll   contentMatchMode = "all"   // every whitespace token must occur (default)
	matchModeAny   contentMatchMode = "any"   // at least one token must occur
)

// contentFreqCap bounds the occurrence-count contribution to the ranking score,
// so one term-stuffed document cannot outrank every structural (title/heading)
// hit (bounded-and-deterministic-output.rule).
const contentFreqCap = 20

const searchDocumentsDescription = `Search .archcore/ documents by content or filters. The search covers the local project AND every mounted read-only global source; read source_kind on each result to tell them apart, and treat local documents as authoritative over same-topic globals.

Use this when you need to find documents matching specific criteria — path references in the body, content words, document types, status, or recency. Unlike list_documents (metadata-only), search_documents scans document bodies.

Returns: JSON {"results": [...], "coverage": {...}}. Each result carries title, type, status, mtime, tags, match details (ref, kind, specificity, excerpt), and manifest relations. "coverage" maps each searched source to its scanned document count (e.g. {"local": 102, "org": 42}) — an empty "results" next to a populated "coverage" is a verified absence, so broaden the query instead of assuming the corpus was not searched. Results sorted by the ` + "`sort`" + ` parameter ("relevance" default). When a source has at least one match, its top match is kept on the page past the limit cut while slots allow — when matching sources outnumber the page, the best-ranked sources win.

Filters combine as AND. At least one filter must be provided. Use path_ref for path references (matches both @-notation and qualified bare paths). Use content for word search across title+body: by default every whitespace-separated word must occur somewhere in the document (any order, any distance — "plugin compatibility" matches "Plugin / CLI Compatibility"). Set match="exact" for a literal substring or match="any" for at-least-one-word. Use source to scope the search to "local", "global", or one declared global source id.

Set ` + "`mode=full`" + ` when your goal is to read the matched document(s): each result then carries the full document body inline (frontmatter stripped), so you get answer-ready content in a single call and do NOT need a follow-up get_document. Full mode defaults to a small limit (3); raise ` + "`limit`" + ` if you need more candidates. Leave mode at the default "snippets" (excerpt windows only) when you just need to discover which docs match.

Prefer this tool over list_documents + get_document loops when you need "which docs match X" — and prefer search_documents(mode=full) over search + get_document when you then need to read those docs.`

// matchKind is the match-evidence vocabulary carried on the wire in
// searchMatch.Kind (§G typed enum; a typed string alias marshals identically,
// so the wire bytes are unchanged). The refKind* constants below are the
// internal pathRef candidate kinds the path_ref match kinds derive from; they
// never leave the process and are not part of the §G migration ledger, so they
// stay untyped.
type matchKind string

const (
	matchKindExplicit matchKind = "path_ref_explicit"
	matchKindMention  matchKind = "path_ref_mention"
	matchKindContent  matchKind = "content"

	refKindExplicit = "explicit"
	refKindMention  = "mention_candidate"
)

// searchMatch is one piece of evidence tying a document to the query.
type searchMatch struct {
	Kind        matchKind `json:"kind"`
	Ref         string    `json:"ref"`
	Specificity int       `json:"specificity"`
	Excerpt     string    `json:"excerpt"`
}

// searchDocumentsResult is the response envelope. Coverage maps each searched
// source id to its scanned document count, so an empty result set is a verified
// absence rather than an unfalsifiable blank (global-recall-guarantees.rfc).
type searchDocumentsResult struct {
	Results  []searchResult `json:"results"`
	Coverage map[string]int `json:"coverage"`
}

// searchResult is the per-document row returned by search_documents.
type searchResult struct {
	Path    string                 `json:"path"`
	Title   string                 `json:"title"`
	Type    templates.DocumentType `json:"type"`
	Status  templates.DocStatus    `json:"status,omitempty"`
	ModTime time.Time              `json:"mtime"`
	Tags    []string               `json:"tags,omitempty"`
	// Source annotation, mirroring LocalDocument so all three read tools
	// (list_documents, get_document, search_documents) carry the same machine-
	// checkable local/global distinction. local-overrides-global.rule requires
	// agents to read these rather than guess authority from the path.
	SourceID   string          `json:"source_id"`
	SourceKind docs.SourceKind `json:"source_kind"`
	Global     bool            `json:"global,omitempty"`
	ReadOnly   bool            `json:"read_only,omitempty"`
	Matches    []searchMatch   `json:"matches"`
	// Body is the full document body (frontmatter stripped), populated only in
	// mode=full so callers can read the matched doc without a get_document call.
	Body              string             `json:"body,omitempty"`
	IncomingRelations []DocumentRelation `json:"incoming_relations"`
	OutgoingRelations []DocumentRelation `json:"outgoing_relations"`

	// Private ranking keys, not serialized. score folds the match specificity
	// (path_ref contributes its maximum, content the per-token sum) and the
	// capped occurrence count; effectiveMtime is the mtime
	// for a local document and the zero time for a global one — a vendored
	// global's mtime is its clone date, not a relevance signal, and on a full
	// tie a local document ranks first (local-overrides-global.rule).
	score          int
	typeRank       int
	effectiveMtime time.Time
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
			mcp.Description("Case-insensitive word search against title + body. Default: every whitespace-separated word must occur somewhere in the document, in any order (match=\"all\"). No stemming — singular/plural forms do not match."),
		),
		mcp.WithString("match",
			mcp.Description("How content words must match. \"all\" (default): every word occurs somewhere in the document. \"any\": at least one word occurs. \"exact\": the whole content string as one literal substring."),
			mcp.Enum(string(matchModeAll), string(matchModeAny), string(matchModeExact)),
		),
		mcp.WithString("source",
			mcp.Description("Scope the search to one source: \"local\" (the project's own documents), \"global\" (every mounted global source), or a declared global source id."),
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
			mcp.Description("Result ordering. \"relevance\" (default) = match score DESC (summed match specificity plus a capped content occurrence count) → type priority → mtime DESC (a global's mtime ranks as zero) → path. \"mtime\" = pure mtime DESC."),
			mcp.Enum("relevance", "mtime"),
		),
		mcp.WithString("mode",
			mcp.Description("Output detail. \"snippets\" (default) returns only excerpt windows around matches. \"full\" additionally returns each matched document's full body inline (frontmatter stripped), so you can read the doc without a follow-up get_document. Full mode defaults to limit=3 (max 20)."),
			mcp.Enum(string(searchModeSnippets), string(searchModeFull)),
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
		rawTypes := request.GetStringSlice("types", nil)
		types := make([]templates.DocumentType, len(rawTypes))
		for i, t := range rawTypes {
			types[i] = templates.DocumentType(t)
		}
		for _, tp := range types {
			if !templates.IsValidType(string(tp)) {
				return errorResult(fmt.Sprintf("invalid type %q (valid: %s)", tp, strings.Join(templates.ValidTypes(), ", "))), nil
			}
		}
		status := templates.DocStatus(request.GetString("status", ""))
		mtimeAfterRaw := strings.TrimSpace(request.GetString("mtime_after", ""))
		sortMode := request.GetString("sort", "relevance")
		mode := searchMode(request.GetString("mode", string(searchModeSnippets)))
		// Normalize defensively (the framework enforces the enum, but any value
		// other than "full" maps to the cheaper snippets output).
		if mode != searchModeFull {
			mode = searchModeSnippets
		}
		matchMode := contentMatchMode(request.GetString("match", string(matchModeAll)))
		if matchMode != matchModeExact && matchMode != matchModeAny {
			matchMode = matchModeAll
		}
		sourceFilter := strings.TrimSpace(request.GetString("source", ""))
		// Validate the scope before scanning ("validate then act"): a typo'd
		// source must fail loudly, not return an empty page that an agent could
		// read as a verified absence.
		if !validSourceScope(baseDir, sourceFilter) {
			return errorResult(fmt.Sprintf("invalid source %q (valid: \"local\", \"global\", or a declared global source id)", sourceFilter)), nil
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

		// The content query as match tokens. exact keeps the whole string as one
		// token — including a whitespace-only one, the pre-tokenization substring
		// behavior; all/any split on whitespace and reject a wordless query, which
		// could otherwise only ever produce an empty result
		// (global-recall-guarantees.rfc).
		var tokens []string
		if contentFilter != "" {
			if matchMode == matchModeExact {
				tokens = []string{strings.ToLower(contentFilter)}
			} else {
				tokens = strings.Fields(strings.ToLower(contentFilter))
				if len(tokens) == 0 {
					return errorResult("content must contain at least one word"), nil
				}
			}
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
			docs, err = scanDocumentsFull(baseDir)
		} else {
			docs, err = scanDocuments(baseDir)
		}
		if err != nil {
			return errorResult(sanitizeError("scanning documents", err)), nil
		}

		// Load manifest for relations. A present-but-invalid manifest is a tool
		// error, consistent with get_document and list_relations — a silently
		// empty relation set is exactly the incomplete-context failure the
		// globals work eliminated (missing file = empty manifest, not an error).
		manifest, mErr := sharedManifestStore.load(baseDir)
		if mErr != nil {
			return errorResult(sanitizeError("loading manifest", mErr)), nil
		}
		// One relation index per call: RelationsFor is a linear scan of ALL
		// relations, and calling it per matched document made enrichment
		// O(matched docs × relations).
		outgoingIdx, incomingIdx := buildRelationIndex(manifest.Relations)

		// Coverage counts every scanned document per source, after the source
		// scope and before the query filters: it answers "what did this call
		// search", not "what matched".
		coverage := make(map[string]int)

		results := make([]searchResult, 0, len(docs))
		for _, doc := range docs {
			if !sourceAdmits(sourceFilter, doc.SourceID, doc.SourceKind) {
				continue
			}
			coverage[doc.SourceID]++

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

			// Match evidence accumulators. specSum and freq feed the internal
			// ranking score; the matches slice is the wire evidence.
			matches := make([]searchMatch, 0)
			specSum := 0
			freq := 0

			// path_ref filter. Every hit stays in the wire evidence, but only the
			// MAXIMUM specificity feeds the score: repeating a path is citation,
			// not extra relevance. Content is the deliberate asymmetry — its
			// per-token evidence sums, and stuffing is bounded by contentFreqCap
			// instead.
			if pathRefFilter != "" {
				refs := extractPathRefs(doc.Content)
				refs = filterBareMentions(refs)
				normalizedTarget := strings.TrimPrefix(pathRefFilter, "@")
				pathSpecMax := 0
				for _, r := range refs {
					refPath := strings.TrimPrefix(r.Raw, "@")
					spec := computeSpecificity(refPath, normalizedTarget)
					if spec == 0 {
						continue
					}
					kind := matchKindMention
					if r.Kind == refKindExplicit {
						kind = matchKindExplicit
					}
					excerpt := buildExcerpt(doc.Content, r.Start, len(r.Raw))
					matches = append(matches, searchMatch{
						Kind:        kind,
						Ref:         r.Raw,
						Specificity: spec,
						Excerpt:     excerpt,
					})
					pathSpecMax = max(pathSpecMax, spec)
				}
				if len(matches) == 0 {
					continue
				}
				specSum += pathSpecMax
			}

			// content filter.
			if contentFilter != "" {
				contentMatches, contentSpecSum, contentFreq, found := scoreContent(doc.Title, doc.Content, tokens, matchMode)
				if !found {
					// No content hit means we drop the doc. If path_ref is also
					// active, AND semantics say content must also match.
					continue
				}
				matches = append(matches, contentMatches...)
				specSum += contentSpecSum
				freq = contentFreq
			}

			// When no content/path_ref filter was provided, matches stays empty
			// — that's the pure-metadata case: we still return the doc.

			rank, ok := typePriority[doc.Type]
			if !ok {
				rank = typePriorityDefault
			}

			// A global document's mtime is its clone date, not a relevance
			// signal; the zero time also makes a fully tied local document rank
			// first (local-overrides-global.rule).
			effectiveMtime := doc.ModTime
			if doc.Global {
				effectiveMtime = time.Time{}
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
				score:             100*specSum + freq,
				typeRank:          rank,
				effectiveMtime:    effectiveMtime,
			}

			// In full mode, attach the body so the caller can read the doc
			// without a separate get_document round-trip. Content is present
			// because needsBody forced scanDocumentsFull above.
			if mode == searchModeFull {
				result.Body = stripFrontmatter(doc.Content)
			}

			relPath := normalizeRelPath(doc.Path)
			for _, r := range outgoingIdx[relPath] {
				result.OutgoingRelations = append(result.OutgoingRelations, DocumentRelation{
					Path: ".archcore/" + r.Target,
					Type: string(r.Type),
				})
			}
			for _, r := range incomingIdx[relPath] {
				result.IncomingRelations = append(result.IncomingRelations, DocumentRelation{
					Path: ".archcore/" + r.Source,
					Type: string(r.Type),
				})
			}

			results = append(results, result)
		}

		sortResults(results, sortMode)

		if limit > 0 && len(results) > limit {
			results = ensureSourceRepresentation(results, limit, sortMode)
		}

		data, err := json.Marshal(searchDocumentsResult{
			Results:  results,
			Coverage: coverage,
		})
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

// validSourceScope reports whether scope names a queryable source: empty (no
// scope), one of the two kinds, the reserved global tree, or a declared global
// source id. The ReadGlobals call is an advisory read, fail-open by design: on
// an unreadable settings.json the declared-id set degrades to empty and only
// the id form is rejected — the read tools stay available, while `archcore
// mcp` startup separately fails closed on invalid settings (checkGlobals).
func validSourceScope(baseDir, scope string) bool {
	switch scope {
	case "", string(docs.SourceKindLocal), string(docs.SourceKindGlobal), docs.SourceIDReserved:
		return true
	}
	for _, gs := range config.ReadGlobals(baseDir) {
		if gs.ID == scope {
			return true
		}
	}
	return false
}

// sourceAdmits reports whether the source scope admits a document. An empty
// scope admits everything; "local" and "global" match the kind; anything else
// matches a declared source id exactly.
func sourceAdmits(scope, sourceID string, kind docs.SourceKind) bool {
	switch scope {
	case "":
		return true
	case string(docs.SourceKindLocal):
		return kind == docs.SourceKindLocal
	case string(docs.SourceKindGlobal):
		return kind == docs.SourceKindGlobal
	default:
		return sourceID == scope
	}
}

// scoreContent matches the content tokens against one document and returns the
// wire evidence, the specificity sum, and the capped occurrence count.
//
// Per-token specificity: 3 in the title, 2 in a markdown heading, 1 in the
// body. matchModeAll requires every token; matchModeAny at least one;
// matchModeExact arrives here as a single whole-query token
// (global-recall-guarantees.rfc).
func scoreContent(title, body string, tokens []string, matchMode contentMatchMode) (matches []searchMatch, specSum, freq int, found bool) {
	if len(tokens) == 0 {
		return nil, 0, 0, false
	}
	lowerTitle := strings.ToLower(title)
	lowerBody := strings.ToLower(body)
	// Heading lines, extracted lazily on the first body-tier token: a title hit
	// never pays the line scan, and the extraction runs at most once per doc.
	lowerHeadings := ""
	headingsBuilt := false

	for _, token := range tokens {
		spec := 0
		var excerpt string
		switch {
		case strings.Contains(lowerTitle, token):
			spec = 3
			excerpt = buildExcerpt(title, strings.Index(lowerTitle, token), len(token))
		case strings.Contains(lowerBody, token):
			if !headingsBuilt {
				lowerHeadings = headingLines(lowerBody)
				headingsBuilt = true
			}
			spec = 1
			if strings.Contains(lowerHeadings, token) {
				spec = 2
			}
			excerpt = buildExcerpt(body, strings.Index(lowerBody, token), len(token))
		default:
			if matchMode == matchModeAny {
				continue
			}
			return nil, 0, 0, false // all/exact: one absent token drops the doc
		}
		matches = append(matches, searchMatch{
			Kind:        matchKindContent,
			Ref:         token,
			Specificity: spec,
			Excerpt:     excerpt,
		})
		specSum += spec
		freq += strings.Count(lowerBody, token) + strings.Count(lowerTitle, token)
	}
	if len(matches) == 0 {
		return nil, 0, 0, false
	}
	if freq > contentFreqCap {
		freq = contentFreqCap
	}
	return matches, specSum, freq, true
}

// headingLines returns the markdown heading lines of body joined by newlines.
func headingLines(body string) string {
	var b strings.Builder
	for line := range strings.Lines(body) {
		if strings.HasPrefix(line, "#") {
			b.WriteString(line)
		}
	}
	return b.String()
}

// ensureSourceRepresentation cuts results to limit while keeping every matching
// source represented: a source whose rows the cut would remove entirely gets its
// top row swapped in over the lowest-ranked row of an over-represented source
// (global-recall-guarantees.rfc). Sources claim their slot in rank order of
// their own top row, so when there are more sources than slots the best-ranked
// ones win deterministically. The page is re-sorted before returning.
func ensureSourceRepresentation(results []searchResult, limit int, sortMode string) []searchResult {
	page := slices.Clone(results[:limit])

	onPage := make(map[string]int, 4) // source id -> rows on the page
	for _, r := range page {
		onPage[r.SourceID]++
	}

	// Missing sources, in rank order of their top row past the cut.
	for _, r := range results[limit:] {
		if _, present := onPage[r.SourceID]; present {
			continue
		}
		// Evict from the end: the lowest-ranked row whose source keeps
		// another row, so one guarantee never breaks an earlier one.
		evicted := false
		for i := len(page) - 1; i >= 0; i-- {
			if onPage[page[i].SourceID] > 1 {
				onPage[page[i].SourceID]--
				page[i] = r
				onPage[r.SourceID] = 1
				evicted = true
				break
			}
		}
		if !evicted {
			break // every page row is its source's last — no slot left
		}
	}

	sortResults(page, sortMode)
	return page
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
			Kind:  refKindExplicit,
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
			Kind:  refKindMention,
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
		if r.Kind == refKindExplicit {
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
// order for deterministic output. Relevance keys: score, then type priority,
// then effective mtime (the zero time for a global — its mtime is a clone-date
// artifact), then path for a total deterministic order.
func sortResults(results []searchResult, mode string) {
	slices.SortStableFunc(results, func(a, b searchResult) int {
		if mode == "mtime" {
			return b.ModTime.Compare(a.ModTime)
		}
		if c := cmp.Compare(b.score, a.score); c != 0 {
			return c
		}
		if c := cmp.Compare(a.typeRank, b.typeRank); c != 0 {
			return c
		}
		if c := b.effectiveMtime.Compare(a.effectiveMtime); c != 0 {
			return c
		}
		return strings.Compare(a.Path, b.Path)
	})
}

// buildRelationIndex builds per-path relation lookups once per call. Keys are
// manifest-relative paths (no ".archcore/" prefix), matching RelationsFor.
func buildRelationIndex(rels []sync.Relation) (outgoing, incoming map[string][]sync.Relation) {
	outgoing = make(map[string][]sync.Relation, len(rels))
	incoming = make(map[string][]sync.Relation, len(rels))
	for _, r := range rels {
		outgoing[r.Source] = append(outgoing[r.Source], r)
		incoming[r.Target] = append(incoming[r.Target], r)
	}
	return outgoing, incoming
}
