package agents

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// instructionsBodySHA256 pins the body to InstructionsBlockVersion. A change to
// the body that keeps the version would leave every committed block reading as
// current, so the stale-block advisory would never fire for it.
const instructionsBodySHA256 = "1b79a0bfc58a4876996b723d833aae3ae01532eff987aa2ac7c769e550524c78"

func TestInstructionsBlockVersion_TracksTheBody(t *testing.T) {
	t.Parallel()
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(instructionsBody))); got != instructionsBodySHA256 {
		t.Errorf("instructionsBody changed (sha256 %s): bump InstructionsBlockVersion and the header, then update instructionsBodySHA256", got)
	}
	if want := "block v" + strconv.Itoa(InstructionsBlockVersion) + " "; !strings.Contains(instructionsHeader, want) {
		t.Errorf("instructionsHeader %q does not carry %q", instructionsHeader, want)
	}
}

func TestInspectInstructionBlocks(t *testing.T) {
	t.Parallel()
	legacy := instructionsMarkerStart + " managed by `archcore init` — edit outside these markers\n## old body\n" + instructionsMarkerEnd
	newer := instructionsMarkerStart + " managed by `archcore init`, block v9 — edit outside these markers\n## future body\n" + instructionsMarkerEnd
	edited := strings.Replace(instructionsFencedBlock, "pure mechanics.", "pure mechanics. Local note.", 1)
	crlf := func(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }
	padTo := func(n int) string { return strings.Repeat("x", n-1) + "\n" }
	// Lines that end the block just under the budget in LF and past it in CRLF.
	linesUnderBudget := (codexProjectDocBudget - len(instructionsFencedBlock) - 1) / 16

	tests := []struct {
		name    string
		file    string
		content string
		want    InstructionBlock
		absent  bool
	}{
		{name: "current block", file: "CLAUDE.md", content: "# Notes\n\n" + instructionsFencedBlock + "\n",
			want: InstructionBlock{State: BlockCurrent, Version: InstructionsBlockVersion, Agent: ClaudeCode}},
		{name: "current block with CRLF line ends", file: "AGENTS.md", content: crlf(instructionsFencedBlock + "\n"),
			want: InstructionBlock{State: BlockCurrent, Version: InstructionsBlockVersion, Agent: CodexCLI}},
		{name: "header without a version reads as v1", file: "AGENTS.md", content: legacy + "\n",
			want: InstructionBlock{State: BlockOutdated, Version: 1, Agent: CodexCLI}},
		{name: "current version edited inside the markers", file: "GEMINI.md", content: edited + "\n",
			want: InstructionBlock{State: BlockOutdated, Version: InstructionsBlockVersion, Agent: GeminiCLI}},
		{name: "current block followed by a stale duplicate", file: "CLAUDE.md",
			content: instructionsFencedBlock + "\n\n" + legacy + "\n",
			want:    InstructionBlock{State: BlockOutdated, Version: InstructionsBlockVersion, Agent: ClaudeCode}},
		{name: "block from a newer CLI", file: "AGENTS.md", content: newer + "\n",
			want: InstructionBlock{State: BlockNewer, Version: 9, Agent: CodexCLI}},
		{name: "LF block ending inside the Codex budget", file: "AGENTS.md",
			content: strings.Repeat("user notes line\n", linesUnderBudget) + instructionsFencedBlock + "\n",
			want:    InstructionBlock{State: BlockCurrent, Version: InstructionsBlockVersion, Agent: CodexCLI}},
		{name: "block ending exactly at the Codex budget", file: "AGENTS.md",
			content: padTo(codexProjectDocBudget-len(instructionsFencedBlock)) + instructionsFencedBlock,
			want:    InstructionBlock{State: BlockCurrent, Version: InstructionsBlockVersion, Agent: CodexCLI}},
		{name: "block ending one byte past the Codex budget", file: "AGENTS.md",
			content: padTo(codexProjectDocBudget-len(instructionsFencedBlock)+1) + instructionsFencedBlock,
			want:    InstructionBlock{State: BlockCurrent, Version: InstructionsBlockVersion, Agent: CodexCLI, PastCodexBudget: true}},
		{name: "CRLF block whose carriage returns push it past the Codex budget", file: "AGENTS.md",
			content: crlf(strings.Repeat("user notes line\n", linesUnderBudget) + instructionsFencedBlock + "\n"),
			want:    InstructionBlock{State: BlockCurrent, Version: InstructionsBlockVersion, Agent: CodexCLI, PastCodexBudget: true}},
		{name: "CLAUDE.md is never held to the Codex budget", file: "CLAUDE.md",
			content: strings.Repeat("user notes line\n", 2*linesUnderBudget) + instructionsFencedBlock + "\n",
			want:    InstructionBlock{State: BlockCurrent, Version: InstructionsBlockVersion, Agent: ClaudeCode}},
		{name: "no block", file: "CLAUDE.md", content: "# Only user notes\n", absent: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			writeFile(t, filepath.Join(base, tt.file), tt.content)
			got := InspectInstructionBlocks(base)
			if tt.absent {
				if len(got) != 0 {
					t.Fatalf("got %+v, want no block", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("got %d blocks, want 1: %+v", len(got), got)
			}
			tt.want.Path = filepath.Join(base, tt.file)
			if got[0] != tt.want {
				t.Errorf("got %+v, want %+v", got[0], tt.want)
			}
		})
	}
}

func TestInstructionFiles_AgentWritesItsFile(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	for _, file := range instructionFiles {
		agent := ByID(file.agent)
		if agent == nil {
			t.Fatalf("%s names unknown agent %q", file.name, file.agent)
		}
		if got, want := agent.InstructionsPath(base), filepath.Join(base, file.name); got != want {
			t.Errorf("agent %s writes %s, not %s", file.agent, got, want)
		}
	}
}
