package cmd

import (
	"strings"
	"testing"
)

// These exercise the pre-write handler, not the alignment engine: what the
// handler decides to produce for a given host, and whether it can ever block.

// TestCodeAlignment_NeverBlocks: the advisory half must never change the
// verdict on someone's edit, whatever it does internally.
func TestCodeAlignment_NeverBlocks(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentHandlerDoc(t, base, "knowledge/api.rule.md", "API Rule", "Applies to src/api/ code.")

	p := decodeHookPayload(strings.NewReader(`{"tool_name":"Write","tool_input":{"file_path":"src/api/h.go"}}`))
	dec := preToolUseHandler(bg(), hookRequest{baseDir: base, dialect: hookDialects[0], event: eventPreToolUse, payload: p})

	if dec.deny {
		t.Error("code alignment produced a deny")
	}
	if dec.context == "" {
		t.Error("expected injected context")
	}
}

// TestCodeAlignment_NotEmittedOnCopilot: Copilot's preToolUse accepts only a
// permission decision, so context sent there is discarded and a second stdout
// document would break its single-parse contract.
//
// The handler is dialect-aware, so it does not produce the context at all —
// which also means it never starts the corpus scan that would build it.
func TestCodeAlignment_NotEmittedOnCopilot(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeAlignmentHandlerDoc(t, base, "knowledge/api.rule.md", "API Rule", "Applies to src/api/ code.")

	const payload = `{"toolName":"edit","toolArgs":{"path":"src/api/h.go"}}`

	copilot := preToolUseHandler(bg(), reqFor(t, "copilot", base, payload))
	if copilot.context != "" {
		t.Errorf("copilot received pre-write context it cannot read:\n%s", copilot.context)
	}

	// Positive control: without it this passes when code alignment is broken
	// everywhere, not just suppressed on this host.
	claude := preToolUseHandler(bg(), reqFor(t, "claude-code", base, payload))
	if claude.context == "" {
		t.Error("a host that accepts pre-write context received none")
	}
}

// writeAlignmentHandlerDoc writes a document that mentions a source root.
func writeAlignmentHandlerDoc(t *testing.T, base, relPath, title, body string) {
	t.Helper()
	writeArchcoreDoc(t, base, relPath, "---\ntitle: \""+title+"\"\nstatus: accepted\n---\n\n"+body+"\n")
}
