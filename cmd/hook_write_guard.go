package cmd

import (
	"errors"
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
// Lazy rather than eager, because the branch below that skips it is load-bearing:
// an ordinary source edit is the overwhelming majority of what this hook sees,
// and reading settings.json in the constructor would make that case pay for the
// rare one on every single write.
func (g *writeGuard) declaredGlobals() []config.GlobalSource {
	if !g.globalsRead {
		g.globals, g.globalsRead = config.ReadGlobals(g.baseDir), true
	}
	return g.globals
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
	if rel, inProject := docs.RelativeToBase(baseDir, filePath); inProject {
		if cleaned, err := docs.ValidateArchcorePath(rel); err == nil {
			if _, err := docs.GuardWritablePath(baseDir, cleaned, g.declaredGlobals()); errors.Is(err, docs.ErrPathNotDocument) {
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
	if docs.IsExternalGlobalDocument(baseDir, filePath, g.declaredGlobals()) {
		return denyHook(writeGuardReason)
	}
	return allowHook()
}
