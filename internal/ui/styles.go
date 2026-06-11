package ui

import "github.com/charmbracelet/lipgloss"

// Styles holds the lipgloss styles used to render the dashboard frame and a
// widget's contents. Keeping them in one struct makes the look consistent and
// easy to adjust (or theme) in one place.
type Styles struct {
	FocusedBorder   lipgloss.Style
	UnfocusedBorder lipgloss.Style
	Title           lipgloss.Style
	SelectedItem    lipgloss.Style
	Meta            lipgloss.Style
	Help            lipgloss.Style
	ActiveTab       lipgloss.Style
	InactiveTab     lipgloss.Style
}

// DefaultStyles returns DevDeck's default palette.
func DefaultStyles() Styles {
	const (
		accent = lipgloss.Color("69")  // periwinkle blue
		muted  = lipgloss.Color("244") // grey
		subtle = lipgloss.Color("240") // dim grey
		bright = lipgloss.Color("255") // near-white
	)

	border := lipgloss.RoundedBorder()

	return Styles{
		FocusedBorder: lipgloss.NewStyle().
			Border(border).
			BorderForeground(accent),
		UnfocusedBorder: lipgloss.NewStyle().
			Border(border).
			BorderForeground(subtle),
		Title: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),
		SelectedItem: lipgloss.NewStyle().
			Foreground(bright).
			Bold(true),
		Meta: lipgloss.NewStyle().
			Foreground(muted),
		Help: lipgloss.NewStyle().
			Foreground(subtle),
		ActiveTab: lipgloss.NewStyle().
			Foreground(bright).
			Bold(true).
			Underline(true),
		InactiveTab: lipgloss.NewStyle().
			Foreground(muted),
	}
}
