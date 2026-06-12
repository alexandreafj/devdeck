package calendar

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexandreafj/devdeck/internal/exec/exectest"
	"github.com/alexandreafj/devdeck/internal/gcal"
	"github.com/alexandreafj/devdeck/internal/ui"
)

// compile-time check: the widget satisfies the dashboard's extension point.
var _ ui.Widget = (*Widget)(nil)

// fixedNow is a Friday so day labels are deterministic.
var fixedNow = time.Date(2026, 6, 12, 8, 0, 0, 0, time.UTC)

type fakeProvider struct {
	events []gcal.Event
	err    error
	calls  int
}

func (f *fakeProvider) FetchEvents(_ context.Context, _, _ time.Time) ([]gcal.Event, error) {
	f.calls++
	return f.events, f.err
}

func ev(day, hour, min int, summary, meet, html string, allDay bool) gcal.Event {
	return gcal.Event{
		Summary:  summary,
		Start:    time.Date(2026, 6, day, hour, min, 0, 0, time.UTC),
		AllDay:   allDay,
		MeetLink: meet,
		HTMLLink: html,
	}
}

func newWidget(p gcal.EventProvider, r *exectest.FakeRunner) *Widget {
	w := New("cal", "Google Calendar", p, r)
	w.now = func() time.Time { return fixedNow }
	return w
}

func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a non-nil command")
	}
	return cmd()
}

func TestInitFetchesAndGroupsByDay(t *testing.T) {
	provider := &fakeProvider{events: []gcal.Event{
		ev(12, 9, 0, "Standup", "https://meet.google.com/a", "", false),
		ev(12, 14, 0, "1:1 Ana", "", "https://cal/1on1", false),
		ev(13, 0, 0, "Offsite", "", "https://cal/off", true),
	}}
	w := newWidget(provider, &exectest.FakeRunner{})

	w.Update(runCmd(t, w.Init()))

	if provider.calls != 1 {
		t.Fatalf("FetchEvents calls = %d, want 1", provider.calls)
	}
	view := w.View(80, 20)
	for _, want := range []string{"Today", "Standup", "09:00", "1:1 Ana", "Tomorrow", "All day", "Offsite"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}

func TestInitShowsLoading(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{})
	w.Init() // sets loading, command not run
	if !strings.Contains(w.View(80, 20), "Loading") {
		t.Errorf("should show loading before results:\n%s", w.View(80, 20))
	}
}

func TestNotConnectedState(t *testing.T) {
	provider := &fakeProvider{err: gcal.ErrNotConnected}
	w := newWidget(provider, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))

	view := w.View(80, 20)
	if !strings.Contains(view, "Not connected") || !strings.Contains(view, "devdeck auth google") {
		t.Errorf("not-connected view should prompt to authorize:\n%s", view)
	}
}

func TestErrorState(t *testing.T) {
	provider := &fakeProvider{err: errors.New("calendar API: 500")}
	w := newWidget(provider, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))
	if !strings.Contains(w.View(80, 20), "calendar API: 500") {
		t.Errorf("error should be surfaced:\n%s", w.View(80, 20))
	}
}

func TestEmptyState(t *testing.T) {
	w := newWidget(&fakeProvider{events: nil}, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))
	if !strings.Contains(w.View(80, 20), "No meetings this week") {
		t.Errorf("empty week should show a message:\n%s", w.View(80, 20))
	}
}

func TestCursorStaysInBoundsAcrossSections(t *testing.T) {
	provider := &fakeProvider{events: []gcal.Event{
		ev(12, 9, 0, "A", "", "https://x/a", false),
		ev(13, 9, 0, "B", "", "https://x/b", false),
		ev(14, 9, 0, "C", "", "https://x/c", false),
	}}
	w := newWidget(provider, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))

	for i := 0; i < 5; i++ {
		w.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if w.cursor != 2 {
		t.Errorf("cursor after many downs = %d, want clamped to 2", w.cursor)
	}
	for i := 0; i < 5; i++ {
		w.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	if w.cursor != 0 {
		t.Errorf("cursor after many ups = %d, want clamped to 0", w.cursor)
	}
}

func TestEnterOpensSelectedMeetingLink(t *testing.T) {
	provider := &fakeProvider{events: []gcal.Event{
		ev(12, 9, 0, "Standup", "https://meet.google.com/abc", "https://cal/std", false),
		ev(12, 10, 0, "Review", "", "https://cal/rev", false),
	}}
	runner := &exectest.FakeRunner{}
	w := newWidget(provider, runner)
	w.Update(runCmd(t, w.Init()))

	// First event: opens the Meet link.
	_, cmd := w.Update(tea.KeyMsg{Type: tea.KeyEnter})
	runCmd(t, cmd)
	call, ok := runner.LastCall()
	if !ok || !strings.Contains(strings.Join(call.Args, " "), "meet.google.com/abc") {
		t.Errorf("enter should open the Meet link, got %+v", call)
	}

	// Second event has no Meet link → opens the calendar page (htmlLink).
	w.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd = w.Update(tea.KeyMsg{Type: tea.KeyEnter})
	runCmd(t, cmd)
	call, _ = runner.LastCall()
	if !strings.Contains(strings.Join(call.Args, " "), "cal/rev") {
		t.Errorf("enter should fall back to htmlLink, got %+v", call)
	}
}

func TestEnterWithNoEventsIsNoop(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))
	if _, cmd := w.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Error("enter with no events should be a no-op")
	}
}

