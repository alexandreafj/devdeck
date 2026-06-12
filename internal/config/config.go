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
}

// Default returns the zero-config dashboard: a single GitHub PR widget showing
// all three modes, auto-refreshing every minute.
func Default() Config {
	return Config{
		Layout: LayoutConfig{MaxWidgets: maxWidgetsCap},
		Widgets: []WidgetConfig{{
			Type:    "github_prs",
			Title:   "PRs",
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
