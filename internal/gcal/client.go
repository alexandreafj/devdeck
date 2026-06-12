package gcal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/oauth2/google"
)

// ErrNotConnected signals that the user has not yet authorized Google Calendar
// access (missing OAuth credentials or token). The widget renders a connect
// prompt rather than an error when it sees this.
var ErrNotConnected = errors.New("google calendar not connected")

// calendarsBase is the Calendar API calendars collection; a calendar ID and
// "/events" are appended to reach events.list.
const calendarsBase = "https://www.googleapis.com/calendar/v3/calendars/"

// EventProvider fetches calendar events in the [start, end) window. The widget
// depends on this interface (not GCalClient) so tests can supply a fake.
type EventProvider interface {
	FetchEvents(ctx context.Context, start, end time.Time) ([]Event, error)
}

// Client is the production EventProvider. It loads the user's own Google
// credentials (an OAuth "Desktop" client or a service account, auto-detected),
// calls the Calendar API events.list endpoint, and decodes the result. Times are
// resolved into loc for grouping and display.
type Client struct {
	loc *time.Location
	// credentialsPath/tokenPath override the default config-dir locations so a
	// user can supply their own credentials JSON. Empty means use the defaults.
	credentialsPath string
	tokenPath       string
	// calendarID selects which calendar to read. Empty means "primary" (OAuth).
	calendarID string
}

// NewClient returns a Client that resolves event times in the local timezone
// (matching the dashboard's clock). credentialsPath, tokenPath, and calendarID
// may be empty to use the defaults (config-dir paths; the "primary" calendar).
func NewClient(credentialsPath, tokenPath, calendarID string) *Client {
	return &Client{
		loc:             time.Local,
		credentialsPath: credentialsPath,
		tokenPath:       tokenPath,
		calendarID:      calendarID,
	}
}

// FetchEvents returns this week's events. It returns ErrNotConnected when the
// credentials/token are missing (OAuth), so the widget can prompt the user to
// run `devdeck auth google`.
func (c *Client) FetchEvents(ctx context.Context, start, end time.Time) ([]Event, error) {
	credsPath, err := resolvePath(c.credentialsPath, CredentialsPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(credsPath)
	if err != nil {
		return nil, ErrNotConnected
	}

	client, err := c.authClient(ctx, data)
	if err != nil {
		return nil, err
	}
	return fetchEvents(ctx, client, c.calendarID, start, end, c.loc)
}

// authClient builds the authenticated HTTP client for the credential type found
// in data: a service account uses JWT auth (no browser); an OAuth "Desktop"
// client uses the cached token (ErrNotConnected if it is missing).
func (c *Client) authClient(ctx context.Context, data []byte) (httpDoer, error) {
	switch credentialKind(data) {
	case credServiceAccount:
		if c.calendarID == "" {
			return nil, fmt.Errorf("service account needs calendar_id set to the calendar's address (shared with the service account)")
		}
		jwtCfg, err := google.JWTConfigFromJSON(data, calendarScope)
		if err != nil {
			return nil, fmt.Errorf("invalid service account credentials: %w", err)
		}
		return jwtCfg.Client(ctx), nil
	case credOAuthDesktop:
		cfg, err := google.ConfigFromJSON(data, calendarScope)
		if err != nil {
			return nil, ErrNotConnected
		}
		tokenPath, err := resolvePath(c.tokenPath, TokenPath)
		if err != nil {
			return nil, err
		}
		tok, err := LoadToken(tokenPath)
		if err != nil {
			return nil, ErrNotConnected
		}
		return cfg.Client(ctx, tok), nil
	default:
		return nil, ErrNotConnected
	}
}

// httpDoer is the minimal HTTP surface fetchEvents needs, satisfied by the
// oauth2 *http.Client and by fakes in tests.
type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// fetchEvents performs the events.list request and decodes the body. It is the
// one network-touching helper; it is kept tiny and driven through httpDoer so
// the request building and decoding remain testable.
func fetchEvents(ctx context.Context, doer httpDoer, calendarID string, start, end time.Time, loc *time.Location) ([]Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eventsURL(calendarID, start, end), nil)
	if err != nil {
		return nil, err
	}
	resp, err := doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, ErrNotConnected
	case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("calendar not accessible (%s) — check calendar_id and that the calendar is shared with the service account", resp.Status)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("calendar API: %s", resp.Status)
	}
	return DecodeEvents(body, loc)
}

// resolvePath returns p when set, otherwise the default produced by fn.
func resolvePath(p string, fn func() (string, error)) (string, error) {
	if p != "" {
		return p, nil
	}
	return fn()
}

// eventsURL builds the events.list query for a single-expanded, start-ordered
// window on the given calendar (default "primary"). It is pure so the URL can be
// asserted in tests.
func eventsURL(calendarID string, start, end time.Time) string {
	if calendarID == "" {
		calendarID = "primary"
	}
	q := url.Values{}
	q.Set("timeMin", start.Format(time.RFC3339))
	q.Set("timeMax", end.Format(time.RFC3339))
	q.Set("singleEvents", "true")
	q.Set("orderBy", "startTime")
	q.Set("maxResults", "50")
	// QueryEscape encodes "@" as %40 (PathEscape leaves it), matching Google's
	// convention for calendar IDs like "you@gmail.com".
	return calendarsBase + url.QueryEscape(calendarID) + "/events?" + q.Encode()
}

// compile-time check that the production client satisfies the interface.
var _ EventProvider = (*Client)(nil)
