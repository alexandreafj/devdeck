package exec

import (
	"context"
	"strings"
	"testing"
)

func TestOSRunnerCapturesStdout(t *testing.T) {
	out, err := New().Run(context.Background(), "echo", "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "hello world" {
		t.Errorf("stdout = %q, want %q", got, "hello world")
	}
}

func TestOSRunnerReturnsErrorForMissingBinary(t *testing.T) {
	_, err := New().Run(context.Background(), "devdeck-nonexistent-binary-xyz")
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
}

func TestOSRunnerIncludesStderrOnFailure(t *testing.T) {
	_, err := New().Run(context.Background(), "sh", "-c", "echo boom >&2; exit 1")
	if err == nil {
		t.Fatal("expected error from failing command, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error %q should include stderr output %q", err.Error(), "boom")
	}
}
