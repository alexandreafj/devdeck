// Package timeutil provides small, pure time-formatting helpers used across
// DevDeck widgets (for example, rendering "2w ago" next to a pull request).
package timeutil

import (
	"fmt"
	"time"
)

const (
	day   = 24 * time.Hour
	week  = 7 * day
	month = 30 * day
	year  = 365 * day
)

// RelativeTime renders the gap between t and now as a short human label such as
// "just now", "5m ago", "3h ago", "2d ago", "4w ago", "6mo ago" or "1y ago".
//
// It is pure: the caller supplies now, so the result is deterministic and easy
// to test. Times at or after now (clock skew, future events) render as
// "just now" rather than a negative duration.
func RelativeTime(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", d/time.Minute)
	case d < day:
		return fmt.Sprintf("%dh ago", d/time.Hour)
	case d < week:
		return fmt.Sprintf("%dd ago", d/day)
	case d < month:
		return fmt.Sprintf("%dw ago", d/week)
	case d < year:
		return fmt.Sprintf("%dmo ago", d/month)
	default:
		return fmt.Sprintf("%dy ago", d/year)
	}
}
