package github

import (
	"context"
	"errors"
	"testing"

	"github.com/alexandreafj/devdeck/internal/exec/exectest"
)

func TestOpenPRRunsGHViewWeb(t *testing.T) {
	fake := &exectest.FakeRunner{}
	url := "https://github.com/acme/api/pull/12"

	if err := OpenPR(context.Background(), fake, url); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call, ok := fake.LastCall()
	if !ok {
		t.Fatal("expected a recorded command, got none")
	}
	want := exectest.Call{Name: "gh", Args: []string{"pr", "view", url, "--web"}}
	if call.Name != want.Name || len(call.Args) != len(want.Args) {
		t.Fatalf("call = %+v, want %+v", call, want)
	}
	for i := range want.Args {
		if call.Args[i] != want.Args[i] {
			t.Errorf("arg[%d] = %q, want %q", i, call.Args[i], want.Args[i])
		}
	}
}

func TestOpenPRPropagatesError(t *testing.T) {
	fake := &exectest.FakeRunner{
		RunFunc: func(context.Context, string, ...string) ([]byte, error) {
			return nil, errors.New("no browser")
		},
	}
	if err := OpenPR(context.Background(), fake, "https://x"); err == nil {
		t.Fatal("expected error to propagate, got nil")
	}
}
