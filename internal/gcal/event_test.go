package gcal

import (
	"testing"
	"time"
)

// fixedNow is a Friday (2026-06-12) so week-window and label tests are stable.
var fixedNow = time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)

func TestDecodeEventsTimedAndAllDay(t *testing.T) {
	data := []byte(`{"items":[
		{"status":"confirmed","summary":"Standup","htmlLink":"https://cal/std",
		 "start":{"dateTime":"2026-06-12T09:00:00Z"},"end":{"dateTime":"2026-06-12T09:30:00Z"},
		 "conferenceData":{"entryPoints":[{"entryPointType":"more","uri":"x"},{"entryPointType":"video","uri":"https://meet.google.com/abc"}]}},
		{"status":"confirmed","summary":"Offsite","htmlLink":"https://cal/off",
		 "start":{"date":"2026-06-13"},"end":{"date":"2026-06-14"}}
	]}`)

	events, err := DecodeEvents(data, time.UTC)
	if err != nil {
		t.Fatalf("DecodeEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	timed := events[0]
	if timed.AllDay {
		t.Error("first event should be timed, not all-day")
	}
	if timed.TimeLabel() != "09:00" {
		t.Errorf("TimeLabel = %q, want 09:00", timed.TimeLabel())
	}
	if timed.MeetLink != "https://meet.google.com/abc" {
		t.Errorf("MeetLink = %q, want the video entry point", timed.MeetLink)
	}
	if timed.Link() != "https://meet.google.com/abc" {
		t.Errorf("Link should prefer the meeting link, got %q", timed.Link())
	}

	allDay := events[1]
	if !allDay.AllDay {
		t.Error("second event should be all-day")
	}
	if allDay.TimeLabel() != "All day" {
		t.Errorf("TimeLabel = %q, want 'All day'", allDay.TimeLabel())
	}
	if allDay.Link() != "https://cal/off" {
		t.Errorf("Link should fall back to htmlLink, got %q", allDay.Link())
	}
}

func TestDecodeEventsSkipsCancelledAndDefaultsTitle(t *testing.T) {
	data := []byte(`{"items":[
		{"status":"cancelled","summary":"Dropped","start":{"dateTime":"2026-06-12T09:00:00Z"}},
		{"status":"confirmed","start":{"dateTime":"2026-06-12T11:00:00Z"},"hangoutLink":"https://meet.google.com/leg"}
	]}`)

	events, err := DecodeEvents(data, time.UTC)
	if err != nil {
		t.Fatalf("DecodeEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (cancelled skipped)", len(events))
	}
	if events[0].Summary != "(no title)" {
		t.Errorf("Summary = %q, want '(no title)'", events[0].Summary)
	}
	if events[0].MeetLink != "https://meet.google.com/leg" {
		t.Errorf("MeetLink = %q, want the hangoutLink fallback", events[0].MeetLink)
	}
}

func TestDecodeEventsConvertsToLocation(t *testing.T) {
	lisbon, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		t.Skipf("tz database unavailable: %v", err)
	}
	data := []byte(`{"items":[{"status":"confirmed","summary":"Call","start":{"dateTime":"2026-06-12T09:00:00Z"},"end":{"dateTime":"2026-06-12T09:30:00Z"}}]}`)
	events, err := DecodeEvents(data, lisbon)
	if err != nil {
		t.Fatalf("DecodeEvents: %v", err)
	}
	// 09:00Z is 10:00 in Lisbon (UTC+1 in June).
	if got := events[0].TimeLabel(); got != "10:00" {
		t.Errorf("TimeLabel in Lisbon = %q, want 10:00", got)
	}
}

func TestDecodeEventsInvalidJSON(t *testing.T) {
	if _, err := DecodeEvents([]byte("not json"), time.UTC); err == nil {
		t.Error("DecodeEvents should error on invalid JSON")
	}
}

func TestGroupByDayLabels(t *testing.T) {
	ev := func(day int, hour int) Event {
		return Event{
			Summary: "e",
			Start:   time.Date(2026, 6, day, hour, 0, 0, 0, time.UTC),
		}
	}
	events := []Event{
		ev(12, 9), ev(12, 14), // Today (two events, one section)
		ev(13, 10), // Tomorrow
		ev(14, 11), // Sun Jun 14
	}

	sections := GroupByDay(events, fixedNow)
	if len(sections) != 3 {
		t.Fatalf("got %d sections, want 3", len(sections))
	}
	if sections[0].Label != "Today" || len(sections[0].Events) != 2 {
		t.Errorf("section 0 = %q with %d events, want Today with 2", sections[0].Label, len(sections[0].Events))
	}
	if sections[1].Label != "Tomorrow" {
		t.Errorf("section 1 label = %q, want Tomorrow", sections[1].Label)
	}
	if sections[2].Label != "Sun Jun 14" {
		t.Errorf("section 2 label = %q, want 'Sun Jun 14'", sections[2].Label)
	}
}

func TestWeekWindow(t *testing.T) {
	// Friday 2026-06-12 → start today, end next Monday (2026-06-15) midnight.
	start, end := WeekWindow(fixedNow)
	wantStart := time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", start, wantStart)
	}
	if !end.Equal(wantEnd) {
		t.Errorf("end = %v, want %v", end, wantEnd)
	}
}

func TestWeekWindowOnSunday(t *testing.T) {
	sunday := time.Date(2026, 6, 14, 15, 0, 0, 0, time.UTC)
	start, end := WeekWindow(sunday)
	if !start.Equal(time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("start = %v, want Sunday midnight", start)
	}
	// On Sunday the window covers only that day → exclusive end is Monday.
	if !end.Equal(time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("end = %v, want next Monday midnight", end)
	}
}

func TestWeekWindowOnMonday(t *testing.T) {
	monday := time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
	start, end := WeekWindow(monday)
	if d := end.Sub(start); d != 7*24*time.Hour {
		t.Errorf("Monday window length = %v, want 7 days", d)
	}
}
