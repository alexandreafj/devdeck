// Package calendar implements the Google Calendar widget: this week's events
// grouped by day (Today / Tomorrow / Mon Jan 2), each row showing the time (or
// "All day") and title, with enter opening the meeting link. It depends on
// gcal.EventProvider and exec.CommandRunner (both interfaces), so it is fully
// unit-testable without OAuth or the network.
package calendar

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexandreafj/devdeck/internal/browser"
	"github.com/alexandreafj/devdeck/internal/exec"
	"github.com/alexandreafj/devdeck/internal/gcal"
	"github.com/alexandreafj/devdeck/internal/ui"
)

// Widget is the Google Calendar widget. It uses a pointer receiver so its
// mutable selection/load state is straightforward to reason about.
type Widget struct {
	id       string
	title    string
	provider gcal.EventProvider
	runner   exec.CommandRunner
	now      func() time.Time
	styles   ui.Styles

	// refresh, when > 0, auto-reloads on a timer.
	refresh time.Duration

	sections     []gcal.DaySection
	cursor       int
	loading      bool
	loaded       bool
	err          error
	notConnected bool
}

// New builds a Google Calendar widget wired to the given event provider and
// command runner (the latter used to open meeting links in the browser).
func New(id, title string, provider gcal.EventProvider, runner exec.CommandRunner) *Widget {
	return &Widget{
		id:       id,
		title:    title,
		provider: provider,
		runner:   runner,
		now:      time.Now,
		styles:   ui.DefaultStyles(),
	}
}

// SetRefreshInterval sets the auto-refresh period (<= 0 disables it) and returns
// the widget so app.Build can wire it fluently.
func (w *Widget) SetRefreshInterval(d time.Duration) *Widget {
	w.refresh = d
	return w
}

// ID returns the widget's stable identifier.
func (w *Widget) ID() string { return w.id }

// Title returns the widget's display title.
func (w *Widget) Title() string { return w.title }

// Init begins loading this week's events and arms auto-refresh if configured.
func (w *Widget) Init() tea.Cmd {
	load := w.load()
	if tick := w.maybeTick(); tick != nil {
		return tea.Batch(load, tick)
	}
	return load
}

// Refresh reloads this week's events.
func (w *Widget) Refresh() tea.Cmd { return w.load() }

// load marks the widget loading and returns the fetch command for the current
// week window.
func (w *Widget) load() tea.Cmd {
	w.loading = true
	start, end := gcal.WeekWindow(w.now())
	return fetchCmd(w.id, w.provider, start, end)
}

func (w *Widget) maybeTick() tea.Cmd {
	if w.refresh <= 0 {
		return nil
	}
	return tickCmd(w.id, w.refresh)
}

// tickMsg is the periodic auto-refresh signal for a specific widget.
type tickMsg struct{ widgetID string }

func tickCmd(id string, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{widgetID: id} })
}

// eventsLoadedMsg carries a fetch result back into Update. widgetID lets a
// widget ignore results addressed to a different instance.
type eventsLoadedMsg struct {
	widgetID string
	events   []gcal.Event
	err      error
}

// openedMsg reports the result of opening a meeting link in the browser.
type openedMsg struct{ err error }

func fetchCmd(id string, provider gcal.EventProvider, start, end time.Time) tea.Cmd {
	return func() tea.Msg {
		events, err := provider.FetchEvents(context.Background(), start, end)
		return eventsLoadedMsg{widgetID: id, events: events, err: err}
	}
}

func openCmd(runner exec.CommandRunner, link string) tea.Cmd {
	return func() tea.Msg {
		return openedMsg{err: browser.Open(context.Background(), runner, link)}
	}
}

// Update handles load results, the refresh tick, and key input.
func (w *Widget) Update(msg tea.Msg) (ui.Widget, tea.Cmd) {
	switch msg := msg.(type) {
	case eventsLoadedMsg:
		w.applyLoad(msg)
		return w, nil
	case tickMsg:
		if msg.widgetID != w.id || w.refresh <= 0 {
			return w, nil
		}
		return w, tea.Batch(w.load(), tickCmd(w.id, w.refresh))
	case tea.KeyMsg:
		return w.handleKey(msg)
	}
	return w, nil
}

func (w *Widget) applyLoad(msg eventsLoadedMsg) {
	if msg.widgetID != w.id {
		return
	}
	w.loading = false
	w.loaded = true
	switch {
	case errors.Is(msg.err, gcal.ErrNotConnected):
		w.notConnected = true
		w.err = nil
		w.sections = nil
	case msg.err != nil:
		w.err = msg.err
		w.notConnected = false
		w.sections = nil
	default:
		w.err = nil
		w.notConnected = false
		w.sections = gcal.GroupByDay(msg.events, w.now())
	}
	if n := w.eventCount(); w.cursor >= n {
		w.cursor = max(0, n-1)
	}
}

func (w *Widget) handleKey(msg tea.KeyMsg) (ui.Widget, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		w.moveCursor(-1)
	case "down", "j":
		w.moveCursor(1)
	case "enter":
		return w, w.openSelected()
	}
	return w, nil
}

func (w *Widget) moveCursor(delta int) {
	n := w.eventCount()
	if n == 0 {
		w.cursor = 0
		return
	}
	w.cursor = min(max(w.cursor+delta, 0), n-1)
}

// events flattens the day sections into a single ordered slice, matching the
// flat cursor index used for selection.
func (w *Widget) events() []gcal.Event {
	var out []gcal.Event
	for _, s := range w.sections {
		out = append(out, s.Events...)
	}
	return out
}

func (w *Widget) eventCount() int {
	n := 0
	for _, s := range w.sections {
		n += len(s.Events)
	}
	return n
}

func (w *Widget) openSelected() tea.Cmd {
	evs := w.events()
	if len(evs) == 0 {
		return nil
	}
	c := w.cursor
	if c >= len(evs) {
		c = len(evs) - 1
	}
	link := evs[c].Link()
	if link == "" {
		return nil
	}
	return openCmd(w.runner, link)
}

// View renders the widget's body, clipped to the given inner height.
func (w *Widget) View(_, height int) string {
	switch {
	case w.loading && !w.loaded:
		return "Loading…"
	case w.notConnected:
		return w.styles.Meta.Render("Not connected.") + "\n" +
			"Run: " + w.styles.SelectedItem.Render("devdeck auth google")
	case w.err != nil:
		return "Error: " + w.err.Error()
	case w.eventCount() == 0:
		return w.styles.Meta.Render("No meetings this week.")
	default:
		return w.renderSections(height)
	}
}

func (w *Widget) renderSections(maxRows int) string {
	var b strings.Builder
	idx, rows := 0, 0
	for _, sec := range w.sections {
		if maxRows > 0 && rows >= maxRows {
			break
		}
		fmt.Fprintf(&b, "%s\n", w.styles.Title.Render(sec.Label))
		rows++
		for _, ev := range sec.Events {
			if maxRows > 0 && rows >= maxRows {
				break
			}
			prefix := "  "
			line := fmt.Sprintf("%-7s %s", ev.TimeLabel(), ev.Summary)
			if idx == w.cursor {
				prefix = "> "
				line = w.styles.SelectedItem.Render(line)
			}
			fmt.Fprintf(&b, "%s%s\n", prefix, line)
			idx++
			rows++
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// compile-time check: the widget satisfies the dashboard's extension point.
var _ ui.Widget = (*Widget)(nil)
