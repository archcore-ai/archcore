package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"archcore-cli/internal/display"
	"archcore-cli/internal/projectroot"

	"github.com/spf13/cobra"
)

// hookInput represents the JSON payload Claude Code sends via stdin.
type hookInput struct {
	SessionID     string `json:"session_id"`
	CWD           string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`
	Source        string `json:"source"`
}

// hookOutput is the JSON response written to stdout.
type hookOutput struct {
	HookSpecificOutput map[string]any `json:"hookSpecificOutput,omitempty"`
	SystemMessage      string         `json:"systemMessage,omitempty"`
}

// resolveBaseDir returns the base directory for a hook, using the projectroot
// resolver. The hook input's CWD (set by the host MCP integration) is treated
// as an explicit signal and takes precedence over walk-up. ModeHooks lets us
// accept the host-supplied directory even if it lacks markers, since the host
// has explicitly chosen it.
func resolveBaseDir(cmd *cobra.Command, input *hookInput) (string, error) {
	flagVal := baseDirFlag(cmd)
	if flagVal == "" {
		flagVal = input.CWD
	}
	res, err := projectroot.Resolve(projectroot.Options{
		Flag: flagVal,
		Mode: projectroot.ModeHooks,
	})
	if err != nil {
		return "", err
	}
	return res.Path, nil
}

// readHookInput parses the hook input JSON from a reader.
func readHookInput(r io.Reader) (*hookInput, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	var input hookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parsing hook input: %w", err)
	}
	return &input, nil
}

func newHooksClaudeCodeCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude-code",
		Short: "Handle Claude Code hook events",
	}
	cmd.AddCommand(
		newSessionStartCmd(version),
	)
	return cmd
}

func newSessionStartCmd(version string) *cobra.Command {
	return newSessionStartHookCmd("session-start", "Handle SessionStart hook event", version)
}

// --- Hook command factories ---

// newSessionStartHookCmd creates a session-start hook command (shared across agents).
func newSessionStartHookCmd(use, short, version string) *cobra.Command {
	return &cobra.Command{
		Use:    use,
		Short:  short,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := readHookInput(os.Stdin)
			if err != nil {
				return err
			}
			baseDir, err := resolveBaseDir(cmd, input)
			if err != nil {
				return err
			}
			out, err := handleSessionStart(baseDir, version)
			if err != nil {
				return err
			}
			_, err = os.Stdout.Write(out)
			return err
		},
	}
}

// --- Session Start Handler (Claude Code adapter) ---

func handleSessionStart(baseDir, version string) ([]byte, error) {
	ctx, docCount := buildSessionContext(baseDir)
	output := hookOutput{
		HookSpecificOutput: map[string]any{
			"hookEventName":     "SessionStart",
			"additionalContext": ctx,
		},
		SystemMessage: display.HookConnectedLine(version, docCount),
	}
	return json.Marshal(output)
}
