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

// TestToolAdvisory_DoubleRegistrationAdvisesOnce pins plugin-cli-compatibility.rule
// §5: the plugin hook and the project hook receive one tool call id, and only
// the first advises; a new call advises again.
func TestToolAdvisory_DoubleRegistrationAdvisesOnce(t *testing.T) {
	dir := t.TempDir()
	prev := toolStampDir
	toolStampDir = func() string { return dir }
	t.Cleanup(func() { toolStampDir = prev })

	base := setupArchcoreDir(t)
	writeAlignmentHandlerDoc(t, base, "knowledge/api.rule.md", "API Rule", "Applies to src/api/ code.")
	call := func(id string) hookDecision {
		p := decodeHookPayload(strings.NewReader(`{"session_id":"s1","tool_use_id":"` + id + `","tool_name":"Write","tool_input":{"file_path":"src/api/h.go"}}`))
		return preToolUseHandler(bg(), hookRequest{baseDir: base, dialect: hookDialects[0], event: eventPreToolUse, payload: p})
	}

	if call("t1").context == "" {
		t.Fatal("the first copy of a call must advise")
	}
	if got := call("t1").context; got != "" {
		t.Errorf("the second copy of one call advised again:\n%s", got)
	}
	if call("t2").context == "" {
		t.Error("a new tool call must advise")
	}
}

// TestPostToolUse_DoubleRegistrationAdvisesOnce pins plugin-cli-compatibility.rule
// §5 for the post-write checks: one document write reports once.
func TestPostToolUse_DoubleRegistrationAdvisesOnce(t *testing.T) {
	dir := t.TempDir()
	prev := toolStampDir
	toolStampDir = func() string { return dir }
	t.Cleanup(func() { toolStampDir = prev })

	base := setupArchcoreDir(t)
	writeArchcoreDoc(t, base, "knowledge/bad.adr.md", "no frontmatter at all\n")
	call := func(id string) hookDecision {
		p := decodeHookPayload(strings.NewReader(`{"session_id":"s1","tool_use_id":"` + id + `","tool_name":"mcp__archcore__update_document","tool_input":{"path":".archcore/knowledge/bad.adr.md"}}`))
		return postToolUseHandler(bg(), hookRequest{baseDir: base, dialect: hookDialects[0], event: eventPostToolUse, payload: p})
	}

	if call("w1").context == "" {
		t.Fatal("the first copy of a write must report")
	}
	if got := call("w1").context; got != "" {
		t.Errorf("the second copy of one write reported again:\n%s", got)
	}
	if call("w2").context == "" {
		t.Error("a new write must report")
	}
}
