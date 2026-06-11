// Command devdeck is the DevDeck terminal dashboard: a Bubble Tea TUI showing
// the things waiting on you, starting with GitHub pull requests across all your
// repositories via the authenticated `gh` CLI.
//
// This file is deliberately thin — the testable logic lives in the internal
// packages (config, app, ui, github, …). It only loads config, wires widgets,
// and runs the program.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexandreafj/devdeck/internal/app"
	"github.com/alexandreafj/devdeck/internal/config"
	"github.com/alexandreafj/devdeck/internal/exec"
	"github.com/alexandreafj/devdeck/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "devdeck:", err)
		os.Exit(1)
	}
}

func run() error {
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
