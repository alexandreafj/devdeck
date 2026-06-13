package browser

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/alexandreafj/devdeck/internal/exec/exectest"
)

// wantCommand mirrors command()'s mapping so the assertions hold on whatever OS
// the test runs on (macOS locally, Linux in CI).
func wantCommand(url string) (string, []string) {
	switch runtime.GOOS {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		return "open", []string{url}
	default:
		return "xdg-open", []string{url}
	}
}

func TestOpenShellsOutWithURL(t *testing.T) {
	runner := &exectest.FakeRunner{}
	const url = "https://meet.google.com/abc-defg-hij"

	if err := Open(context.Background(), runner, url); err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	call, ok := runner.LastCall()
	if !ok {
		t.Fatal("Open should shell out to a browser command")
	}
	wantName, wantArgs := wantCommand(url)
	if call.Name != wantName {
		t.Errorf("command name = %q, want %q", call.Name, wantName)
	}
	if len(call.Args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", call.Args, wantArgs)
	}
	for i := range wantArgs {
		if call.Args[i] != wantArgs[i] {
			t.Errorf("arg[%d] = %q, want %q", i, call.Args[i], wantArgs[i])
		}
	}
	// The URL must always be passed through to the opener.
	if call.Args[len(call.Args)-1] != url {
		t.Errorf("last arg = %q, want the URL %q", call.Args[len(call.Args)-1], url)
	}
}

func TestOpenPropagatesRunnerError(t *testing.T) {
	runner := &exectest.FakeRunner{Responses: []exectest.Response{{Err: errors.New("boom")}}}
	if err := Open(context.Background(), runner, "https://example.com"); err == nil {
		t.Error("Open should surface the runner's error")
	}
}
