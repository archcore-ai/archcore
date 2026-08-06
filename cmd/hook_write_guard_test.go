package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The case names below mirror test/unit/check-archcore-write.bats in the plugin
// one for one, so parity between the shell guard and this port is a grep across
// two repositories rather than a reading exercise. A case that exists there and
// not here is a hole.

func TestWriteGuard_Decisions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		filePath string
		wantDeny bool
	}{
		{name: "blocks write to .archcore/*.md", filePath: ".archcore/knowledge/a.adr.md", wantDeny: true},
		{name: "blocks edit to .archcore/*.md", filePath: ".archcore/a.rule.md", wantDeny: true},
		{name: "blocks nested .archcore/ path", filePath: ".archcore/deep/nested/doc.prd.md", wantDeny: true},
		{name: "allows .archcore/settings.json", filePath: ".archcore/settings.json"},
		{name: "allows .archcore/.sync-state.json", filePath: ".archcore/.sync-state.json"},
		{name: "allows regular file", filePath: "src/main.go"},
		{name: "allows when no file_path", filePath: ""},
		{name: "allows a non-markdown file inside .archcore/", filePath: ".archcore/notes.txt"},
		// The shell pattern ".archcore/*.md" matched any path containing that
		// literal, so a sibling directory whose name merely ended in ".archcore"
		// was guarded as if it were the store. Segment matching ends that.
		{name: "allows a directory that only ends in .archcore", filePath: "my.archcore/x.md"},
		// Read-only global space is blocked too — for a different reason, but a
		// direct write there is equally wrong.
		{name: "blocks a write into the reserved global tree", filePath: ".archcore/global/company/x.rule.md", wantDeny: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			dec := writeGuardDecision(base, tt.filePath)
			if dec.deny != tt.wantDeny {
				t.Errorf("writeGuardDecision(%q).deny = %v, want %v", tt.filePath, dec.deny, tt.wantDeny)
			}
		})
	}
}

// TestWriteGuard_BlockMessageMentionsMCPTools mirrors "block message mentions
// MCP tools". A refusal that does not say what to do instead just looks broken.
func TestWriteGuard_BlockMessageMentionsMCPTools(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	dec := writeGuardDecision(base, ".archcore/knowledge/a.adr.md")

	if !dec.deny {
		t.Fatal("expected a deny")
	}
	for _, want := range []string{"create_document", "update_document", "remove_document"} {
		if !strings.Contains(dec.reason, want) {
			t.Errorf("block message missing %q:\n%s", want, dec.reason)
		}
	}
}

// TestWriteGuard_AbsolutePaths: hosts send absolute paths as often as relative
// ones, and a path outside the project is none of the guard's business.
func TestWriteGuard_AbsolutePaths(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)

	inside := filepath.Join(base, ".archcore", "knowledge", "a.adr.md")
	if dec := writeGuardDecision(base, inside); !dec.deny {
		t.Errorf("absolute path inside the store was allowed: %s", inside)
	}

	outside := filepath.Join(t.TempDir(), ".archcore", "knowledge", "a.adr.md")
	if dec := writeGuardDecision(base, outside); dec.deny {
		t.Errorf("absolute path outside the project was blocked: %s", outside)
	}
}

