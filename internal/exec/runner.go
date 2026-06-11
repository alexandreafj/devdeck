// Package exec defines the single seam through which DevDeck shells out to
// external tools (notably the `gh` CLI). Production code depends on the
// CommandRunner interface, so tests can substitute a fake and never touch a
// real process or the network. See exec/exectest for that fake.
package exec

import (
	"bytes"
	"context"
	"fmt"
	osexec "os/exec"
)

// CommandRunner runs an external command and returns its stdout. Implementations
// must be safe to call concurrently.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// OSRunner is the production CommandRunner backed by os/exec.
type OSRunner struct{}

// New returns a ready-to-use OSRunner.
func New() *OSRunner { return &OSRunner{} }

// Run executes name with args, returning captured stdout. On failure the
// returned error wraps the process error and includes any stderr output, which
// makes `gh` failures (auth, rate limits) legible to the user.
func (OSRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := osexec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%s: %w: %s", name, err, stderr.String())
		}
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return stdout.Bytes(), nil
}
