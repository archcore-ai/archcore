package sync

import "slices"

// DiffAction classifies what happened to a file since the last sync.
type DiffAction string

const (
	ActionCreated   DiffAction = "created"
	ActionModified  DiffAction = "modified"
	ActionDeleted   DiffAction = "deleted"
	ActionUnchanged DiffAction = "unchanged"
)

// DiffEntry represents one file's diff result.
type DiffEntry struct {
	RelPath string
	Action  DiffAction
	Hash    string // current hash (empty for deleted files)
}

// Diff compares the current on-disk file states against the manifest
// and returns a categorized list of changes.
func Diff(current []FileState, manifest *Manifest) []DiffEntry {
	var result []DiffEntry

	seen := make(map[string]bool, len(current))

	for _, f := range current {
		seen[f.RelPath] = true
		prevHash, exists := manifest.Files[f.RelPath]

		switch {
		case !exists:
			result = append(result, DiffEntry{
				RelPath: f.RelPath,
				Action:  ActionCreated,
				Hash:    f.Hash,
			})
		case prevHash != f.Hash:
			result = append(result, DiffEntry{
				RelPath: f.RelPath,
				Action:  ActionModified,
				Hash:    f.Hash,
			})
		default:
			result = append(result, DiffEntry{
				RelPath: f.RelPath,
				Action:  ActionUnchanged,
				Hash:    f.Hash,
			})
		}
	}

	// Files in manifest but not on disk are deleted. Map iteration order is
	// random; sort so the deleted-file listing is stable between runs.
	var deleted []string
	for relPath := range manifest.Files {
		if !seen[relPath] {
			deleted = append(deleted, relPath)
		}
	}
	slices.Sort(deleted)
	for _, relPath := range deleted {
		result = append(result, DiffEntry{
			RelPath: relPath,
			Action:  ActionDeleted,
		})
	}

	return result
}

// HasChanges returns true if there are any created, modified, or deleted entries.
func HasChanges(entries []DiffEntry) bool {
	for _, e := range entries {
		if e.Action != ActionUnchanged {
			return true
		}
	}
	return false
}

// FilterByAction returns only the entries matching the given action.
func FilterByAction(entries []DiffEntry, action DiffAction) []DiffEntry {
	var out []DiffEntry
	for _, e := range entries {
		if e.Action == action {
			out = append(out, e)
		}
	}
	return out
}
