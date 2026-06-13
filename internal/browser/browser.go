// Package browser opens URLs in the user's default web browser. GitHub links go
// through `gh pr view --web`, but other widgets (Calendar, …) need a generic,
// cross-platform opener. All process execution goes through exec.CommandRunner,
// so callers stay fully unit-testable.
package browser

import (
	"context"
	"runtime"

	"github.com/alexandreafj/devdeck/internal/exec"
)

// Open launches url in the default browser, shelling out through runner. The
// command is chosen per operating system (macOS `open`, Windows `rundll32`,
// otherwise `xdg-open`).
func Open(ctx context.Context, runner exec.CommandRunner, url string) error {
	name, args := command(url)
	_, err := runner.Run(ctx, name, args...)
	return err
}

// command returns the OS-appropriate program and arguments to open url. It is
// split out from Open so the platform mapping is testable in isolation.
func command(url string) (string, []string) {
	switch runtime.GOOS {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		return "open", []string{url}
	default:
		return "xdg-open", []string{url}
	}
}
