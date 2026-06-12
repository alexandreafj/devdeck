package fuzzy

import "testing"

func TestMatch(t *testing.T) {
	cases := []struct {
		name   string
		query  string
		target string
		want   bool
	}{
		{"empty query matches anything", "", "acme/api", true},
		{"empty query matches empty target", "", "", true},
		{"exact substring", "api", "acme/api", true},
		{"subsequence with gaps", "amapi", "acme/api", true},
		{"case-insensitive query", "ACME", "acme/api", true},
		{"case-insensitive target", "page", "Add AB Page", true},
		{"order matters", "ipa", "acme/api", false},
		{"missing rune", "xyz", "acme/api", false},
		{"longer than target", "acme/apiserver", "acme/api", false},
		{"non-empty query, empty target", "a", "", false},
		{"unicode", "café", "Le Café Périgord", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Match(c.query, c.target); got != c.want {
				t.Errorf("Match(%q, %q) = %v, want %v", c.query, c.target, got, c.want)
			}
		})
	}
}
