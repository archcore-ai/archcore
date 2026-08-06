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

// writeGuardDecision reports whether a file write must be blocked.
//
// The allow cases are enumerated and everything else denies, rather than the
// reverse. GuardWritablePath reports four classes of failure and only two carry
// a comparable sentinel, so a default-allow silently permits the rest — which is
// how a document reachable through a symlink out of the store stayed writable
// while MCP refused it.
func writeGuardDecision(baseDir, filePath string) hookDecision {
	if filePath == "" {
		return allowHook() // no file involved
	}
	if !config.DirExists(baseDir) {
		return allowHook() // no store yet, so nothing to protect
	}
	// Lexical containment is GuardWritablePath's own first step, so checking it
	// here separates "not under .archcore/" from every other refusal.
	if rel, inProject := docs.RelativeToBase(baseDir, filePath); inProject {
		if cleaned, err := docs.ValidateArchcorePath(rel); err == nil {
			if _, err := docs.GuardWritablePath(baseDir, cleaned, config.ReadGlobals(baseDir)); errors.Is(err, docs.ErrPathNotDocument) {
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
	if docs.IsExternalGlobalDocument(baseDir, filePath, config.ReadGlobals(baseDir)) {
		return denyHook(writeGuardReason)
	}
	return allowHook()
}