func TestMoveCursorWithNoEventsStaysZero(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))
	w.Update(tea.KeyMsg{Type: tea.KeyDown})
	w.Update(tea.KeyMsg{Type: tea.KeyUp})
	if w.cursor != 0 {
		t.Errorf("cursor with no events = %d, want 0", w.cursor)
	}
}

func TestEnterWithNoLinkIsNoop(t *testing.T) {
	provider := &fakeProvider{events: []gcal.Event{ev(12, 9, 0, "Blocked", "", "", false)}}
	w := newWidget(provider, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))
	if _, cmd := w.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Error("enter on an event with no link should be a no-op")
	}
}

func TestIgnoresMessagesForOtherWidget(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{})
	w.Update(eventsLoadedMsg{widgetID: "someone-else", events: []gcal.Event{ev(12, 9, 0, "X", "", "https://x", false)}})
	if w.loaded {
		t.Error("a load addressed to another widget must be ignored")
	}
}

func TestIDAndTitle(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{})
	if w.ID() != "cal" || w.Title() != "Google Calendar" {
		t.Errorf("ID=%q Title=%q, want cal/Google Calendar", w.ID(), w.Title())
	}
}

func TestRefreshReloads(t *testing.T) {
	provider := &fakeProvider{events: []gcal.Event{ev(12, 9, 0, "A", "", "https://x", false)}}
	w := newWidget(provider, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))
	runCmd(t, w.Refresh())
	if provider.calls != 2 {
		t.Errorf("Refresh should refetch, calls = %d want 2", provider.calls)
	}
}

func TestMaybeTickHonoursInterval(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{})
	if w.maybeTick() != nil {
		t.Error("no interval should yield no tick")
	}
	w.SetRefreshInterval(time.Minute)
	if w.maybeTick() == nil {
		t.Error("a positive interval should yield a tick")
	}
}

func TestAutoRefreshTickReloadsAndRearms(t *testing.T) {
	provider := &fakeProvider{events: []gcal.Event{ev(12, 9, 0, "A", "", "https://x", false)}}
	w := newWidget(provider, &exectest.FakeRunner{}).SetRefreshInterval(time.Minute)
	w.now = func() time.Time { return fixedNow }

	_, cmd := w.Update(tickMsg{widgetID: w.id})
	if !w.loading {
		t.Error("a tick should reload (loading=true)")
	}
	if cmd == nil {
		t.Fatal("a tick should batch reload + re-arm")
	}
	if _, ok := cmd().(tea.BatchMsg); !ok {
		t.Errorf("tick command should be a batch, got %T", cmd())
	}
}

func TestTickIgnoredForOtherWidgetAndWhenDisabled(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{}).SetRefreshInterval(time.Minute)
	w.now = func() time.Time { return fixedNow }
	if _, cmd := w.Update(tickMsg{widgetID: "other"}); cmd != nil {
		t.Error("a tick for another widget must be ignored")
	}
	if w.loading {
		t.Error("a tick for another widget must not reload")
	}

	noRefresh := newWidget(&fakeProvider{}, &exectest.FakeRunner{})
	if _, cmd := noRefresh.Update(tickMsg{widgetID: noRefresh.id}); cmd != nil {
		t.Error("with auto-refresh disabled, a tick is a no-op")
	}
}

func TestInitArmsAutoRefreshWhenConfigured(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{}).SetRefreshInterval(time.Minute)
	w.now = func() time.Time { return fixedNow }
	cmd := w.Init()
	if cmd == nil {
		t.Fatal("Init should return a command")
	}
	if _, ok := cmd().(tea.BatchMsg); !ok {
		t.Errorf("Init with auto-refresh should batch load + tick, got %T", cmd())
	}
}

func TestViewClipsToHeight(t *testing.T) {
	provider := &fakeProvider{events: []gcal.Event{
		ev(12, 9, 0, "A", "", "https://x/a", false),
		ev(12, 10, 0, "B", "", "https://x/b", false),
		ev(12, 11, 0, "C", "", "https://x/c", false),
	}}
	w := newWidget(provider, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))

	clipped := w.View(80, 2) // header + 1 event row
	if strings.Contains(clipped, "C") {
		t.Errorf("view with height 2 should not include the third event:\n%s", clipped)
	}
}
