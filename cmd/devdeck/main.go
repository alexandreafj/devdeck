// Command devdeck is the DevDeck terminal dashboard: a Bubble Tea TUI showing
// the things waiting on you, starting with GitHub pull requests across all your
// repositories via the authenticated `gh` CLI.
//
// This file is deliberately thin: the testable logic lives in the internal
// packages (config, app, ui, github, cli, ...). It only loads config, wires
// widgets, and runs the program.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexandreafj/devdeck/internal/app"
	"github.com/alexandreafj/devdeck/internal/cli"
	"github.com/alexandreafj/devdeck/internal/config"
	"github.com/alexandreafj/devdeck/internal/exec"
	"github.com/alexandreafj/devdeck/internal/gcal"
	"github.com/alexandreafj/devdeck/internal/ui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "devdeck:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "auth":
			return runAuth(args[1:])
		case "-h", "--help", "help":
			fmt.Print(cli.HelpText())
			return nil
		}
	}
	return runDashboard()
}

// runAuth handles `devdeck auth google [--reset] [path/to/credentials.json]`,
// connecting the user's Google account for the Calendar widget. Credentials/token
// paths come from the configured google_calendar widget (so auth and the widget
// agree); an optional path argument overrides the credentials location. With
// --reset it instead removes the saved credentials and token so you can reconnect
// from scratch (e.g. switch from a service account to OAuth).
func runAuth(args []string) error {
	if len(args) < 1 || args[0] != "google" {
		return fmt.Errorf("usage: devdeck auth google [--reset] [path/to/credentials.json]")
	}

	reset := false
	var override string
	for _, a := range args[1:] {
		switch a {
		case "--reset", "-reset", "reset":
			reset = true
		default:
			if s := strings.TrimSpace(a); s != "" {
				override = s
			}
		}
	}

	var credsPath, tokenPath string
	if path, err := config.DefaultPath(); err == nil {
		if cfg, err := config.Load(path); err == nil {
			credsPath, tokenPath = app.CalendarAuthPaths(cfg)
		}
	}
	if override != "" {
		credsPath = override
	}

	if reset {
		return gcal.Reset(os.Stdout, credsPath, tokenPath)
	}
	return gcal.Authorize(context.Background(), exec.New(), credsPath, tokenPath)
}

func runDashboard() error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	// Always materialize a config file the user can edit to add/remove widgets.
	if created, err := config.Ensure(path); err != nil {
		fmt.Fprintln(os.Stderr, "devdeck: could not create config:", err)
	} else if created {
		fmt.Fprintln(os.Stderr, "devdeck: created a starter config at", path)
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}

	runner := exec.New()
	widgets := app.Build(cfg, runner)

	program := tea.NewProgram(ui.NewDashboard(widgets...), tea.WithAltScreen())
	_, err = program.Run()
	return err
}
