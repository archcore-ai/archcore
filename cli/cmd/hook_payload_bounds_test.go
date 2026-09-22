package cmd

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

// countingReader records how much of the input was actually consumed.
type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

// TestDecodeHookPayload_StopsAtTheLimit: the decoder must stop reading rather
// than pull an arbitrarily large payload into memory. A PostToolUse payload
// echoes tool_response, so its size is the host's choice, not ours.
func TestDecodeHookPayload_StopsAtTheLimit(t *testing.T) {
	t.Parallel()
	oversized := fmt.Sprintf(`{"tool_name":"Write","filler":%q}`,
		strings.Repeat("x", 2*maxPayloadBytes))
	cr := &countingReader{r: strings.NewReader(oversized)}

	p := decodeHookPayload(cr)

	if cr.n > maxPayloadBytes+1 {
		t.Errorf("read %d bytes, want at most %d", cr.n, maxPayloadBytes+1)
	}
	// Truncated JSON does not parse, and an unparsable payload allows.
	if p.toolName() != "" {
		t.Errorf("an over-cap payload decoded to %q, want an empty payload", p.toolName())
	}
}

// TestDecodeHookPayload_AcceptsAPayloadUnderTheLimit guards against a cap so
// tight it rejects an ordinary edit — which would silently disable the guard.
func TestDecodeHookPayload_AcceptsAPayloadUnderTheLimit(t *testing.T) {
	t.Parallel()
	body := strings.Repeat("a", 1<<20) // a 1 MiB file body, well within reach
	payload := fmt.Sprintf(`{"tool_name":"Write","tool_input":{"file_path":".archcore/a.adr.md","content":%q}}`, body)

	p := decodeHookPayload(strings.NewReader(payload))

	if got := p.filePath(); got != ".archcore/a.adr.md" {
		t.Errorf("filePath() = %q, want the path from a 1 MiB payload", got)
	}
}

// TestDecodeHookPayload_TrailingBytesStillGuard pins the deliberate laxity: the
// decoder reads the leading object and ignores whatever follows.
//
// This looks like a hole and is the opposite of one. Requiring the whole input
// to be exactly one JSON object would turn each of these into an empty payload,
// and an empty payload allows — so a write into .archcore/ would go through by
// appending a byte to the payload. The strict reading IS the bypass; the lax one
// keeps the guard firing.
func TestDecodeHookPayload_TrailingBytesStillGuard(t *testing.T) {
	t.Parallel()
	const object = `{"tool_name":"Write","tool_input":{"file_path":".archcore/a.adr.md"}}`
	tests := []struct {
		name    string
		payload string
	}{
		{name: "exactly one object", payload: object},
		{name: "trailing whitespace", payload: object + "\n\n"},
		{name: "trailing garbage", payload: object + "trailing garbage"},
		{name: "a second object", payload: object + `{"tool_name":"Read"}`},
		{name: "a NUL and binary junk", payload: object + "\x00\x01\x02\xff"},
		{name: "an oversized suffix", payload: object + strings.Repeat("x", 2*maxPayloadBytes)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			p := decodeHookPayload(strings.NewReader(tt.payload))

			if got := p.filePath(); got != ".archcore/a.adr.md" {
				t.Errorf("filePath() = %q, want the path from the leading object", got)
			}
			dec := preToolUseHandler(bg(), hookRequest{baseDir: base, dialect: hookDialects[0], event: eventPreToolUse, payload: p})
			if !dec.deny {
				t.Error("a document write was allowed because of what followed the payload")
			}
		})
	}
}

// TestPostToolUse_SkipsReadTools: the post-write checks scan the corpus. Cursor
// installs its post-MCP event without a matcher, so a read tool reaching them
// costs a full scan and reports on a call that changed nothing.
func TestPostToolUse_SkipsReadTools(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/bad.adr.md", "no frontmatter at all\n")

	readTools := []string{
		"mcp__archcore__get_document",
		"mcp__archcore__list_documents",
		"mcp__archcore__search_documents",
		"mcp__archcore__list_relations",
	}
	for _, tool := range readTools {
		t.Run(tool, func(t *testing.T) {
			dec := postToolUseHandler(bg(), postReq(t, base, tool, ".archcore/knowledge/bad.adr.md"))
			if dec.context != "" {
				t.Errorf("a read tool produced advisory output:\n%s", dec.context)
			}
		})
	}
}

// TestPostToolUse_RunsForEveryMutatingTool is the positive control: without it
// the test above passes when the checks are disabled entirely.
func TestPostToolUse_RunsForEveryMutatingTool(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/bad.adr.md", "no frontmatter at all\n")

	for _, tool := range []string{
		"mcp__archcore__create_document",
		"mcp__archcore__update_document",
		"mcp__archcore__remove_document",
		"mcp__archcore__add_relation",
		"mcp__archcore__remove_relation",
	} {
		t.Run(tool, func(t *testing.T) {
			dec := postToolUseHandler(bg(), postReq(t, base, tool, ".archcore/knowledge/bad.adr.md"))
			if dec.context == "" {
				t.Error("a mutating tool produced no advisory output")
			}
		})
	}
}
