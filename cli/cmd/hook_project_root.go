package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"archcore-cli/internal/docs"
	"archcore-cli/internal/projectroot"
	"archcore-cli/internal/stamp"
)

// projectTouchWindow bounds how long one session is told about one foreign
// project once. A session longer than the window hears it again, which costs
// one line. [assumption]
const projectTouchWindow = 24 * time.Hour

// hookRoots resolves the project each target of one hook invocation belongs to
// (call-root-resolution.spec §13–§15) and keeps one writeGuard per root, so a
// patch naming many files of one project reads that project's state once
// (hook-runtime.spec §9).
type hookRoots struct {
	session string
	byDir   map[string]string
	guards  map[string]*writeGuard
}

func newHookRoots(session string) *hookRoots {
	return &hookRoots{session: session, byDir: map[string]string{}, guards: map[string]*writeGuard{}}
}

// rootOf returns the project target belongs to: the nearest directory above
// the target that holds .archcore/, else the session root. A relative target
// is read against the payload working directory first, so "../b/.archcore/x.md"
// belongs to b (hook-runtime.spec §15). A target that is itself a symlink, or
// whose directory is one, belongs to the project its link resolves into. A
// root that is the session root under another spelling stays the session
// root, so a session sees one root, not two.
func (h *hookRoots) rootOf(target string) string {
	if target == "" {
		return h.session
	}
	abs := h.absolute(target)
	if isLink(abs) {
		// A file link answers for its own target, not for its directory.
		return h.resolve(filepath.Dir(projectroot.EvalExisting(abs)))
	}
	return h.dirRoot(filepath.Dir(abs))
}

// dirRoot memoizes the root of dir. A directory that does not exist yet holds
// no store and no link, so it belongs where its parent does: a patch that
// creates deep new trees pays one stat per new directory, not a walk
// (hook-runtime.spec, the one-second pre-write budget).
func (h *hookRoots) dirRoot(dir string) string {
	if root, ok := h.byDir[dir]; ok {
		return root
	}
	var root string
	if parent := filepath.Dir(dir); parent != dir && !exists(dir) {
		root = h.dirRoot(parent)
	} else {
		root = h.resolve(dir)
	}
	h.byDir[dir] = root
	return root
}

// resolve walks up from dir as written and with symlinks evaluated; the
// evaluated answer wins when the two differ.
func (h *hookRoots) resolve(dir string) string {
	root, ok := projectroot.Nearest(dir)
	if real, realOK := projectroot.Nearest(projectroot.EvalExisting(dir)); realOK && (!ok || !projectroot.SameDir(real, root)) {
		root, ok = real, true
	}
	if !ok || projectroot.SameDir(root, h.session) {
		return h.session
	}
	return root
}

// guardPath returns the spelling of target the guard of root judges: the
// resolved path of a link, the absolute path for another project's guard, and
// the target as written otherwise, so the session's own verdicts stay as they
// were.
func (h *hookRoots) guardPath(target, root string) string {
	if target == "" {
		return target
	}
	abs := h.absolute(target)
	if isLink(abs) {
		return projectroot.EvalExisting(abs)
	}
	if root != h.session {
		return abs
	}
	return target
}

// isLink and exists are advisory reads: an unreadable path counts as a plain,
// missing one, and the guard judges it as written.
func isLink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// absolute returns target read against the payload working directory.
func (h *hookRoots) absolute(target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(h.session, target)
}

func (h *hookRoots) guard(root string) *writeGuard {
	g, ok := h.guards[root]
	if !ok {
		g = newWriteGuard(root)
		h.guards[root] = g
	}
	return g
}

// withProjectHint names the project argument in a deny for a file of another
// project, so the agent's retry through the MCP tools reaches that project and
// not the session's. A project the session declares as a source is named by its
// id too, because a change there reaches every project that declares it
// (source-mounts.spec, behavior 13).
func withProjectHint(dec hookDecision, root, session string) hookDecision {
	if root != session {
		root = projectroot.Canonical(root)
		dec.reason += fmt.Sprintf("\nThis file belongs to the Archcore project at %s: pass project=\"%s\" to these tools.", root, root)
		if id, ok := docs.SourceIDFor(session, root); ok {
			dec.reason += fmt.Sprintf(" That project is source %q of this session's project; a change there reaches every project that declares it.", id)
		}
	}
	return dec
}

// projectTouchAdvisory tells a session, once per project, that the file it
// edits belongs to another Archcore project (hook-runtime.spec §16). Without a
// session id there is nothing to key the stamp on, and the note is always
// emitted: a repeated line is cheaper than a missed project.
func projectTouchAdvisory(r hookRequest, root string) string {
	root = projectroot.Canonical(root)
	if id := r.payload.sessionID(); id != "" {
		key := id + "\x00" + root + "\x00" + string(r.dialect.id)
		if !stamp.Claim(stamp.DirFor("project-touch-stamps"), key, projectTouchWindow) {
			return ""
		}
	}
	return fmt.Sprintf("[Archcore Project] This file belongs to the Archcore project at %s, not to this session's project. Pass project=\"%s\" to the Archcore tools when you read or write its documents.", root, root)
}

const (
	// inventoryDepth bounds the walk below a folder without .archcore/: a
	// workspace folder holds its projects within three levels. [assumption]
	inventoryDepth = 3
	// inventoryDirs bounds the directories that walk reads, so a session started
	// in a home directory pays a bounded cost at session start. [assumption]
	inventoryDirs = 2000
	// inventoryShown bounds the project list in the recap.
	inventoryShown = 10
)

// writeProjectInventory replaces the corpus line when the session folder holds
// no .archcore/: it names the projects the session can reach and the project
// argument that reaches them, and gives no advice to create a store here
// (hooks-validation-system.spec, Hook 1 phase 2).
func writeProjectInventory(b *strings.Builder, baseDir string) {
	b.WriteString("\nNO PROJECT HERE: this folder holds no .archcore/.\n")
	above, hasAbove := projectroot.Nearest(baseDir)
	if hasAbove {
		fmt.Fprintf(b, "PROJECT ABOVE: %s\n", above)
	}
	below, complete := projectroot.Below(baseDir, inventoryDepth, inventoryDirs)
	if len(below) > 0 {
		shown := below[:min(len(below), inventoryShown)]
		fmt.Fprintf(b, "PROJECTS BELOW: %s", strings.Join(shown, ", "))
		if rest := len(below) - len(shown); rest > 0 {
			fmt.Fprintf(b, " … and %d more", rest)
		}
		if !complete {
			b.WriteString(" (the search stopped at its bound)")
		}
		b.WriteString("\n")
	}
	if !hasAbove && len(below) == 0 {
		b.WriteString("No Archcore project was found near this folder.\n")
	}
	b.WriteString("To read or write a project, pass project=<its absolute root> to the Archcore tools, or call search_documents with an absolute for_path. Ask the user which project to use before you write.\n")
}
