package tools

// The byte budget of a search_documents response
// (read-tool-responses-survive-host-truncation.adr).

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	// searchResponseByteBudget protects the host's inline tool-result limit. One
	// host stored results from about 50,000 characters upward and refused larger
	// ones by token count; bytes bound characters from above.
	searchResponseByteBudget = 40_000
	// searchBodyFloorBytes protects the use of a full-mode row: a row leaves the
	// page before its body shrinks below this.
	searchBodyFloorBytes = 1024
	// searchBodyFieldBytes reserves the JSON around one body: the body key, its
	// quotes, and the body_truncated and body_bytes fields of a shortened row.
	searchBodyFieldBytes = 64
	// searchIndexByteBudget protects the rows from the index: 200 admitted rows
	// with long titles otherwise fill the whole response budget by themselves.
	searchIndexByteBudget = searchResponseByteBudget / 2
)

// capIndex keeps the leading entries of index that fit searchIndexByteBudget,
// and at least one. The entries are in rank order, so the cut drops the
// lowest-ranked rows.
func capIndex(index []searchIndexEntry) (kept []searchIndexEntry, cut bool, err error) {
	used := 0
	for i, entry := range index {
		data, mErr := json.Marshal(entry)
		if mErr != nil {
			return nil, false, fmt.Errorf("measuring index entry %s: %w", entry.Path, mErr)
		}
		used += len(data) + len(",")
		if used > searchIndexByteBudget && i > 0 {
			return index[:i], true, nil
		}
	}
	return index, false, nil
}

// fitSearchPage returns the rows of the response, at most pageLen of them,
// sized so that headBytes plus the rows stay inside searchResponseByteBudget.
// Rows leave from the tail only while bare rows do not fit; bodies are then
// shortened to share what is left. One row always stays, and its body gives up
// its floor before the response gives up the budget. The rows of ranked are
// never modified.
func fitSearchPage(ranked []searchResult, pageLen, headBytes int, sortMode string) (page []searchResult, truncated bool, err error) {
	if pageLen == 0 {
		return []searchResult{}, false, nil
	}
	bareBytes := make(map[string]int, pageLen)
	bodyBytes := make(map[string]int, pageLen)
	sizeOf := func(r searchResult) (int, error) {
		if size, ok := bareBytes[r.Path]; ok {
			return size, nil
		}
		body := r.Body
		r.Body = ""
		data, mErr := json.Marshal(r)
		if mErr != nil {
			return 0, fmt.Errorf("measuring result %s: %w", r.Path, mErr)
		}
		size := len(data) + len(",")
		if body != "" {
			bodyBytes[r.Path] = jsonStringLen(body)
			size += searchBodyFieldBytes + min(searchBodyFloorBytes, bodyBytes[r.Path])
		}
		bareBytes[r.Path] = size
		return size, nil
	}

	for k := pageLen; k >= 1; k-- {
		if k < len(ranked) {
			page = ensureSourceRepresentation(ranked, k, sortMode)
		} else {
			page = slices.Clone(ranked)
		}
		total := headBytes
		for _, r := range page {
			size, sErr := sizeOf(r)
			if sErr != nil {
				return nil, false, sErr
			}
			total += size
		}
		if total <= searchResponseByteBudget || k == 1 {
			shareBodies(page, bodyBytes, searchResponseByteBudget-total)
			return page, k < pageLen, nil
		}
	}
	return []searchResult{}, false, nil
}

// shareBodies shortens the bodies of page to fit spare bytes beyond the floor
// every row already holds. The split is max-min: a body smaller than an equal
// share arrives whole, and what it leaves unused goes to the larger ones.
func shareBodies(page []searchResult, bodyBytes map[string]int, spare int) {
	var order []int
	for i, r := range page {
		if r.Body != "" {
			order = append(order, i)
		}
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Compare(bodyBytes[page[a].Path], bodyBytes[page[b].Path])
	})
	// A deficit exists only for the one row fitSearchPage keeps by force.
	deficit := max(-spare, 0)
	spare = max(spare, 0)
	for n, i := range order {
		row := &page[i]
		encoded := bodyBytes[row.Path]
		held := max(min(searchBodyFloorBytes, encoded)-deficit, 0)
		grant := held + spare/(len(order)-n)
		if encoded <= grant {
			spare -= encoded - held
			continue
		}
		spare -= grant - held
		row.BodyBytes = len(row.Body)
		row.Body = row.Body[:cutForJSONBudget(row.Body, grant)]
		row.BodyTruncated = true
	}
}

// cutForJSONBudget returns the largest prefix length of s whose JSON encoding
// fits budget bytes. The cut never splits a rune, and it moves back to the last
// line break when one sits in the second half of the prefix.
func cutForJSONBudget(s string, budget int) int {
	end, used := 0, 0
	for end < len(s) {
		size, width := jsonRuneLen(s[end:])
		if used+size > budget {
			break
		}
		used += size
		end += width
	}
	if end == len(s) {
		return end
	}
	if i := strings.LastIndexByte(s[:end], '\n'); i >= end/2 {
		return i + 1
	}
	return end
}

// jsonStringLen returns the number of bytes encoding/json writes for s between
// the quotes, with its default HTML escaping.
func jsonStringLen(s string) int {
	total := 0
	for i := 0; i < len(s); {
		size, width := jsonRuneLen(s[i:])
		total += size
		i += width
	}
	return total
}

// jsonRuneLen returns the encoded size and the source width of the first rune
// of s, which must not be empty.
func jsonRuneLen(s string) (size, width int) {
	const escapedRune = 6 // a six-byte \u escape
	b := s[0]
	if b < utf8.RuneSelf {
		switch {
		case b == '"', b == '\\', b == '\n', b == '\r', b == '\t', b == '\b', b == '\f':
			return len(`\n`), 1
		case b < ' ', b == '<', b == '>', b == '&':
			return escapedRune, 1
		default:
			return 1, 1
		}
	}
	r, width := utf8.DecodeRuneInString(s)
	if (r == utf8.RuneError && width == 1) || r == '\u2028' || r == '\u2029' {
		return escapedRune, width
	}
	return width, width
}
