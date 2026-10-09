package docs

import (
	"path/filepath"
	"regexp"
	"strings"

	"archcore-cli/templates"
)

// Path-reference candidate kinds. They never leave the process, so they stay
// untyped under the naming rule's §G MAY.
const (
	RefKindExplicit = "explicit"
	RefKindMention  = "mention_candidate"
)

// PathRef is a single path-reference candidate extracted from a document body.
type PathRef struct {
	Raw   string // e.g. "@src/payments/" or "src/payments/stripe.ts"
	Kind  string // RefKindExplicit or RefKindMention
	Start int    // byte offset of the first character in the source body
}

var (
	// Explicit @-prefixed references.
	pathRefExplicitRe = regexp.MustCompile(`@[\w./-]+`)
	// Bare mention candidates: identifier / path.
	pathRefBareRe = regexp.MustCompile(`[\w-]+/[\w./-]+`)
)

// ExtractPathRefs runs the two path-ref regexes over body and returns all
// matches as PathRef values. Explicit matches are tagged "explicit"; bare
// candidates are tagged "mention_candidate" and need FilterBareMentions.
func ExtractPathRefs(body string) []PathRef {
	var refs []PathRef
	// Explicit first — we record their offsets so we can skip bare hits that
	// overlap (the bare regex would otherwise re-match the same token minus
	// the leading "@").
	explicitSpans := pathRefExplicitRe.FindAllStringIndex(body, -1)
	for _, m := range explicitSpans {
		refs = append(refs, PathRef{
			Raw:   body[m[0]:m[1]],
			Kind:  RefKindExplicit,
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
		refs = append(refs, PathRef{
			Raw:   body[start:end],
			Kind:  RefKindMention,
			Start: start,
		})
	}
	return refs
}

// FilterBareMentions keeps explicit refs unchanged and drops bare
// mention-candidates unless one of these heuristics holds:
//   - the candidate ends with '/'
//   - the candidate has ≥2 '/' separators
//   - the final segment's extension is a known source extension
func FilterBareMentions(candidates []PathRef) []PathRef {
	out := make([]PathRef, 0, len(candidates))
	for _, r := range candidates {
		if r.Kind == RefKindExplicit {
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
