// Package app wires configuration into concrete widgets. Keeping this assembly
// here (rather than in main) makes the type dispatch and max-widget capping
// unit-testable, leaving cmd/devdeck as a thin entrypoint.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandreafj/devdeck/internal/config"
	"github.com/alexandreafj/devdeck/internal/exec"
	"github.com/alexandreafj/devdeck/internal/gcal"
	"github.com/alexandreafj/devdeck/internal/github"
	"github.com/alexandreafj/devdeck/internal/ui"
	"github.com/alexandreafj/devdeck/internal/widgets/calendar"
	"github.com/alexandreafj/devdeck/internal/widgets/githubprs"
)

// defaultTitles supplies a fallback title per widget type when the config omits
// one.
var defaultTitles = map[string]string{
	"github_prs":      "Github Pull Requests",
	"google_calendar": "Google Calendar",
}

// Build turns a Config into the dashboard's widgets, honoring Layout.MaxWidgets
// and skipping unknown widget types. The runner is shared by every widget so
// they all shell out through the same seam.
func Build(cfg config.Config, runner exec.CommandRunner) []ui.Widget {
	var widgets []ui.Widget
	for i, wc := range cfg.Widgets {
		if wc.Disabled {
			continue
		}
		if len(widgets) >= cfg.Layout.MaxWidgets {
			break
		}
		if w := buildWidget(i, wc, runner); w != nil {
			widgets = append(widgets, w)
		}
	}
	return widgets
}

func buildWidget(index int, wc config.WidgetConfig, runner exec.CommandRunner) ui.Widget {
	id := fmt.Sprintf("%s-%d", wc.Type, index)
	title := wc.Title
	if title == "" {
		title = defaultTitles[wc.Type]
	}

	switch wc.Type {
	case "github_prs":
		return githubprs.
			New(id, title, github.NewClient(runner), runner, github.ParseModes(wc.Modes)).
			SetRefreshInterval(parseRefresh(wc.Refresh))
	case "google_calendar":
		client := gcal.NewClient(expandUser(wc.CredentialsPath), expandUser(wc.TokenPath), wc.CalendarID)
		return calendar.
			New(id, title, client, runner).
			SetRefreshInterval(parseRefresh(wc.Refresh))
	default:
		return nil
	}
}

// CalendarAuthPaths returns the credentials and token paths configured for the
// first google_calendar widget (with a leading "~/" expanded), or empty strings
// when none is configured, in which case callers fall back to the default
// locations. `devdeck auth google` uses this so auth and the widget agree.
func CalendarAuthPaths(cfg config.Config) (credentials, token string) {
	for _, wc := range cfg.Widgets {
		if wc.Type == "google_calendar" {
			return expandUser(wc.CredentialsPath), expandUser(wc.TokenPath)
		}
	}
	return "", ""
}

// expandUser expands a leading "~/" in path to the user's home directory. Other
// paths (including empty) are returned unchanged.
func expandUser(path string) string {
	if path == "" || !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}

// parseRefresh interprets a widget's refresh interval (e.g. "1m", "30s"). An
// empty, malformed, or non-positive value disables auto-refresh (returns 0).
func parseRefresh(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil || d <= 0 {
		return 0
	}
	return d
}
