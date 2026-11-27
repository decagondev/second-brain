package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/second-brain/internal/core/ports"
)

type fakeModel struct {
	err   error
	calls int
}

func (m *fakeModel) Complete(context.Context, ports.ModelRequest) (ports.ModelResponse, error) {
	m.calls++
	if m.err != nil {
		return ports.ModelResponse{}, m.err
	}
	return ports.ModelResponse{Text: "ok"}, nil
}

// SB-5601, SB-5602: a failing primary goes to a gated fallback, never to an ungated one;
// three failures in five minutes switch to the fallback for ten, then the primary is tried
// again; with no model left the call fails and the state says down.
func TestFallbackOnlyWhenGated(t *testing.T) {
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	primary := &fakeModel{err: ErrUnavailable}
	second := &fakeModel{}
	gated := false
	f := NewFallback(primary, second, func() bool { return gated }, func() time.Time { return now })
	ctx, used := WithUsed(context.Background())
	if _, err := f.Complete(ctx, ports.ModelRequest{}); err == nil || second.calls != 0 || used.Fallback() || !f.State().Down {
		t.Fatalf("an ungated fallback answered: %v calls %d", err, second.calls)
	}
	gated = true
	if r, err := f.Complete(ctx, ports.ModelRequest{}); err != nil || r.Text != "ok" || !used.Fallback() || f.State().Down {
		t.Fatalf("gated fallback: %v %+v", err, f.State())
	}
	if _, err := f.Complete(ctx, ports.ModelRequest{}); err != nil || !f.State().Switched {
		t.Fatalf("three failures didn't switch: %+v", f.State())
	}
	before := primary.calls
	if _, err := f.Complete(ctx, ports.ModelRequest{}); err != nil || primary.calls != before {
		t.Fatal("the primary was tried while switched")
	}
	now = now.Add(SwitchedFor + time.Second)
	primary.err = nil
	if r, err := f.Complete(context.Background(), ports.ModelRequest{}); err != nil || r.Text != "ok" || primary.calls != before+1 || f.State().Switched {
		t.Fatalf("the primary wasn't tried again: %+v", f.State())
	}
	if _, err := (&Fallback{Primary: &fakeModel{err: ErrSecretInPrompt}, Secondary: second, Gated: func() bool { return true }}).Complete(context.Background(), ports.ModelRequest{}); !errors.Is(err, ErrSecretInPrompt) {
		t.Fatal("a refused secret went to the fallback")
	}
}
