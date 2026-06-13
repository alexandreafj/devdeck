package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpText is the persistent key hint shown beneath the widgets.
const helpText = "tab section · shift+tab widget · 1-4 jump · ↑/↓ move · / filter · enter open · r refresh · q quit"

// inputCapturer is implemented by widgets that temporarily consume all key
// input (e.g. an open text filter). While a focused widget is capturing, the
// dashboard forwards every key to it instead of applying its global bindings,
// so the user can type characters like "q" or digits without quitting/jumping.
type inputCapturer interface{ CapturingInput() bool }

// Dashboard is the top-level Bubble Tea model. It owns the widget set, which
// widget has focus, and the terminal size, and routes input: global keys it
// handles itself, other keys go to the focused widget, and any non-key message
// is broadcast to every widget (so async results reach whichever widget issued
// the request).
type Dashboard struct {
	widgets       []Widget
	focus         int
	width, height int
	keys          KeyMap
	styles        Styles
}

// NewDashboard builds a Dashboard for the given widgets, focused on the first.
func NewDashboard(widgets ...Widget) Dashboard {
	return Dashboard{
		widgets: widgets,
		keys:    DefaultKeyMap(),
		styles:  DefaultStyles(),
	}
}

// Init batches every widget's Init so they can kick off their first load.
func (d Dashboard) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(d.widgets))
	for _, w := range d.widgets {
		cmds = append(cmds, w.Init())
	}
	return tea.Batch(cmds...)
}

// Update routes messages. See the Dashboard doc comment for the routing rules.
func (d Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width, d.height = msg.Width, msg.Height
		return d, nil
	case tea.KeyMsg:
		return d.handleKey(msg)
	default:
		return d.broadcast(msg)
	}
}

func (d Dashboard) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While the focused widget is capturing text (e.g. its filter is open),
	// every key is its input, so skip the global bindings entirely.
	if c, ok := d.focused().(inputCapturer); ok && c.CapturingInput() {
		return d.updateFocused(msg)
	}

	switch {
	case key.Matches(msg, d.keys.Quit):
		return d, tea.Quit
	case key.Matches(msg, d.keys.Next): // tab: cycle sections inside the focused widget
		return d.updateFocused(msg)
	case key.Matches(msg, d.keys.Prev): // shift+tab: switch between widgets
		d.focusBy(1)
		return d, nil
	case key.Matches(msg, d.keys.RefreshAll):
		return d, d.refreshAll()
	case key.Matches(msg, d.keys.Refresh):
		if w := d.focused(); w != nil {
			return d, w.Refresh()
		}
		return d, nil
	}

	if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= len(d.widgets) {
		d.focus = n - 1
		return d, nil
	}

	return d.updateFocused(msg)
}

// focusBy advances focus by delta, wrapping. It is a no-op with no widgets.
func (d *Dashboard) focusBy(delta int) {
	if len(d.widgets) == 0 {
		return
	}
	d.focus = (d.focus + delta + len(d.widgets)) % len(d.widgets)
}

func (d Dashboard) focused() Widget {
	if len(d.widgets) == 0 {
		return nil
	}
	return d.widgets[d.focus]
}

func (d Dashboard) updateFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(d.widgets) == 0 {
		return d, nil
	}
	w, cmd := d.widgets[d.focus].Update(msg)
	d.widgets[d.focus] = w
	return d, cmd
}

// broadcast forwards a message to every widget, collecting their commands.
func (d Dashboard) broadcast(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0, len(d.widgets))
	for i, w := range d.widgets {
		updated, cmd := w.Update(msg)
		d.widgets[i] = updated
		cmds = append(cmds, cmd)
	}
	return d, tea.Batch(cmds...)
}

func (d Dashboard) refreshAll() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(d.widgets))
	for _, w := range d.widgets {
		cmds = append(cmds, w.Refresh())
	}
	return tea.Batch(cmds...)
}

// View renders the widgets into their responsive layout slots, framing each in
// a border (highlighted for the focused widget) with its title, and appends the
// global help line.
func (d Dashboard) View() string {
	if len(d.widgets) == 0 {
		return "No widgets configured.\n\n" + d.styles.Help.Render(helpText)
	}

	bodyHeight := d.height - 1 // reserve a line for help
	if bodyHeight < 1 {
		bodyHeight = d.height
	}
	rects := Layout(d.width, bodyHeight, len(d.widgets))

	panes := make([]string, len(d.widgets))
	for i, w := range d.widgets {
		panes[i] = d.renderPane(w, i, rects[i])
	}

	return d.assemble(rects, panes)
}

// renderPane frames a single widget within its rect.
func (d Dashboard) renderPane(w Widget, idx int, r Rect) string {
	frame := d.styles.UnfocusedBorder
	if idx == d.focus {
		frame = d.styles.FocusedBorder
	}

	// Inner content area excludes the 1-cell border on each side.
	innerW := max(r.Width-2, 0)
	innerH := max(r.Height-2, 0)

	title := d.styles.Title.Render(w.Title())
	body := w.View(innerW, innerH-1) // -1 for the title line
	content := lipgloss.JoinVertical(lipgloss.Left, title, body)

	// Hard-clip to the inner area so a widget that emits lines wider or taller
	// than its pane (a long PR title, an unwrapped error) can't overflow and
	// break the surrounding grid. Truncates rather than wraps, keeping the layout
	// stable regardless of widget content.
	content = lipgloss.NewStyle().MaxWidth(innerW).MaxHeight(innerH).Render(content)

	return frame.Width(innerW).Height(innerH).Render(content)
}

// assemble joins panes that share a row horizontally, then stacks rows, then
// appends the help line. Rows are detected by equal Y.
func (d Dashboard) assemble(rects []Rect, panes []string) string {
	var rows []string
	i := 0
	for i < len(panes) {
		y := rects[i].Y
		var rowPanes []string
		for i < len(panes) && rects[i].Y == y {
			rowPanes = append(rowPanes, panes[i])
			i++
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowPanes...))
	}
	grid := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return strings.Join([]string{grid, d.styles.Help.Render(helpText)}, "\n")
}
