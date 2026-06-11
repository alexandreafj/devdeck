package githubprs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexandreafj/devdeck/internal/exec/exectest"
	"github.com/alexandreafj/devdeck/internal/github"
	"github.com/alexandreafj/devdeck/internal/ui"
)

// compile-time check: the widget satisfies the dashboard's extension point.
var _ ui.Widget = (*Widget)(nil)

var fixedNow = time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

type fakeProvider struct {
	prs   map[github.Mode][]github.PR
	errs  map[github.Mode]error
	calls []github.Mode
}

func (f *fakeProvider) FetchPRs(_ context.Context, mode github.Mode) ([]github.PR, error) {
	f.calls = append(f.calls, mode)
	return f.prs[mode], f.errs[mode]
}

func pr(repo, title, author string) github.PR {
	return github.PR{
		Title:      title,
		URL:        "https://github.com/" + repo + "/pull/1",
		Repository: github.Repository{NameWithOwner: repo},
		Author:     github.Author{Login: author},
		CreatedAt:  fixedNow.Add(-24 * time.Hour),
	}
}

// newWidget builds a widget wired to the given provider/runner with a fixed
// clock for deterministic relative-time output.
func newWidget(p github.PRProvider, r *exectest.FakeRunner) *Widget {
	w := New("prs", "PRs", p, r, github.Modes)
	w.now = func() time.Time { return fixedNow }
	return w
}

// runCmd executes a tea.Cmd and returns its message.
func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a non-nil command")
	}
	return cmd()
}

func TestInitFetchesActiveModeAndStoresItems(t *testing.T) {
	provider := &fakeProvider{prs: map[github.Mode][]github.PR{
		github.Authored: {pr("acme/api", "Fix mask", "bruno"), pr("acme/web", "Add page", "ana")},
	}}
	w := newWidget(provider, &exectest.FakeRunner{})

	msg := runCmd(t, w.Init())
	w.Update(msg)

	if got := provider.calls; len(got) != 1 || got[0] != github.Authored {
		t.Fatalf("provider calls = %v, want [Authored]", got)
	}
	view := w.View(80, 20)
	if !strings.Contains(view, "acme/api") || !strings.Contains(view, "Fix mask") {
		t.Errorf("view missing loaded PR content:\n%s", view)
	}
}

func TestInitSetsLoadingState(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{})
	w.Init() // sets loading, returns a cmd we don't run yet
	if !strings.Contains(w.View(80, 20), "Loading") {
		t.Errorf("view should show a loading indicator before results arrive:\n%s", w.View(80, 20))
	}
}

func TestErrorStateRendered(t *testing.T) {
	provider := &fakeProvider{errs: map[github.Mode]error{
		github.Authored: errors.New("gh not authenticated"),
	}}
	w := newWidget(provider, &exectest.FakeRunner{})

	w.Update(runCmd(t, w.Init()))

	view := w.View(80, 20)
	if !strings.Contains(view, "gh not authenticated") {
		t.Errorf("view should surface the fetch error:\n%s", view)
	}
}

func TestEmptyStateRendered(t *testing.T) {
	provider := &fakeProvider{prs: map[github.Mode][]github.PR{github.Authored: {}}}
	w := newWidget(provider, &exectest.FakeRunner{})

	w.Update(runCmd(t, w.Init()))

	if !strings.Contains(w.View(80, 20), "No pull requests") {
		t.Errorf("empty tab should show a 'No pull requests' message:\n%s", w.View(80, 20))
	}
}

