package cmd

import (
	"encoding/json"
	"os"
	"strconv"

	"archcore-cli/internal/advisory"

	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	var projectFlag string
	cmd := &cobra.Command{
		Use:   "check [scope...]",
		Short: "Settle the claims in .archcore/ that a lookup can settle, as JSON",
		Long: "check reads every local document and reports, as one JSON object, the claims a lookup settles: " +
			"cited paths that resolve nowhere, line anchors, cited code identifiers and «quoted» interface text " +
			"that no tracked file holds, drafts that code cites as its contract, citations of missing documents, " +
			"document pairs that share text, and code blocks that copy a tracked file. A scope is a document or a " +
			"directory under .archcore/. Each kind holds at most " + strconv.Itoa(advisory.MaxFindingsPerKind) + " findings; omitted counts the rest.",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveProjectRoot(projectFlag, os.Getenv("ARCHCORE_PROJECT_ROOT"))
			if err != nil {
				return err
			}
			report, err := advisory.CheckCorpusFor(cmd.Context(), root, args)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetEscapeHTML(false)
			return enc.Encode(report)
		},
	}
	cmd.Flags().StringVar(&projectFlag, "project", "",
		"project root to check (default: current directory; env: ARCHCORE_PROJECT_ROOT)")
	return cmd
}
