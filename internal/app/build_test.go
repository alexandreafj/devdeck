package app

import (
	"testing"
	"time"

	"github.com/alexandreafj/devdeck/internal/config"
	"github.com/alexandreafj/devdeck/internal/exec/exectest"
)

func cfg(maxWidgets int, widgets ...config.WidgetConfig) config.Config {
	return config.Config{
		Layout:  config.LayoutConfig{MaxWidgets: maxWidgets},
		Widgets: widgets,
	}
}

func TestBuildCreatesGithubWidget(t *testing.T) {
	c := cfg(4, config.WidgetConfig{Type: "github_prs", Title: "My PRs", Modes: []string{"authored"}})
	widgets := Build(c, &exectest.FakeRunner{})
	if len(widgets) != 1 {
		t.Fatalf("got %d widgets, want 1", len(widgets))
	}
	if widgets[0].Title() != "My PRs" {
		t.Errorf("Title = %q, want My PRs", widgets[0].Title())
	}
}

func TestBuildDefaultsTitleWhenMissing(t *testing.T) {
	c := cfg(4, config.WidgetConfig{Type: "github_prs"})
	widgets := Build(c, &exectest.FakeRunner{})
	if len(widgets) != 1 || widgets[0].Title() != "Github Pull Requests" {
		t.Errorf("expected a single widget titled 'Github Pull Requests', got %+v", widgets)
	}
}

func TestBuildCreatesCalendarWidget(t *testing.T) {
	c := cfg(4, config.WidgetConfig{Type: "google_calendar"})
	widgets := Build(c, &exectest.FakeRunner{})
	if len(widgets) != 1 || widgets[0].Title() != "Google Calendar" {
		t.Errorf("expected a single widget titled 'Google Calendar', got %+v", widgets)
	}
}

func TestBuildCapsAtMaxWidgets(t *testing.T) {
	c := cfg(2,
		config.WidgetConfig{Type: "github_prs"},
		config.WidgetConfig{Type: "github_prs"},
		config.WidgetConfig{Type: "github_prs"},
		config.WidgetConfig{Type: "github_prs"},
	)
	if got := len(Build(c, &exectest.FakeRunner{})); got != 2 {
		t.Errorf("built %d widgets, want capped at 2", got)
	}
}

func TestBuildSkipsUnknownTypes(t *testing.T) {
	c := cfg(4,
		config.WidgetConfig{Type: "github_prs"},
		config.WidgetConfig{Type: "weather"},
	)
	if got := len(Build(c, &exectest.FakeRunner{})); got != 1 {
		t.Errorf("built %d widgets, want 1 (unknown type skipped)", got)
	}
}

func TestBuildSkipsDisabledWidgets(t *testing.T) {
	c := cfg(4,
		config.WidgetConfig{Type: "github_prs"},
		config.WidgetConfig{Type: "google_calendar", Disabled: true},
	)
	widgets := Build(c, &exectest.FakeRunner{})
	if len(widgets) != 1 || widgets[0].Title() != "Github Pull Requests" {
		t.Errorf("a disabled widget should be skipped, got %+v", widgets)
	}
}

func TestBuildDisabledWidgetDoesNotConsumeSlot(t *testing.T) {
	c := cfg(1,
		config.WidgetConfig{Type: "github_prs", Disabled: true},
		config.WidgetConfig{Type: "github_prs", Title: "Visible"},
	)
	widgets := Build(c, &exectest.FakeRunner{})
	if len(widgets) != 1 || widgets[0].Title() != "Visible" {
		t.Errorf("a disabled widget must not take a max-widgets slot; got %+v", widgets)
	}
}

func TestParseRefresh(t *testing.T) {
	cases := map[string]time.Duration{
		"1m":  time.Minute,
		"30s": 30 * time.Second,
		"5m":  5 * time.Minute,
		"":    0,
		"abc": 0,
		"-5s": 0,
		"0":   0,
	}
	for in, want := range cases {
		if got := parseRefresh(in); got != want {
			t.Errorf("parseRefresh(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestBuildGivesWidgetsUniqueIDs(t *testing.T) {
	c := cfg(4,
		config.WidgetConfig{Type: "github_prs"},
		config.WidgetConfig{Type: "github_prs"},
	)
	widgets := Build(c, &exectest.FakeRunner{})
	if len(widgets) != 2 {
		t.Fatalf("got %d widgets, want 2", len(widgets))
	}
	if widgets[0].ID() == widgets[1].ID() {
		t.Errorf("widget IDs must be unique, both are %q", widgets[0].ID())
	}
}
