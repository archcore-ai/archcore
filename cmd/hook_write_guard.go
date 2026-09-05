package cmd

import (
	"errors"
	"path/filepath"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/internal/docs"
)

// Write guard.
//
// A document written straight to disk skips frontmatter validation, template
// structure, and the relation manifest — the file looks right and the knowledge
// base quietly stops being consistent with itself. This is the only guard that
// blocks; everything else the hooks do is advisory.
//
// The decision reuses docs.GuardWritablePath, the same predicate the MCP write
// tools consult, so a path the tools would refuse cannot be reached by going
// around them.

const writeGuardReason = `Direct writes to .archcore/ documents are not allowed. Use Archcore MCP tools instead:
- create_document: create a new document
- update_document: modify an existing document
- remove_document: delete a document
This ensures validation, templates, and the sync manifest stay consistent.`

// writeGuardUnverifiableReason is the refusal when the guard cannot read the
// list it needs to judge the path. It names the file to repair, because the
// write that unblocks this state is an edit to settings.json.
const writeGuardUnverifiableReason = `Cannot verify global sources: .archcore/settings.json is unreadable.
Writes to Markdown files outside the project are refused until it parses.
Fix .archcore/settings.json, then retry.`

// writeGuard answers for one hook invocation. It exists because a single call
// can carry many targets: an apply-patch document names up to maxPatchLines of
// them, and the project state every verdict consults is the same for all of
// them. Read per target, settings.json alone cost 620k allocations and enough
// wall clock on a maximal patch to threaten the one-second pre-write budget the
// CLI writes into host configs; read once, the same patch is a few million
// allocations lighter and roughly thirty times faster.
//
// The cache lives no longer than the invocation, so the question "did the store
// change underneath us" never arises.
//
// Both cached facts are read on demand rather than in the constructor. The order
// of the checks in decide is deliberate — a call naming no file at all answers
// before either one — and computing them up front would spend syscalls the old
// code correctly skipped.
type writeGuard struct {
	baseDir string

	store     bool
	storeRead bool

	globals     []config.GlobalSource
	globalsErr  error
	globalsRead bool
}

func newWriteGuard(baseDir string) *writeGuard {
	return &writeGuard{baseDir: baseDir}
}

// storeExists stats .archcore/ at most once.
func (g *writeGuard) storeExists() bool {
	if !g.storeRead {
		g.store, g.storeRead = config.DirExists(g.baseDir), true
	}
	return g.store
}

// declaredGlobals reads settings.json at most once, and only when a verdict
// actually needs it.
//
// A guard, not an advisory (fail-open-or-fail-closed-reads.rule §2): config.ReadGlobals
// reports an unreadable settings.json as "no globals are declared", which reads as
// "not a global" and permits the write this guard exists to refuse.
//
// Lazy rather than eager, because the branch below that skips it is load-bearing:
// an ordinary source edit is the overwhelming majority of what this hook sees,
// and reading settings.json in the constructor would make that case pay for the
// rare one on every single write.
func (g *writeGuard) declaredGlobals() ([]config.GlobalSource, error) {
	if !g.globalsRead {
		g.globals, g.globalsErr = config.LoadGlobals(g.baseDir)
		g.globalsRead = true
	}
	return g.globals, g.globalsErr
}

// outsideProject reports whether filePath resolves to a location outside
// baseDir.
//
// It resolves before it classifies. docs.RelativeToBase reports every
// non-absolute path as in-project, because a relative path is read against
// baseDir — but the patch bodies patchPaths parses carry exactly that form, so
// classifying "../company/.archcore/x.rule.md" straight from the raw string put
// an escaping target on the in-project side of the fail-closed branch below.
// Joining first is what IsExternalGlobalDocument already does with the same
// input, so the two answers now agree on the same path.
func (g *writeGuard) outsideProject(filePath string) bool {
	resolved := filePath
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(g.baseDir, filePath)
	}
	_, inProject := docs.RelativeToBase(g.baseDir, resolved)
	return !inProject
}

// writeGuardDecision reports whether a file write must be blocked. It is the
// single-target entry point; a caller holding several targets should build one
// writeGuard and call decide for each.
func writeGuardDecision(baseDir, filePath string) hookDecision {
	return newWriteGuard(baseDir).decide(filePath)
}

// decide reports whether a file write must be blocked.
//
// The allow cases are enumerated and everything else denies, rather than the
// reverse. GuardWritablePath reports four classes of failure and only two carry
// a comparable sentinel, so a default-allow silently permits the rest — which is
// how a document reachable through a symlink out of the store stayed writable
// while MCP refused it.
func (g *writeGuard) decide(filePath string) hookDecision {
	baseDir := g.baseDir // named for the calls below that take it positionally
	if filePath == "" {
		return allowHook() // no file involved
	}
	if !g.storeExists() {
		return allowHook() // no store yet, so nothing to protect
	}
	// Lexical containment is GuardWritablePath's own first step, so checking it
	// here separates "not under .archcore/" from every other refusal.
	rel, inProject := docs.RelativeToBase(baseDir, filePath)
	if inProject {
		if cleaned, err := docs.ValidateArchcorePath(rel); err == nil {
			// The error is dropped here and only here: a missing list cannot turn
			// a document into a non-document, so an unreadable settings.json still
			// denies. Failing closed instead would deny settings.json itself.
			globals, _ := g.declaredGlobals()
			if _, err := docs.GuardWritablePath(baseDir, cleaned, globals); errors.Is(err, docs.ErrPathNotDocument) {
				return allowHook() // settings.json, .sync-state.json, a non-markdown file
			}
			return denyHook(writeGuardReason)
		}
	}

	// Not lexically under .archcore/: outside the project, or a "../" sibling of
	// it. Either can still be a declared global source mounted from outside the
	// store, and those documents are as read-only as in-tree ones — the MCP
	// tools cannot even address them. Returning allow here is what left them
	// writable straight from the editor.
	//
	// Gated on the extension so an ordinary source edit — the overwhelming
	// majority of what this hook sees — never pays for the settings read: a
	// global source mounts nothing but Markdown documents.
	if !strings.HasSuffix(filePath, ".md") {
		return allowHook()
	}
	globals, globalsErr := g.declaredGlobals()
	if globalsErr != nil && g.outsideProject(filePath) {
		// An unreadable list answers "not a declared mount" for every path, which
		// is the allow this guard exists to refuse. Narrowed to a path that
		// resolves outside the project, where an external mount lives: refusing
		// every in-project .md would block a README edit on a settings.json typo.
		return denyHook(writeGuardUnverifiableReason)
	}
	if docs.IsExternalGlobalDocument(baseDir, filePath, globals) {
		return denyHook(writeGuardReason)
	}
	return allowHook()
}
