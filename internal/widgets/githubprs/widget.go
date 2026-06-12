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
	"github.com/alexandreafj/devdeck/internal/fuzzy"
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

	// refresh, when > 0, auto-reloads the active tab on a timer.
	refresh time.Duration

	// filtering is true while the user is typing a filter; filterQuery is the
	// current fuzzy query applied to the active tab's items ("" shows all).
	filtering   bool
	filterQuery string
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

// SetRefreshInterval sets the auto-refresh period. A value <= 0 disables
// auto-refresh. It returns the widget so app.Build can wire it fluently.
func (w *Widget) SetRefreshInterval(d time.Duration) *Widget {
	w.refresh = d
	return w
}

// ID returns the widget's stable identifier.
func (w *Widget) ID() string { return w.id }

// Title returns the widget's display title.
func (w *Widget) Title() string { return w.title }

// CapturingInput reports whether the widget is consuming all key input (the
// filter is open). The dashboard checks this so typed keys reach the filter
// instead of triggering global bindings.
func (w *Widget) CapturingInput() bool { return w.filtering }

// Init begins loading the active tab and arms auto-refresh if configured.
func (w *Widget) Init() tea.Cmd {
	load := w.loadActive()
	if tick := w.maybeTick(); tick != nil {
		return tea.Batch(load, tick)
	}
	return load
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

// maybeTick returns the periodic refresh command, or nil when auto-refresh is
// disabled.
func (w *Widget) maybeTick() tea.Cmd {
	if w.refresh <= 0 {
		return nil
	}
	return tickCmd(w.id, w.refresh)
}

// tickMsg is the periodic auto-refresh signal for a specific widget. widgetID
// lets a widget ignore ticks meant for another (the dashboard broadcasts
// non-key messages to every widget).
type tickMsg struct{ widgetID string }

func tickCmd(id string, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{widgetID: id} })
}

// prsLoadedMsg carries the result of a fetch back into Update. widgetID lets a
// widget ignore results addressed to a different instance.
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

// Update handles load results, the refresh tick, and key input.
func (w *Widget) Update(msg tea.Msg) (ui.Widget, tea.Cmd) {
	switch msg := msg.(type) {
	case prsLoadedMsg:
		w.applyLoad(msg)
		return w, nil
	case tickMsg:
		if msg.widgetID != w.id || w.refresh <= 0 {
			return w, nil
		}
		// Reload the active tab and re-arm the next tick.
		return w, tea.Batch(w.loadActive(), tickCmd(w.id, w.refresh))
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
	if w.filtering {
		return w.handleFilterKey(msg)
	}
	switch msg.String() {
	case "up", "k":
		w.moveCursor(-1)
	case "down", "j":
		w.moveCursor(1)
	case "left", "h":
		return w, w.switchTab(-1)
	case "right", "l", "tab":
		return w, w.switchTab(1)
	case "/":
		w.filtering = true
	case "enter":
		return w, w.openSelected()
	}
	return w, nil
}

// handleFilterKey edits the filter query while the filter is open. esc clears
// and closes it; enter closes it but keeps the query (so the list stays
// filtered while you navigate); backspace narrows, and emptying the query
// restores the full list.
func (w *Widget) handleFilterKey(msg tea.KeyMsg) (ui.Widget, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		w.filtering = false
		w.filterQuery = ""
	case tea.KeyEnter:
		w.filtering = false
	case tea.KeyBackspace:
		w.filterQuery = trimLastRune(w.filterQuery)
	case tea.KeySpace:
		w.filterQuery += " "
	case tea.KeyRunes:
		w.filterQuery += string(msg.Runes)
	default:
		return w, nil
	}
	w.tabs[w.active].cursor = 0
	return w, nil
}

func trimLastRune(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}

func (w *Widget) moveCursor(delta int) {
	t := &w.tabs[w.active]
	n := len(w.visible(*t))
	if n == 0 {
		t.cursor = 0
		return
	}
	t.cursor = min(max(t.cursor+delta, 0), n-1)
}

// switchTab moves to an adjacent tab (wrapping) and lazily loads it the first
// time it is shown. The filter query carries across tabs.
func (w *Widget) switchTab(delta int) tea.Cmd {
	w.active = (w.active + delta + len(w.tabs)) % len(w.tabs)
	if t := &w.tabs[w.active]; !t.loaded && !t.loading {
		return w.loadActive()
	}
	return nil
}

func (w *Widget) openSelected() tea.Cmd {
	vis := w.visible(w.tabs[w.active])
	if len(vis) == 0 {
		return nil
	}
	c := w.tabs[w.active].cursor
	if c >= len(vis) {
		c = len(vis) - 1
	}
	return openCmd(w.runner, vis[c].URL)
}

// visible applies the active filter to a tab's items. An empty query returns
// all items unchanged (no allocation).
func (w *Widget) visible(t tabState) []domain.Item {
	if w.filterQuery == "" {
		return t.items
	}
	out := make([]domain.Item, 0, len(t.items))
	for _, it := range t.items {
		if fuzzy.Match(w.filterQuery, it.Title+" "+it.Subtitle) {
			out = append(out, it)
		}
	}
	return out
}

// View renders the tab bar, an optional filter line, and the active tab's list
// (or its loading/error/empty/no-match state), clipped to the given inner size.
func (w *Widget) View(width, height int) string {
	t := w.tabs[w.active]
	vis := w.visible(t)

	header := w.tabBar(width)
	reserved := 2 // tab bar + spacing
	if w.filtering || w.filterQuery != "" {
		header = lipgloss.JoinVertical(lipgloss.Left, header, w.filterLine(t, vis))
		reserved = 3
	}

	var body string
	switch {
	case t.loading && !t.loaded:
		body = "Loading…"
	case t.err != nil:
		body = "Error: " + t.err.Error()
	case len(t.items) == 0:
		body = w.styles.Meta.Render("No pull requests.")
	case len(vis) == 0:
		body = w.styles.Meta.Render("No matches.")
	default:
		body = w.renderItems(vis, t.cursor, height-reserved)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, "", body)
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

// filterLine shows the current query, a caret while typing, and a match count.
func (w *Widget) filterLine(t tabState, vis []domain.Item) string {
	caret := ""
	if w.filtering {
		caret = "▌"
	}
	return w.styles.Meta.Render(fmt.Sprintf("/%s%s  (%d/%d)", w.filterQuery, caret, len(vis), len(t.items)))
}

func (w *Widget) renderItems(items []domain.Item, cursor, maxRows int) string {
	if maxRows < 1 {
		maxRows = len(items)
	}
	var b strings.Builder
	for i, item := range items {
		if i >= maxRows {
			fmt.Fprintf(&b, "  …and %d more\n", len(items)-i)
			break
		}
		prefix := "  "
		title := item.Title
		if i == cursor {
			prefix = "> "
			title = w.styles.SelectedItem.Render(item.Title)
		}
		fmt.Fprintf(&b, "%s%s\n", prefix, title)
		fmt.Fprintf(&b, "  %s\n", item.Subtitle)
		fmt.Fprintf(&b, "  %s\n", w.styles.Meta.Render(item.Meta))
	}
	return strings.TrimRight(b.String(), "\n")
}
