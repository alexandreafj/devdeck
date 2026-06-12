package gcal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ErrNotConnected signals that the user has not yet authorized Google Calendar
// access (missing credentials or token). The widget renders a connect prompt
// rather than an error when it sees this.
var ErrNotConnected = errors.New("google calendar not connected")

// eventsEndpoint is the Calendar API events.list URL for the primary calendar.
const eventsEndpoint = "https://www.googleapis.com/calendar/v3/calendars/primary/events"

// EventProvider fetches calendar events in the [start, end) window. The widget
// depends on this interface (not GCalClient) so tests can supply a fake.
type EventProvider interface {
	FetchEvents(ctx context.Context, start, end time.Time) ([]Event, error)
}

// Client is the production EventProvider. It loads the cached OAuth token,
// calls the Calendar API events.list endpoint, and decodes the result. Times are
// resolved into loc for grouping and display.
type Client struct {
	loc *time.Location
}

// NewClient returns a Client that resolves event times in the local timezone
// (matching the dashboard's clock).
func NewClient() *Client {
	return &Client{loc: time.Local}
}

// FetchEvents returns this week's events. It returns ErrNotConnected when the
// credentials or token are missing, so the widget can prompt the user to run
// `devdeck auth google`.
func (c *Client) FetchEvents(ctx context.Context, start, end time.Time) ([]Event, error) {
	credsPath, err := CredentialsPath()
	if err != nil {
		return nil, err
	}
	tokenPath, err := TokenPath()
	if err != nil {
		return nil, err
	}

	cfg, err := LoadCredentials(credsPath)
	if err != nil {
		return nil, ErrNotConnected
	}
	tok, err := LoadToken(tokenPath)
	if err != nil {
		return nil, ErrNotConnected
	}

	client := cfg.Client(ctx, tok)
	return fetchEvents(ctx, client, start, end, c.loc)
}

// httpDoer is the minimal HTTP surface fetchEvents needs, satisfied by the
// oauth2 *http.Client and by fakes in tests.
type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// fetchEvents performs the events.list request and decodes the body. It is the
// one network-touching helper; it is kept tiny and driven through httpDoer so
// the request building and decoding remain testable.
func fetchEvents(ctx context.Context, doer httpDoer, start, end time.Time, loc *time.Location) ([]Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eventsURL(start, end), nil)
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
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrNotConnected
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendar API: %s", resp.Status)
	}
	return DecodeEvents(body, loc)
}

// eventsURL builds the events.list query for a single-expanded, start-ordered
// window. It is pure so the query parameters can be asserted in tests.
func eventsURL(start, end time.Time) string {
	q := url.Values{}
	q.Set("timeMin", start.Format(time.RFC3339))
	q.Set("timeMax", end.Format(time.RFC3339))
	q.Set("singleEvents", "true")
	q.Set("orderBy", "startTime")
	q.Set("maxResults", "50")
	return eventsEndpoint + "?" + q.Encode()
}

// compile-time check that the production client satisfies the interface.
var _ EventProvider = (*Client)(nil)
