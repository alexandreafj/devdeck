// Package ui is DevDeck's Bubble Tea layer: the Widget extension point, the
// responsive layout, and the top-level Dashboard model that owns focus and
// routes input. Widgets (GitHub, Calendar, …) live in sibling packages and only
// depend on the Widget interface here.
package ui

import tea "github.com/charmbracelet/bubbletea"

// Widget is the extension point of the dashboard. Each implementation is a
// self-contained Bubble Tea component: it manages its own data and selection,
// renders into the area the dashboard gives it, and can refresh on demand.
//
// Update returns the (possibly updated) Widget so implementations may use value
// receivers; the dashboard stores whatever is returned. View receives the inner
// content size (the dashboard draws the surrounding frame and focus styling).
type Widget interface {
	ID() string
	Title() string
	Init() tea.Cmd
	Update(msg tea.Msg) (Widget, tea.Cmd)
	View(width, height int) string
	Refresh() tea.Cmd
}
