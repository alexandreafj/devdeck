package github

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/alexandreafj/devdeck/internal/exec/exectest"
)

func TestFetchPRsBuildsGHSearchCommand(t *testing.T) {
	fake := &exectest.FakeRunner{
		Responses: []exectest.Response{{Out: []byte(sampleJSON)}},
	}
	client := NewClient(fake)

	prs, err := client.FetchPRs(context.Background(), ReviewRequested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("got %d PRs, want 2", len(prs))
	}

	call, ok := fake.LastCall()
	if !ok {
		t.Fatal("expected a recorded command, got none")
	}
	if call.Name != "gh" {
		t.Errorf("command = %q, want gh", call.Name)
	}
	for _, want := range []string{"search", "prs", "--state", "open", "--review-requested", "@me", "--json"} {
		if !slices.Contains(call.Args, want) {
			t.Errorf("args %v missing %q", call.Args, want)
		}
	}
}

func TestFetchPRsUsesModeFlag(t *testing.T) {
	cases := []struct {
		mode Mode
		flag string
	}{
		{Authored, "--author"},
		{ReviewRequested, "--review-requested"},
		{Assigned, "--assignee"},
	}
	for _, tc := range cases {
		t.Run(tc.mode.Label(), func(t *testing.T) {
			fake := &exectest.FakeRunner{Responses: []exectest.Response{{Out: []byte(`[]`)}}}
			if _, err := NewClient(fake).FetchPRs(context.Background(), tc.mode); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			call, _ := fake.LastCall()
			if !slices.Contains(call.Args, tc.flag) {
				t.Errorf("mode %v: args %v missing flag %q", tc.mode, call.Args, tc.flag)
			}
		})
	}
}

func TestFetchPRsPropagatesRunnerError(t *testing.T) {
	fake := &exectest.FakeRunner{
		RunFunc: func(context.Context, string, ...string) ([]byte, error) {
			return nil, errors.New("gh not authenticated")
		},
	}
	if _, err := NewClient(fake).FetchPRs(context.Background(), Authored); err == nil {
		t.Fatal("expected error to propagate from runner, got nil")
	}
}

func TestFetchPRsPropagatesDecodeError(t *testing.T) {
	fake := &exectest.FakeRunner{Responses: []exectest.Response{{Out: []byte(`not json`)}}}
	if _, err := NewClient(fake).FetchPRs(context.Background(), Authored); err == nil {
		t.Fatal("expected decode error, got nil")
	}
}
