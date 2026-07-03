package git

import (
	"context"
	"os/exec"
	"strings"
)

// DetectRepoURL returns the URL of the "origin" remote for the git repository
// at dir, or an empty string if detection fails (not a git repo, no remote, etc.).
func DetectRepoURL(ctx context.Context, dir string) string {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
