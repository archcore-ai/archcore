package projectroot

import (
	"errors"
	"os"
	"path/filepath"
)

// Diagnostics shapes shared between `archcore where` and the which_project
// MCP tool. They live here so the two surfaces cannot drift in JSON shape.

// Guards mirrors the guard flags surfaced to diagnostic consumers.
type Guards struct {
	Strict    bool
	AllowHome bool
	Legacy    bool
}

// GuardsFor returns guard state for a Resolution. Safe with nil — a nil
// resolution reports strict=true, legacy=false (the default refusing posture).
func GuardsFor(res *Resolution) Guards {
	if res == nil {
		return Guards{Strict: true}
	}
	return Guards{
		Strict:    !res.LegacyMode,
		AllowHome: res.LegacyMode,
		Legacy:    res.LegacyMode,
	}
}

// MarkerStates reports which canonical project markers exist directly under
// path. Returns true=found / false=missing keyed by marker name. An empty
// path returns an empty map.
func MarkerStates(path string) map[string]bool {
	out := make(map[string]bool, len(allMarkers))
	if path == "" {
		return out
	}
	for _, m := range allMarkers {
		_, err := os.Stat(filepath.Join(path, m))
		out[m] = err == nil
	}
	return out
}

// MarkerStateLabel converts a found/missing boolean to the stable string
// label used in JSON output ("found" / "missing"). Both `archcore where
// --json` and which_project rely on this label set.
func MarkerStateLabel(found bool) string {
	if found {
		return "found"
	}
	return "missing"
}

// Problem is the diagnostic shape for a resolve error: a stable code plus
// a human-readable message. Mirrors the `problems[]` entries in the JSON
// output of `archcore where` and which_project.
type Problem struct {
	Code    string
	Message string
}

// ProblemsFor extracts structured Problems from a resolve error. Returns
// an empty slice for nil. Unknown errors map to ERR_UNKNOWN.
func ProblemsFor(err error) []Problem {
	if err == nil {
		return []Problem{}
	}
	var re *ResolveError
	if !errors.As(err, &re) {
		return []Problem{{Code: "ERR_UNKNOWN", Message: err.Error()}}
	}
	return []Problem{{Code: re.Code, Message: re.Sentinel.Error()}}
}
