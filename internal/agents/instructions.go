package agents

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Instruction-nudge files. archcore writes a short, always-on "use Archcore"
// hint into each host's instruction file so that CLI-only users (no Archcore
// plugin) discover and invoke the MCP tools. Under Tool Search, MCP tools are
// deferred — only names load at startup — so an always-on instruction nudge is
// the discovery trigger. See
// .archcore/integrations/instruction-nudge-on-init.adr.md.
//
// Two file kinds:
//   - Shared user files (AGENTS.md, GEMINI.md): fenced-block upsert — archcore
//     only ever touches the span between its markers, never user content.
//   - Owned files (.claude/rules/archcore.md): archcore owns the whole file.

const (
	// instructionsMarkerStart and instructionsMarkerEnd delimit the archcore-
	// managed span inside a shared instruction file. A single shared marker pair
	// (rather than per-host markers) lets the Archcore plugin target the same
	// block later.
	instructionsMarkerStart = "<!-- archcore:start -->"
	instructionsMarkerEnd   = "<!-- archcore:end -->"

	// instructionsHeader is the full start-marker line, including the human note
	// that steers manual edits outside the managed span.
	instructionsHeader = instructionsMarkerStart + " managed by `archcore init` — edit outside these markers"
)

// instructionsBody is the host-neutral nudge. It is outcome-first, references
// Archcore through its MCP tools (not plugin slash commands — CLI-only users
// have no plugin), and splits the cheap discovery search from the selective
// deep read so invocation tracks relevance, not volume: lean on the search,
// skip only turns the repo would have no opinion on. The skip is keyed on the
// nature of the turn — a prior the agent can form up front — not on whether a
// rule exists, which it cannot know without the very lookup it is told to skip.
// It also flags that a project MAY mount read-only global sources (phrased
// conditionally — most projects have none) that the session-start context omits,
// so the agent learns to surface them via the MCP read tools when present.
// Built via concatenation because a raw string literal cannot contain the
// backticks that wrap `.archcore/`.
const instructionsBody = "## Archcore — project context for this repo\n" +
	"\n" +
	"This repo's architecture, decisions, rules, specs and patterns live in `.archcore/`,\n" +
	"reachable through the Archcore MCP tools. Consult them even on code you think you\n" +
	"know — a decision or rule may already constrain it.\n" +
	"\n" +
	"- Touching this repo's real code or behavior → search first; read only what matches.\n" +
	"- A decision was made (\"we'll use X\", \"from now on Y\") → record it.\n" +
	"- A module / API / system has no doc — or a search comes back empty → capture it.\n" +
	"- Planning a feature or refactor → scope it against what's already decided.\n" +
	"\n" +
	"A `.archcore/` may also mount read-only **global sources** — shared, org-wide\n" +
	"context not shown in the session-start list. `list_documents` / `search_documents`\n" +
	"surface them alongside local docs, tagged `source_kind: \"global\"`. When present,\n" +
	"treat them as defaults a local doc can override — never edit or relate to one.\n" +
	"\n" +
	"The search is cheap — lean on it. Skip it only for turns this repo would have no\n" +
	"opinion on: syntax trivia, throwaway snippets, pure mechanics."

// instructionsFencedBlock is the full managed block (markers + body) written
// into shared instruction files. Compile-time constant concatenation.
const instructionsFencedBlock = instructionsHeader + "\n" + instructionsBody + "\n" + instructionsMarkerEnd

// ownedInstructionsContent is the whole-file content for owned instruction
// files (no markers — archcore owns the entire file).
const ownedInstructionsContent = instructionsBody + "\n"

// Shared instruction-file names. Six agents read AGENTS.md; Gemini CLI reads
// GEMINI.md by default; Claude Code does not read AGENTS.md and gets its own
// owned file under .claude/rules/.
const (
	agentsInstructionsFile = "AGENTS.md"
	geminiInstructionsFile = "GEMINI.md"
)

// claudeInstructionsRelPath is the Claude Code owned-file location. Files under
// .claude/rules/*.md auto-load at CLAUDE.md priority without imports.
var claudeInstructionsRelPath = filepath.Join(".claude", "rules", "archcore.md")

