package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"archcore-cli/internal/projectroot"
)

const flagBaseDir = "base-dir"

// addBaseDirFlag registers --base-dir as a persistent flag on root. Called
// once during root construction.
func addBaseDirFlag(root *cobra.Command) {
	root.PersistentFlags().String(flagBaseDir, "", "base directory for archcore project (overrides ARCHCORE_BASE_DIR)")
}

// baseDirFlag returns the --base-dir value visible to cmd, walking up
// parents to find the persistent registration.
func baseDirFlag(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	if f := cmd.Flags().Lookup(flagBaseDir); f != nil {
		return f.Value.String()
	}
	return ""
}

// resolveProjectRoot reads --base-dir from cmd, runs the projectroot resolver
// in the requested mode, prints the unified error block to stderr on failure,
// and memoizes the *Resolution on cmd.Context() so downstream callers (sub-checks,
// MCP tool handlers, doctor stages) read it via projectroot.From.
//
// Calling this function repeatedly within the same command invocation returns
// the cached resolution without re-resolving.
func resolveProjectRoot(cmd *cobra.Command, mode projectroot.Mode) (*projectroot.Resolution, error) {
	if r, ok := projectroot.From(cmd.Context()); ok {
		return r, nil
	}

	res, err := projectroot.Resolve(projectroot.Options{
		Flag: baseDirFlag(cmd),
		Mode: mode,
	})
	if err != nil {
		if block := projectroot.FormatError(err); block != "" {
			fmt.Fprint(cmd.ErrOrStderr(), block)
		} else {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
		}
		return nil, err
	}
	if res.LegacyMode {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"warning: ARCHCORE_LEGACY_BASE_DIR=1 is set; guard checks bypassed (path=%s)\n",
			res.Path)
	}
	cmd.SetContext(projectroot.WithResolution(cmd.Context(), res))
	return res, nil
}
