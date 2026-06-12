package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds the dashboard-level (global) key bindings. Bindings that act
// inside a widget (cursor movement, opening an item, switching tabs) are owned
// by the widget itself.
type KeyMap struct {
	Quit       key.Binding
	Next       key.Binding
	Prev       key.Binding
	Refresh    key.Binding
	RefreshAll key.Binding
}

// DefaultKeyMap returns DevDeck's standard global bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Next: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next section"),
		),
		Prev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "switch widget"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		RefreshAll: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "refresh all"),
		),
	}
}