// findManagedSpans returns the byte ranges [start, end) of every well-formed
// archcore managed block in content — a markerStart paired with the next
// markerEnd that has no other markerStart between them. Orphaned start markers
// (no following end) and stray end markers are left unpaired, so they are
// treated as ordinary user content and never consumed. Locating start and end
// independently (a plain double strings.Index) is unsafe: an orphaned start
// would pair with a later block's end and delete everything in between. Pairing
// here is what keeps upsert/remove non-destructive and idempotent even when a
// file holds malformed or duplicated markers.
func findManagedSpans(content string) [][2]int {
	var spans [][2]int
	for i := 0; ; {
		rel := strings.Index(content[i:], instructionsMarkerStart)
		if rel == -1 {
			break
		}
		start := i + rel
		afterStart := start + len(instructionsMarkerStart)

		endRel := strings.Index(content[afterStart:], instructionsMarkerEnd)
		if endRel == -1 {
			break // orphaned start, no end after it — leave as user content
		}
		end := afterStart + endRel

		// If another start appears before this end, the current start is
		// malformed; skip it and pair the inner start with the end instead.
		if nextRel := strings.Index(content[afterStart:], instructionsMarkerStart); nextRel != -1 && afterStart+nextRel < end {
			i = afterStart + nextRel
			continue
		}

		spans = append(spans, [2]int{start, end + len(instructionsMarkerEnd)})
		i = end + len(instructionsMarkerEnd)
	}
	return spans
}

// upsertFencedBlock writes the archcore managed block into the file at path. If
// one or more managed blocks already exist, the first is replaced in place and
// any duplicates are dropped (collapsing to a single block); otherwise the
// block is appended after a blank-line separator. Content outside managed
// blocks — including orphaned or stray markers — is preserved untouched, and
// writing twice yields byte-identical output (idempotent). The file and its
// parent directory are created if absent.
func upsertFencedBlock(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	content := string(data)

	var out string
	if spans := findManagedSpans(content); len(spans) > 0 {
		// Replace the first managed block in place; drop any duplicates,
		// keeping the user content between them.
		var b strings.Builder
		b.WriteString(content[:spans[0][0]])
		b.WriteString(instructionsFencedBlock)
		prev := spans[0][1]
		for _, span := range spans[1:] {
			b.WriteString(content[prev:span[0]])
			prev = span[1]
		}
		b.WriteString(content[prev:])
		out = b.String()
	} else {
		// Append after the user's content, separated by a blank line.
		var b strings.Builder
		if prefix := strings.TrimRight(content, "\n"); prefix != "" {
			b.WriteString(prefix)
			b.WriteString("\n\n")
		}
		b.WriteString(instructionsFencedBlock)
		b.WriteString("\n")
		out = b.String()
	}

	return os.WriteFile(path, []byte(out), 0o644)
}

// removeFencedBlock strips every archcore managed block from the file at path,
// leaving surrounding user content intact. If nothing but whitespace remains,
// the file is deleted. A missing file or absent block is a no-op. The "\r\n"
// trim cutset keeps the result clean on CRLF (Windows-edited) files.
func removeFencedBlock(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	content := string(data)

	if len(findManagedSpans(content)) == 0 {
		return nil // no managed block
	}

	// Remove the first block each pass, re-trimming the seam, until none
	// remain. Content strictly shrinks each pass, so this terminates.
	for {
		spans := findManagedSpans(content)
		if len(spans) == 0 {
			break
		}
		start, end := spans[0][0], spans[0][1]
		before := strings.TrimRight(content[:start], "\r\n")
		after := strings.TrimLeft(content[end:], "\r\n")
		switch {
		case before == "":
			content = after
		case after == "":
			content = before + "\n"
		default:
			content = before + "\n\n" + after
		}
	}

	if content = strings.TrimRight(content, "\r\n"); content == "" {
		return removeOwnedFile(path)
	}
	return os.WriteFile(path, []byte(content+"\n"), 0o644)
}

// writeOwnedFile creates parent directories and writes content as the entire
// file, overwriting any existing content. Used for files archcore owns wholly
// (e.g. .claude/rules/archcore.md).
func writeOwnedFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// removeOwnedFile deletes the file at path. A missing file is a no-op.
func removeOwnedFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("removing %s: %w", path, err)
	}
	return nil
}

// --- Per-target helpers wired into the agent registry ---

func agentsMDInstructionsPath(baseDir string) string {
	return filepath.Join(baseDir, agentsInstructionsFile)
}

func writeAgentsMDInstructions(baseDir string) error {
	return upsertFencedBlock(agentsMDInstructionsPath(baseDir))
}

func removeAgentsMDInstructions(baseDir string) error {
	return removeFencedBlock(agentsMDInstructionsPath(baseDir))
}

func geminiInstructionsPath(baseDir string) string {
	return filepath.Join(baseDir, geminiInstructionsFile)
}

func writeGeminiInstructions(baseDir string) error {
	return upsertFencedBlock(geminiInstructionsPath(baseDir))
}

func removeGeminiInstructions(baseDir string) error {
	return removeFencedBlock(geminiInstructionsPath(baseDir))
}

func claudeInstructionsPath(baseDir string) string {
	return filepath.Join(baseDir, claudeInstructionsRelPath)
}

func writeClaudeInstructions(baseDir string) error {
	return writeOwnedFile(claudeInstructionsPath(baseDir), ownedInstructionsContent)
}

func removeClaudeInstructions(baseDir string) error {
	return removeOwnedFile(claudeInstructionsPath(baseDir))
}
