package github

import "strings"

// ParseMode converts a config string ("authored", "review_requested" /
// "review-requested", "assigned") into a Mode. Matching is case-insensitive.
// The second return is false for unknown values.
func ParseMode(s string) (Mode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "authored":
		return Authored, true
	case "review_requested", "review-requested":
		return ReviewRequested, true
	case "assigned":
		return Assigned, true
	default:
		return 0, false
	}
}

// ParseModes converts config strings into Modes, silently skipping any it does
// not recognize. Callers that need a non-empty set (e.g. the widget) supply
// their own default when the result is empty.
func ParseModes(ss []string) []Mode {
	var modes []Mode
	for _, s := range ss {
		if m, ok := ParseMode(s); ok {
			modes = append(modes, m)
		}
	}
	return modes
}
