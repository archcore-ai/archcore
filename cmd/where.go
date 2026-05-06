package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"archcore-cli/internal/projectroot"
)

const whereExitNotResolved = 2

// newWhereCmd creates the `archcore where` diagnostic command. It resolves the
// base directory, applies guards, and prints the result. Exit codes:
//
//	0 — resolved + guards passed
//	1 — resolved but guards failed
//	2 — not resolved (no project found)
func newWhereCmd(version string) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "where",
		Short: "show resolved base directory and project guard state",
		Long: `Prints the resolved base directory, how it was resolved (flag, env, or
walk-up), which project markers were found, and whether all guards passed.

Useful for debugging why archcore is refusing a directory or picking up
an unexpected path. Use --json for machine-readable output suitable for
scripts or agents.`,
		Example: `  archcore where
  archcore where --json
  archcore --base-dir /path/to/project where`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := projectroot.Resolve(projectroot.Options{
				Flag: baseDirFlag(cmd),
				Mode: projectroot.ModeRuntime,
			})
			if jsonOut {
				return whereRenderJSON(cmd, version, res, err)
			}
			return whereRenderHuman(cmd, res, err)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON instead of human-readable text")
	return cmd
}

// whereRenderHuman prints the human-readable diagnostic and returns an error
// that carries the appropriate exit semantics. The error itself is suppressed
// from cobra's auto-print (SilenceErrors); we manage exit codes via main.go.
func whereRenderHuman(cmd *cobra.Command, res *projectroot.Resolution, resolveErr error) error {
	out := cmd.OutOrStdout()
	if res == nil {
		fmt.Fprintf(out, "base dir : (none)\n")
		fmt.Fprintf(out, "source   : (none)\n")
		fmt.Fprintf(out, "guards   : %s\n", whereGuardsLine(nil))
		fmt.Fprintf(out, "status   : %s\n", whereStatusLine(resolveErr))
		return whereExitFromErr(resolveErr)
	}

	fmt.Fprintf(out, "base dir : %s\n", res.Path)
	fmt.Fprintf(out, "source   : %s\n", res.Source)
	fmt.Fprintf(out, "markers  : %s\n", whereMarkersLine(res.Path))
	fmt.Fprintf(out, "guards   : %s\n", whereGuardsLine(res))
	fmt.Fprintf(out, "status   : OK\n")
	return nil
}

// whereRenderJSON emits the same shape as the which_project MCP tool.
func whereRenderJSON(cmd *cobra.Command, version string, res *projectroot.Resolution, resolveErr error) error {
	guards := projectroot.GuardsFor(res)
	resp := map[string]any{
		"ok":          res != nil && resolveErr == nil,
		"cli_version": version,
		"guards": map[string]bool{
			"strict":     guards.Strict,
			"allow_home": guards.AllowHome,
			"legacy":     guards.Legacy,
		},
		"markers":  whereMarkers(res),
		"problems": whereProblems(resolveErr),
	}
	if res != nil {
		resp["base_dir"] = res.Path
		resp["source"] = string(res.Source)
	}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling where output: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	if resolveErr != nil {
		return whereExitFromErr(resolveErr)
	}
	return nil
}

// whereGuardsLine renders "strict=... allow_home=... legacy=...".
func whereGuardsLine(res *projectroot.Resolution) string {
	g := projectroot.GuardsFor(res)
	return fmt.Sprintf("strict=%v  allow_home=%v  legacy=%v", g.Strict, g.AllowHome, g.Legacy)
}

func whereMarkersLine(path string) string {
	states := projectroot.MarkerStates(path)
	var b strings.Builder
	for i, m := range projectroot.Markers() {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%s(%s)", m, projectroot.MarkerStateLabel(states[m]))
	}
	return b.String()
}

func whereMarkers(res *projectroot.Resolution) map[string]string {
	if res == nil {
		return map[string]string{}
	}
	states := projectroot.MarkerStates(res.Path)
	out := make(map[string]string, len(states))
	for m, found := range states {
		out[m] = projectroot.MarkerStateLabel(found)
	}
	return out
}

func whereProblems(err error) []map[string]string {
	probs := projectroot.ProblemsFor(err)
	out := make([]map[string]string, 0, len(probs))
	for _, p := range probs {
		out = append(out, map[string]string{"code": p.Code, "message": p.Message})
	}
	return out
}

func whereStatusLine(err error) string {
	if err == nil {
		return "OK"
	}
	var re *projectroot.ResolveError
	if errors.As(err, &re) {
		return fmt.Sprintf("%s — %s", re.Code, re.Sentinel)
	}
	return err.Error()
}

// whereExitFromErr maps a resolve error to a terminal exit code carrier.
// NewRootCmd's Execute returns this; main.go interprets exit codes via the
// ExitError wrapper.
func whereExitFromErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, projectroot.ErrNoProject) {
		return &exitCodeError{code: whereExitNotResolved, inner: err}
	}
	return &exitCodeError{code: 1, inner: err}
}

// exitCodeError carries a custom exit code that main.go can read.
type exitCodeError struct {
	code  int
	inner error
}

func (e *exitCodeError) Error() string { return e.inner.Error() }
func (e *exitCodeError) Unwrap() error { return e.inner }
func (e *exitCodeError) ExitCode() int { return e.code }
