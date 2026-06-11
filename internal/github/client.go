package github

import (
	"context"

	"github.com/alexandreafj/devdeck/internal/exec"
)

// jsonFields is the set of fields requested from `gh search prs`. It matches the
// json tags decoded by DecodePRs.
const jsonFields = "title,url,repository,author,createdAt"

// PRProvider fetches pull requests for a given mode. The widget depends on this
// interface (not GHClient) so tests can supply a fake provider.
type PRProvider interface {
	FetchPRs(ctx context.Context, mode Mode) ([]PR, error)
}

// GHClient is the production PRProvider, backed by the `gh` CLI through a
// CommandRunner.
type GHClient struct {
	runner exec.CommandRunner
}

// NewClient returns a GHClient that runs `gh` via the given runner.
func NewClient(runner exec.CommandRunner) *GHClient {
	return &GHClient{runner: runner}
}

// FetchPRs runs `gh search prs --state open <mode-flag> @me --json …` across all
// repositories and decodes the result.
func (c *GHClient) FetchPRs(ctx context.Context, mode Mode) ([]PR, error) {
	args := []string{
		"search", "prs",
		"--state", "open",
		mode.searchFlag(), "@me",
		"--json", jsonFields,
	}
	out, err := c.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, err
	}
	return DecodePRs(out)
}
