// Package config loads DevDeck's YAML configuration, falling back to sensible
// defaults so the app runs out of the box with no config file. The schema is
// deliberately small; widgets read their own typed options from WidgetConfig.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// maxWidgetsCap is the hard ceiling on visible widgets (left-to-right slots).
const maxWidgetsCap = 4

// DefaultConfigYAML is the documented starter config written on first run (see
// Ensure). It keeps comments and instructions YAML marshalling would drop, so
// the user has a clear, editable file: add, remove, reorder, or duplicate
// widgets, each authenticating on its own.
const DefaultConfigYAML = `# DevDeck configuration.
#
# Created automatically on first run — edit it to choose your widgets. The
# dashboard shows up to 4 widgets, left to right. Add, remove, reorder, or
# duplicate entries under "widgets:" then restart devdeck. Each widget
# authenticates on its own (see the notes below).

layout:
  # Maximum widgets shown left-to-right. Capped at 4.
  max_widgets: 4

widgets:
  # GitHub pull requests across all your repos.
  # Auth: run ` + "`gh auth login`" + ` once (the GitHub CLI).
  - type: github_prs
    title: "Github Pull Requests"
    # Sections, switched with tab or ←/→ in the widget. Any subset of:
    #   authored, review_requested, assigned
    modes: [authored, review_requested, assigned]
    refresh: 1m            # auto-refresh (e.g. 30s, 1m, 5m); omit/0 to disable

  # Google Calendar — this week's meetings grouped by day.
  # You bring your OWN Google OAuth client (DevDeck ships none): create an OAuth
  # "Desktop app" client in your Google Cloud project. Run the guided
  # ` + "`devdeck auth google`" + ` once, then uncomment this block.
  # - type: google_calendar
  #   title: "Google Calendar"
  #   refresh: 5m
  #   # Optional — point at your own files (default: this config dir):
  #   # credentials_path: ~/secrets/my-oauth-client.json
  #   # token_path: ~/secrets/devdeck-google-token.json
`

// Config is the top-level configuration.
type Config struct {
	Layout  LayoutConfig   `yaml:"layout"`
	Widgets []WidgetConfig `yaml:"widgets"`
}

// LayoutConfig controls the dashboard layout.
type LayoutConfig struct {
	MaxWidgets int `yaml:"max_widgets"`
}

// WidgetConfig describes one widget instance. Type selects the implementation;
// the remaining fields are common options interpreted per widget.
type WidgetConfig struct {
	Type    string   `yaml:"type"`
	Title   string   `yaml:"title"`
	Modes   []string `yaml:"modes"`
	Refresh string   `yaml:"refresh"`

	// CredentialsPath and TokenPath let a widget that authenticates with its own
	// OAuth client (e.g. google_calendar) point at a user-chosen credentials JSON
	// and token cache. Empty means the default location in the config dir. A
	// leading "~/" is expanded to the user's home.
	CredentialsPath string `yaml:"credentials_path"`
	TokenPath       string `yaml:"token_path"`
}

// Default returns the zero-config dashboard: a single GitHub PR widget showing
// all three modes, auto-refreshing every minute.
func Default() Config {
	return Config{
		Layout: LayoutConfig{MaxWidgets: maxWidgetsCap},
		Widgets: []WidgetConfig{{
			Type:    "github_prs",
			Title:   "Github Pull Requests",
			Modes:   []string{"authored", "review_requested", "assigned"},
			Refresh: "1m",
		}},
	}
}

// DefaultPath is the OS-conventional config location, e.g.
// ~/Library/Application Support/devdeck/config.yml on macOS or
// ~/.config/devdeck/config.yml on Linux.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, "devdeck", "config.yml"), nil
}

// Ensure writes the documented starter config (DefaultConfigYAML) to path when
// no file exists yet, creating the parent directory, so the user always has a
// config to edit. An existing file is left untouched. It reports whether a new
// file was created.
func Ensure(path string) (bool, error) {
	switch _, err := os.Stat(path); {
	case err == nil:
		return false, nil
	case !errors.Is(err, fs.ErrNotExist):
		return false, fmt.Errorf("stat config %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(DefaultConfigYAML), 0o644); err != nil {
		return false, fmt.Errorf("write config %s: %w", path, err)
	}
	return true, nil
}

// Load reads and parses the config at path. A missing file is not an error: it
// yields Default(). A present-but-malformed file is an error. After parsing,
// defaults are applied so the returned Config is always usable.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

// applyDefaults fills in missing or out-of-range values so callers never see an
// unusable Config.
func (c *Config) applyDefaults() {
	if c.Layout.MaxWidgets <= 0 || c.Layout.MaxWidgets > maxWidgetsCap {
		c.Layout.MaxWidgets = maxWidgetsCap
	}
	if len(c.Widgets) == 0 {
		c.Widgets = Default().Widgets
	}
}
