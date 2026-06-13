package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// overflowWidget renders a body with lines wider and taller than any pane, to
// prove the dashboard clips each widget to its pane rather than letting content
// overflow and break the grid.
type overflowWidget struct{ title string }

func (overflowWidget) ID() string                         { return "overflow" }
func (w overflowWidget) Title() string                    { return w.title }
func (overflowWidget) Init() tea.Cmd                      { return nil }
func (overflowWidget) Refresh() tea.Cmd                   { return nil }
func (w overflowWidget) Update(tea.Msg) (Widget, tea.Cmd) { return w, nil }
func (overflowWidget) View(int, int) string {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = strings.Repeat("x", 200) // far wider than any pane
	}
	return strings.Join(lines, "\n")
}

func TestDashboardClipsWidgetsToPanes(t *testing.T) {
	const w, h = 100, 24
	d := NewDashboard(overflowWidget{title: "Left"}, overflowWidget{title: "Right"})
	m, _ := d.Update(tea.WindowSizeMsg{Width: w, Height: h})
	d = m.(Dashboard)

	out := d.View()
	for i, line := range strings.Split(out, "\n") {
		if got := lipgloss.Width(line); got > w {
			t.Errorf("line %d is %d cols wide, exceeds terminal width %d:\n%q", i, got, w, line)
		}
	}
	if got := lipgloss.Height(out); got > h {
		t.Errorf("rendered height %d exceeds terminal height %d", got, h)
	}
}
