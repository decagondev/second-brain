package laws

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"example.com/second-brain/internal/core/gateway"
	"example.com/second-brain/internal/core/ports"
)

// LAW-06, the fallback model (SB-5601; sb-laws(7)): only a model that passed the release's
// eval gate answers. Random runs of a failing or working primary and a gate that opens and
// closes, through the real fallback: the second model is called only while the gate is open,
// and an answer it writes says so.

// FallbackModel is what the law checks: a model over a primary and a second one, given the
// gate.
type FallbackModel func(primary, second ports.Model, gated func() bool, now func() time.Time) ports.Model

// RealFallback is the real one.
var RealFallback FallbackModel = func(p, s ports.Model, g func() bool, now func() time.Time) ports.Model {
	return gateway.NewFallback(p, s, g, now)
}

type lawModel struct {
	fail  *bool
	calls int
}

func (m *lawModel) Complete(context.Context, ports.ModelRequest) (ports.ModelResponse, error) {
	m.calls++
	if m.fail != nil && *m.fail {
		return ports.ModelResponse{}, gateway.ErrUnavailable
	}
	return ports.ModelResponse{Text: "ok"}, nil
}

func law06Fallback(r *rand.Rand, mk FallbackModel) error {
	now := time.Date(2026, 11, 27, 9, 0, 0, 0, time.UTC)
	fail, gated := false, false
	primary, second := &lawModel{fail: &fail}, &lawModel{}
	m := mk(primary, second, func() bool { return gated }, func() time.Time { return now })
	for i := 0; i < 40; i++ {
		fail, gated = r.Intn(2) == 0, r.Intn(3) == 0
		now = now.Add(time.Duration(r.Intn(300)) * time.Second)
		before := second.calls
		ctx, used := gateway.WithUsed(context.Background())
		_, err := m.Complete(ctx, ports.ModelRequest{})
		if second.calls > before && !gated {
			return fmt.Errorf("call %d: an ungated fallback answered", i)
		}
		if err == nil && second.calls > before && !used.Fallback() {
			return fmt.Errorf("call %d: the fallback answered without saying so", i)
		}
	}
	return nil
}
