package advisory

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"archcore-cli/internal/docs"
	"archcore-cli/internal/git"
	"archcore-cli/internal/stamp"
)

// Staleness advisory.
//
// Documentation drifts silently: code moves, documents do not, and nothing in
// the normal workflow says so. This compares the last commit that touched
// .archcore/ against everything committed since, and names the documents that
// mention the directories that moved.
//
// It is advisory only. The correlation is by directory name, so it over-reports
// by design — a false "review this" costs a glance, a missed drift costs a
// decision made against stale context.

const (
	// stalenessWindow rate-limits the advisory. Drift is a slow signal; repeating
	// it every session would train the reader to skip it.
	stalenessWindow = 24 * time.Hour
	// maxStalenessDocsPerDir caps documents reported per changed directory.
	maxStalenessDocsPerDir = 5
	// maxStalenessLines caps the whole list.
	maxStalenessLines = 10
	// maxStalenessDirs caps how many changed directories are correlated. Each one
	// walks the corpus, so a commit touching thirty top-level directories would
	// otherwise scan it thirty times to fill a ten-line list.
	maxStalenessDirs = 12
)

// Staleness returns the drift warning, or an empty string when there is
// nothing to say. stampDir carries the rate-limit state; an empty stampDir
// disables rate limiting, which is what tests want.
//
// corpus is the caller's already-scanned local documents. Passing it in keeps
// session start to one corpus read: correlation needs bodies, and the caller
// has them.
func Staleness(ctx context.Context, baseDir, stampDir string, corpus []docs.Document) string {
	// Freshness is checked before any git work, and the stamp is claimed only
	// once there is something to report — a session that finds no drift must not
	// spend the day's single advisory.
	key := "staleness\x00" + baseDir
	if stampDir != "" && stamp.Fresh(stampDir, key, stalenessWindow) {
		return ""
	}

	// No IsRepo probe: LastCommitTouching already errors outside a repository
	// and when git is absent, so asking first only costs an extra subprocess.
	lastDocCommit, err := git.LastCommitTouching(ctx, baseDir, ".archcore/")
	if err != nil || lastDocCommit == "" {
		return "" // documentation has no history yet — nothing to drift from
	}

	changed, err := git.ChangedSince(ctx, baseDir, lastDocCommit, ":(exclude).archcore/")
	if err != nil || len(changed) == 0 {
		return ""
	}

	affected := correlateStaleness(corpus, changed)
	if len(affected) == 0 {
		return ""
	}

	if stampDir != "" && !stamp.Claim(stampDir, key, stalenessWindow) {
		return "" // a peer session claimed the day's advisory first
	}

	var b strings.Builder
	fmt.Fprintf(&b, "[Archcore Staleness] %d source file(s) changed since the last documentation update.\n", len(changed))
	b.WriteString("Potentially affected documents:\n")
	for _, line := range affected {
		fmt.Fprintf(&b, "  - %s\n", line)
	}
	b.WriteString("Review them against the changes, then update what no longer holds.\n")
	return b.String()
}

// correlateStaleness maps changed source files to documents that mention their
// top-level directory. Global sources are excluded: they are read-only, so
// asking the reader to update one is advice they cannot take.
func correlateStaleness(corpus []docs.Document, changed []string) []string {
	counts := countChangedDirs(changed)
	if len(counts) == 0 {
		return nil
	}

	dirs := make([]string, 0, len(counts))
	for dir := range counts {
		dirs = append(dirs, dir)
	}
	// Most-changed first, then alphabetical for a stable tie-break; keep the top
	// slice, then restore name order so the output reads the same as before for
	// any commit under the cap.
	slices.SortFunc(dirs, func(a, b string) int {
		if counts[a] != counts[b] {
			return counts[b] - counts[a]
		}
		return strings.Compare(a, b)
	})
	dirs = dirs[:min(len(dirs), maxStalenessDirs)]
	slices.Sort(dirs)

	var out []string
	for _, dir := range dirs {
		needle := dir + "/"
		shown := 0
		for _, doc := range corpus {
			if shown == maxStalenessDocsPerDir || len(out) == maxStalenessLines {
				break
			}
			if !strings.Contains(doc.Content, needle) {
				continue
			}
			out = append(out, fmt.Sprintf("%s — references %s (%d file(s) changed)", doc.Path, needle, counts[dir]))
			shown++
		}
		if len(out) == maxStalenessLines {
			break
		}
	}
	return out
}

// countChangedDirs groups changed files by their top-level directory. A file at
// the repository root has no directory to correlate on and is skipped.
func countChangedDirs(changed []string) map[string]int {
	counts := make(map[string]int)
	for _, path := range changed {
		dir, _, found := strings.Cut(path, "/")
		if !found || dir == "" {
			continue
		}
		counts[dir]++
	}
	return counts
}
