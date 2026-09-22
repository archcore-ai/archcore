package cmd

import (
	"fmt"
	"io"
	"strings"

	"archcore-cli/internal/config"
	"archcore-cli/internal/display"
)

// warnUnknownConfigFields prints a one-line warning to w when settings.json
// contains fields this binary does not recognize (captured into Settings.Extra).
// Such fields are typically introduced by a newer archcore version; they are
// tolerated on read and preserved on write, but the user is told their CLI may
// be older than the project's config.
//
// Callers MUST pass os.Stderr (never stdout): the MCP server speaks JSON-RPC on
// stdout, and `config get` output on stdout is machine-readable — a warning on
// either would corrupt them.
func warnUnknownConfigFields(w io.Writer, s *config.Settings) {
	names := s.UnknownFieldNames()
	if len(names) == 0 {
		return
	}
	fmt.Fprintln(w, display.WarnLine(fmt.Sprintf(
		"settings.json has unrecognized field(s): %s — this archcore may be older than the project's config (or a typo). Consider 'archcore update'.",
		strings.Join(names, ", "),
	)))
}
