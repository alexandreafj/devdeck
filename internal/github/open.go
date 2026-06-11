package github

import (
	"context"

	"github.com/alexandreafj/devdeck/internal/exec"
)

// OpenPR opens a pull request in the user's browser via `gh pr view <url> --web`.
// Reusing `gh` keeps a single, cross-platform code path for opening links.
func OpenPR(ctx context.Context, runner exec.CommandRunner, url string) error {
	_, err := runner.Run(ctx, "gh", "pr", "view", url, "--web")
	return err
}
