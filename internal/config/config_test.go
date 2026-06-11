package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReturnsDefaultsWhenFileMissing(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Default()
	if cfg.Layout.MaxWidgets != want.Layout.MaxWidgets {
		t.Errorf("MaxWidgets = %d, want %d", cfg.Layout.MaxWidgets, want.Layout.MaxWidgets)
	}
	if len(cfg.Widgets) != 1 || cfg.Widgets[0].Type != "github_prs" {
		t.Errorf("widgets = %+v, want a single github_prs widget", cfg.Widgets)
	}
}

func TestLoadReadsValidFile(t *testing.T) {
	path := writeTemp(t, `
layout:
  max_widgets: 2
widgets:
  - type: github_prs
    title: "My PRs"
    modes: [authored]
    refresh: 10m
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Layout.MaxWidgets != 2 {
		t.Errorf("MaxWidgets = %d, want 2", cfg.Layout.MaxWidgets)
	}
	if len(cfg.Widgets) != 1 {
		t.Fatalf("got %d widgets, want 1", len(cfg.Widgets))
	}
	w := cfg.Widgets[0]
	if w.Title != "My PRs" || w.Refresh != "10m" || len(w.Modes) != 1 || w.Modes[0] != "authored" {
		t.Errorf("widget = %+v, not parsed as expected", w)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := writeTemp(t, "layout: [this is : not valid")
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoadCapsMaxWidgets(t *testing.T) {
	path := writeTemp(t, "layout:\n  max_widgets: 10\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Layout.MaxWidgets != 4 {
		t.Errorf("MaxWidgets = %d, want capped at 4", cfg.Layout.MaxWidgets)
	}
}

func TestLoadZeroMaxWidgetsGetsDefault(t *testing.T) {
	path := writeTemp(t, "layout:\n  max_widgets: 0\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Layout.MaxWidgets != 4 {
		t.Errorf("MaxWidgets = %d, want default 4", cfg.Layout.MaxWidgets)
	}
}

func TestLoadEmptyWidgetsGetsDefaultWidget(t *testing.T) {
	path := writeTemp(t, "layout:\n  max_widgets: 3\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Widgets) != 1 || cfg.Widgets[0].Type != "github_prs" {
		t.Errorf("widgets = %+v, want a default github_prs widget when none configured", cfg.Widgets)
	}
}

func TestDefaultPathEndsWithDevdeckConfig(t *testing.T) {
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(p, filepath.Join("devdeck", "config.yml")) {
		t.Errorf("DefaultPath() = %q, want it to end with devdeck/config.yml", p)
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}
