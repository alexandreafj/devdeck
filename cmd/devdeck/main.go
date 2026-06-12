// Command devdeck is the DevDeck terminal dashboard: a Bubble Tea TUI showing
// the things waiting on you, starting with GitHub pull requests across all your
// repositories via the authenticated `gh` CLI.
//
// This file is deliberately thin — the testable logic lives in the internal
// packages (config, app, ui, github, …). It only loads config, wires widgets,
// and runs the program.
package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexandreafj/devdeck/internal/app"
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
	if len(args) > 0 && args[0] == "auth" {
		return runAuth(args[1:])
	}
	return runDashboard()
}

// runAuth handles `devdeck auth google`, connecting the user's Google account
// for the Calendar widget.
func runAuth(args []string) error {
	if len(args) != 1 || args[0] != "google" {
		return fmt.Errorf("usage: devdeck auth google")
	}
	return gcal.Authorize(context.Background(), exec.New())
}

func runDashboard() error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
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
