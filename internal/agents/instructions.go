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
// All targets are shared user files written with a fenced-block upsert —
// archcore only ever touches the span between its markers, never surrounding
// user content: AGENTS.md (six agents), GEMINI.md (Gemini CLI), and CLAUDE.md
// (Claude Code).
//
// Claude Code gets CLAUDE.md AND AGENTS.md. Per Anthropic's docs Claude Code
// reads CLAUDE.md natively but does NOT auto-read AGENTS.md, so CLAUDE.md is
// what delivers the nudge to Claude Code; the AGENTS.md block is written too so
// the repo also carries the standard block the plugin and the other hosts
// converge on. Earlier CLI versions instead wrote an owned .claude/rules/
// archcore.md file; that is now migrated away (removed) on install/remove.

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

// Shared instruction-file names. Six agents read AGENTS.md; Gemini CLI reads
// GEMINI.md by default; Claude Code reads CLAUDE.md (and also gets AGENTS.md).
const (
	agentsInstructionsFile = "AGENTS.md"
	geminiInstructionsFile = "GEMINI.md"
	claudeInstructionsFile = "CLAUDE.md"
)

// legacyClaudeRulesRelPath is the pre-CLAUDE.md Claude Code owned-file location
// written by older CLI versions. Claude Code now gets its nudge from CLAUDE.md,
// so this file is migrated away (removed) whenever Claude Code is (re)wired.
var legacyClaudeRulesRelPath = filepath.Join(".claude", "rules", "archcore.md")

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

// InstructionBlockPresent reports whether the file at path currently holds an
// archcore managed block. It is the ground-truth check wiring uses to report
// exactly the instruction files that landed on disk: a multi-file write
// (claude-code: CLAUDE.md + AGENTS.md) can fail partway, and the report must
// name the file that WAS written rather than treat the whole agent as unwritten.
// A missing or unreadable file (including a path that is a directory) reads as
// absent.
func InstructionBlockPresent(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return len(findManagedSpans(string(data))) > 0
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

	return writeFileAtomic(path, []byte(out))
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
	return writeFileAtomic(path, []byte(content+"\n"))
}

// removeOwnedFile deletes the file at path. A missing file is a no-op.
func removeOwnedFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("removing %s: %w", path, err)
	}
	return nil
}

// writeFileAtomic replaces the file at path with data by writing a sibling temp
// file and renaming it into place. The rename is atomic on POSIX, so a crash or
// ENOSPC mid-write can never leave a shared user file (CLAUDE.md / AGENTS.md /
// GEMINI.md) half-written with the managed block torn across the failure — the
// old contents survive intact until the single rename swaps them. It preserves
// two behaviors of the plain os.WriteFile it replaces:
//
//   - Permission bits: os.WriteFile keeps an existing file's mode (the mode
//     argument only applies on create); a naive temp+rename would reset it to
//     the temp file's mode, so we copy the target's current perm onto the temp
//     before renaming, falling back to 0o644 for a new file.
//   - Symlinks: os.WriteFile follows a symlink and writes through to its target.
//     We resolve the link and rename onto the resolved target, so a symlinked
//     instruction file stays a symlink instead of being replaced by a regular
//     file. (A dangling link resolves to itself and is materialized as a real
//     file — writing through an unresolvable target is impossible anyway.)
func writeFileAtomic(path string, data []byte) error {
	// Follow a symlink to the file it points at so we replace that file and the
	// link is preserved. EvalSymlinks fails for a not-yet-existing path, in
	// which case target stays the literal path (the fresh-file case).
	target := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		target = resolved
	}

	perm := os.FileMode(0o644)
	if info, err := os.Stat(target); err == nil {
		perm = info.Mode().Perm()
	}

	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".archcore-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file in %s: %w", dir, err)
	}
	// Remove the temp file if we return before the rename consumes it.
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", target, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("setting permissions on %s: %w", target, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file for %s: %w", target, err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("replacing %s: %w", target, err)
	}
	tmpName = "" // consumed by the rename — nothing to clean up
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

// claudeInstructionsPath is the path reported for Claude Code — its CLAUDE.md
// file. It doubles as the dedupe key, and being distinct from AGENTS.md
// guarantees Claude Code always runs its own write/remove (which also touches
// AGENTS.md) regardless of agent-list order.
func claudeInstructionsPath(baseDir string) string {
	return filepath.Join(baseDir, claudeInstructionsFile)
}

// writeClaudeInstructions writes the nudge to BOTH Claude Code targets:
// CLAUDE.md (read natively by Claude Code — this is what actually delivers the
// nudge) and the shared AGENTS.md fenced block (the standard the plugin and the
// six AGENTS.md hosts converge on). Both are idempotent fenced upserts, so a
// co-installed AGENTS.md agent writing the same block is harmless. It also
// migrates away the legacy owned .claude/rules/archcore.md file older CLIs
// wrote, so the nudge is not loaded twice.
func writeClaudeInstructions(baseDir string) error {
	if err := upsertFencedBlock(claudeInstructionsPath(baseDir)); err != nil {
		return err
	}
	if err := upsertFencedBlock(agentsMDInstructionsPath(baseDir)); err != nil {
		return err
	}
	return removeOwnedFile(filepath.Join(baseDir, legacyClaudeRulesRelPath))
}

// removeClaudeInstructions strips the CLAUDE.md fenced block (keeping user
// content) and deletes the legacy .claude/rules/archcore.md file. The shared
// AGENTS.md block is left to the AGENTS.md agents' own remove (both run in the
// "remove all" path), so removing Claude Code alone never strips a block a
// co-installed Cursor/Codex still relies on.
func removeClaudeInstructions(baseDir string) error {
	if err := removeFencedBlock(claudeInstructionsPath(baseDir)); err != nil {
		return err
	}
	return removeOwnedFile(filepath.Join(baseDir, legacyClaudeRulesRelPath))
}
