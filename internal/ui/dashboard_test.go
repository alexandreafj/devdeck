package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// fakeWidget is a pointer-based Widget that records how the dashboard drives it.
// Update returns itself so the dashboard keeps the same instance the test holds.
type fakeWidget struct {
	id        string
	inits     int
	updates   int
	refreshes int
	lastMsg   tea.Msg
	capturing bool
}

func (f *fakeWidget) ID() string           { return f.id }
func (f *fakeWidget) Title() string        { return f.id }
func (f *fakeWidget) Init() tea.Cmd        { f.inits++; return nil }
func (f *fakeWidget) View(int, int) string { return f.id }
func (f *fakeWidget) Refresh() tea.Cmd     { f.refreshes++; return nil }
func (f *fakeWidget) CapturingInput() bool { return f.capturing }
func (f *fakeWidget) Update(msg tea.Msg) (Widget, tea.Cmd) {
	f.updates++
	f.lastMsg = msg
	return f, nil
}

type pingMsg struct{}

func runeKey(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func update(t *testing.T, d Dashboard, msg tea.Msg) (Dashboard, tea.Cmd) {
	t.Helper()
	m, cmd := d.Update(msg)
	got, ok := m.(Dashboard)
	if !ok {
		t.Fatalf("Update returned %T, want Dashboard", m)
	}
	return got, cmd
}

func TestDashboardInitCallsEachWidgetInit(t *testing.T) {
	w1, w2 := &fakeWidget{id: "a"}, &fakeWidget{id: "b"}
	NewDashboard(w1, w2).Init()
	if w1.inits != 1 || w2.inits != 1 {
		t.Errorf("inits = (%d, %d), want (1, 1)", w1.inits, w2.inits)
	}
}

func TestDashboardTabForwardsToFocusedWidget(t *testing.T) {
	w0, w1 := &fakeWidget{id: "a"}, &fakeWidget{id: "b"}
	d := NewDashboard(w0, w1) // focus starts on w0
	d, _ = update(t, d, tea.KeyMsg{Type: tea.KeyTab})

	if d.focus != 0 {
		t.Errorf("tab must not change widget focus, got %d want 0", d.focus)
	}
	if w0.updates != 1 || w1.updates != 0 {
		t.Errorf("tab must forward to the focused widget only, updates = (%d, %d)", w0.updates, w1.updates)
	}
}

func TestDashboardShiftTabSwitchesWidget(t *testing.T) {
	d := NewDashboard(&fakeWidget{id: "a"}, &fakeWidget{id: "b"})
	d, _ = update(t, d, tea.KeyMsg{Type: tea.KeyShiftTab})
	if d.focus != 1 {
		t.Errorf("after shift+tab, focus = %d, want 1", d.focus)
	}
	d, _ = update(t, d, tea.KeyMsg{Type: tea.KeyShiftTab})
	if d.focus != 0 {
		t.Errorf("after second shift+tab, focus = %d, want 0 (wraps)", d.focus)
	}
}

func TestDashboardForwardsAllKeysWhileWidgetCaptures(t *testing.T) {
	w := &fakeWidget{id: "a", capturing: true}
	d := NewDashboard(w)

	// 'q' would normally quit; while the widget captures input it must reach the
	// widget instead so it can be typed into the filter.
	_, cmd := update(t, d, runeKey("q"))
	if cmd != nil {
		t.Error("while capturing, 'q' must not quit (expected nil cmd)")
	}
	if w.updates != 1 {
		t.Errorf("while capturing, keys must forward to the widget, updates = %d want 1", w.updates)
	}
}

func TestDashboardNumberKeyJumpsToWidget(t *testing.T) {
	d := NewDashboard(&fakeWidget{id: "a"}, &fakeWidget{id: "b"}, &fakeWidget{id: "c"})
	d, _ = update(t, d, runeKey("3"))
	if d.focus != 2 {
		t.Errorf("after '3', focus = %d, want 2", d.focus)
	}
	// Out-of-range number is ignored, not a crash.
	d, _ = update(t, d, runeKey("9"))
	if d.focus != 2 {
		t.Errorf("after '9' (out of range), focus = %d, want unchanged 2", d.focus)
	}
}

func TestDashboardQuitKeyReturnsQuitCommand(t *testing.T) {
	d := NewDashboard(&fakeWidget{id: "a"})
	_, cmd := update(t, d, runeKey("q"))
	if cmd == nil {
		t.Fatal("quit returned nil cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("quit cmd produced %T, want tea.QuitMsg", cmd())
	}
}

func TestDashboardForwardsOtherKeysToFocusedWidgetOnly(t *testing.T) {
	w0, w1 := &fakeWidget{id: "a"}, &fakeWidget{id: "b"}
	d := NewDashboard(w0, w1) // focus starts on w0
	update(t, d, runeKey("j"))
	if w0.updates != 1 {
		t.Errorf("focused widget updates = %d, want 1", w0.updates)
	}
	if w1.updates != 0 {
		t.Errorf("unfocused widget updates = %d, want 0", w1.updates)
	}
}

func TestDashboardBroadcastsNonKeyMessagesToAllWidgets(t *testing.T) {
	w0, w1 := &fakeWidget{id: "a"}, &fakeWidget{id: "b"}
	d := NewDashboard(w0, w1)
	update(t, d, pingMsg{})
	if w0.updates != 1 || w1.updates != 1 {
		t.Errorf("updates = (%d, %d), want (1, 1) — non-key msgs broadcast", w0.updates, w1.updates)
	}
}

func TestDashboardRefreshKeyRefreshesFocusedWidgetOnly(t *testing.T) {
	w0, w1 := &fakeWidget{id: "a"}, &fakeWidget{id: "b"}
	d := NewDashboard(w0, w1)
	update(t, d, runeKey("r"))
	if w0.refreshes != 1 || w1.refreshes != 0 {
		t.Errorf("refreshes = (%d, %d), want (1, 0)", w0.refreshes, w1.refreshes)
	}
}

func TestDashboardRefreshAllKeyRefreshesEveryWidget(t *testing.T) {
	w0, w1 := &fakeWidget{id: "a"}, &fakeWidget{id: "b"}
	d := NewDashboard(w0, w1)
	update(t, d, runeKey("R"))
	if w0.refreshes != 1 || w1.refreshes != 1 {
		t.Errorf("refreshes = (%d, %d), want (1, 1)", w0.refreshes, w1.refreshes)
	}
}

func TestDashboardViewRendersWidgetTitlesAfterSizing(t *testing.T) {
	d := NewDashboard(&fakeWidget{id: "alpha"}, &fakeWidget{id: "beta"})
	d, _ = update(t, d, tea.WindowSizeMsg{Width: 120, Height: 40})
	view := d.View()
	if !strings.Contains(view, "alpha") || !strings.Contains(view, "beta") {
		t.Errorf("view should contain both widget bodies, got:\n%s", view)
	}
}

func TestDashboardEmptyIsSafe(t *testing.T) {
	d := NewDashboard()
	d, _ = update(t, d, tea.KeyMsg{Type: tea.KeyTab})
	if d.focus != 0 {
		t.Errorf("empty dashboard focus = %d, want 0", d.focus)
	}
	if d.View() == "" {
		t.Error("empty dashboard View() should render a placeholder, got empty string")
	}
}
