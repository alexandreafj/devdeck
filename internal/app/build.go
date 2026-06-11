// Package app wires configuration into concrete widgets. Keeping this assembly
// here (rather than in main) makes the type dispatch and max-widget capping
// unit-testable, leaving cmd/devdeck as a thin entrypoint.
package app

import (
	"fmt"

	"github.com/alexandreafj/devdeck/internal/config"
	"github.com/alexandreafj/devdeck/internal/exec"
	"github.com/alexandreafj/devdeck/internal/github"
	"github.com/alexandreafj/devdeck/internal/ui"
	"github.com/alexandreafj/devdeck/internal/widgets/githubprs"
)

// defaultTitles supplies a fallback title per widget type when the config omits
// one.
var defaultTitles = map[string]string{
	"github_prs": "PRs",
}

// Build turns a Config into the dashboard's widgets, honouring Layout.MaxWidgets
// and skipping unknown widget types. The runner is shared by every widget so
// they all shell out through the same seam.
func Build(cfg config.Config, runner exec.CommandRunner) []ui.Widget {
	var widgets []ui.Widget
	for i, wc := range cfg.Widgets {
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
		return githubprs.New(id, title, github.NewClient(runner), runner, github.ParseModes(wc.Modes))
	default:
		return nil
	}
}
