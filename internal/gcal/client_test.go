package gcal

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEventsURLParams(t *testing.T) {
	start := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	raw := eventsURL(start, end)

	if !strings.HasPrefix(raw, eventsEndpoint+"?") {
		t.Fatalf("URL %q should target the events endpoint", raw)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	checks := map[string]string{
		"timeMin":      start.Format(time.RFC3339),
		"timeMax":      end.Format(time.RFC3339),
		"singleEvents": "true",
		"orderBy":      "startTime",
	}
	for k, want := range checks {
		if got := q.Get(k); got != want {
			t.Errorf("query %s = %q, want %q", k, got, want)
		}
	}
}

// fakeDoer returns a scripted HTTP response (and records the request URL).
type fakeDoer struct {
	status   int
	body     string
	err      error
	gotURL   string
	gotQuery url.Values
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.gotURL = req.URL.String()
	f.gotQuery = req.URL.Query()
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{
		StatusCode: f.status,
		Status:     http.StatusText(f.status),
		Body:       io.NopCloser(strings.NewReader(f.body)),
	}, nil
}

func TestFetchEventsDecodesOK(t *testing.T) {
	doer := &fakeDoer{status: http.StatusOK, body: `{"items":[{"status":"confirmed","summary":"Standup","start":{"dateTime":"2026-06-12T09:00:00Z"}}]}`}
	events, err := fetchEvents(context.Background(), doer, time.Now(), time.Now().Add(time.Hour), time.UTC)
	if err != nil {
		t.Fatalf("fetchEvents: %v", err)
	}
	if len(events) != 1 || events[0].Summary != "Standup" {
		t.Errorf("events = %+v, want one Standup", events)
	}
	if doer.gotQuery.Get("singleEvents") != "true" {
		t.Error("fetchEvents should request expanded single events")
	}
}

func TestFetchEventsUnauthorizedMapsToNotConnected(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		doer := &fakeDoer{status: status, body: "nope"}
		_, err := fetchEvents(context.Background(), doer, time.Now(), time.Now().Add(time.Hour), time.UTC)
		if !errors.Is(err, ErrNotConnected) {
			t.Errorf("status %d: err = %v, want ErrNotConnected", status, err)
		}
	}
}

func TestFetchEventsOtherStatusErrors(t *testing.T) {
	doer := &fakeDoer{status: http.StatusInternalServerError, body: "boom"}
	if _, err := fetchEvents(context.Background(), doer, time.Now(), time.Now().Add(time.Hour), time.UTC); err == nil {
		t.Error("a 500 should surface an error")
	}
}

func TestFetchEventsDoerError(t *testing.T) {
	doer := &fakeDoer{err: errors.New("network down")}
	if _, err := fetchEvents(context.Background(), doer, time.Now(), time.Now().Add(time.Hour), time.UTC); err == nil {
		t.Error("a transport error should propagate")
	}
}

func TestFetchEventsNotConnectedWithoutCredentials(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config")) // empty: no credentials
	c := NewClient("", "")
	_, err := c.FetchEvents(context.Background(), time.Now(), time.Now().Add(time.Hour))
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("FetchEvents without credentials = %v, want ErrNotConnected", err)
	}
}

func TestFetchEventsUsesConfiguredCredentialsPath(t *testing.T) {
	// A configured (but absent) credentials path is honored and yields
	// ErrNotConnected, independent of the default config dir.
	dir := t.TempDir()
	c := NewClient(filepath.Join(dir, "mine.json"), filepath.Join(dir, "tok.json"))
	_, err := c.FetchEvents(context.Background(), time.Now(), time.Now().Add(time.Hour))
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("FetchEvents with a missing configured credentials path = %v, want ErrNotConnected", err)
	}
}
