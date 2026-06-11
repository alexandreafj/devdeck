package github

import (
	"slices"
	"testing"
)

func TestParseMode(t *testing.T) {
	cases := map[string]struct {
		mode Mode
		ok   bool
	}{
		"authored":         {Authored, true},
		"review_requested": {ReviewRequested, true},
		"review-requested": {ReviewRequested, true},
		"assigned":         {Assigned, true},
		"ASSIGNED":         {Assigned, true}, // case-insensitive
		"nonsense":         {0, false},
		"":                 {0, false},
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			got, ok := ParseMode(in)
			if ok != want.ok || (ok && got != want.mode) {
				t.Errorf("ParseMode(%q) = (%v, %v), want (%v, %v)", in, got, ok, want.mode, want.ok)
			}
		})
	}
}

func TestParseModesSkipsUnknown(t *testing.T) {
	got := ParseModes([]string{"authored", "bogus", "assigned"})
	want := []Mode{Authored, Assigned}
	if !slices.Equal(got, want) {
		t.Errorf("ParseModes = %v, want %v", got, want)
	}
}

func TestParseModesEmptyReturnsNil(t *testing.T) {
	if got := ParseModes(nil); len(got) != 0 {
		t.Errorf("ParseModes(nil) = %v, want empty", got)
	}
}