// TestWriteGuard_SymlinkEscapeIsBlocked: a document path whose ancestor is a
// symlink out of the store must be refused here, because MCP refuses it. The
// guard's premise is that going around the tools cannot reach a path the tools
// would reject; an allow here makes that false.
func TestWriteGuard_SymlinkEscapeIsBlocked(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	outside := t.TempDir()
	link := filepath.Join(base, ".archcore", "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	dec := writeGuardDecision(base, ".archcore/escape/x.adr.md")

	if !dec.deny {
		t.Error("a document reachable only through a symlink out of .archcore/ was allowed")
	}
}

// TestWriteGuard_AllowsWhenArchcoreMissing: with no .archcore/ there is no store
// to protect, and a deny would turn an uninitialized project into a blocked one.
func TestWriteGuard_AllowsWhenArchcoreMissing(t *testing.T) {
	t.Parallel()
	base := t.TempDir()

	if dec := writeGuardDecision(base, ".archcore/knowledge/a.adr.md"); dec.deny {
		t.Error("write was blocked in a project that has no .archcore/")
	}
}

// TestWriteGuard_ThroughPreToolUse walks the payload shapes hosts actually send,
// mirroring the per-host rows of the bats suite. The guard must reach the same
// verdict whichever dialect delivered the path.
func TestWriteGuard_ThroughPreToolUse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		payload  string
		wantDeny bool
	}{
		{
			name:     "claude-code: blocks write to .archcore/*.md",
			payload:  `{"tool_name":"Write","tool_input":{"file_path":".archcore/a.adr.md"}}`,
			wantDeny: true,
		},
		{
			name:     "cursor: blocks write to .archcore/*.md",
			payload:  `{"conversation_id":"c","tool_name":"Write","tool_input":{"file_path":".archcore/a.adr.md"}}`,
			wantDeny: true,
		},
		{
			name:     "codex: blocks apply_patch to .archcore/*.md",
			payload:  `{"turn_id":"t","tool_name":"apply_patch","tool_input":{"file_path":".archcore/a.adr.md"}}`,
			wantDeny: true,
		},
		{
			name:     "copilot: denies write to .archcore/*.md via native toolArgs",
			payload:  `{"toolName":"create","toolArgs":{"path":".archcore/a.adr.md"}}`,
			wantDeny: true,
		},
		{
			name:     "copilot: legacy hybrid payload is also denied",
			payload:  `{"hookEventName":"preToolUse","tool_name":"edit","tool_input":{"file_path":".archcore/a.adr.md"}}`,
			wantDeny: true,
		},
		{
			name:    "claude-code: allows regular file",
			payload: `{"tool_name":"Write","tool_input":{"file_path":"src/main.go"}}`,
		},
		{
			name:    "copilot: allows regular file silently",
			payload: `{"toolName":"create","toolArgs":{"path":"src/main.go"}}`,
		},
		{
			name:    "allows empty stdin",
			payload: ``,
		},
		{
			name:    "allows an unparsable payload",
			payload: `{{{`,
		},
		// A document mutation through MCP is the sanctioned path. The bare "path"
		// key means a document there, not a file, and reading it as a file would
		// make the guard block the very route it exists to enforce.
		{
			name:    "an MCP document mutation is never blocked",
			payload: `{"tool_name":"mcp__archcore__update_document","tool_input":{"path":".archcore/a.adr.md"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := setupArchcoreDir(t)
			p := decodeHookPayload(strings.NewReader(tt.payload))
			dec := preToolUseHandler(bg(), hookRequest{baseDir: base, dialect: hookDialects[0], event: eventPreToolUse, payload: p})
			if dec.deny != tt.wantDeny {
				t.Errorf("preToolUseHandler.deny = %v, want %v (payload %s)", dec.deny, tt.wantDeny, tt.payload)
			}
		})
	}
}

// TestWriteGuard_DeclaredExternalGlobalIsBlocked: a global source mounted from
// outside the store is read-only like any other, but its documents render with a
// leading ".." — so the lexical check rejects them and the guard used to fall
// straight through to "outside the project, none of my business". The MCP write
// tools refuse those paths outright, so an allow here meant the store's
// read-only mount was editable by going around them.
func TestWriteGuard_DeclaredExternalGlobalIsBlocked(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		file     string // relative to the fixture root, joined below
		abs      bool   // pass the absolute path instead of the relative one
		wantDeny bool
	}{
		{name: "absolute path into a declared external global", file: "company/.archcore/x.rule.md", abs: true, wantDeny: true},
		{name: "relative path into a declared external global", file: "company/.archcore/x.rule.md", wantDeny: true},
		{name: "nested document in a declared external global", file: "company/.archcore/knowledge/deep.adr.md", abs: true, wantDeny: true},
		// Case-folded for the same reason the in-tree check folds: on APFS or
		// NTFS this reaches the very same directory.
		{name: "case-variant path into a declared external global", file: "COMPANY/.archcore/x.rule.md", abs: true, wantDeny: true},
		// GuardWritablePath classifies these ErrPathNotDocument and the guard
		// allows them in-tree; the verdicts have to match.
		{name: "settings.json inside a declared external global", file: "company/.archcore/settings.json", abs: true},
		{name: "a non-markdown file inside a declared external global", file: "company/.archcore/notes.txt", abs: true},
		// An undeclared sibling is an ordinary directory outside the project.
		{name: "an undeclared sibling directory", file: "vendor/.archcore/x.rule.md", abs: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			base := filepath.Join(root, "project")
			if err := os.MkdirAll(filepath.Join(base, ".archcore"), 0o755); err != nil {
				t.Fatal(err)
			}
			settings := filepath.Join(base, ".archcore", "settings.json")
			if err := os.WriteFile(settings,
				[]byte(`{"sync":"none","globals":[{"id":"company","path":"../company/.archcore"}]}`), 0o644); err != nil {
				t.Fatal(err)
			}

			target := filepath.Join(root, filepath.FromSlash(tt.file))
			if !tt.abs {
				rel, err := filepath.Rel(base, target)
				if err != nil {
					t.Fatal(err)
				}
				target = rel
			}

			if dec := writeGuardDecision(base, target); dec.deny != tt.wantDeny {
				t.Errorf("writeGuardDecision(%q).deny = %v, want %v", target, dec.deny, tt.wantDeny)
			}
		})
	}
}

// TestWriteGuard_CaseVariantGlobalBlocked: on a case-insensitive filesystem
// ".archcore/Global/x" resolves to the reserved tree, so the guard folds case
// or the read-only invariant is bypassable by typing a capital letter.
func TestWriteGuard_CaseVariantGlobalBlocked(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	if err := os.MkdirAll(filepath.Join(base, ".archcore", "Global"), 0o755); err != nil {
		t.Fatal(err)
	}

	if dec := writeGuardDecision(base, ".archcore/Global/x.rule.md"); !dec.deny {
		t.Error("a case-variant global path was allowed")
	}
}
