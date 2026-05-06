package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"archcore-cli/internal/display"
)

var pseudoVersionSuffix = regexp.MustCompile(`\.0\.\d{14}-[0-9a-f]+$`)

func cleanVersion(v string) string {
	if i := strings.Index(v, "+"); i != -1 {
		v = v[:i]
	}
	v = pseudoVersionSuffix.ReplaceAllString(v, "")
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// FormatExecuteError returns styled output for known cobra errors
// (unknown command, unknown flag). Returns empty string for unrecognized errors.
func FormatExecuteError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()

	var b strings.Builder

	switch {
	case strings.HasPrefix(msg, "unknown command"):
		lines := strings.SplitN(msg, "\n", 2)
		b.WriteString(display.FailLine(lines[0]))
		if len(lines) > 1 {
			// Preserve cobra's "Did you mean this?" suggestion lines
			for line := range strings.SplitSeq(lines[1], "\n") {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					b.WriteString("\n")
					b.WriteString(display.HintLine(trimmed))
				}
			}
		}
		b.WriteString("\n")
		b.WriteString(display.HintLine("Run 'archcore --help' to see available commands"))
		return b.String()

	case strings.HasPrefix(msg, "unknown flag"), strings.HasPrefix(msg, "unknown shorthand flag"):
		b.WriteString(display.FailLine(msg))
		b.WriteString("\n")
		b.WriteString(display.HintLine("Run 'archcore --help' to see available options"))
		return b.String()

	default:
		return ""
	}
}

func NewRootCmd(version string) *cobra.Command {
	ver := cleanVersion(version)
	root := &cobra.Command{
		Use:           "archcore",
		Short:         "Archcore — Git-native context for AI coding agents",
		Version:       ver,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), display.WelcomeBanner())
			fmt.Fprintln(cmd.OutOrStdout())
			_ = cmd.Usage()
			return nil
		},
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	root.SetVersionTemplate("{{.Version}}\n")
	addBaseDirFlag(root)

	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == root {
			fmt.Fprintln(cmd.OutOrStdout(), display.WelcomeBanner())
			fmt.Fprintln(cmd.OutOrStdout())
			_ = cmd.Usage()
			return
		}
		defaultHelp(cmd, args)
	})

	root.AddCommand(
		newInitCmd(),
		newConfigCmd(),
		newDoctorCmd(),
		newStatusCmd(),
		newHooksCmd(ver),
		newMCPCmd(ver),
		newSyncCmd(),
		newUpdateCmd(ver),
		newWhereCmd(ver),
	)

	return root
}
