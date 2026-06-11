// Package exectest provides a fake exec.CommandRunner for use in unit tests.
// It records every call and lets a test script responses, so packages that
// shell out to `gh` can be exercised without a real process or network.
package exectest

import "context"

// Call captures a single invocation of FakeRunner.Run.
type Call struct {
	Name string
	Args []string
}

// FakeRunner is an exec.CommandRunner that records calls and returns scripted
// output. Set RunFunc for full control; otherwise it replays Responses in order.
type FakeRunner struct {
	// Calls records every invocation, in order, for assertions.
	Calls []Call
	// RunFunc, when set, fully handles each call (highest precedence).
	RunFunc func(ctx context.Context, name string, args ...string) ([]byte, error)
	// Responses are replayed in order when RunFunc is nil. Running past the end
	// returns empty output and no error.
	Responses []Response
}

// Response is one scripted result for FakeRunner.
type Response struct {
	Out []byte
	Err error
}

// Run records the call and returns the next scripted result.
func (f *FakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	f.Calls = append(f.Calls, Call{Name: name, Args: append([]string(nil), args...)})
	if f.RunFunc != nil {
		return f.RunFunc(ctx, name, args...)
	}
	idx := len(f.Calls) - 1
	if idx < len(f.Responses) {
		r := f.Responses[idx]
		return r.Out, r.Err
	}
	return nil, nil
}

// LastCall returns the most recent recorded call and whether one exists.
func (f *FakeRunner) LastCall() (Call, bool) {
	if len(f.Calls) == 0 {
		return Call{}, false
	}
	return f.Calls[len(f.Calls)-1], true
}
