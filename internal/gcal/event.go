// Package gcal fetches Google Calendar events for the current week and maps them
// onto display-ready values. Network access and OAuth live behind the
// EventProvider interface and thin, isolated helpers, so the parsing, grouping,
// and formatting logic is all pure and unit-testable without hitting Google.
package gcal

import (
	"encoding/json"
	"fmt"
	"time"
)

// Event is a single calendar event in display-ready form. AllDay events carry a
// date with no clock time. MeetLink is the video-conference URL when present;
// HTMLLink is the event's page on Google Calendar, used as a fallback target.
type Event struct {
	Summary  string
	Start    time.Time
	End      time.Time
	AllDay   bool
	Location string
	MeetLink string
	HTMLLink string
}

// Link returns the best URL to open for the event: the video meeting link if it
// has one, otherwise the Google Calendar event page.
func (e Event) Link() string {
	if e.MeetLink != "" {
		return e.MeetLink
	}
	return e.HTMLLink
}

// TimeLabel renders the event's time column: "All day" for all-day events, else
// the 24-hour start time (e.g. "09:00").
func (e Event) TimeLabel() string {
	if e.AllDay {
		return "All day"
	}
	return e.Start.Format("15:04")
}

// DaySection groups the events that fall on one calendar day under a header
// label ("Today", "Tomorrow", or "Mon Jan 2").
type DaySection struct {
	Label  string
	Events []Event

	// day is the section's midnight, used internally to coalesce consecutive
	// events that fall on the same calendar day.
	day time.Time
}

// apiResponse mirrors the subset of the Calendar API events.list payload we use.
type apiResponse struct {
	Items []apiEvent `json:"items"`
}

type apiEvent struct {
	Status         string         `json:"status"`
	Summary        string         `json:"summary"`
	Location       string         `json:"location"`
	HTMLLink       string         `json:"htmlLink"`
	HangoutLink    string         `json:"hangoutLink"`
	Start          apiTime        `json:"start"`
	End            apiTime        `json:"end"`
	ConferenceData *apiConference `json:"conferenceData"`
}

type apiTime struct {
	// DateTime is set (RFC 3339) for timed events; Date (YYYY-MM-DD) for all-day.
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
}

type apiConference struct {
	EntryPoints []apiEntryPoint `json:"entryPoints"`
}

type apiEntryPoint struct {
	EntryPointType string `json:"entryPointType"`
	URI            string `json:"uri"`
}

// DecodeEvents parses the JSON returned by the Calendar API events.list call and
// maps it onto []Event, resolving all times into loc. Cancelled events and
// events with no parseable start are skipped; the result preserves input order
// (the API is queried ordered by start time).
func DecodeEvents(data []byte, loc *time.Location) ([]Event, error) {
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	if loc == nil {
		loc = time.Local
	}

	events := make([]Event, 0, len(resp.Items))
	for _, it := range resp.Items {
		if it.Status == "cancelled" {
			continue
		}
		start, end, allDay, ok := parseSpan(it.Start, it.End, loc)
		if !ok {
			continue
		}
		summary := it.Summary
		if summary == "" {
			summary = "(no title)"
		}
		events = append(events, Event{
			Summary:  summary,
			Start:    start,
			End:      end,
			AllDay:   allDay,
			Location: it.Location,
			MeetLink: meetLink(it),
			HTMLLink: it.HTMLLink,
		})
	}
	return events, nil
}

// parseSpan resolves an event's start/end and whether it is all-day. It reports
// ok=false when neither a dateTime nor a date is present on the start.
func parseSpan(start, end apiTime, loc *time.Location) (s, e time.Time, allDay, ok bool) {
	switch {
	case start.DateTime != "":
		s, err := time.Parse(time.RFC3339, start.DateTime)
		if err != nil {
			return time.Time{}, time.Time{}, false, false
		}
		e := s
		if end.DateTime != "" {
			if parsed, err := time.Parse(time.RFC3339, end.DateTime); err == nil {
				e = parsed
			}
		}
		return s.In(loc), e.In(loc), false, true
	case start.Date != "":
		s, err := time.ParseInLocation("2006-01-02", start.Date, loc)
		if err != nil {
			return time.Time{}, time.Time{}, false, false
		}
		e := s
		if end.Date != "" {
			if parsed, err := time.ParseInLocation("2006-01-02", end.Date, loc); err == nil {
				e = parsed
			}
		}
		return s, e, true, true
	default:
		return time.Time{}, time.Time{}, false, false
	}
}

// meetLink extracts the video-conference URL: a conferenceData "video" entry
// point if present, otherwise the legacy hangoutLink.
func meetLink(it apiEvent) string {
	if it.ConferenceData != nil {
		for _, ep := range it.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" && ep.URI != "" {
				return ep.URI
			}
		}
	}
	return it.HangoutLink
}

// GroupByDay splits events (assumed ordered by start time) into per-day sections
// with human labels relative to now ("Today", "Tomorrow", else "Mon Jan 2").
func GroupByDay(events []Event, now time.Time) []DaySection {
	var sections []DaySection
	for _, ev := range events {
		day := startOfDay(ev.Start)
		if n := len(sections); n > 0 && sections[n-1].day.Equal(day) {
			sections[n-1].Events = append(sections[n-1].Events, ev)
			continue
		}
		sections = append(sections, DaySection{
			Label:  dayLabel(day, now),
			Events: []Event{ev},
			day:    day,
		})
	}
	return sections
}

// dayLabel names a day relative to now.
func dayLabel(day, now time.Time) string {
	today := startOfDay(now)
	switch {
	case day.Equal(today):
		return "Today"
	case day.Equal(today.AddDate(0, 0, 1)):
		return "Tomorrow"
	default:
		return day.Format("Mon Jan 2")
	}
}

// startOfDay truncates t to midnight in its own location.
func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// WeekWindow returns the [start, end) time range for "this week": from the start
// of today through the end of the current calendar week (Monday-based), i.e. the
// exclusive upper bound is next Monday at midnight.
func WeekWindow(now time.Time) (start, end time.Time) {
	start = startOfDay(now)
	iso := int(now.Weekday()) // Sunday=0 … Saturday=6
	if iso == 0 {
		iso = 7 // treat Sunday as the last day of the week
	}
	end = start.AddDate(0, 0, 8-iso)
	return start, end
}
