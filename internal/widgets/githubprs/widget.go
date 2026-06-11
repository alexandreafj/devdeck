// Package githubprs implements the GitHub pull-request widget: three switchable
// tabs (authored, review-requested, assigned) listing PRs across all of the
// user's repositories. It depends on github.PRProvider and exec.CommandRunner
// (both interfaces), so it is fully unit-testable without invoking `gh`.
package githubprs

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alexandreafj/devdeck/internal/domain"
	"github.com/alexandreafj/devdeck/internal/exec"
	"github.com/alexandreafj/devdeck/internal/github"
	"github.com/alexandreafj/devdeck/internal/ui"
)

// tabState is the per-mode view state.
type tabState struct {
	mode    github.Mode
	items   []domain.Item
	cursor  int
	loading bool
	loaded  bool
	err     error
}

// Widget is the GitHub PR widget. It implements ui.Widget with a pointer
// receiver so its mutable tab state is straightforward to reason about.
type Widget struct {
	id       string
	title    string
	provider github.PRProvider
	runner   exec.CommandRunner
	now      func() time.Time
	tabs     []tabState
	active   int
	styles   ui.Styles
}

// New builds a GitHub PR widget. If modes is empty it defaults to all modes.
func New(id, title string, provider github.PRProvider, runner exec.CommandRunner, modes []github.Mode) *Widget {
	if len(modes) == 0 {
		modes = github.Modes
	}
	tabs := make([]tabState, len(modes))
	for i, m := range modes {
		tabs[i] = tabState{mode: m}
	}
	return &Widget{
		id:       id,
		title:    title,
		provider: provider,
		runner:   runner,
		now:      time.Now,
		tabs:     tabs,
		styles:   ui.DefaultStyles(),
	}
}

// ID returns the widget's stable identifier.
func (w *Widget) ID() string { return w.id }

// Title returns the widget's display title.
func (w *Widget) Title() string { return w.title }

// Init begins loading the active tab.
func (w *Widget) Init() tea.Cmd {
	return w.loadActive()
}

// Refresh reloads the active tab.
func (w *Widget) Refresh() tea.Cmd {
	return w.loadActive()
}

// loadActive marks the active tab loading and returns its fetch command.
func (w *Widget) loadActive() tea.Cmd {
	t := &w.tabs[w.active]
	t.loading = true
	return fetchCmd(w.id, w.provider, t.mode, w.now)
}

// prsLoadedMsg carries the result of a fetch back into Update. widgetID lets a
// widget ignore results addressed to a different instance (the dashboard
// broadcasts non-key messages to every widget).
type prsLoadedMsg struct {
	widgetID string
	mode     github.Mode
	items    []domain.Item
	err      error
}

// prOpenedMsg reports the result of opening a PR in the browser.
type prOpenedMsg struct {
	err error
}

func fetchCmd(id string, provider github.PRProvider, mode github.Mode, now func() time.Time) tea.Cmd {
	return func() tea.Msg {
		prs, err := provider.FetchPRs(context.Background(), mode)
		if err != nil {
			return prsLoadedMsg{widgetID: id, mode: mode, err: err}
		}
		n := now()
		items := make([]domain.Item, len(prs))
		for i, p := range prs {
			items[i] = p.ToItem(n, mode)
		}
		return prsLoadedMsg{widgetID: id, mode: mode, items: items}
	}
}

func openCmd(runner exec.CommandRunner, url string) tea.Cmd {
	return func() tea.Msg {
		return prOpenedMsg{err: github.OpenPR(context.Background(), runner, url)}
	}
}

// Update handles load results and key input.
func (w *Widget) Update(msg tea.Msg) (ui.Widget, tea.Cmd) {
	switch msg := msg.(type) {
	case prsLoadedMsg:
		w.applyLoad(msg)
		return w, nil
	case tea.KeyMsg:
		return w.handleKey(msg)
	}
	return w, nil
}

func (w *Widget) applyLoad(msg prsLoadedMsg) {
	if msg.widgetID != w.id {
		return
	}
	for i := range w.tabs {
		if w.tabs[i].mode != msg.mode {
			continue
		}
		w.tabs[i].loading = false
		w.tabs[i].loaded = true
		w.tabs[i].err = msg.err
		w.tabs[i].items = msg.items
		if w.tabs[i].cursor >= len(msg.items) {
			w.tabs[i].cursor = max(0, len(msg.items)-1)
		}
	}
}

func (w *Widget) handleKey(msg tea.KeyMsg) (ui.Widget, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		w.moveCursor(-1)
	case "down", "j":
		w.moveCursor(1)
	case "left", "h":
		return w, w.switchTab(-1)
	case "right", "l":
		return w, w.switchTab(1)
	case "enter":
		return w, w.openSelected()
	}
	return w, nil
}

func (w *Widget) moveCursor(delta int) {
	t := &w.tabs[w.active]
	if len(t.items) == 0 {
		return
	}
	t.cursor = min(max(t.cursor+delta, 0), len(t.items)-1)
}

// switchTab moves to an adjacent tab (wrapping) and lazily loads it the first
// time it is shown.
func (w *Widget) switchTab(delta int) tea.Cmd {
	w.active = (w.active + delta + len(w.tabs)) % len(w.tabs)
	if t := &w.tabs[w.active]; !t.loaded && !t.loading {
		return w.loadActive()
	}
	return nil
}

func (w *Widget) openSelected() tea.Cmd {
	t := &w.tabs[w.active]
	if len(t.items) == 0 {
		return nil
	}
	return openCmd(w.runner, t.items[t.cursor].URL)
}

// View renders the tab bar and the active tab's list (or its loading/error/
// empty state), clipped to the given inner size.
func (w *Widget) View(width, height int) string {
	t := w.tabs[w.active]

	var body string
	switch {
	case t.loading && !t.loaded:
		body = "Loading…"
	case t.err != nil:
		body = "Error: " + t.err.Error()
	case len(t.items) == 0:
		body = w.styles.Meta.Render("No pull requests.")
	default:
		body = w.renderItems(t, height-2) // -2 for tab bar + spacing
	}

	return lipgloss.JoinVertical(lipgloss.Left, w.tabBar(width), "", body)
}

func (w *Widget) tabBar(width int) string {
	labels := make([]string, len(w.tabs))
	for i, t := range w.tabs {
		style := w.styles.InactiveTab
		if i == w.active {
			style = w.styles.ActiveTab
		}
		labels[i] = style.Render(t.mode.Label())
	}
	bar := strings.Join(labels, w.styles.Meta.Render(" · "))
	if width > 0 {
		bar = lipgloss.NewStyle().MaxWidth(width).Render(bar)
	}
	return bar
}

func (w *Widget) renderItems(t tabState, maxRows int) string {
	if maxRows < 1 {
		maxRows = len(t.items)
	}
	var b strings.Builder
	for i, item := range t.items {
		if i >= maxRows {
			fmt.Fprintf(&b, "  …and %d more\n", len(t.items)-i)
			break
		}
		cursor := "  "
		title := item.Title
		if i == t.cursor {
			cursor = "> "
			title = w.styles.SelectedItem.Render(item.Title)
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, title)
		fmt.Fprintf(&b, "  %s\n", item.Subtitle)
		fmt.Fprintf(&b, "  %s\n", w.styles.Meta.Render(item.Meta))
	}
	return strings.TrimRight(b.String(), "\n")
}
