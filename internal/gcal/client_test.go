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
	raw := eventsURL("", start, end) // empty → primary

	if !strings.HasPrefix(raw, calendarsBase+"primary/events?") {
		t.Fatalf("URL %q should target the primary calendar's events endpoint", raw)
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

func TestEventsURLEscapesCalendarID(t *testing.T) {
	raw := eventsURL("me@gmail.com", time.Now(), time.Now().Add(time.Hour))
	if !strings.Contains(raw, calendarsBase+"me%40gmail.com/events") {
		t.Errorf("calendar id should be path-escaped in %q", raw)
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
	events, err := fetchEvents(context.Background(), doer, "", time.Now(), time.Now().Add(time.Hour), time.UTC)
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
	doer := &fakeDoer{status: http.StatusUnauthorized, body: "nope"}
	_, err := fetchEvents(context.Background(), doer, "", time.Now(), time.Now().Add(time.Hour), time.UTC)
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("401 err = %v, want ErrNotConnected", err)
	}
}

func TestFetchEventsForbiddenOrNotFoundGivesSharingHint(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusNotFound} {
		doer := &fakeDoer{status: status, body: "no"}
		_, err := fetchEvents(context.Background(), doer, "x@y.com", time.Now(), time.Now().Add(time.Hour), time.UTC)
		if err == nil || errors.Is(err, ErrNotConnected) {
			t.Errorf("status %d should be a descriptive error, got %v", status, err)
		}
		if !strings.Contains(err.Error(), "calendar_id") {
			t.Errorf("status %d error should mention calendar_id/sharing, got %v", status, err)
		}
	}
}

func TestFetchEventsOtherStatusErrors(t *testing.T) {
	doer := &fakeDoer{status: http.StatusInternalServerError, body: "boom"}
	if _, err := fetchEvents(context.Background(), doer, "", time.Now(), time.Now().Add(time.Hour), time.UTC); err == nil {
		t.Error("a 500 should surface an error")
	}
}

func TestFetchEventsDoerError(t *testing.T) {
	doer := &fakeDoer{err: errors.New("network down")}
	if _, err := fetchEvents(context.Background(), doer, "", time.Now(), time.Now().Add(time.Hour), time.UTC); err == nil {
		t.Error("a transport error should propagate")
	}
}

func TestFetchEventsNotConnectedWithoutCredentials(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config")) // empty: no credentials
	c := NewClient("", "", "")
	_, err := c.FetchEvents(context.Background(), time.Now(), time.Now().Add(time.Hour))
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("FetchEvents without credentials = %v, want ErrNotConnected", err)
	}
}

func TestFetchEventsUsesConfiguredCredentialsPath(t *testing.T) {
	// A configured (but absent) credentials path is honored and yields
	// ErrNotConnected, independent of the default config dir.
	dir := t.TempDir()
	c := NewClient(filepath.Join(dir, "mine.json"), filepath.Join(dir, "tok.json"), "")
	_, err := c.FetchEvents(context.Background(), time.Now(), time.Now().Add(time.Hour))
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("FetchEvents with a missing configured credentials path = %v, want ErrNotConnected", err)
	}
}

func TestAuthClientServiceAccount(t *testing.T) {
	// With a calendar_id, a service account yields a usable client (no network).
	c := NewClient("", "", "you@example.com")
	doer, err := c.authClient(context.Background(), []byte(sampleServiceAccount))
	if err != nil || doer == nil {
		t.Fatalf("service account authClient = (%v, %v), want a client", doer, err)
	}
}

func TestAuthClientServiceAccountNeedsCalendarID(t *testing.T) {
	c := NewClient("", "", "") // no calendar_id
	_, err := c.authClient(context.Background(), []byte(sampleServiceAccount))
	if err == nil || !strings.Contains(err.Error(), "calendar_id") {
		t.Errorf("service account without calendar_id should error about calendar_id, got %v", err)
	}
}

func TestAuthClientOAuthMissingTokenIsNotConnected(t *testing.T) {
	c := NewClient("", filepath.Join(t.TempDir(), "no-token.json"), "")
	_, err := c.authClient(context.Background(), []byte(sampleCredentials))
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("OAuth client with no cached token = %v, want ErrNotConnected", err)
	}
}
