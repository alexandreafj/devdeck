package exectest

import (
	"context"
	"errors"
	"testing"
)

func TestFakeRunnerReplaysResponsesInOrder(t *testing.T) {
	f := &FakeRunner{Responses: []Response{
		{Out: []byte("first")},
		{Out: []byte("second")},
	}}
	ctx := context.Background()

	out, _ := f.Run(ctx, "gh", "a")
	if string(out) != "first" {
		t.Errorf("call 1 = %q, want first", out)
	}
	out, _ = f.Run(ctx, "gh", "b")
	if string(out) != "second" {
		t.Errorf("call 2 = %q, want second", out)
	}
	// Past the end: empty output, no error.
	out, err := f.Run(ctx, "gh", "c")
	if out != nil || err != nil {
		t.Errorf("past end = (%q, %v), want (nil, nil)", out, err)
	}
}

func TestFakeRunnerRunFuncTakesPrecedence(t *testing.T) {
	want := errors.New("scripted")
	f := &FakeRunner{
		Responses: []Response{{Out: []byte("ignored")}},
		RunFunc: func(context.Context, string, ...string) ([]byte, error) {
			return nil, want
		},
	}
	if _, err := f.Run(context.Background(), "gh"); !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
}

func TestFakeRunnerRecordsCallsAndArgs(t *testing.T) {
	f := &FakeRunner{}
	args := []string{"search", "prs"}
	_, _ = f.Run(context.Background(), "gh", args...)

	// Mutating the caller's slice must not affect the recorded call.
	args[0] = "MUTATED"

	call, ok := f.LastCall()
	if !ok {
		t.Fatal("LastCall returned false after a call")
	}
	if call.Name != "gh" {
		t.Errorf("Name = %q, want gh", call.Name)
	}
	if len(call.Args) != 2 || call.Args[0] != "search" || call.Args[1] != "prs" {
		t.Errorf("Args = %v, want [search prs] (defensively copied)", call.Args)
	}
}

func TestFakeRunnerLastCallEmpty(t *testing.T) {
	f := &FakeRunner{}
	if _, ok := f.LastCall(); ok {
		t.Error("LastCall returned ok=true with no calls recorded")
	}
}
