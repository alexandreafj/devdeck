// Package cli renders DevDeck's command-line help. Keeping the text here (rather
// than in main) makes it testable and keeps the entrypoint thin.
package cli

// HelpText returns the `devdeck --help` output: the available commands and where
// the config file lives.
func HelpText() string {
	return `DevDeck: a terminal dashboard for the things waiting on you.

Usage:
  devdeck                       Launch the dashboard
  devdeck auth google           Connect Google Calendar (guided setup)
  devdeck auth google --reset   Disconnect and remove saved Google credentials
  devdeck --help                Show this help

Config (edit it, then restart devdeck):
  macOS:  ~/Library/Application Support/devdeck/config.yml
  Linux:  ~/.config/devdeck/config.yml

More: https://github.com/alexandreafj/devdeck
`
}
