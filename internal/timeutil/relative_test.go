package timeutil

import (
	"testing"
	"time"
)

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		when time.Time
		want string
	}{
		{"just now (same instant)", now, "just now"},
		{"30 seconds ago", now.Add(-30 * time.Second), "just now"},
		{"1 minute ago", now.Add(-1 * time.Minute), "1m ago"},
		{"59 minutes ago", now.Add(-59 * time.Minute), "59m ago"},
		{"1 hour ago", now.Add(-1 * time.Hour), "1h ago"},
		{"23 hours ago", now.Add(-23 * time.Hour), "23h ago"},
		{"1 day ago", now.Add(-24 * time.Hour), "1d ago"},
		{"6 days ago", now.Add(-6 * 24 * time.Hour), "6d ago"},
		{"1 week ago", now.Add(-7 * 24 * time.Hour), "1w ago"},
		{"3 weeks ago", now.Add(-21 * 24 * time.Hour), "3w ago"},
		{"5 weeks ago becomes months", now.Add(-35 * 24 * time.Hour), "1mo ago"},
		{"11 months ago", now.Add(-330 * 24 * time.Hour), "11mo ago"},
		{"1 year ago", now.Add(-365 * 24 * time.Hour), "1y ago"},
		{"2 years ago", now.Add(-2 * 365 * 24 * time.Hour), "2y ago"},
		{"future time clamps to just now", now.Add(1 * time.Hour), "just now"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RelativeTime(tc.when, now)
			if got != tc.want {
				t.Errorf("RelativeTime(%v, %v) = %q, want %q", tc.when, now, got, tc.want)
			}
		})
	}
}