func TestCursorStaysInBounds(t *testing.T) {
	provider := &fakeProvider{prs: map[github.Mode][]github.PR{
		github.Authored: {pr("a/1", "t1", "x"), pr("a/2", "t2", "y"), pr("a/3", "t3", "z")},
	}}
	w := newWidget(provider, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))

	for i := 0; i < 5; i++ {
		w.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if got := w.tabs[w.active].cursor; got != 2 {
		t.Errorf("cursor after many downs = %d, want clamped to 2", got)
	}
	for i := 0; i < 5; i++ {
		w.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	if got := w.tabs[w.active].cursor; got != 0 {
		t.Errorf("cursor after many ups = %d, want clamped to 0", got)
	}
}

func TestSwitchTabFetchesNewModeOnlyOnce(t *testing.T) {
	provider := &fakeProvider{prs: map[github.Mode][]github.PR{
		github.Authored:        {pr("a/1", "t1", "x")},
		github.ReviewRequested: {pr("b/2", "t2", "y")},
	}}
	w := newWidget(provider, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init())) // Authored loaded

	// Switch right to ReviewRequested → should fetch it.
	_, cmd := w.Update(tea.KeyMsg{Type: tea.KeyRight})
	if w.active != 1 {
		t.Fatalf("active tab = %d, want 1 after right", w.active)
	}
	w.Update(runCmd(t, cmd)) // load ReviewRequested

	// Switch away and back → already loaded, no new fetch.
	w.Update(tea.KeyMsg{Type: tea.KeyLeft})
	_, cmd = w.Update(tea.KeyMsg{Type: tea.KeyRight})
	if cmd != nil {
		t.Error("returning to an already-loaded tab should not refetch")
	}

	authored, review := 0, 0
	for _, m := range provider.calls {
		switch m {
		case github.Authored:
			authored++
		case github.ReviewRequested:
			review++
		}
	}
	if authored != 1 || review != 1 {
		t.Errorf("fetch counts authored=%d review=%d, want 1 and 1", authored, review)
	}
}

func TestEnterOpensSelectedPR(t *testing.T) {
	provider := &fakeProvider{prs: map[github.Mode][]github.PR{
		github.Authored: {pr("acme/api", "Fix mask", "bruno")},
	}}
	runner := &exectest.FakeRunner{}
	w := newWidget(provider, runner)
	w.Update(runCmd(t, w.Init()))

	_, cmd := w.Update(tea.KeyMsg{Type: tea.KeyEnter})
	runCmd(t, cmd) // executes the open command

	call, ok := runner.LastCall()
	if !ok || call.Name != "gh" {
		t.Fatalf("expected gh invocation, got %+v (ok=%v)", call, ok)
	}
	joined := strings.Join(call.Args, " ")
	if !strings.Contains(joined, "pr view") || !strings.Contains(joined, "--web") {
		t.Errorf("open command args = %q, want `pr view <url> --web`", joined)
	}
	if !strings.Contains(joined, "acme/api/pull/1") {
		t.Errorf("open command should target the selected PR URL, got %q", joined)
	}
}

func TestEnterWithNoItemsIsNoop(t *testing.T) {
	provider := &fakeProvider{prs: map[github.Mode][]github.PR{github.Authored: {}}}
	w := newWidget(provider, &exectest.FakeRunner{})
	w.Update(runCmd(t, w.Init()))

	if _, cmd := w.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
		t.Error("enter on an empty tab should be a no-op (nil cmd)")
	}
}

func TestIgnoresMessagesForOtherWidget(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{})
	w.Update(prsLoadedMsg{widgetID: "someone-else", mode: github.Authored, items: nil})
	if w.tabs[0].loaded {
		t.Error("a load message addressed to another widget must be ignored")
	}
}

func TestIDAndTitle(t *testing.T) {
	w := newWidget(&fakeProvider{}, &exectest.FakeRunner{})
	if w.ID() != "prs" || w.Title() != "PRs" {
		t.Errorf("ID=%q Title=%q, want prs/PRs", w.ID(), w.Title())
	}
}

func TestNewDefaultsToAllModesWhenNoneGiven(t *testing.T) {
	w := New("prs", "PRs", &fakeProvider{}, &exectest.FakeRunner{}, nil)
	if len(w.tabs) != len(github.Modes) {
		t.Errorf("got %d tabs, want %d (default all modes)", len(w.tabs), len(github.Modes))
	}
}
