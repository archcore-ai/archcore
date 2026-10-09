package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"archcore-cli/internal/agents"
)

const legacyManagedBlock = "<!-- archcore:start --> managed by `archcore init` — edit outside these markers\n## Archcore — project context for this repo\n<!-- archcore:end -->\n"

func writeInstructionFile(t *testing.T, base, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(base, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func currentManagedBlock(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	if err := agents.ByID(agents.GeminiCLI).WriteInstructions(base); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(base, "GEMINI.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDescribeInstructionBlocks(t *testing.T) {
	t.Parallel()
	current := currentManagedBlock(t)
	tests := []struct {
		name    string
		file    string
		content string
		want    []string
	}{
		{name: "legacy block asks for a refresh", file: "AGENTS.md", content: legacyManagedBlock,
			want: []string{"AGENTS.md holds an older Archcore block (v1; this CLI writes v2)",
				"run 'archcore instructions install --agent codex-cli'"}},
		{name: "refresh names the agent that writes the file", file: "GEMINI.md", content: legacyManagedBlock,
			want: []string{"GEMINI.md holds an older Archcore block", "--agent gemini-cli"}},
		{name: "edited current block is not called older", file: "CLAUDE.md",
			content: strings.Replace(current, "pure mechanics.", "pure mechanics. Local note.", 1),
			want: []string{"CLAUDE.md holds an Archcore block that differs from the one this CLI writes",
				"--agent claude-code"}},
		{name: "newer block asks for a CLI update", file: "AGENTS.md",
			content: strings.Replace(legacyManagedBlock, "`archcore init`", "`archcore init`, block v7", 1),
			want:    []string{"from a newer CLI (v7; this CLI writes v2)", "run 'archcore update'"}},
		{name: "block past the Codex budget asks for a move", file: "AGENTS.md",
			content: strings.Repeat("user notes line\n", 40<<10/16) + current,
			want:    []string{"past 32 KiB", "move the block to the top of the file"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			writeInstructionFile(t, base, tt.file, tt.content)
			var got strings.Builder
			for _, note := range describeInstructionBlocks(base) {
				got.WriteString(note.problem + " " + note.fix + "\n")
			}
			for _, want := range tt.want {
				if !strings.Contains(got.String(), want) {
					t.Errorf("notes %q lack %q", got.String(), want)
				}
			}
			if strings.Contains(got.String(), "older") && strings.Contains(got.String(), "v2; this CLI writes v2") {
				t.Errorf("a v2 block is called older: %q", got.String())
			}
		})
	}
}

func TestDescribeInstructionBlocks_CurrentBlockIsSilent(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	if err := agents.ByID("claude-code").WriteInstructions(base); err != nil {
		t.Fatal(err)
	}
	if got := describeInstructionBlocks(base); len(got) != 0 {
		t.Errorf("current blocks produced notes: %q", got)
	}
}

func TestInstructionsAdvisory_OncePerWindow(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	writeInstructionFile(t, base, "CLAUDE.md", legacyManagedBlock)
	stampDir := t.TempDir()

	first := instructionsAdvisory(base, stampDir)
	want := "[Archcore Instructions] CLAUDE.md holds an older Archcore block (v1; this CLI writes v2). " +
		"With the user's consent, run 'archcore instructions install --agent claude-code'.\n"
	if first != want {
		t.Fatalf("first advisory = %q, want %q", first, want)
	}
	if second := instructionsAdvisory(base, stampDir); second != "" {
		t.Errorf("second advisory inside the window = %q, want empty", second)
	}
}

func TestInstructionsAdvisory_CurrentBlocksLeaveTheWindowUnclaimed(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	writeInstructionFile(t, base, "GEMINI.md", currentManagedBlock(t))
	stampDir := t.TempDir()

	if got := instructionsAdvisory(base, stampDir); got != "" {
		t.Fatalf("advisory for current blocks = %q, want empty", got)
	}
	writeInstructionFile(t, base, "GEMINI.md", legacyManagedBlock)
	if got := instructionsAdvisory(base, stampDir); got == "" {
		t.Error("a session with nothing to report claimed the day's advisory")
	}
}

func TestReportInstructionBlocks_WarnsWithTheFix(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	writeInstructionFile(t, base, "CLAUDE.md", legacyManagedBlock)

	var out bytes.Buffer
	reportInstructionBlocks(&out, base)
	got := out.String()
	if !strings.Contains(got, "CLAUDE.md holds an older Archcore block (v1; this CLI writes v2) — run 'archcore instructions install --agent claude-code'") {
		t.Errorf("doctor report = %q", got)
	}
	if strings.Contains(got, "consent") {
		t.Errorf("doctor report addresses an agent, not the user: %q", got)
	}
}

func TestBuildSessionContext_StaleBlockPrecedesTags(t *testing.T) {
	t.Parallel()
	base := setupArchcoreDir(t)
	writeDoc(t, base, "knowledge", "tagged.adr.md", "---\ntitle: Tagged\nstatus: draft\ntags:\n  - billing\n---\n\nBody.\n")
	writeInstructionFile(t, base, "AGENTS.md", legacyManagedBlock)

	ctx, _ := buildSessionContext(bg(), base)
	advisory := strings.Index(ctx, "[Archcore Instructions] AGENTS.md holds an older Archcore block")
	tags := strings.Index(ctx, "EXISTING TAGS")
	if advisory == -1 || tags == -1 {
		t.Fatalf("session context lacks the advisory or the tags; ctx=%q", ctx)
	}
	if advisory > tags {
		t.Errorf("advisory at %d follows EXISTING TAGS at %d; ctx=%q", advisory, tags, ctx)
	}
}
