package agents

import (
	"os"
	"path/filepath"
	"testing"
)

// InstructionBlockPresent is the ground-truth predicate the wiring report relies
// on to name exactly the instruction files that landed on disk (internal/wiring
// Apply). A regression here silently mis-reports partial writes, so pin every
// branch directly rather than only through the wiring integration tests.
func TestInstructionBlockPresent(t *testing.T) {
	t.Parallel()

	t.Run("present block is detected", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "AGENTS.md")
		writeFile(t, path, "# user notes\n\n"+instructionsFencedBlock+"\n")
		if !InstructionBlockPresent(path) {
			t.Error("a file carrying the managed block must read as present")
		}
	})

	t.Run("file without a block reads as absent", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "AGENTS.md")
		writeFile(t, path, "# just user content, no markers\n")
		if InstructionBlockPresent(path) {
			t.Error("a file with no managed block must read as absent")
		}
	})

	// A lone start marker (no paired end) is ordinary user content, not a managed
	// block — findManagedSpans leaves it unpaired. This is what separates
	// InstructionBlockPresent from a naive strings.Contains(markerStart): a
	// "simplification" to Contains would wrongly report present here.
	t.Run("orphaned start marker is not a block", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "AGENTS.md")
		writeFile(t, path, instructionsMarkerStart+"\nno end marker follows\n")
		if InstructionBlockPresent(path) {
			t.Error("an unpaired start marker must not count as a present block")
		}
	})

	t.Run("missing file reads as absent", func(t *testing.T) {
		t.Parallel()
		path := filepath.Join(t.TempDir(), "does-not-exist.md")
		if InstructionBlockPresent(path) {
			t.Error("a missing file must read as absent, not error")
		}
	})

	// wiring forces AGENTS.md to be a directory to simulate a failed write; the
	// predicate must treat that unreadable path as absent so the report omits the
	// file instead of crediting an unwritten agent.
	t.Run("directory path reads as absent", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(t.TempDir(), "AGENTS.md")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if InstructionBlockPresent(dir) {
			t.Error("a directory path must read as absent")
		}
	})
}
