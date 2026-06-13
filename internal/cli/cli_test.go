package cli

import (
	"strings"
	"testing"
)

func TestHelpTextListsCommands(t *testing.T) {
	h := HelpText()
	for _, want := range []string{
		"devdeck",
		"auth google",
		"--reset",
		"--help",
		"config.yml",
	} {
		if !strings.Contains(h, want) {
			t.Errorf("help text missing %q:\n%s", want, h)
		}
	}
}

func TestHelpTextHasNoEmDash(t *testing.T) {
	if strings.Contains(HelpText(), "—") {
		t.Error("help text must not contain em-dashes")
	}
}
