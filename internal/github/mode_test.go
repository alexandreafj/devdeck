package github

import "testing"

func TestModeLabels(t *testing.T) {
	cases := map[Mode]string{
		Authored:        "Authored",
		ReviewRequested: "Review requested",
		Assigned:        "Assigned",
		Mode(99):        "Unknown",
	}
	for mode, want := range cases {
		if got := mode.Label(); got != want {
			t.Errorf("Mode(%d).Label() = %q, want %q", mode, got, want)
		}
	}
}

func TestModeInvalidMetaAndFlag(t *testing.T) {
	invalid := Mode(99)
	if got := invalid.meta(); got != "unknown" {
		t.Errorf("meta() = %q, want unknown", got)
	}
	if got := invalid.searchFlag(); got != "" {
		t.Errorf("searchFlag() = %q, want empty", got)
	}
}

func TestModesCanonicalOrder(t *testing.T) {
	if len(Modes) != 3 || Modes[0] != Authored || Modes[1] != ReviewRequested || Modes[2] != Assigned {
		t.Errorf("Modes = %v, want [Authored ReviewRequested Assigned]", Modes)
	}
}
