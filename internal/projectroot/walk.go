package projectroot

import (
	"path/filepath"
)

const walkDepthLimit = 8

// walkUp searches from start upward (≤ walkDepthLimit levels), stopping at
// $HOME or filesystem root. Returns the highest-priority match:
//
//  1. closest level with .archcore (best — most specific)
//  2. else closest level with .git
//  3. else closest level with any generic marker
//
// Returns (nil, false) if no markers were found.
func walkUp(start string, o Options) (*Resolution, bool) {
	home, _ := o.Home()
	if home != "" {
		home = filepath.Clean(home)
	}

	type levelHit struct {
		path     string
		depth    int
		archcore bool
		git      bool
		generic  string
	}
	hits := make([]levelHit, 0, walkDepthLimit)

	current := filepath.Clean(start)
	for depth := range walkDepthLimit {
		h := levelHit{path: current, depth: depth}
		if hasMarker(current, ".archcore", o) {
			h.archcore = true
		}
		if hasMarker(current, ".git", o) {
			h.git = true
		}
		for _, m := range genericMarkers {
			if hasMarker(current, m, o) {
				h.generic = m
				break
			}
		}
		if h.archcore || h.git || h.generic != "" {
			hits = append(hits, h)
			if h.archcore {
				// Closest .archcore wins — no point walking further.
				break
			}
		}

		// HOME boundary applies *after* marker evaluation: a HOME hit must
		// reach validate() so it's converted to ERR_HOME_REFUSED instead of
		// being silently dropped to ERR_NO_PROJECT.
		if home != "" && current == home {
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break // fs root reached
		}
		current = parent
	}

	for _, h := range hits {
		if h.archcore {
			return &Resolution{
				Path:      h.path,
				Source:    SourceWalkArchcore,
				Marker:    ".archcore",
				WalkDepth: h.depth,
			}, true
		}
	}
	for _, h := range hits {
		if h.git {
			return &Resolution{
				Path:      h.path,
				Source:    SourceWalkGit,
				Marker:    ".git",
				WalkDepth: h.depth,
			}, true
		}
	}
	for _, h := range hits {
		if h.generic != "" {
			return &Resolution{
				Path:      h.path,
				Source:    SourceWalkMarker,
				Marker:    h.generic,
				WalkDepth: h.depth,
			}, true
		}
	}
	return nil, false
}
